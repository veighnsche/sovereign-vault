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

# TEAM_037: Tailscale REMOVED from SQL VM
# Forge and Vault connect via TAP network (192.168.100.2:5432), not Tailscale.
# This simplifies the SQL VM and removes unnecessary complexity.
# If external Tailnet access to PostgreSQL is needed later, re-enable this section.
echo "=== Tailscale Disabled (not needed for SQL) ==="
echo "Forge/Vault connect via TAP: 192.168.100.2:5432"

# TEAM_037: Sync time via NTP directly (no Tailscale needed)
if command -v ntpd >/dev/null 2>&1; then
    ntpd -d -q -n -p pool.ntp.org 2>&1 || true
fi

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

# TEAM_035: Read database passwords from kernel cmdline (centralized in .env)
FORGEJO_DB_PASS=""
VAULTWARDEN_DB_PASS=""
for param in $(cat /proc/cmdline); do
    case "$param" in
        forgejo.db_password=*) FORGEJO_DB_PASS="${param#forgejo.db_password=}" ;;
        vaultwarden.db_password=*) VAULTWARDEN_DB_PASS="${param#vaultwarden.db_password=}" ;;
    esac
done

# TEAM_029: Create forgejo database user for Forgejo VM
if [ -n "$FORGEJO_DB_PASS" ]; then
    su postgres -c "psql -c \"CREATE USER forgejo WITH PASSWORD '$FORGEJO_DB_PASS';\"" 2>&1 || true
    echo "Created forgejo user with password from .env"
else
    su postgres -c "psql -c \"CREATE USER forgejo WITH PASSWORD 'forgejo';\"" 2>&1 || true
    echo "WARNING: No forgejo.db_password in cmdline, using default (insecure)"
fi
su postgres -c "psql -c \"CREATE DATABASE forgejo OWNER forgejo;\"" 2>&1 || true
su postgres -c "psql -c \"GRANT ALL PRIVILEGES ON DATABASE forgejo TO forgejo;\"" 2>&1 || true

# TEAM_035: Create vaultwarden database user for Vaultwarden VM
if [ -n "$VAULTWARDEN_DB_PASS" ]; then
    su postgres -c "psql -c \"CREATE USER vaultwarden WITH PASSWORD '$VAULTWARDEN_DB_PASS';\"" 2>&1 || true
    echo "Created vaultwarden user with password from .env"
else
    su postgres -c "psql -c \"CREATE USER vaultwarden WITH PASSWORD 'vaultwarden';\"" 2>&1 || true
    echo "WARNING: No vaultwarden.db_password in cmdline, using default (insecure)"
fi
su postgres -c "psql -c \"CREATE DATABASE vaultwarden OWNER vaultwarden;\"" 2>&1 || true
su postgres -c "psql -c \"GRANT ALL PRIVILEGES ON DATABASE vaultwarden TO vaultwarden;\"" 2>&1 || true

echo "PostgreSQL version:"
su postgres -c "psql -c \"SELECT version();\"" 2>&1

# TEAM_034: Removed tailscale serve - not needed!
# Direct port binding works: PostgreSQL listens on 0.0.0.0, Tailscale routes inbound
# traffic directly to tailscale0 interface. See TAILSCALE_AVF_LIMITATIONS.md

# CRITICAL: These messages are monitored by the host to detect successful boot
log "PostgreSQL started"
log "=== INIT COMPLETE ==="

# TEAM_037: Supervision loop for PostgreSQL only (Tailscale removed)
# If PostgreSQL dies (OOM, crash, anything), restart it automatically.
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
    
    sleep 30
done
