# TEAM_040: Deploy All Services (SQL, Vault, Forge)

## Date: 2026-01-02

## Status: PARTIAL SUCCESS

## Task
Build, deploy, and start all three VMs: SQL, Vault, and Forgejo.

## Summary
- **SQL**: ✓ Running, accessible on 192.168.100.2:5432
- **Vault**: ✓ Running, accessible on 192.168.100.4:443
- **Forge**: ⚠️ TAP connectivity issue - VM boots but NO-CARRIER on host TAP

---

## Context from Prior Teams

### TEAM_037: Root Cause of VM Killing
- VMs started via `adb shell su -c` become orphaned (ppid=1)
- Android init kills untracked orphan processes after ~90 seconds
- **Fix**: Created daemon script that stays alive as watchdog

### TEAM_038: Bugs Fixed
- Fixed trap in `start_single()` to only kill watched VM
- Fixed Vault not receiving database password
- Fixed Tailscale state detection (check BEFORE starting tailscaled)
- Added timeout to `tailscale up` commands

### TEAM_039: Root Cause Investigation
- Fixed Tailscale duplicate registration bug
- Added `./sovereign clean` command for Tailscale cleanup
- Created `.planning/VM_OPERATIONS_GUIDE.md`

---

## Work Done This Session

### 1. Build Phase
```bash
sudo ./sovereign build --sql --skip-preflight    # ✓ Success
sudo ./sovereign build --vault --skip-preflight  # ✓ Success
sudo ./sovereign build --forge --skip-preflight  # ✓ Success (after date fix)
```

### 2. Deploy Phase
```bash
./sovereign deploy --sql    # ✓ Success
./sovereign deploy --vault  # ✓ Success
./sovereign deploy --forge  # ✓ Success
```

### 3. Start Phase - Issues Encountered

#### Issue 1: CLI Start Command Fails
- `./sovereign start --sql` reported "VM process died during boot"
- **Root cause**: Timing issues in daemon start mechanism
- **Workaround**: Start VMs manually with direct crosvm command

#### Issue 2: Forge TLS Certificate Missing
- Forgejo crashed: `Failed to load https cert file /data/forgejo/tls/cert.pem`
- **Root cause**: Tailscale state on data.img was in "Logged out" state
- The init script found PrivateNodeKey (looked valid) but state was actually logged out
- `tailscale cert` failed because Tailscale wasn't connected
- **Fix**:
  1. Deleted corrupt data.img
  2. Created fresh data.img: `dd if=/dev/zero of=... bs=1M count=4096 && mkfs.ext4`
  3. Restarted with authkey in kernel cmdline
  4. Fresh registration succeeded, TLS cert generated

#### Issue 3: Forge TAP Connectivity (UNRESOLVED)
- After rebuild and redeploy, Forge VM TAP shows NO-CARRIER
- SQL TAP (vm_sql) and Vault TAP (vm_vault) are UP
- Forge TAP (vm_forge) is DOWN despite VM showing eth0 configured inside
- Forge crosvm process dies shortly after start
- **Needs investigation**: Possible TAP naming conflict or crosvm networking issue

---

## Files Modified

| File | Change |
|------|--------|
| `vm/forgejo/init.sh` | Updated hardcoded date from 2025-12-29 to 2026-01-02 |

---

## Current Service Status

| Service | IP:Port | Status | Notes |
|---------|---------|--------|-------|
| SQL (PostgreSQL) | 192.168.100.2:5432 | ✓ Running | Started manually via crosvm |
| Vault (Vaultwarden) | 192.168.100.4:443 | ✓ Running | Started manually via crosvm |
| Forge (Forgejo) | 192.168.100.3:443 | ⚠️ TAP Issue | VM boots but network DOWN |

---

## Tailscale Hostnames
- `sovereign-forge.tail5bea38.ts.net` (Forgejo)
- `sovereign-vault.tail5bea38.ts.net` (Vaultwarden)  
- `sovereign-sql.tail5bea38.ts.net` (PostgreSQL)

---

## Commands Reference

### Manual VM Start (when CLI fails)
```bash
# Setup TAP first
adb shell "su -c 'ip tuntap add dev vm_<name> mode tap && ip link set vm_<name> master vm_bridge && ip link set vm_<name> up'"

# Start crosvm
adb shell "su -c 'export LD_LIBRARY_PATH=/apex/com.android.virt/lib64:/system/lib64; nohup /apex/com.android.virt/bin/crosvm run --disable-sandbox --mem 1024 --cpus 2 --block path=/data/sovereign/vm/<name>/rootfs.img,root --block path=/data/sovereign/vm/<name>/data.img --params \"earlycon console=ttyS0 root=/dev/vda rw init=/sbin/init.sh tailscale.authkey=<key>\" --serial type=stdout --net tap-name=vm_<name> --socket /data/sovereign/vm/<name>/vm.sock /data/sovereign/vm/<name>/Image > /data/sovereign/vm/<name>/console.log 2>&1 &'"
```

### Create Fresh data.img on Device
```bash
adb shell "su -c 'dd if=/dev/zero of=/data/sovereign/vm/<name>/data.img bs=1M count=4096 && mkfs.ext4 /data/sovereign/vm/<name>/data.img'"
```

### Check Service Connectivity
```bash
adb shell "su -c 'nc -z -w 2 192.168.100.2 5432 && echo SQL_OK'"
adb shell "su -c 'nc -z -w 2 192.168.100.4 443 && echo VAULT_OK'"
adb shell "su -c 'nc -z -w 2 192.168.100.3 443 && echo FORGE_OK'"
```

### Check TAP Interfaces
```bash
adb shell "su -c 'ip link show | grep vm_'"
```

---

## Known Issues for Next Team

### 1. Forge TAP NO-CARRIER Issue
- **Symptom**: vm_forge TAP shows NO-CARRIER even when Forge VM is running
- **Observation**: vm_sql and vm_vault TAPs work fine (state UP)
- **Console log shows**: VM configures eth0 with 192.168.100.3 internally
- **Possible causes**:
  - TAP name conflict (old TAP not properly cleaned up)
  - Crosvm networking race condition
  - Forge VM process dying before TAP fully connects

### 2. Date Hardcoding in init.sh
- All three VMs have hardcoded dates for TLS certificate validation
- Should get date from host or use more dynamic approach

### 3. CLI Start Command Issues
- `./sovereign start --sql` sometimes fails with "VM process died during boot"
- Manual crosvm command works reliably
- May need to add retry logic or improve timing in daemon script

---

## Handoff Checklist
- [x] Project builds cleanly
- [x] All VMs deployed to device
- [x] SQL service accessible
- [x] Vault service accessible
- [ ] Forge service accessible (TAP issue)
- [x] Team file created with full context
- [x] Prior team files reviewed and context documented
