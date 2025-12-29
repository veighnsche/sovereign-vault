#!/system/bin/sh
# TEAM_016: Sovereign SQL VM Start Script
# TEAM_023: Added Phantom Process Killer defense
# Uses TAP networking with key fix from crosvm-on-android repo

SOVEREIGN_DIR="/data/sovereign"
VM_DIR="${SOVEREIGN_DIR}/vm/sql"
LOG="${VM_DIR}/console.log"
KERNEL="${VM_DIR}/Image"
CROSVM="/apex/com.android.virt/bin/crosvm"
TAP_NAME="vm_sql"
TAP_HOST_IP="192.168.100.1"
TAP_NETMASK="255.255.255.0"

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

# Setup TAP interface
ip link del ${TAP_NAME} 2>/dev/null || true
ip tuntap add mode tap name ${TAP_NAME}
ip addr add ${TAP_HOST_IP}/24 dev ${TAP_NAME}
ip link set ${TAP_NAME} up

# Enable IP forwarding and NAT for VM internet access
echo 1 > /proc/sys/net/ipv4/ip_forward
echo 0 > /proc/sys/net/ipv4/conf/${TAP_NAME}/rp_filter
echo 0 > /proc/sys/net/ipv4/conf/all/rp_filter

# KEY FIX: Bypass Android policy routing by using main table first
# From: https://github.com/bvucode/crosvm-on-android
ip rule del from all lookup main pref 1 2>/dev/null || true
ip rule add from all lookup main pref 1

# TEAM_018: Add default route to main table (Android keeps it in wlan0 table)
# Get the gateway from wlan0 table and add to main
GATEWAY=$(ip route show table wlan0 | grep default | awk '{print $3}')
if [ -n "$GATEWAY" ]; then
    ip route del default 2>/dev/null || true
    ip route add default via $GATEWAY dev wlan0
fi

# NAT for VM traffic
iptables -t nat -D POSTROUTING -s 192.168.100.0/24 -o wlan0 -j MASQUERADE 2>/dev/null || true
iptables -t nat -A POSTROUTING -s 192.168.100.0/24 -o wlan0 -j MASQUERADE

# FORWARD rules for VM traffic
iptables -D FORWARD -i ${TAP_NAME} -o wlan0 -j ACCEPT 2>/dev/null || true
iptables -D FORWARD -i wlan0 -o ${TAP_NAME} -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true
iptables -I FORWARD 1 -i ${TAP_NAME} -o wlan0 -j ACCEPT
iptables -I FORWARD 2 -i wlan0 -o ${TAP_NAME} -m state --state RELATED,ESTABLISHED -j ACCEPT

# Build kernel params
# TEAM_023: Use ttyS0 for console - crosvm --serial captures serial port, NOT virtio hvc0
KPARAMS="earlycon console=ttyS0 root=/dev/vda rw init=/sbin/init.sh"
if [ -n "$TAILSCALE_AUTHKEY" ]; then
    KPARAMS="$KPARAMS tailscale.authkey=$TAILSCALE_AUTHKEY"
fi

# Start VM with TAP networking
$CROSVM run \
    --disable-sandbox \
    --mem 1024 \
    --cpus 2 \
    --block path="${VM_DIR}/rootfs.img",root \
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
echo "SQL VM started (PID: $VM_PID)"
echo "Log: $LOG"
