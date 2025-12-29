# Phase 3, Step 2: Start Script

**Feature:** Forgejo VM Implementation  
**Status:** 🔲 NOT STARTED  
**Parent:** [Phase 3: Implementation](phase-3.md)

---

## Goal

Rewrite `vm/forgejo/start.sh` to use TAP networking instead of gvproxy/vsock.

---

## Reference Implementation

**Copy patterns from:** `vm/sql/start.sh` (95 lines, working)

---

## Current State (WRONG)

```bash
# Uses gvproxy and vsock - this doesn't work in AVF
$VM_DIR/bin/gvproxy -listen vsock://:1024 ...
crosvm run ... --cid 4 --serial type=stdout,hardware=virtio-console ...
```

---

## Tasks

### Task 1: Replace header and variables

```bash
#!/system/bin/sh
# TEAM_0XX: Sovereign Forgejo VM Start Script
# Uses TAP networking with key fix from crosvm-on-android repo

SOVEREIGN_DIR="/data/sovereign"
VM_DIR="${SOVEREIGN_DIR}/vm/forgejo"
LOG="${VM_DIR}/console.log"
KERNEL="${VM_DIR}/Image"
CROSVM="/apex/com.android.virt/bin/crosvm"
TAP_NAME="vm_forge"
TAP_HOST_IP="192.168.101.1"
TAP_NETMASK="255.255.255.0"

# Load auth key if exists
[ -f "${SOVEREIGN_DIR}/.env" ] && . ${SOVEREIGN_DIR}/.env
```

### Task 2: Add Phantom Process Killer defense

```bash
# Disable Phantom Process Killer (Android 12+)
# Without this, Android silently kills crosvm vCPU forks
device_config set_sync_disabled_for_tests persistent 2>/dev/null || true
device_config put activity_manager max_phantom_processes 2147483647 2>/dev/null || true
```

### Task 3: Clean up old instances

```bash
# Clean up old instances
pkill -9 -f "crosvm.*forgejo" 2>/dev/null || true
rm -f ${VM_DIR}/vm.sock
sleep 1
```

### Task 4: Setup TAP interface

```bash
# Setup TAP interface
ip link del ${TAP_NAME} 2>/dev/null || true
ip tuntap add mode tap name ${TAP_NAME}
ip addr add ${TAP_HOST_IP}/24 dev ${TAP_NAME}
ip link set ${TAP_NAME} up

# Enable IP forwarding and NAT
echo 1 > /proc/sys/net/ipv4/ip_forward
echo 0 > /proc/sys/net/ipv4/conf/${TAP_NAME}/rp_filter
echo 0 > /proc/sys/net/ipv4/conf/all/rp_filter

# KEY FIX: Bypass Android policy routing
ip rule del from all lookup main pref 1 2>/dev/null || true
ip rule add from all lookup main pref 1

# Add default route to main table
GATEWAY=$(ip route show table wlan0 | grep default | awk '{print $3}')
if [ -n "$GATEWAY" ]; then
    ip route del default 2>/dev/null || true
    ip route add default via $GATEWAY dev wlan0
fi

# NAT for VM traffic
iptables -t nat -D POSTROUTING -s 192.168.101.0/24 -o wlan0 -j MASQUERADE 2>/dev/null || true
iptables -t nat -A POSTROUTING -s 192.168.101.0/24 -o wlan0 -j MASQUERADE

# FORWARD rules
iptables -D FORWARD -i ${TAP_NAME} -o wlan0 -j ACCEPT 2>/dev/null || true
iptables -D FORWARD -i wlan0 -o ${TAP_NAME} -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true
iptables -I FORWARD 1 -i ${TAP_NAME} -o wlan0 -j ACCEPT
iptables -I FORWARD 2 -i wlan0 -o ${TAP_NAME} -m state --state RELATED,ESTABLISHED -j ACCEPT
```

### Task 5: Build kernel params and start crosvm

```bash
# Build kernel params
# Use ttyS0 for console - crosvm --serial captures this
KPARAMS="earlycon console=ttyS0 root=/dev/vda rw init=/sbin/init.sh"
if [ -n "$TAILSCALE_AUTHKEY" ]; then
    KPARAMS="$KPARAMS tailscale.authkey=$TAILSCALE_AUTHKEY"
fi

# Start VM with TAP networking
# data.img as second block device (/dev/vdb) for persistent storage
$CROSVM run \
    --disable-sandbox \
    --mem 1024 \
    --cpus 2 \
    --block path="${VM_DIR}/rootfs.img",root \
    --block path="${VM_DIR}/data.img" \
    --params "$KPARAMS" \
    --serial type=stdout \
    --net tap-name=${TAP_NAME} \
    --socket "${VM_DIR}/vm.sock" \
    "${KERNEL}" > "$LOG" 2>&1 &

VM_PID=$!
echo $VM_PID > "${VM_DIR}/vm.pid"

# Protect from OOM killer
echo -1000 > /proc/${VM_PID}/oom_score_adj 2>/dev/null || true

echo "TAP interface: ${TAP_NAME} (${TAP_HOST_IP})"
echo "Forge VM started (PID: $VM_PID)"
echo "Log: $LOG"
```

---

## Key Differences from Current Script

| Aspect | Current (Wrong) | New (Correct) |
|--------|-----------------|---------------|
| Networking | gvproxy/vsock | TAP with NAT |
| Console | `console=hvc0`, virtio-console | `console=ttyS0`, --serial stdout |
| CID | `--cid 4` | Not needed |
| Init | `/sbin/init` | `/sbin/init.sh` |
| Device path | `/data/sovereign/forgejo/` | `/data/sovereign/vm/forgejo/` |

---

## Expected Output

- Rewritten file: `vm/forgejo/start.sh` (~95 lines)
- Executable permissions
- No gvproxy references
- Uses TAP `vm_forge` at 192.168.101.x

---

## Verification

1. No gvproxy or vsock references
2. Uses TAP interface `vm_forge`
3. Uses host IP 192.168.101.1
4. Uses `console=ttyS0` in kernel params
5. Uses `init=/sbin/init.sh`
6. Passes data.img as second block device

---

## Next Step

→ [Phase 3, Step 3: Dockerfile](phase-3-step-3.md)
