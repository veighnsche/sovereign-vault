#!/system/bin/sh
# TEAM_016: Sovereign SQL VM Start Script
# TEAM_023: Added Phantom Process Killer defense
# TEAM_030: Uses shared bridge for VM-to-VM communication
# Uses TAP networking with key fix from crosvm-on-android repo

SOVEREIGN_DIR="/data/sovereign"
VM_DIR="${SOVEREIGN_DIR}/vm/sql"
LOG="${VM_DIR}/console.log"
KERNEL="${VM_DIR}/Image"
CROSVM="/apex/com.android.virt/bin/crosvm"
TAP_NAME="vm_sql"
# TEAM_030: Bridge-based networking - all VMs on same subnet
BRIDGE_NAME="vm_bridge"
BRIDGE_IP="192.168.100.1"

# Load auth key if exists
[ -f "${SOVEREIGN_DIR}/.env" ] && . ${SOVEREIGN_DIR}/.env

# TEAM_023: Disable Phantom Process Killer (Android 12+)
# This is THE MOST CRITICAL defense - without it, Android silently kills
# child processes (crosvm forks for vCPUs) regardless of OOM settings.
# Reference: Field Guide Section 1.2 - "the most critical and non-obvious gotcha"
device_config set_sync_disabled_for_tests persistent 2>/dev/null || true
device_config put activity_manager max_phantom_processes 2147483647 2>/dev/null || true

# Clean up old instances
pkill -9 -f "crosvm.*sql" 2>/dev/null || true
rm -f ${VM_DIR}/vm.sock
sleep 1

# TEAM_030: Setup shared bridge if not exists
if ! ip link show ${BRIDGE_NAME} >/dev/null 2>&1; then
    echo "Creating shared VM bridge: ${BRIDGE_NAME}"
    ip link add ${BRIDGE_NAME} type bridge
    ip addr add ${BRIDGE_IP}/24 dev ${BRIDGE_NAME}
    ip link set ${BRIDGE_NAME} up
fi

# TEAM_033: ALWAYS ensure networking rules are in place (stop command removes them)
# These must run on EVERY start, not just when creating the bridge
echo "Ensuring networking rules..."

# Enable IP forwarding
echo 1 > /proc/sys/net/ipv4/ip_forward
echo 0 > /proc/sys/net/ipv4/conf/${BRIDGE_NAME}/rp_filter 2>/dev/null || true
echo 0 > /proc/sys/net/ipv4/conf/all/rp_filter

# KEY FIX: Bypass Android policy routing by using main table first
# From: https://github.com/bvucode/crosvm-on-android
# Android's netd/fwmark routing blocks VM traffic - this rule makes main table take precedence
ip rule del from all lookup main pref 1 2>/dev/null || true
ip rule add from all lookup main pref 1

# Add default route to main table (Android keeps it in wlan0 table)
GATEWAY=$(ip route show table wlan0 2>/dev/null | grep default | awk '{print $3}')
if [ -n "$GATEWAY" ]; then
    ip route del default 2>/dev/null || true
    ip route add default via $GATEWAY dev wlan0
fi

# NAT for VM traffic to internet
iptables -t nat -D POSTROUTING -s 192.168.100.0/24 -o wlan0 -j MASQUERADE 2>/dev/null || true
iptables -t nat -A POSTROUTING -s 192.168.100.0/24 -o wlan0 -j MASQUERADE

# FORWARD rules for bridge traffic
iptables -D FORWARD -i ${BRIDGE_NAME} -o wlan0 -j ACCEPT 2>/dev/null || true
iptables -D FORWARD -i wlan0 -o ${BRIDGE_NAME} -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true
iptables -I FORWARD 1 -i ${BRIDGE_NAME} -o wlan0 -j ACCEPT
iptables -I FORWARD 2 -i wlan0 -o ${BRIDGE_NAME} -m state --state RELATED,ESTABLISHED -j ACCEPT

# Setup TAP interface and attach to bridge
ip link del ${TAP_NAME} 2>/dev/null || true
ip tuntap add mode tap name ${TAP_NAME}
ip link set ${TAP_NAME} master ${BRIDGE_NAME}
ip link set ${TAP_NAME} up
echo "TAP ${TAP_NAME} attached to bridge ${BRIDGE_NAME}"

# Build kernel params
# TEAM_023: Use ttyS0 for console - crosvm --serial captures serial port, NOT virtio hvc0
KPARAMS="earlycon console=ttyS0 root=/dev/vda rw init=/sbin/init.sh"
if [ -n "$TAILSCALE_AUTHKEY" ]; then
    KPARAMS="$KPARAMS tailscale.authkey=$TAILSCALE_AUTHKEY"
fi

# Start VM with TAP networking
# TEAM_023: Added data.img as second block device (/dev/vdb) for persistent storage
# This is CRITICAL for Tailscale machine identity to survive rebuilds!
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

# TEAM_033: Fixed undefined variable TAP_HOST_IP
echo "TAP interface: ${TAP_NAME} (bridge: ${BRIDGE_NAME})"
echo "SQL VM started (PID: $VM_PID)"
echo "Log: $LOG"
