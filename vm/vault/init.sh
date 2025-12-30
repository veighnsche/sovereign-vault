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
# Mount filesystems
# ============================================================================
mount -t proc proc /proc
mount -t sysfs sys /sys
mount -t devtmpfs dev /dev
mkdir -p /dev/pts /dev/shm /run
mount -t devpts devpts /dev/pts
mount -t tmpfs -o mode=1777 tmpfs /dev/shm
mount -t tmpfs tmpfs /run

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
# ============================================================================
log "=== Configuring Network ==="

# Bring up loopback
ip link set lo up

# Configure eth0 (TAP interface from host)
# Vault VM gets 192.168.100.4 (SQL=.2, Forge=.3, Vault=.4)
ip link set eth0 up
ip addr add 192.168.100.4/24 dev eth0
ip route add default via 192.168.100.1

log "Network configured: $(ip addr show eth0 | grep inet)"

# Test connectivity
log "Testing internet connectivity..."
ping -c 2 8.8.8.8 2>&1 || log "WARNING: Internet not reachable"

# ============================================================================
# Time synchronization (CRITICAL for TLS)
# ============================================================================
log "=== Syncing Time ==="
ntpd -n -q -p pool.ntp.org 2>&1 &

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
    /usr/bin/tailscale up --hostname=sovereign-vault --accept-routes 2>&1
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
        /usr/bin/tailscale up --authkey="$AUTHKEY" --hostname=sovereign-vault 2>&1
    else
        log "WARNING: No authkey for first-time registration"
    fi
fi

/usr/bin/tailscale status 2>&1

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
    /usr/bin/tailscale cert \
        --cert-file=/data/vault/tls/cert.pem \
        --key-file=/data/vault/tls/key.pem \
        "$TS_FQDN" 2>&1 || {
        log "WARNING: Failed to generate TLS cert for $TS_FQDN"
    }
    echo "$TS_FQDN" > /data/vault/tls/fqdn.txt
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
export WEB_VAULT_FOLDER=/data/vault/web-vault
export WEB_VAULT_ENABLED=true

# Database connection (created automatically by SQL VM init.sh)
# TEAM_035: Password documented in vm/vault/CREDENTIALS.md
export DATABASE_URL="postgresql://vaultwarden:PCc5zNNG6v8gwguclMQWMPjk4DUvg5F5@192.168.100.2:5432/vaultwarden"

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
