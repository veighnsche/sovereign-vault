#!/system/bin/sh
# TEAM_030: Shared bridge setup for VM-to-VM communication
# This script creates a Linux bridge that connects all VM TAP interfaces
# allowing direct L2 communication between VMs.

BRIDGE_NAME="vm_bridge"
BRIDGE_IP="192.168.100.1"
BRIDGE_SUBNET="24"

# Check if bridge already exists
if ip link show ${BRIDGE_NAME} >/dev/null 2>&1; then
    echo "Bridge ${BRIDGE_NAME} already exists"
    exit 0
fi

echo "Creating VM bridge: ${BRIDGE_NAME}"

# Create the bridge
ip link add ${BRIDGE_NAME} type bridge
ip addr add ${BRIDGE_IP}/${BRIDGE_SUBNET} dev ${BRIDGE_NAME}
ip link set ${BRIDGE_NAME} up

# Enable IP forwarding
echo 1 > /proc/sys/net/ipv4/ip_forward
echo 0 > /proc/sys/net/ipv4/conf/${BRIDGE_NAME}/rp_filter
echo 0 > /proc/sys/net/ipv4/conf/all/rp_filter

# KEY FIX: Bypass Android policy routing by using main table first
# From: https://github.com/bvucode/crosvm-on-android
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

# FORWARD rules for bridge traffic to internet
iptables -D FORWARD -i ${BRIDGE_NAME} -o wlan0 -j ACCEPT 2>/dev/null || true
iptables -D FORWARD -i wlan0 -o ${BRIDGE_NAME} -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true
iptables -I FORWARD 1 -i ${BRIDGE_NAME} -o wlan0 -j ACCEPT
iptables -I FORWARD 2 -i wlan0 -o ${BRIDGE_NAME} -m state --state RELATED,ESTABLISHED -j ACCEPT

echo "Bridge ${BRIDGE_NAME} created with IP ${BRIDGE_IP}/${BRIDGE_SUBNET}"
echo "VMs should use 192.168.100.x addresses (gateway: ${BRIDGE_IP})"
