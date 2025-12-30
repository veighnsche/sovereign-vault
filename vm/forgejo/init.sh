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

# TEAM_030: Configure TAP networking (bridge-based)
# All VMs on same 192.168.100.x subnet via shared bridge
# SQL VM: 192.168.100.2, Forge VM: 192.168.100.3, Gateway: 192.168.100.1
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
    # TEAM_030: Use 192.168.100.3 - same subnet as SQL VM (192.168.100.2)
    ip addr add 192.168.100.3/24 dev "$IFACE"
    ip link set "$IFACE" up
    ip route add default via 192.168.100.1
    echo "nameserver 8.8.8.8" > /etc/resolv.conf
    echo "Network configured on $IFACE (192.168.100.3)"
    ip addr show "$IFACE"
    ip route show
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

# TEAM_025: Check if we have PERSISTENT state (machine identity survives rebuilds)
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
    /usr/bin/tailscale up --hostname=sovereign-forge 2>&1
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
        /usr/bin/tailscale up --authkey="$AUTHKEY" --hostname=sovereign-forge 2>&1
    else
        echo "WARNING: No authkey for first-time registration"
    fi
fi

# TEAM_034: Removed tailscale serve - not needed!
# Direct port binding works: Forgejo listens on 0.0.0.0, Tailscale routes inbound
# traffic directly to tailscale0 interface. See TAILSCALE_AVF_LIMITATIONS.md

/usr/bin/tailscale status 2>&1

# ============================================================================
# TEAM_025: Wait for PostgreSQL (sovereign-sql VM)
# ============================================================================
# Forgejo needs PostgreSQL for its database backend.
# The SQL VM should be running and accessible via Tailscale.
# ============================================================================
echo "=== Waiting for PostgreSQL ==="
# TEAM_029: Use TAP IP for VM-to-VM (Tailscale userspace can't initiate outgoing)
# SQL VM TAP IP: 192.168.100.2, routed via Android host gateway
DB_HOST="192.168.100.2"
DB_PORT="5432"
DB_USER="forgejo"
DB_NAME="forgejo"
DB_URI="postgres://${DB_USER}:***@${DB_HOST}:${DB_PORT}/${DB_NAME}"

log "PostgreSQL URI: $DB_URI"
echo "PostgreSQL URI: $DB_URI"

# TEAM_029: FAIL FAST - don't start a broken service
if ! nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; then
    log "PostgreSQL not immediately available, waiting..."
    for i in $(seq 1 30); do
        if nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; then
            log "PostgreSQL is ready at ${DB_HOST}:${DB_PORT}"
            break
        fi
        if [ $i -eq 30 ]; then
            log "FATAL: PostgreSQL at ${DB_HOST}:${DB_PORT} not responding after 30s"
            log "Cannot start Forgejo without database - FAIL FAST"
            echo "FATAL: PostgreSQL dependency unavailable"
            # Keep VM running for debugging but don't start Forgejo
            while true; do sleep 3600; done
        fi
        sleep 1
    done
else
    log "PostgreSQL immediately available at ${DB_HOST}:${DB_PORT}"
fi

# TEAM_029: Test actual PostgreSQL protocol connection (not just TCP)
log "Testing PostgreSQL protocol connection..."
echo "Testing PostgreSQL protocol connection..."
if command -v psql >/dev/null 2>&1; then
    PGPASSWORD=forgejo psql -h "$DB_HOST" -U forgejo -d forgejo -c "SELECT 1;" 2>&1 | head -5 || log "psql test failed"
else
    log "psql not available, testing with nc verbose..."
    echo "QUIT" | nc -v -w 5 "$DB_HOST" "$DB_PORT" 2>&1 | head -5 || true
fi

# ============================================================================
# TEAM_034: Generate TLS certificates using Tailscale
# ============================================================================
echo "=== Generating TLS Certificates ==="
mkdir -p /data/forgejo/tls

