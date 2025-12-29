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

# TEAM_018: Log ALL output to file for debugging
LOG=/var/log/init.log
mkdir -p /var/log
exec > $LOG 2>&1
set -x
echo "=== INIT START $(date) ==="

mount -t proc proc /proc
mount -t sysfs sysfs /sys

# Set hostname
hostname sovereign-sql

# TEAM_017: Mount tmpfs for shared memory (required for PostgreSQL)
mkdir -p /dev/shm /tmp
mount -t tmpfs -o mode=1777 tmpfs /dev/shm
mount -t tmpfs tmpfs /tmp

# TEAM_023: Get current time from host via kernel cmdline or use NTP after network
# The hardcoded date was a bug - instead we'll sync after Tailscale connects
# For now, set a reasonable recent date to avoid TLS cert "not yet valid" errors
date -s "2025-01-01 00:00:00" 2>/dev/null || true

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

# Only register if NOT already connected (first boot)
if ! /usr/bin/tailscale status >/dev/null 2>&1; then
    # TEAM_023: Safer authkey parsing - handle special characters
    AUTHKEY=""
    for param in $(cat /proc/cmdline); do
        case "$param" in
            tailscale.authkey=*) AUTHKEY="${param#tailscale.authkey=}" ;;
        esac
    done
    if [ -n "$AUTHKEY" ]; then
        echo "Tailscale: First-time registration..."
        /usr/bin/tailscale up --authkey="$AUTHKEY" --hostname=sovereign-sql 2>&1
        /usr/bin/tailscale serve --bg --tcp 5432 5432 2>&1
    else
        echo "WARNING: No authkey for first-time registration"
    fi
else
    echo "Tailscale: Already registered, auto-reconnecting..."
    # Ensure serve is running
    /usr/bin/tailscale serve --bg --tcp 5432 5432 2>&1 || true
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

# TEAM_023: FIX - $DB_PASSWORD must use double quotes to expand!
# The original had single quotes which prevented variable expansion
su postgres -c "psql -c \"ALTER USER postgres PASSWORD '$DB_PASSWORD';\"" 2>&1
echo "PostgreSQL version:"
su postgres -c "psql -c \"SELECT version();\"" 2>&1

echo "=== INIT COMPLETE ==="

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
        /usr/sbin/tailscaled \
            --tun=userspace-networking \
            --state=/data/tailscale/tailscaled.state \
            --socket=/var/run/tailscale/tailscaled.sock &
        TAILSCALED_PID=$!
        sleep 3
        /usr/bin/tailscale serve --bg --tcp 5432 5432 2>&1 || true
    fi
    
    sleep 30
done
