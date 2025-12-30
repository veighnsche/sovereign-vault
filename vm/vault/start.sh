#!/system/bin/sh
# TEAM_035: Sovereign Vaultwarden VM Start Script
# Based on sql/start.sh pattern
# Uses shared bridge networking (192.168.100.0/24)

SOVEREIGN_DIR="/data/sovereign"
VM_DIR="${SOVEREIGN_DIR}/vm/vault"
LOG="${VM_DIR}/console.log"
KERNEL="${VM_DIR}/Image"
CROSVM="/apex/com.android.virt/bin/crosvm"
TAP_NAME="vm_vault"
# TEAM_035: Shared bridge with SQL and Forge VMs
BRIDGE_NAME="vm_bridge"
BRIDGE_IP="192.168.100.1"

# Load auth key if exists
[ -f "${SOVEREIGN_DIR}/.env" ] && . ${SOVEREIGN_DIR}/.env

# TEAM_023: Disable Phantom Process Killer (Android 12+)
device_config set_sync_disabled_for_tests persistent 2>/dev/null || true
device_config put activity_manager max_phantom_processes 2147483647 2>/dev/null || true

# Clean up old instances
pkill -9 -f "crosvm.*vault" 2>/dev/null || true
rm -f ${VM_DIR}/vm.sock
sleep 1

# TEAM_035: Ensure shared bridge exists (SQL VM usually creates it)
if ! ip link show ${BRIDGE_NAME} >/dev/null 2>&1; then
    echo "Creating shared VM bridge: ${BRIDGE_NAME}"
    ip link add ${BRIDGE_NAME} type bridge
    ip addr add ${BRIDGE_IP}/24 dev ${BRIDGE_NAME}
    ip link set ${BRIDGE_NAME} up
fi

# Ensure networking rules are in place
echo "Ensuring networking rules..."

# Enable IP forwarding
echo 1 > /proc/sys/net/ipv4/ip_forward
echo 0 > /proc/sys/net/ipv4/conf/${BRIDGE_NAME}/rp_filter 2>/dev/null || true
echo 0 > /proc/sys/net/ipv4/conf/all/rp_filter

# KEY FIX: Bypass Android policy routing
ip rule del from all lookup main pref 1 2>/dev/null || true
ip rule add from all lookup main pref 1

# Add default route to main table
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
KPARAMS="earlycon console=ttyS0 root=/dev/vda rw init=/sbin/init.sh"
if [ -n "$TAILSCALE_AUTHKEY" ]; then
    KPARAMS="$KPARAMS tailscale.authkey=$TAILSCALE_AUTHKEY"
fi
# TEAM_035: Pass secrets from .env (centralized secrets)
if [ -n "$POSTGRES_VAULTWARDEN_PASSWORD" ]; then
    KPARAMS="$KPARAMS vaultwarden.db_password=$POSTGRES_VAULTWARDEN_PASSWORD"
fi
if [ -n "$VAULTWARDEN_ADMIN_TOKEN" ]; then
    KPARAMS="$KPARAMS vaultwarden.admin_token=$VAULTWARDEN_ADMIN_TOKEN"
fi

# Start VM with TAP networking
# TEAM_035: Vaultwarden uses less resources than PostgreSQL
$CROSVM run \
    --disable-sandbox \
    --mem 512 \
    --cpus 1 \
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

echo "TAP interface: ${TAP_NAME} (bridge: ${BRIDGE_NAME})"
echo "Vaultwarden VM started (PID: $VM_PID)"
echo "Log: $LOG"
