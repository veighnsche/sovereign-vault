#!/bin/sh
# Sovereign Vaultwarden VM Init Script
# TEAM_034: Based on working Forgejo pattern
#
# This script runs as PID 1 in the VM. It must:
# 1. Mount filesystems
# 2. Configure networking
# 3. Start Tailscale and get TLS cert
# 4. Wait for PostgreSQL
# 5. Start Vaultwarden
# 6. Supervise processes
#
# See: docs/VAULTWARDEN_IMPLEMENTATION_GUIDE.md

set -e

# ============================================================================
# Logging
# ============================================================================
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

log "=== Vaultwarden VM Init Starting ==="

# ============================================================================
# TEAM_035: Read secrets from kernel cmdline (centralized in .env)
# ============================================================================
VAULTWARDEN_DB_PASS=""
VAULTWARDEN_ADMIN_TOKEN=""
for param in $(cat /proc/cmdline 2>/dev/null); do
    case "$param" in
        vaultwarden.db_password=*) VAULTWARDEN_DB_PASS="${param#vaultwarden.db_password=}" ;;
        vaultwarden.admin_token=*) VAULTWARDEN_ADMIN_TOKEN="${param#vaultwarden.admin_token=}" ;;
    esac
done
log "Secrets loaded: db_pass=${VAULTWARDEN_DB_PASS:+SET} admin_token=${VAULTWARDEN_ADMIN_TOKEN:+SET}"

# ============================================================================
# Mount filesystems (use || true to handle already-mounted cases)
# TEAM_035: Match SQL init.sh pattern - kernel may pre-mount devtmpfs
# ============================================================================
mount -t proc proc /proc 2>/dev/null || true
mount -t sysfs sysfs /sys 2>/dev/null || true
mount -t devtmpfs devtmpfs /dev 2>/dev/null || true
mkdir -p /dev/pts /dev/shm /run /tmp
mount -t devpts devpts /dev/pts 2>/dev/null || true
mount -t tmpfs -o mode=1777 tmpfs /dev/shm 2>/dev/null || true
mount -t tmpfs tmpfs /run 2>/dev/null || true
mount -t tmpfs tmpfs /tmp 2>/dev/null || true

# Create required directories
mkdir -p /var/run/tailscale /var/lib/tailscale /data/tailscale

# ============================================================================
# Mount data disk (persistent storage)
# ============================================================================
log "Mounting data disk..."
mkdir -p /data
if [ -b /dev/vdb ]; then
    mount /dev/vdb /data || log "WARNING: Could not mount /dev/vdb"
else
    log "WARNING: /dev/vdb not found, using tmpfs for /data"
    mount -t tmpfs tmpfs /data
fi

# Create data directories
mkdir -p /data/vault/data /data/vault/tls /data/tailscale
chown -R vaultwarden:vaultwarden /data/vault 2>/dev/null || true

# ============================================================================
# Network configuration
# TEAM_037: Fixed to detect interface like SQL/Forgejo (not hardcode eth0)
# ============================================================================
log "=== Configuring Network ==="

# Bring up loopback
ip link set lo up

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
log "Found interface: $IFACE"

if [ -n "$IFACE" ]; then
    # Vault VM gets 192.168.100.4 (SQL=.2, Forge=.3, Vault=.4)
    ip addr add 192.168.100.4/24 dev "$IFACE"
    ip link set "$IFACE" up
    ip route add default via 192.168.100.1
    # TEAM_035: Set DNS resolver (required for ACME/Let's Encrypt cert generation)
    echo "nameserver 8.8.8.8" > /etc/resolv.conf
    log "Network configured on $IFACE (192.168.100.4)"
    ip addr show "$IFACE"
else
    log "WARNING: No network interface found"
    ip link
fi

# Test connectivity
log "Testing internet connectivity..."
ping -c 2 8.8.8.8 2>&1 || log "WARNING: Internet not reachable"

# ============================================================================
# Time synchronization (CRITICAL for TLS - must complete BEFORE Tailscale)
# TEAM_035: Set approximate time first (ntpd needs DNS which needs Tailscale)
# We'll do proper NTP sync after Tailscale is up
# ============================================================================
log "=== Setting Initial Time ==="
# Set approximate time for TLS cert validation (will be refined by NTP later)
date -s "2025-12-30 12:00:00" 2>/dev/null || true
log "Initial time set: $(date)"

