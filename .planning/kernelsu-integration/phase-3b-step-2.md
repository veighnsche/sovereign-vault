# Phase 3B, Step 2 — Deploy & Start PostgreSQL VM

**Phase:** 3B (PostgreSQL)
**Step:** 2 of 3
**Team:** TEAM_001
**Status:** Pending
**Depends On:** Step 1

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** I am about to deploy a VM to a phone and run it under pKVM. This is where complexity explodes. My failure modes:
> - Pushing files to wrong location
> - Forgetting to create the TAP interface
> - Not waiting for PostgreSQL to be ready before declaring success
> - Ignoring crosvm error messages
>
> **The rule:** The VM must actually START and PostgreSQL must be LISTENING. crosvm running is not success.

---

## 1. Goal

Deploy PostgreSQL VM to device and start it under pKVM.

---

## 2. Pre-Conditions

- [ ] Step 1 complete (VM image built)
- [ ] Phase 3A complete (root working)
- [ ] Device connected via USB
- [ ] `.env` file has Tailscale auth key

---

## 3. Task 1: Export VM Image

```bash
# Create a container from the image
CONTAINER_ID=$(docker create sovereign-sql)

# Export filesystem to tarball
docker export $CONTAINER_ID > vm/sql/rootfs.tar

# Create ext4 image (512MB should be enough for OS)
truncate -s 512M vm/sql/rootfs.img
mkfs.ext4 vm/sql/rootfs.img

# Mount and extract
mkdir -p /tmp/sql-mount
sudo mount vm/sql/rootfs.img /tmp/sql-mount
sudo tar -xf vm/sql/rootfs.tar -C /tmp/sql-mount
sudo umount /tmp/sql-mount
rmdir /tmp/sql-mount

# Cleanup
docker rm $CONTAINER_ID
rm vm/sql/rootfs.tar

echo "✓ VM image exported to vm/sql/rootfs.img"
```

---

## 4. Task 2: Create Data Disk

```bash
# Create 4GB data disk for PostgreSQL data
truncate -s 4G vm/sql/data.img
mkfs.ext4 vm/sql/data.img

echo "✓ Data disk created: vm/sql/data.img"
```

---

## 5. Task 3: Deploy to Device

```bash
go run sovereign.go deploy --sql
```

**Or manually:**

```bash
# Load env
source .env

# Create directories on device
adb shell su -c 'mkdir -p /data/sovereign/vm/sql'

# Push VM image (this takes a while)
adb push vm/sql/rootfs.img /data/local/tmp/
adb shell su -c 'mv /data/local/tmp/rootfs.img /data/sovereign/vm/sql/'

# Push data disk
adb push vm/sql/data.img /data/local/tmp/
adb shell su -c 'mv /data/local/tmp/data.img /data/sovereign/vm/sql/'

# Push guest kernel (if separate from host)
# adb push vm/guest_kernel /data/sovereign/

echo "✓ Files deployed to device"
```

---

## 6. Task 4: Create Start Script on Device

```bash
adb shell su -c 'cat > /data/sovereign/vm/sql/start.sh << "SCRIPT"
#!/system/bin/sh
# Sovereign SQL VM Start Script

SOVEREIGN_DIR="/data/sovereign"
VM_DIR="${SOVEREIGN_DIR}/vm/sql"
LOG="${VM_DIR}/console.log"
CROSVM="/apex/com.android.virt/bin/crosvm"

# Load auth key
source ${SOVEREIGN_DIR}/.env

# Clean up old instance
pkill -f "crosvm.*sql" 2>/dev/null || true
sleep 1

# Create TAP interface if not exists
ip link show sovereign_sql >/dev/null 2>&1 || {
    ip tuntap add mode tap user root vnet_hdr sovereign_sql
    ip addr add 192.168.10.1/24 dev sovereign_sql
    ip link set sovereign_sql up
}

# Start VM
$CROSVM run \
    --disable-sandbox \
    --mem 1024 \
    --cpus 2 \
    --rwdisk "${VM_DIR}/rootfs.img" \
    --rwdisk "${VM_DIR}/data.img" \
    --params "console=hvc0 sovereign.role=database tailscale.authkey=${TAILSCALE_AUTHKEY}" \
    --vsock 10 \
    --net tap-name=sovereign_sql \
    --serial type=stdout \
    "${SOVEREIGN_DIR}/guest_Image" > "$LOG" 2>&1 &

VM_PID=$!
echo $VM_PID > "${VM_DIR}/vm.pid"

# Protect from OOM killer
echo -1000 > /proc/${VM_PID}/oom_score_adj

echo "SQL VM started (PID: $VM_PID)"
SCRIPT'

adb shell su -c 'chmod +x /data/sovereign/vm/sql/start.sh'
```

---

## 7. Task 5: Start the VM

```bash
go run sovereign.go start --sql
```

**Or manually:**

```bash
adb shell su -c '/data/sovereign/vm/sql/start.sh'
```

---

## 8. Task 6: Verify VM is Running

```bash
# Check crosvm process
adb shell su -c 'ps -ef | grep crosvm'
# Expected: crosvm process with sql in args

# Check PID file
adb shell su -c 'cat /data/sovereign/vm/sql/vm.pid'

# Check TAP interface
adb shell su -c 'ip link show sovereign_sql'

# Check console log (first 50 lines)
adb shell su -c 'head -50 /data/sovereign/vm/sql/console.log'
```

---

## 9. Verification Checklist

```bash
echo "=== Phase 3B Step 2 Verification ==="

echo -n "1. VM image on device: "
adb shell su -c '[ -f /data/sovereign/vm/sql/rootfs.img ] && echo "✓" || echo "✗ FAIL"'

echo -n "2. Data disk on device: "
adb shell su -c '[ -f /data/sovereign/vm/sql/data.img ] && echo "✓" || echo "✗ FAIL"'

echo -n "3. Start script exists: "
adb shell su -c '[ -x /data/sovereign/vm/sql/start.sh ] && echo "✓" || echo "✗ FAIL"'

echo -n "4. crosvm running: "
adb shell su -c 'pgrep -f "crosvm.*sql" >/dev/null && echo "✓" || echo "✗ FAIL"'

echo -n "5. TAP interface up: "
adb shell su -c 'ip link show sovereign_sql 2>/dev/null | grep -q UP && echo "✓" || echo "✗ FAIL"'

echo "=== End Verification ==="
```

---

## 10. Troubleshooting

| Problem | Cause | Fix |
|---------|-------|-----|
| crosvm not found | APEX not mounted | Check `/apex/com.android.virt/bin/crosvm` exists |
| Permission denied | Not running as root | Use `su -c` |
| VM crashes immediately | Bad kernel or image | Check console.log for errors |
| TAP create fails | No tun module | Verify kernel has CONFIG_TUN=y |

---

## 11. Checkpoint

- [ ] VM rootfs.img exported and on device
- [ ] Data disk created and on device
- [ ] Start script created with correct paths
- [ ] TAP interface sovereign_sql exists
- [ ] crosvm process running
- [ ] Console log shows boot messages

---

## Next Step

Proceed to **[Phase 3B, Step 3 — Tailscale + Verify](phase-3b-step-3.md)**
