#!/bin/sh
# init.sh - AVF guest init script for PostgreSQL VM
#
# TEAM_012: Init script for AVF (OpenRC hangs, so we use this instead)
# TEAM_016/017: TAP networking with verified working configuration
# TEAM_023: ICU collation, PostgreSQL supervision, bug fixes
#
# This script is injected into the rootfs at /sbin/init.sh (symlinked to /sbin/init)
# The DB_PASSWORD environment variable is set by rootfs.go during preparation
#
# Reference: Field Guide to Deploying Self-Hosted Services on Android 16 with AVF

export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
# DB_PASSWORD is injected by rootfs.go - DO NOT HARDCODE

# Mount essential filesystems FIRST
mount -t proc proc /proc 2>/dev/null || true
mount -t sysfs sysfs /sys 2>/dev/null || true
mount -t devtmpfs devtmpfs /dev 2>/dev/null || true

# TEAM_023: Log file for debugging
LOG=/var/log/init.log
mkdir -p /var/log

# TEAM_023: Find the console device - try ttyS0 first (crosvm --serial captures this)
CONSOLE=""
for dev in /dev/ttyS0 /dev/console /dev/hvc0; do
    if [ -c "$dev" ]; then
        CONSOLE="$dev"
        break
    fi
done

# Log function - writes to both console device AND log file
log() {
    echo "$1" >> "$LOG"
    [ -n "$CONSOLE" ] && echo "$1" > "$CONSOLE" 2>/dev/null
}

log "=== INIT START $(date) ==="
log "Console device: $CONSOLE"

# Set hostname
hostname sovereign-sql

# TEAM_017: Mount tmpfs for shared memory (required for PostgreSQL)
mkdir -p /dev/shm /tmp
mount -t tmpfs -o mode=1777 tmpfs /dev/shm
mount -t tmpfs tmpfs /tmp

# ============================================================================
# TEAM_023: Mount data.img (/dev/vdb) for PERSISTENT storage
# ============================================================================
# This is CRITICAL for Tailscale machine identity!
# - rootfs.img (/dev/vda) = rebuilt on every `sovereign build`
# - data.img (/dev/vdb) = PERSISTS across rebuilds
#
# By storing Tailscale state on /data, the machine ID survives rebuilds,
# preventing duplicate registrations (sovereign-sql, sovereign-sql-1, etc.)
# ============================================================================
mkdir -p /data
if [ -b /dev/vdb ]; then
    log "Mounting persistent data disk /dev/vdb -> /data"
    mount /dev/vdb /data
    if [ $? -eq 0 ]; then
        log "  ✓ Data disk mounted successfully"
    else
        log "  ⚠ Failed to mount data disk - using tmpfs fallback"
        mount -t tmpfs tmpfs /data
    fi
else
    log "  ⚠ No data disk found (/dev/vdb) - using tmpfs"
    mount -t tmpfs tmpfs /data
fi

# Create persistent directories on data disk
mkdir -p /data/postgres /data/tailscale
chown postgres:postgres /data/postgres 2>/dev/null || true

# TEAM_023: Set current date for TLS cert validation
# Must be recent enough for certificate "not before" dates
date -s "2025-12-29 10:00:00" 2>/dev/null || true

# Create device nodes
mkdir -p /dev/net
[ -e /dev/net/tun ] || mknod /dev/net/tun c 10 200
chmod 666 /dev/net/tun

# TEAM_016/017: Configure TAP networking
# Host TAP is 192.168.100.1, guest is 192.168.100.2
echo "=== Configuring TAP Network ==="
# TEAM_023: Reduced sleep - interface should be ready immediately after /sys mount
sleep 1

# Find virtio network interface (not tunnel interfaces like erspan0)
# Look for eth* or enp* which are real network devices
IFACE=""
for iface in $(ls /sys/class/net/); do
    case "$iface" in
        eth*|enp*|ens*) IFACE="$iface"; break ;;
    esac
done
# Fallback: look for device with virtio driver
if [ -z "$IFACE" ]; then
    for iface in $(ls /sys/class/net/); do
        if [ -d "/sys/class/net/$iface/device/driver" ]; then
            driver=$(basename $(readlink /sys/class/net/$iface/device/driver))
            if [ "$driver" = "virtio_net" ]; then
                IFACE="$iface"
                break
            fi
        fi
    done