# ============================================================================
# Start Tailscale
# ============================================================================
log "=== Starting Tailscale ==="

/usr/sbin/tailscaled \
    --state=/data/tailscale/tailscaled.state \
    --socket=/var/run/tailscale/tailscaled.sock &
TAILSCALED_PID=$!

# Wait for tailscaled to be ready
sleep 3

# Check if we have valid state or need authkey
STATE_FILE="/data/tailscale/tailscaled.state"
STATE_VALID=false
if [ -f "$STATE_FILE" ] && [ -s "$STATE_FILE" ]; then
    if grep -q 'PrivateNodeKey' "$STATE_FILE" 2>/dev/null; then
        STATE_VALID=true
    fi
fi

if [ "$STATE_VALID" = "true" ]; then
    log "Tailscale: Using existing machine identity"
    /usr/bin/tailscale up --hostname=sovereign-vault --accept-routes --reset 2>&1
else
    log "Tailscale: No valid state, using authkey for registration..."
    # Get authkey from kernel command line
    AUTHKEY=""
    for param in $(cat /proc/cmdline); do
        case "$param" in
            tailscale.authkey=*) AUTHKEY="${param#tailscale.authkey=}" ;;
        esac
    done
    if [ -n "$AUTHKEY" ]; then
        rm -f "$STATE_FILE" 2>/dev/null
        /usr/bin/tailscale up --authkey="$AUTHKEY" --hostname=sovereign-vault --reset 2>&1
    else
        log "WARNING: No authkey for first-time registration"
    fi
fi

/usr/bin/tailscale status 2>&1

# TEAM_035: Sync time properly now that DNS works via Tailscale
log "=== Syncing Time via NTP ==="
if command -v ntpd >/dev/null 2>&1; then
    ntpd -d -q -n -p pool.ntp.org 2>&1 || true
fi
log "Current time: $(date)"

# ============================================================================
# TEAM_034: Generate TLS certificates using Tailscale
# ============================================================================
log "=== Generating TLS Certificates ==="
mkdir -p /data/vault/tls

# Get the ACTUAL Tailscale hostname (may be sovereign-vault-1, -2, etc.)
TS_FQDN=$(/usr/bin/tailscale status --json | grep -o '"DNSName":"[^"]*"' | head -1 | cut -d'"' -f4 | sed 's/\.$//')
if [ -z "$TS_FQDN" ]; then
    TS_FQDN=$(/usr/bin/tailscale cert 2>&1 | grep -o '[a-z0-9-]*\.tail[a-z0-9]*\.ts\.net' | head -1)
fi
log "Tailscale FQDN: $TS_FQDN"

# Generate cert for actual hostname
if [ -n "$TS_FQDN" ]; then
    log "Requesting TLS cert for: $TS_FQDN"
    if /usr/bin/tailscale cert \
        --cert-file=/data/vault/tls/cert.pem \
        --key-file=/data/vault/tls/key.pem \
        "$TS_FQDN" 2>&1; then
        log "TLS cert generated successfully"
        echo "$TS_FQDN" > /data/vault/tls/fqdn.txt
    else
        log "WARNING: Failed to generate TLS cert for $TS_FQDN"
        log "Vaultwarden will NOT work without HTTPS - WebCrypto requires secure context"
        # Create self-signed cert as fallback (won't work for Bitwarden clients but allows debugging)
        log "Creating self-signed fallback cert..."
        openssl req -x509 -newkey rsa:2048 -keyout /data/vault/tls/key.pem \
            -out /data/vault/tls/cert.pem -days 365 -nodes \
            -subj "/CN=$TS_FQDN" 2>/dev/null || true
        echo "$TS_FQDN" > /data/vault/tls/fqdn.txt
    fi
else
    log "ERROR: Could not determine Tailscale FQDN"
fi

chown -R vaultwarden:vaultwarden /data/vault/tls
chmod 600 /data/vault/tls/key.pem 2>/dev/null || true

