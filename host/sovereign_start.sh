#!/system/bin/sh
# TEAM_037: Sovereign Vault Boot Script
# This script runs at boot via KernelSU (/data/adb/service.d/)
# It starts VMs and STAYS ALIVE to prevent Android init from killing orphaned crosvm processes
#
# The key insight: Android 12+ init kills processes with ppid=1 that aren't tracked.
# By keeping this script running, crosvm processes remain children of this script,
# not orphans adopted by init.

SOVEREIGN_DIR="/data/sovereign"
LOG="${SOVEREIGN_DIR}/daemon.log"
CROSVM="/apex/com.android.virt/bin/crosvm"
BRIDGE_NAME="vm_bridge"
BRIDGE_IP="192.168.100.1"

# VM directories
SQL_DIR="${SOVEREIGN_DIR}/vm/sql"
FORGE_DIR="${SOVEREIGN_DIR}/vm/forgejo"
VAULT_DIR="${SOVEREIGN_DIR}/vm/vault"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') [sovereign] $1" >> "$LOG"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [sovereign] $1"
}

# Load environment
[ -f "${SOVEREIGN_DIR}/.env" ] && . "${SOVEREIGN_DIR}/.env"

# CRITICAL: Set linker path for crosvm
export LD_LIBRARY_PATH=/apex/com.android.virt/lib64:/system/lib64

# Wait for boot completion (only when running at boot)
wait_for_boot() {
    log "Waiting for boot completion..."
    while [ "$(getprop sys.boot_completed)" != "1" ]; do sleep 1; done
    while [ "$(getprop apexd.status)" != "ready" ]; do sleep 1; done
    log "Boot complete"
}

# CRITICAL: Disable Phantom Process Killer and related Android 12+ protections
disable_process_killers() {
    log "Disabling Android process killers..."
    
    # Phantom Process Killer - kills child processes of backgrounded apps
    /system/bin/device_config set_sync_disabled_for_tests persistent 2>/dev/null || true
    /system/bin/device_config put activity_manager max_phantom_processes 2147483647 2>/dev/null || true
    
    # Monitor phantom procs setting
    settings put global settings_enable_monitor_phantom_procs false 2>/dev/null || true
    
    log "Process killers disabled"
}

# Setup shared bridge network (all VMs on same subnet)
setup_networking() {
    log "Setting up networking..."
    
    # Create bridge if not exists
    if ! ip link show ${BRIDGE_NAME} >/dev/null 2>&1; then
        ip link add ${BRIDGE_NAME} type bridge
        ip addr add ${BRIDGE_IP}/24 dev ${BRIDGE_NAME}
        ip link set ${BRIDGE_NAME} up
        log "Created bridge ${BRIDGE_NAME}"
    fi
    
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
    
    # NAT for VM traffic
    iptables -t nat -D POSTROUTING -s 192.168.100.0/24 -o wlan0 -j MASQUERADE 2>/dev/null || true
    iptables -t nat -A POSTROUTING -s 192.168.100.0/24 -o wlan0 -j MASQUERADE
    
    # FORWARD rules
    iptables -D FORWARD -i ${BRIDGE_NAME} -o wlan0 -j ACCEPT 2>/dev/null || true
    iptables -D FORWARD -i wlan0 -o ${BRIDGE_NAME} -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true
    iptables -I FORWARD 1 -i ${BRIDGE_NAME} -o wlan0 -j ACCEPT
    iptables -I FORWARD 2 -i wlan0 -o ${BRIDGE_NAME} -m state --state RELATED,ESTABLISHED -j ACCEPT
    
    log "Networking configured"
}

# Create TAP interface for a VM and attach to bridge
setup_tap() {
    local TAP_NAME="$1"
    ip link del ${TAP_NAME} 2>/dev/null || true
    ip tuntap add mode tap name ${TAP_NAME}
    ip link set ${TAP_NAME} master ${BRIDGE_NAME}
    ip link set ${TAP_NAME} up
    log "TAP ${TAP_NAME} attached to bridge"
}