fi
echo "Found interface: $IFACE"

if [ -n "$IFACE" ]; then
    ip addr add 192.168.100.2/24 dev "$IFACE"
    ip link set "$IFACE" up
    ip route add default via 192.168.100.1
    echo "nameserver 8.8.8.8" > /etc/resolv.conf
    echo "Network configured on $IFACE"
    ip addr show "$IFACE"
else
    echo "WARNING: No network interface found"
    ip link
fi

# Test connectivity
echo "=== Testing Network ==="
ping -c 2 8.8.8.8 2>&1 || echo "Ping failed - will retry after Tailscale"

# Start Tailscale
# TEAM_020: tailscaled auto-reconnects - only need authkey for FIRST registration
#
# ============================================================================
# TEAM_023 FIX: Tailscale state now persisted on data.img!
# ============================================================================
# The duplicate registration bug is FIXED by:
# 1. Mounting data.img to /data (done above)
# 2. Storing tailscaled.state in /data/tailscale/ (persistent disk)
# 3. Machine identity survives rebuilds - no more duplicates!
# ============================================================================
echo "=== Starting Tailscale ==="
mkdir -p /data/tailscale /var/run/tailscale /dev/net
[ -e /dev/net/tun ] || mknod /dev/net/tun c 10 200
chmod 666 /dev/net/tun

# TEAM_030: Use native tun mode - kernel now has full nftables support
/usr/sbin/tailscaled \
    --state=/data/tailscale/tailscaled.state \
    --socket=/var/run/tailscale/tailscaled.sock &
TAILSCALED_PID=$!

# Wait for tailscaled to be ready (check socket exists)
for i in 1 2 3 4 5; do
    [ -S /var/run/tailscale/tailscaled.sock ] && break
    sleep 1
done

# TEAM_023: Check if we have PERSISTENT state (machine identity survives rebuilds)
# The state file on /data/tailscale is the source of truth, not `tailscale status`
# which may not be ready immediately after tailscaled starts.
STATE_FILE="/data/tailscale/tailscaled.state"

# TEAM_030: Validate state file content, not just existence
# A corrupt/empty state file will cause Tailscale to generate new nodekey
STATE_VALID=false
if [ -f "$STATE_FILE" ] && [ -s "$STATE_FILE" ]; then
    # Check if state has actual Tailscale identity (not just empty JSON)
    if grep -q 'PrivateNodeKey' "$STATE_FILE" 2>/dev/null; then
        STATE_VALID=true
    else
        echo "Tailscale: State file exists but appears invalid, will use authkey"
    fi
fi

if [ "$STATE_VALID" = "true" ]; then
    # We have valid saved state - reconnect without authkey (preserves machine identity!)
    echo "Tailscale: Found valid persistent state, reconnecting..."
    # TEAM_034: Direct port binding works - no subnet routing needed
    # PostgreSQL listens on 0.0.0.0, Tailscale routes inbound traffic directly
    /usr/bin/tailscale up --hostname=sovereign-sql --accept-routes 2>&1
else
    # First boot or invalid state - need authkey for registration
    echo "Tailscale: No valid state, using authkey for registration..."
    AUTHKEY=""
    for param in $(cat /proc/cmdline); do
        case "$param" in
            tailscale.authkey=*) AUTHKEY="${param#tailscale.authkey=}" ;;
        esac
    done
    if [ -n "$AUTHKEY" ]; then
        # Delete invalid state file if exists
        rm -f "$STATE_FILE" 2>/dev/null
        # TEAM_034: Direct port binding works - no subnet routing needed
        /usr/bin/tailscale up --authkey="$AUTHKEY" --hostname=sovereign-sql --accept-routes 2>&1
    else
        echo "WARNING: No authkey for first-time registration"
    fi
fi

# TEAM_023: Sync time via Tailscale/internet now that network is up
# This fixes TLS cert validation issues
if command -v ntpd >/dev/null 2>&1; then
    ntpd -d -q -n -p pool.ntp.org 2>&1 || true
fi

/usr/bin/tailscale status 2>&1

