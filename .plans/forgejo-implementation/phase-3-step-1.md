# Phase 3, Step 1: Init Script

**Feature:** Forgejo VM Implementation  
**Status:** 🔲 NOT STARTED  
**Parent:** [Phase 3: Implementation](phase-3.md)

---

## Goal

Create `vm/forgejo/init.sh` that properly initializes the Forgejo VM in AVF/crosvm environment.

---

## Reference Implementation

**Copy patterns from:** `vm/sql/init.sh` (264 lines, working)

---

## Tasks

### Task 1: Create base init script structure

```bash
#!/bin/sh
# init.sh - AVF guest init script for Forgejo VM
# TEAM_0XX: Init script for AVF (OpenRC hangs, so we use this instead)

export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

# Mount essential filesystems FIRST
mount -t proc proc /proc 2>/dev/null || true
mount -t sysfs sysfs /sys 2>/dev/null || true
mount -t devtmpfs devtmpfs /dev 2>/dev/null || true
```

### Task 2: Add logging and console detection

```bash
LOG=/var/log/init.log
mkdir -p /var/log

CONSOLE=""
for dev in /dev/ttyS0 /dev/console /dev/hvc0; do
    if [ -c "$dev" ]; then
        CONSOLE="$dev"
        break
    fi
done

log() {
    echo "$1" >> "$LOG"
    [ -n "$CONSOLE" ] && echo "$1" > "$CONSOLE" 2>/dev/null
}

log "=== INIT START $(date) ==="
```

### Task 3: Mount data disk for persistent storage

```bash
hostname sovereign-forge

mkdir -p /dev/shm /tmp
mount -t tmpfs -o mode=1777 tmpfs /dev/shm
mount -t tmpfs tmpfs /tmp

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

mkdir -p /data/forgejo /data/tailscale
chown forgejo:forgejo /data/forgejo 2>/dev/null || true
```

### Task 4: Configure TAP networking

```bash
# Set time for TLS
date -s "2025-12-29 10:00:00" 2>/dev/null || true

# Create device nodes
mkdir -p /dev/net
[ -e /dev/net/tun ] || mknod /dev/net/tun c 10 200
chmod 666 /dev/net/tun

echo "=== Configuring TAP Network ==="
sleep 1

# Find virtio network interface
IFACE=""
for iface in $(ls /sys/class/net/); do
    case "$iface" in
        eth*|enp*|ens*) IFACE="$iface"; break ;;
    esac
done

if [ -n "$IFACE" ]; then
    ip addr add 192.168.101.2/24 dev "$IFACE"
    ip link set "$IFACE" up
    ip route add default via 192.168.101.1
    echo "nameserver 8.8.8.8" > /etc/resolv.conf
    echo "Network configured on $IFACE"
else
    echo "WARNING: No network interface found"
fi
```

### Task 5: Start Tailscale with persistent identity

```bash
echo "=== Starting Tailscale ==="
mkdir -p /data/tailscale /var/run/tailscale /dev/net
[ -e /dev/net/tun ] || mknod /dev/net/tun c 10 200
chmod 666 /dev/net/tun

/usr/sbin/tailscaled \
    --tun=userspace-networking \
    --state=/data/tailscale/tailscaled.state \
    --socket=/var/run/tailscale/tailscaled.sock &
TAILSCALED_PID=$!

# Wait for tailscaled
for i in 1 2 3 4 5; do
    [ -S /var/run/tailscale/tailscaled.sock ] && break
    sleep 1
done

STATE_FILE="/data/tailscale/tailscaled.state"

if [ -f "$STATE_FILE" ] && [ -s "$STATE_FILE" ]; then
    echo "Tailscale: Found persistent state, reconnecting..."
    /usr/bin/tailscale up --hostname=sovereign-forge 2>&1
else
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

# Expose ports via Tailscale
/usr/bin/tailscale serve --bg --tcp 3000 3000 2>&1 || true
/usr/bin/tailscale serve --bg --tcp 22 22 2>&1 || true

/usr/bin/tailscale status 2>&1
```

### Task 6: Wait for PostgreSQL and start Forgejo

```bash
echo "=== Waiting for PostgreSQL ==="
DB_HOST="sovereign-sql"
for i in $(seq 1 60); do
    if nc -z "$DB_HOST" 5432 2>/dev/null; then
        log "PostgreSQL is ready"
        break
    fi
    if [ $i -eq 60 ]; then
        log "WARNING: PostgreSQL timeout, starting Forgejo anyway"
    fi
    sleep 1
done

echo "=== Starting Forgejo ==="
mkdir -p /data/forgejo/repositories /var/log/forgejo
chown -R forgejo:forgejo /data/forgejo /var/log/forgejo

# Start Forgejo
su -s /bin/sh forgejo -c '/usr/bin/forgejo web' &
FORGEJO_PID=$!

log "Forgejo started"
log "=== INIT COMPLETE ==="
```

### Task 7: Add supervision loop

```bash
# Supervision loop
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
```

---

## Expected Output

- New file: `vm/forgejo/init.sh` (~200 lines)
- Executable permissions set
- Mirrors `vm/sql/init.sh` structure

---

## Verification

1. File exists and is executable
2. Uses TAP IP 192.168.101.2 (not 192.168.100.2)
3. Uses hostname `sovereign-forge`
4. Persists Tailscale state on `/data/tailscale`
5. Exposes ports 3000 and 22

---

## Next Step

→ [Phase 3, Step 2: Start Script](phase-3-step-2.md)