# ============================================================================
# Wait for PostgreSQL
# ============================================================================
log "=== Waiting for PostgreSQL ==="
DB_HOST="192.168.100.2"
DB_PORT="5432"

if ! nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; then
    log "PostgreSQL not immediately available, waiting..."
    for i in $(seq 1 30); do
        if nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; then
            log "PostgreSQL is ready at ${DB_HOST}:${DB_PORT}"
            break
        fi
        if [ $i -eq 30 ]; then
            log "FATAL: PostgreSQL at ${DB_HOST}:${DB_PORT} not responding after 30s"
            log "Cannot start Vaultwarden without database - FAIL FAST"
            echo "FATAL: PostgreSQL dependency unavailable"
            while true; do sleep 3600; done
        fi
        sleep 1
    done
else
    log "PostgreSQL immediately available at ${DB_HOST}:${DB_PORT}"
fi

# ============================================================================
# TEAM_034: Allow non-root to bind to port 443
# ============================================================================
echo 443 > /proc/sys/net/ipv4/ip_unprivileged_port_start
log "Unprivileged port start set to 443"

# ============================================================================
# Start Vaultwarden
# ============================================================================
log "=== Starting Vaultwarden ==="
mkdir -p /data/vault/data
chown -R vaultwarden:vaultwarden /data/vault

# Set environment variables
export DATA_FOLDER=/data/vault/data
# TEAM_035: web-vault is in /usr/share (not /data) to avoid being shadowed by data.img mount
export WEB_VAULT_FOLDER=/usr/share/vaultwarden/web-vault
export WEB_VAULT_ENABLED=true

# Database connection (created automatically by SQL VM init.sh)
# TEAM_035: Password from .env (passed via cmdline), fallback to default
DB_PASS="${VAULTWARDEN_DB_PASS:-vaultwarden}"
export DATABASE_URL="postgresql://vaultwarden:${DB_PASS}@192.168.100.2:5432/vaultwarden"

# HTTPS configuration with actual Tailscale hostname
if [ -f /data/vault/tls/fqdn.txt ]; then
    TS_FQDN=$(cat /data/vault/tls/fqdn.txt)
    export DOMAIN="https://$TS_FQDN"
else
    export DOMAIN="https://sovereign-vault.tail5bea38.ts.net"
fi

export ROCKET_PORT=443
export ROCKET_ADDRESS=0.0.0.0
export ROCKET_TLS="{certs=\"/data/vault/tls/cert.pem\",key=\"/data/vault/tls/key.pem\"}"

# Security settings
export SIGNUPS_ALLOWED=true
export WEBSOCKET_ENABLED=true
export LOG_LEVEL=info

# TEAM_035: Admin token from .env (enables /admin panel if set)
if [ -n "$VAULTWARDEN_ADMIN_TOKEN" ]; then
    export ADMIN_TOKEN="$VAULTWARDEN_ADMIN_TOKEN"
    log "Admin panel enabled"
fi

log "Starting Vaultwarden with DOMAIN=$DOMAIN"

# Start as vaultwarden user
su -s /bin/sh vaultwarden -c '/usr/bin/vaultwarden' &
VAULTWARDEN_PID=$!

log "Vaultwarden started (PID: $VAULTWARDEN_PID)"
log "=== INIT COMPLETE ==="

# ============================================================================
# Supervision loop
# ============================================================================
while true; do
    # Check Vaultwarden
    if ! kill -0 $VAULTWARDEN_PID 2>/dev/null; then
        log "Vaultwarden died, restarting..."
        su -s /bin/sh vaultwarden -c '/usr/bin/vaultwarden' &
        VAULTWARDEN_PID=$!
        sleep 5
    fi
    
    # Check Tailscale daemon
    if ! kill -0 $TAILSCALED_PID 2>/dev/null; then
        log "Tailscaled died, restarting..."
        /usr/sbin/tailscaled \
            --state=/data/tailscale/tailscaled.state \
            --socket=/var/run/tailscale/tailscaled.sock &
        TAILSCALED_PID=$!
        sleep 3
    fi
    
    sleep 30
done