# Start a VM and return its PID
# Arguments: VM_DIR TAP_NAME KPARAMS_EXTRA
start_vm() {
    local VM_DIR="$1"
    local TAP_NAME="$2"
    local KPARAMS_EXTRA="$3"
    local VM_NAME=$(basename "$VM_DIR")
    
    if [ ! -f "${VM_DIR}/rootfs.img" ]; then
        log "ERROR: ${VM_DIR}/rootfs.img not found - skipping ${VM_NAME}"
        return 1
    fi
    
    if [ ! -f "${VM_DIR}/Image" ]; then
        log "ERROR: ${VM_DIR}/Image not found - skipping ${VM_NAME}"
        return 1
    fi
    
    # Setup TAP
    setup_tap "${TAP_NAME}"
    
    # Build kernel params
    local KPARAMS="earlycon console=ttyS0 root=/dev/vda rw init=/sbin/init.sh"
    [ -n "$TAILSCALE_AUTHKEY" ] && KPARAMS="$KPARAMS tailscale.authkey=$TAILSCALE_AUTHKEY"
    [ -n "$KPARAMS_EXTRA" ] && KPARAMS="$KPARAMS $KPARAMS_EXTRA"
    
    # Clean old socket
    rm -f "${VM_DIR}/vm.sock"
    
    # Build crosvm command
    local CROSVM_CMD="$CROSVM run --disable-sandbox --mem 1024 --cpus 2"
    CROSVM_CMD="$CROSVM_CMD --block path=${VM_DIR}/rootfs.img,root"
    [ -f "${VM_DIR}/data.img" ] && CROSVM_CMD="$CROSVM_CMD --block path=${VM_DIR}/data.img"
    CROSVM_CMD="$CROSVM_CMD --params \"$KPARAMS\""
    CROSVM_CMD="$CROSVM_CMD --serial type=stdout"
    CROSVM_CMD="$CROSVM_CMD --net tap-name=${TAP_NAME}"
    CROSVM_CMD="$CROSVM_CMD --socket ${VM_DIR}/vm.sock"
    CROSVM_CMD="$CROSVM_CMD ${VM_DIR}/Image"
    
    log "Starting ${VM_NAME}: $CROSVM_CMD"
    
    # Start VM in background
    eval "$CROSVM_CMD" > "${VM_DIR}/console.log" 2>&1 &
    local VM_PID=$!
    
    # Protect from OOM killer
    echo -1000 > /proc/${VM_PID}/oom_score_adj 2>/dev/null || true
    
    # Save PID
    echo $VM_PID > "${VM_DIR}/vm.pid"
    
    log "${VM_NAME} started (PID: ${VM_PID})"
    echo $VM_PID
}

# Wait for a service to be ready
wait_for_service() {
    local IP="$1"
    local PORT="$2"
    local TIMEOUT="$3"
    local NAME="$4"
    
    log "Waiting for ${NAME} at ${IP}:${PORT}..."
    for i in $(seq 1 $TIMEOUT); do
        if nc -z -w 1 "$IP" "$PORT" 2>/dev/null; then
            log "${NAME} is ready"
            return 0
        fi
        sleep 1
    done
    log "WARNING: ${NAME} not ready after ${TIMEOUT}s"
    return 1
}