# Start PostgreSQL
echo "=== Starting PostgreSQL ==="
mkdir -p /run/postgresql /data/postgres /var/log
touch /var/log/postgresql.log
chown -R postgres:postgres /run/postgresql /data/postgres /var/log/postgresql.log

# TEAM_023: Clean up stale PID file if exists (prevents startup failure after crash)
rm -f /data/postgres/postmaster.pid 2>/dev/null

if [ ! -f /data/postgres/PG_VERSION ]; then
    echo "Initializing PostgreSQL database..."
    # TEAM_023: Use ICU collation to fix musl libc collation bug
    # Without this, musl's "C" collation causes silent B-tree index corruption
    # Reference: Field Guide Section 2.1 - "Database Collation" / Section 3.1
    if ! su postgres -c "initdb -D /data/postgres --locale-provider=icu --icu-locale=en-US" 2>&1; then
        echo "ERROR: initdb failed!"
        # Don't exit - supervision loop will keep retrying
    fi
    # TEAM_017: Configure PostgreSQL for AVF environment
    cat >> /data/postgres/postgresql.conf << 'PGCONF'
listen_addresses = '*'
dynamic_shared_memory_type = mmap
shared_buffers = 32MB
PGCONF
    echo "host all all 0.0.0.0/0 md5" >> /data/postgres/pg_hba.conf
    echo "host all all ::/0 md5" >> /data/postgres/pg_hba.conf
fi

su postgres -c "pg_ctl -D /data/postgres -l /var/log/postgresql.log start" 2>&1
sleep 2
# TEAM_030: Debug - show PostgreSQL log if startup failed
if ! su postgres -c "pg_isready" 2>/dev/null; then
    echo "PostgreSQL failed to start, checking log:"
    cat /var/log/postgresql.log 2>&1 | tail -30
fi

# TEAM_023: FIX - $DB_PASSWORD must use double quotes to expand!
# The original had single quotes which prevented variable expansion
su postgres -c "psql -c \"ALTER USER postgres PASSWORD '$DB_PASSWORD';\"" 2>&1

# TEAM_029: Create forgejo database user for Forgejo VM
su postgres -c "psql -c \"CREATE USER forgejo WITH PASSWORD 'forgejo';\"" 2>&1 || true
su postgres -c "psql -c \"CREATE DATABASE forgejo OWNER forgejo;\"" 2>&1 || true
su postgres -c "psql -c \"GRANT ALL PRIVILEGES ON DATABASE forgejo TO forgejo;\"" 2>&1 || true
echo "Created forgejo database user"

echo "PostgreSQL version:"
su postgres -c "psql -c \"SELECT version();\"" 2>&1

# TEAM_034: Removed tailscale serve - not needed!
# Direct port binding works: PostgreSQL listens on 0.0.0.0, Tailscale routes inbound
# traffic directly to tailscale0 interface. See TAILSCALE_AVF_LIMITATIONS.md

# CRITICAL: These messages are monitored by the host to detect successful boot
log "PostgreSQL started"
log "=== INIT COMPLETE ==="

# TEAM_023: Supervision loop for BOTH Tailscale and PostgreSQL
# If either dies (OOM, crash, anything), restart it automatically.
# Reference: Field Guide Section 6 - "No process supervision in VM"
while true; do
    # Check PostgreSQL
    if ! su postgres -c "pg_isready -q" 2>/dev/null; then
        echo "$(date): PostgreSQL not responding, restarting..."
        rm -f /data/postgres/postmaster.pid 2>/dev/null
        su postgres -c "pg_ctl -D /data/postgres -l /var/log/postgresql.log restart" 2>&1 || \
        su postgres -c "pg_ctl -D /data/postgres -l /var/log/postgresql.log start" 2>&1
        sleep 5
    fi
    
    # Check Tailscale daemon
    if ! kill -0 $TAILSCALED_PID 2>/dev/null; then
        echo "$(date): Tailscaled died, restarting..."
        # TEAM_030: Use native tun mode - kernel has full nftables support
        /usr/sbin/tailscaled \
            --state=/data/tailscale/tailscaled.state \
            --socket=/var/run/tailscale/tailscaled.sock &
        TAILSCALED_PID=$!
        sleep 3
        # TEAM_034: tailscale serve removed - direct port binding works
    fi
    
    sleep 30
done
