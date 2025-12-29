#!/bin/sh
# init.sh - AVF guest init script for Forgejo VM
#
# TEAM_025: Init script for AVF (OpenRC hangs, so we use this instead)
# Modeled after vm/sql/init.sh - the gold standard for AVF VMs
#
# This script is injected into the rootfs at /sbin/init.sh (symlinked to /sbin/init)
#
# Reference: Field Guide to Deploying Self-Hosted Services on Android 16 with AVF

export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

# Mount essential filesystems FIRST
mount -t proc proc /proc 2>/dev/null || true
mount -t sysfs sysfs /sys 2>/dev/null || true
mount -t devtmpfs devtmpfs /dev 2>/dev/null || true

# TEAM_025: Log file for debugging
LOG=/var/log/init.log
mkdir -p /var/log

# TEAM_025: Find the console device - try ttyS0 first (crosvm --serial captures this)
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
hostname sovereign-forge

# TEAM_025: Mount tmpfs for shared memory
mkdir -p /dev/shm /tmp
mount -t tmpfs -o mode=1777 tmpfs /dev/shm
mount -t tmpfs tmpfs /tmp

# ============================================================================
# TEAM_025: Mount data.img (/dev/vdb) for PERSISTENT storage
# ============================================================================
# This is CRITICAL for Tailscale machine identity!
# - rootfs.img (/dev/vda) = rebuilt on every `sovereign build`
# - data.img (/dev/vdb) = PERSISTS across rebuilds
#
# By storing Tailscale state on /data, the machine ID survives rebuilds,
# preventing duplicate registrations (sovereign-forge, sovereign-forge-1, etc.)
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
mkdir -p /data/forgejo/repositories /data/tailscale
chown forgejo:forgejo /data/forgejo 2>/dev/null || true

# TEAM_025: Set current date for TLS cert validation
# Must be recent enough for certificate "not before" dates
date -s "2025-12-29 14:00:00" 2>/dev/null || true

# Create device nodes
mkdir -p /dev/net
[ -e /dev/net/tun ] || mknod /dev/net/tun c 10 200
chmod 666 /dev/net/tun

# TEAM_025: Configure TAP networking
# Host TAP is 192.168.101.1, guest is 192.168.101.2
echo "=== Configuring TAP Network ==="
sleep 1

# Find virtio network interface (not tunnel interfaces like erspan0)
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
    ip addr add 192.168.101.2/24 dev "$IFACE"
    ip link set "$IFACE" up
    ip route add default via 192.168.101.1
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

# ============================================================================
# TEAM_025: Start Tailscale with persistent identity
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

/usr/sbin/tailscaled \
    --tun=userspace-networking \
    --state=/data/tailscale/tailscaled.state \
    --socket=/var/run/tailscale/tailscaled.sock &
TAILSCALED_PID=$!

# Wait for tailscaled to be ready (check socket exists)
for i in 1 2 3 4 5; do
    [ -S /var/run/tailscale/tailscaled.sock ] && break
    sleep 1
done

# TEAM_025: Check if we have PERSISTENT state (machine identity survives rebuilds)
STATE_FILE="/data/tailscale/tailscaled.state"

if [ -f "$STATE_FILE" ] && [ -s "$STATE_FILE" ]; then
    # We have saved state - reconnect without authkey (preserves machine identity!)
    echo "Tailscale: Found persistent state, reconnecting..."
    /usr/bin/tailscale up --hostname=sovereign-forge 2>&1
else
    # First boot - need authkey for initial registration
    echo "Tailscale: No saved state, first-time registration..."
    AUTHKEY=""
    for param in $(cat /proc/cmdline); do
        case "$param" in
            tailscale.authkey=*) AUTHKEY="${param#tailscale.authkey=}" ;;
        esac
    done
    if [ -n "$AUTHKEY" ]; then
        /usr/bin/tailscale up --authkey="$AUTHKEY" --hostname=sovereign-forge 2>&1
    else
        echo "WARNING: No authkey for first-time registration"
    fi
fi

# TEAM_025: Expose Forgejo ports via Tailscale serve
# - Port 3000: Forgejo web UI
# - Port 22: Git SSH access
/usr/bin/tailscale serve --bg --tcp 3000 3000 2>&1 || true
/usr/bin/tailscale serve --bg --tcp 22 22 2>&1 || true

/usr/bin/tailscale status 2>&1

# ============================================================================
# TEAM_025: Wait for PostgreSQL (sovereign-sql VM)
# ============================================================================
# Forgejo needs PostgreSQL for its database backend.
# The SQL VM should be running and accessible via Tailscale.
# ============================================================================
echo "=== Waiting for PostgreSQL ==="
DB_HOST="sovereign-sql"
for i in $(seq 1 60); do
    if nc -z "$DB_HOST" 5432 2>/dev/null; then
        log "PostgreSQL is ready"
        break
    fi
    if [ $i -eq 60 ]; then
        log "WARNING: PostgreSQL timeout after 60s, starting Forgejo anyway"
    fi
    sleep 1
done

# ============================================================================
# TEAM_025: Start Forgejo
# ============================================================================
echo "=== Starting Forgejo ==="
mkdir -p /data/forgejo/repositories /var/log/forgejo
chown -R forgejo:forgejo /data/forgejo /var/log/forgejo

# Start Forgejo as forgejo user
su -s /bin/sh forgejo -c '/usr/bin/forgejo web' &
FORGEJO_PID=$!

log "Forgejo started (PID: $FORGEJO_PID)"
log "=== INIT COMPLETE ==="

# ============================================================================
# TEAM_025: Supervision loop for BOTH Tailscale and Forgejo
# ============================================================================
# If either dies (OOM, crash, anything), restart it automatically.
# Reference: Field Guide Section 6 - "No process supervision in VM"
# ============================================================================
while true; do
    # Check Forgejo
    if ! kill -0 $FORGEJO_PID 2>/dev/null; then
        echo "$(date): Forgejo died, restarting..."
        su -s /bin/sh forgejo -c '/usr/bin/forgejo web' &
        FORGEJO_PID=$!
        sleep 5
    fi
    
    # Check Tailscale daemon
    if ! kill -0 $TAILSCALED_PID 2>/dev/null; then
        echo "$(date): Tailscaled died, restarting..."
        /usr/sbin/tailscaled \
            --tun=userspace-networking \
            --state=/data/tailscale/tailscaled.state \
            --socket=/var/run/tailscale/tailscaled.sock &
        TAILSCALED_PID=$!
        sleep 3
        /usr/bin/tailscale serve --bg --tcp 3000 3000 2>&1 || true
        /usr/bin/tailscale serve --bg --tcp 22 22 2>&1 || true
    fi
    
    sleep 30
done