# Check if a VM is running
is_vm_running() {
    local VM_DIR="$1"
    local PID_FILE="${VM_DIR}/vm.pid"
    
    if [ -f "$PID_FILE" ]; then
        local PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Main daemon function
run_daemon() {
    log "=== Sovereign Vault Daemon Starting ==="
    
    # Wait for boot if running at boot
    if [ "$(getprop sys.boot_completed)" != "1" ]; then
        wait_for_boot
    fi
    
    disable_process_killers
    setup_networking
    
    # Track PIDs
    local SQL_PID=""
    local FORGE_PID=""
    local VAULT_PID=""
    
    # Start SQL VM first (others depend on it)
    if [ -d "$SQL_DIR" ]; then
        # Build extra params for SQL (database passwords)
        local SQL_EXTRA=""
        [ -n "$POSTGRES_FORGEJO_PASSWORD" ] && SQL_EXTRA="$SQL_EXTRA forgejo.db_password=$POSTGRES_FORGEJO_PASSWORD"
        [ -n "$POSTGRES_VAULTWARDEN_PASSWORD" ] && SQL_EXTRA="$SQL_EXTRA vaultwarden.db_password=$POSTGRES_VAULTWARDEN_PASSWORD"
        
        SQL_PID=$(start_vm "$SQL_DIR" "vm_sql" "$SQL_EXTRA")
        
        # Wait for PostgreSQL
        wait_for_service "192.168.100.2" "5432" "60" "PostgreSQL"
    else
        log "SQL VM not deployed, skipping"
    fi
    
    # Start Forge VM
    if [ -d "$FORGE_DIR" ]; then
        FORGE_PID=$(start_vm "$FORGE_DIR" "vm_forge" "")
    else
        log "Forge VM not deployed, skipping"
    fi
    
    # Start Vault VM
    if [ -d "$VAULT_DIR" ]; then
        VAULT_PID=$(start_vm "$VAULT_DIR" "vm_vault" "")
    else
        log "Vault VM not deployed, skipping"
    fi
    
    log "=== All VMs started, entering watchdog loop ==="
    
    # CRITICAL: Stay alive forever as watchdog
    # This keeps crosvm processes as our children, not orphans
    while true; do
        sleep 60
        
        # Optional: restart dead VMs
        if [ -n "$SQL_PID" ] && ! kill -0 "$SQL_PID" 2>/dev/null; then
            log "WARNING: SQL VM died (was PID $SQL_PID)"
            # Could auto-restart here if desired
        fi
        
        if [ -n "$FORGE_PID" ] && ! kill -0 "$FORGE_PID" 2>/dev/null; then
            log "WARNING: Forge VM died (was PID $FORGE_PID)"
        fi
        
        if [ -n "$VAULT_PID" ] && ! kill -0 "$VAULT_PID" 2>/dev/null; then
            log "WARNING: Vault VM died (was PID $VAULT_PID)"
        fi
    done
}

# Handle stop signal
stop_all() {
    log "=== Stopping all VMs ==="
    
    for VM_DIR in "$SQL_DIR" "$FORGE_DIR" "$VAULT_DIR"; do
        if [ -f "${VM_DIR}/vm.pid" ]; then
            local PID=$(cat "${VM_DIR}/vm.pid")
            if kill -0 "$PID" 2>/dev/null; then
                log "Stopping $(basename $VM_DIR) (PID: $PID)"
                kill "$PID" 2>/dev/null
                sleep 1
                kill -9 "$PID" 2>/dev/null || true
            fi
            rm -f "${VM_DIR}/vm.pid"
        fi
    done
    
    # Cleanup TAPs
    ip link del vm_sql 2>/dev/null || true
    ip link del vm_forge 2>/dev/null || true
    ip link del vm_vault 2>/dev/null || true
    
    log "All VMs stopped"
}

# Handle single VM start (for CLI integration)
# CRITICAL: This function STAYS ALIVE as a watchdog to prevent Android killing the VM
start_single() {
    local VM="$1"
    local VM_PID=""
    
    disable_process_killers
    setup_networking
    
    case "$VM" in
        sql)
            local SQL_EXTRA=""
            [ -n "$POSTGRES_FORGEJO_PASSWORD" ] && SQL_EXTRA="$SQL_EXTRA forgejo.db_password=$POSTGRES_FORGEJO_PASSWORD"
            [ -n "$POSTGRES_VAULTWARDEN_PASSWORD" ] && SQL_EXTRA="$SQL_EXTRA vaultwarden.db_password=$POSTGRES_VAULTWARDEN_PASSWORD"
            VM_PID=$(start_vm "$SQL_DIR" "vm_sql" "$SQL_EXTRA")
            ;;
        forge)
            VM_PID=$(start_vm "$FORGE_DIR" "vm_forge" "")
            ;;
        vault)
            VM_PID=$(start_vm "$VAULT_DIR" "vm_vault" "")
            ;;
        *)
            log "Unknown VM: $VM"
            exit 1
            ;;
    esac
    
    # CRITICAL: Stay alive as watchdog - this keeps crosvm as our child
    # Without this, crosvm becomes orphaned and Android init kills it after ~90s
    log "Watchdog started for ${VM} (PID: ${VM_PID})"
    while true; do
        sleep 60
        if [ -n "$VM_PID" ] && ! kill -0 "$VM_PID" 2>/dev/null; then
            log "WARNING: ${VM} VM died (was PID ${VM_PID})"
            # VM died - exit watchdog (CLI will see it as stopped)
            break
        fi
    done
}

# Trap signals
trap stop_all TERM INT

# Main entry point
case "${1:-daemon}" in
    daemon)
        run_daemon
        ;;
    start)
        if [ -n "$2" ]; then
            start_single "$2"
        else
            run_daemon
        fi
        ;;
    stop)
        stop_all
        ;;
    status)
        for VM_DIR in "$SQL_DIR" "$FORGE_DIR" "$VAULT_DIR"; do
            VM_NAME=$(basename "$VM_DIR")
            if is_vm_running "$VM_DIR"; then
                PID=$(cat "${VM_DIR}/vm.pid")
                echo "${VM_NAME}: running (PID: $PID)"
            else
                echo "${VM_NAME}: stopped"
            fi
        done
        ;;
    *)
        echo "Usage: $0 {daemon|start [vm]|stop|status}"
        exit 1
        ;;
esac
