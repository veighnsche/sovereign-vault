# TEAM_004 — Implement Phase 3A: KernelSU Root Access

**Created:** 2024-12-27
**Status:** Active
**Task:** Implement Phase 3A - sovereign.go foundation + KernelSU integration

---

## Mission

Execute Phase 3A plan:
1. Create `sovereign.go` - the foundation CLI
2. Apply KernelSU patches
3. Build, deploy, verify root access

---

## Progress Log

| Date       | Action                                      |
|------------|---------------------------------------------|
| 2024-12-27 | Team registered, beginning implementation   |
| 2024-12-27 | Step 1 complete: sovereign.go created, compiles, help/status work |
| 2024-12-27 | Step 2 complete: All 7 patches applied and verified |
| 2024-12-27 | Step 3 in progress: Build, deploy, verify |

---

## Pre-Implementation Checks

- [x] Go installed (1.23.5)
- [x] KernelSU/kernel directory exists
- [x] aosp/drivers exists
- [x] private/devices/google/raviole exists

---

## Changes Made

| File | Change |
|------|--------|
| `sovereign.go` | Created foundation CLI, fixed deploy to flash ALL images |
| `KernelSU/kernel/Kbuild` | Version 16 → 32245 |
| `aosp/drivers/kernelsu` | Symlink → ../../KernelSU/kernel |
| `aosp/drivers/Makefile` | Added obj-$(CONFIG_KSU) += kernelsu/ |
| `aosp/drivers/Kconfig` | Added source "drivers/kernelsu/Kconfig" |
| `private/devices/google/raviole/kernelsu.fragment` | Created CONFIG_KSU=y, LOCALVERSION=-sovereign |
| `private/devices/google/raviole/BUILD.bazel` | Added kernelsu.fragment to defconfig_fragments |
| `build_raviole.sh` | **CRITICAL FIX** - Added 3 GKI gates |

---

## CRITICAL FINDING: The 3 GKI Gates

Google hides kernel building behind 3 flags in `device.bazelrc`. Default = use prebuilt GKI (no custom kernel!):

| Gate | Default | Required for KernelSU |
|------|---------|----------------------|
| `--use_prebuilt_gki` | `true` | **`false`** |
| `--use_signed_prebuilts` | `true` | **`false`** |
| `--download_prebuilt_gki_fips140` | `true` | **`false`** |

**Location:** `/home/vince/Projects/android/kernel/private/devices/google/common/device.bazelrc`

---

## CRITICAL FINDING: Correct Flash Sequence (from Google docs)

Per https://source.android.com/docs/setup/build/building-pixel-kernels:

```bash
# Step 1-3: In bootloader mode
fastboot flash boot        out/raviole/dist/boot.img
fastboot flash dtbo        out/raviole/dist/dtbo.img
fastboot flash --dtb out/raviole/dist/dtb.img vendor_boot:dlkm out/raviole/dist/initramfs.img

# Step 4: Reboot to fastboot mode (different from bootloader!)
fastboot reboot fastboot

# Step 5: In fastboot mode
fastboot flash vendor_dlkm out/raviole/dist/vendor_dlkm.img

# Step 6: Final reboot
fastboot reboot
```

**Key insight:** There are TWO reboot stages - first to fastboot mode, then final reboot.

---

## Handoff Notes

### What Went Wrong
1. Built with prebuilt GKI (default) - KernelSU patches ignored
2. Flashed only boot.img manually - missing vendor_dlkm.img caused bootloop
3. Did not use sovereign.go CLI for flashing

### Next Steps (User Action Required)
1. First restore device with factory image
2. Run: `go run sovereign.go build --kernel` (now uses GKI gates)
3. Run: `go run sovereign.go deploy --kernel` (now flashes all images)
