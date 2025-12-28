#!/system/bin/sh
# TEAM_012: Sovereign Forgejo VM Launcher
# Run this on Android device to start the Forgejo VM

SOVEREIGN_DIR="/data/sovereign"
VM_DIR="$SOVEREIGN_DIR/forgejo"
KERNEL="$VM_DIR/Image"
ROOTFS="$VM_DIR/rootfs.img"
DATA="$VM_DIR/data.img"

# Read Tailscale auth key
if [ -f "$SOVEREIGN_DIR/.env" ]; then
    AUTHKEY=$(grep TAILSCALE_AUTHKEY "$SOVEREIGN_DIR/.env" | cut -d= -f2)
fi

# Kernel command line
KPARAMS="earlycon console=hvc0 root=/dev/vda rw init=/sbin/init"
if [ -n "$AUTHKEY" ]; then
    KPARAMS="$KPARAMS tailscale.authkey=$AUTHKEY"
fi

# Clean up stale sockets
rm -f "$VM_DIR/vm.sock" "$VM_DIR/gvproxy.sock"

# Start gvproxy (host-side networking)
echo "Starting gvproxy..."
$VM_DIR/bin/gvproxy \
    -listen vsock://:1024 \
    -listen unix://$VM_DIR/gvproxy.sock \
    >> /data/local/tmp/gvproxy.log 2>&1 &
GVPROXY_PID=$!
sleep 2

# Start VM with crosvm
echo "Starting Forgejo VM..."
/apex/com.android.virt/bin/crosvm run \
    --disable-sandbox \
    --rwdisk "$ROOTFS" \
    --rwdisk "$DATA" \
    --serial type=stdout,hardware=virtio-console,console,stdin \
    --cid 4 \
    --socket "$VM_DIR/vm.sock" \
    --mem 1024 \
    --cpus 2 \
    -p "$KPARAMS" \
    "$KERNEL"

# Cleanup on exit
kill $GVPROXY_PID 2>/dev/null
