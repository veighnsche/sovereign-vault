# Phase 3A, Step 3 — Build, Deploy, Verify Root

**Phase:** 3A (KernelSU)
**Step:** 3 of 3
**Team:** TEAM_001
**Status:** Pending
**Depends On:** Step 2

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** This is the moment of truth. I am about to flash a custom kernel to a real device. My failure modes:
> - Claiming "build succeeded" without checking CONFIG_KSU=y
> - Flashing without backing up the original boot.img
> - Declaring victory when the device boots without testing root
> - Panicking when something fails and suggesting we "try a different approach"
>
> **The rule:** A kernel that boots is not success. A kernel with working root is success. I verify EVERYTHING.
>
> **Device bricking risk:** If the kernel is broken, the device may bootloop. I MUST have a backup plan.

---

## 1. Goal

Build the kernel, flash it, and verify root access works.

---

## 2. Pre-Conditions

- [ ] Step 2 complete (all patches applied, verifications passed)
- [ ] Device connected via USB
- [ ] Bootloader unlocked
- [ ] **BACKUP of stock boot.img exists** (CRITICAL)

---

## 3. Task 1: Build Kernel

```bash
go run sovereign.go build --kernel
```

**Or manually:**
```bash
./build_raviole.sh
```

**Expected output location:** `out/raviole/dist/boot.img`

**Verify build artifacts:**
```bash
# boot.img exists
ls -la out/raviole/dist/boot.img

# CONFIG_KSU is enabled
grep "CONFIG_KSU=y" out/raviole/dist/.config

# LOCALVERSION is set
grep "CONFIG_LOCALVERSION" out/raviole/dist/.config
# Expected: CONFIG_LOCALVERSION="-sovereign"
```

> 🤖 **AI Warning:** If `CONFIG_KSU=y` is NOT in .config, the build is WRONG. Do not flash. Go back to Step 2.

---

## 4. Task 2: Backup Current Kernel (MANDATORY)

```bash
# Create backup directory
mkdir -p backups/

# Get current boot image from device
adb reboot bootloader
fastboot fetch boot backups/boot_stock.img
# Or if fetch not supported:
# Boot to recovery, use TWRP to backup boot partition
```

> 🤖 **AI Warning:** Do NOT skip this step. If the new kernel bootloops, this backup is your recovery path.

---

## 5. Task 3: Deploy Kernel

```bash
go run sovereign.go deploy --kernel
```

**Or manually:**
```bash
adb reboot bootloader
fastboot flash boot out/raviole/dist/boot.img
fastboot reboot
```

Wait for device to boot (1-2 minutes).

---

## 6. Task 4: Install KernelSU Manager

Download from: https://github.com/tiann/KernelSU/releases

```bash
adb install KernelSU_*.apk
```

Open the app and check:
- Status shows "Working" (green)
- Version is ~32245 (NOT 16)

---

## 7. Task 5: Verify Root Access

```bash
go run sovereign.go test --kernel
```

**Or manually:**
```bash
# Test 1: Kernel version
adb shell cat /proc/version
# Expected: Contains "sovereign"

# Test 2: Root access
adb shell su -c id
# Expected: uid=0(root) gid=0(root) ...

# Test 3: KernelSU version
adb shell su -v
# Expected: ~32245 (NOT 16)
```

---

## 8. Expected Test Output

```
=== Testing Kernel/KernelSU ===

1. Kernel version contains 'sovereign': ✓ PASS
2. Root access via su: ✓ PASS
3. KernelSU version (not 16): ✓ PASS (version: 32245)

=== ALL TESTS PASSED ===
Phase 3A complete! Root access working.
```

---

## 9. Troubleshooting

| Problem | Cause | Fix |
|---------|-------|-----|
| **Bootloop** | Kernel crash | Flash backup: `fastboot flash boot backups/boot_stock.img` |
| **No `su` command** | KernelSU not compiled in | Check CONFIG_KSU=y, redo Step 2 |
| **Version shows 16** | Kbuild patch not applied | Redo Step 2, Task 1 |
| **"su: permission denied"** | Shell not granted root | Open KernelSU Manager, grant Shell root |
| **Manager shows "Not Installed"** | Kernel not flashed correctly | Verify with `uname -r`, reflash |

---

## 10. Recovery Procedure

If device bootloops:

```bash
# 1. Force into bootloader (hold Power + Volume Down)
# 2. Flash backup
fastboot flash boot backups/boot_stock.img
fastboot reboot

# 3. Diagnose what went wrong
# Check build logs, verify patches, try again
```

---

## 11. Checkpoint

- [ ] Kernel built successfully
- [ ] `CONFIG_KSU=y` confirmed in .config
- [ ] Backup of stock boot.img saved
- [ ] Device boots with "-sovereign" kernel
- [ ] KernelSU Manager shows "Working"
- [ ] KernelSU version is ~32245 (not 16)
- [ ] `adb shell su -c id` returns `uid=0(root)`

---

## Phase 3A Complete!

If all tests pass, Phase 3A is complete. You have root access.

**Next:** Proceed to **[Phase 3B — PostgreSQL](phase-3b.md)**