# Get the ACTUAL Tailscale hostname (may be sovereign-forge-1, sovereign-forge-2, etc.)
TS_FQDN=$(/usr/bin/tailscale status --json | grep -o '"DNSName":"[^"]*"' | head -1 | cut -d'"' -f4 | sed 's/\.$//')
if [ -z "$TS_FQDN" ]; then
    # Fallback: get from tailscale cert --help output which shows available domains
    TS_FQDN=$(/usr/bin/tailscale cert 2>&1 | grep -o '[a-z0-9-]*\.tail[a-z0-9]*\.ts\.net' | head -1)
fi
echo "Tailscale FQDN: $TS_FQDN"

# Generate cert for the actual hostname
if [ -n "$TS_FQDN" ]; then
    /usr/bin/tailscale cert --cert-file=/data/forgejo/tls/cert.pem --key-file=/data/forgejo/tls/key.pem "$TS_FQDN" 2>&1 || {
        echo "WARNING: Failed to generate TLS cert for $TS_FQDN"
    }
    # Save the FQDN for app.ini updates
    echo "$TS_FQDN" > /data/forgejo/tls/fqdn.txt
else
    echo "ERROR: Could not determine Tailscale FQDN"
fi

chown -R forgejo:forgejo /data/forgejo/tls
chmod 600 /data/forgejo/tls/key.pem 2>/dev/null || true

# ============================================================================
# TEAM_034: Update app.ini with actual Tailscale hostname
# ============================================================================
if [ -f /data/forgejo/tls/fqdn.txt ]; then
    TS_FQDN=$(cat /data/forgejo/tls/fqdn.txt)
    echo "Updating app.ini with hostname: $TS_FQDN"
    # Update DOMAIN and ROOT_URL in app.ini
    sed -i "s|^DOMAIN = .*|DOMAIN = $TS_FQDN|" /etc/forgejo/app.ini
    sed -i "s|^ROOT_URL = .*|ROOT_URL = https://$TS_FQDN/|" /etc/forgejo/app.ini
    sed -i "s|^SSH_DOMAIN = .*|SSH_DOMAIN = $TS_FQDN|" /etc/forgejo/app.ini
fi

# ============================================================================
# TEAM_034: Allow non-root to bind to port 443
# ============================================================================
# Lower the unprivileged port start from 1024 to 443
# This allows the forgejo user to bind directly to port 443
echo 443 > /proc/sys/net/ipv4/ip_unprivileged_port_start
echo "Unprivileged port start set to 443"

# ============================================================================
# TEAM_025: Start Forgejo
# ============================================================================
echo "=== Starting Forgejo ==="
mkdir -p /data/forgejo/repositories /var/log/forgejo /var/lib/forgejo
chown -R forgejo:forgejo /data/forgejo /var/log/forgejo /var/lib/forgejo

# TEAM_029: Set Forgejo paths - official binary looks in wrong places by default
export FORGEJO_WORK_DIR=/var/lib/forgejo
export GITEA_WORK_DIR=/var/lib/forgejo

# Start Forgejo with explicit config path
# TEAM_029: Redirect stderr to see crash reasons
echo "Starting Forgejo with config: /etc/forgejo/app.ini"
su -s /bin/sh forgejo -c 'FORGEJO_WORK_DIR=/var/lib/forgejo GITEA_WORK_DIR=/var/lib/forgejo /usr/bin/forgejo web -c /etc/forgejo/app.ini 2>&1' &
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
        # TEAM_029: Must include -c flag and env vars on restart too!
        su -s /bin/sh forgejo -c 'FORGEJO_WORK_DIR=/var/lib/forgejo GITEA_WORK_DIR=/var/lib/forgejo /usr/bin/forgejo web -c /etc/forgejo/app.ini' &
        FORGEJO_PID=$!
        sleep 5
    fi
    
    # Check Tailscale daemon
    # TEAM_033: Fixed inconsistency - restart must match initial start (native tun, TAP IP)
    if ! kill -0 $TAILSCALED_PID 2>/dev/null; then
        echo "$(date): Tailscaled died, restarting..."
        /usr/sbin/tailscaled \
            --state=/data/tailscale/tailscaled.state \
            --socket=/var/run/tailscale/tailscaled.sock &
        TAILSCALED_PID=$!
        sleep 3
        # TEAM_034: tailscale serve removed - direct port binding works
    fi
    
    sleep 30
done
