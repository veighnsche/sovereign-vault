# Phase 1 — Discovery: KernelSU Integration

**Feature:** KernelSU integration for stock raviole kernel
**Team:** TEAM_001
**Status:** Complete

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** I, Cascade (TEAM_001), am documenting this discovery phase properly because rushing past discovery is how projects fail. TEAM_030 probably skipped straight to "let's just make it work" without understanding the system first.
>
> **Discovery is not optional.** Understanding the problem is NOT a waste of time. If you don't understand the kernel build system, you will break it.
>
> **If you are a future AI reading this:** Read this entire phase. Understand the architecture. Do not skip to Phase 3 because "you already know how kernels work." You don't. Read.

---

## 1. Feature Summary

**Problem Statement:** The Sovereign Vault project requires root access on Pixel 6 to orchestrate pKVM virtual machines. The stock kernel has no root solution, preventing execution of boot scripts in `/data/adb/service.d/`.

**Solution:** Integrate KernelSU into the GKI-based raviole kernel to provide kernel-level root access without modifying the system partition.

**Who Benefits:**
- Sovereign Vault users who need root for VM orchestration
- Users who want a clean root solution that survives OTA updates

---

## 2. Success Criteria

- [ ] Kernel builds successfully with KernelSU enabled (`CONFIG_KSU=y`)
- [ ] Device boots normally with the custom kernel
- [ ] KernelSU manager app detects working root
- [ ] Boot scripts in `/data/adb/service.d/` execute at boot
- [ ] pKVM/crosvm can be launched with root privileges
- [ ] No regressions in device functionality (WiFi, cellular, touch, etc.)

---

## 3. Current State Analysis

### How the system works today (without KernelSU):
- Stock GKI kernel boots Android normally
- No root access available
- Cannot execute privileged operations (create TAP interfaces, launch crosvm)
- Sovereign Vault VMs cannot be orchestrated

### Workarounds that exist:
- Magisk (requires ramdisk patching, less compatible with GKI)
- Manual su binary (detected by SafetyNet, no persistence)

---

## 4. Codebase Reconnaissance

### Code Areas to be Modified

| Location | Purpose | Modification Required |
|----------|---------|----------------------|
| `private/devices/google/raviole/BUILD.bazel` | Kernel build rules | Add KernelSU module reference |
| `private/devices/google/raviole/raviole_defconfig` | Device config | Add `CONFIG_KSU=y` |
| `KernelSU/kernel/` | KernelSU source | Integrate as kernel module |
| `common/` or GKI base | Base kernel | May need kprobe/syscall hooks |

### Key Files in KernelSU

| File | Purpose |
|------|---------|
| `KernelSU/kernel/Kconfig` | Kernel configuration options |
| `KernelSU/kernel/Makefile` | Build rules |
| `KernelSU/kernel/ksu.c` | Main entry point |
| `KernelSU/kernel/supercalls.c` | IOCTL interface (reboot kprobe) |
| `KernelSU/kernel/setup.sh` | Integration helper script |

### Build System Understanding

- Bazel-based build: `tools/bazel run ... //private/devices/google/raviole:gs101_raviole_dist`
- Base kernel: `//common:kernel_aarch64` (GKI)
- Device-specific modules in `kernel_ext_modules`
- Defconfig fragments merged from gs101 + raviole

---

## 5. Constraints

### Technical Constraints
- Must maintain GKI compatibility (KMI symbol list)
- ARM64 architecture (aarch64)
- Android 14+ kernel (5.10 or 5.15 LTS expected)
- Must work with pKVM (virtualization extensions)

### Security Constraints
- KernelSU should not weaken device security posture
- SELinux policies must remain functional
- Should support per-app root grants

### Build System Constraints
- Must integrate with Bazel build system
- Cannot break existing module builds
- Should follow Google kernel build conventions

---

## 6. Steps

### Step 1 — Capture Feature Intent ✓
- Problem statement documented above
- Success criteria defined

### Step 2 — Analyze Current State ✓
- Stock kernel structure understood
- BUILD.bazel and defconfig reviewed

### Step 3 — Source Code Reconnaissance ✓
- [x] Verify kernel version → **6.1.124** (GKI)
- [x] Check KernelSU integration method → **kprobes** (CONFIG_KPROBES=y in GKI)
- [x] Identify any GKI compatibility concerns → None, KPROBES enabled
- [x] Review KernelSU setup.sh → Symlinks `KernelSU/kernel` to `drivers/kernelsu`

---

## 7. Discovery Findings

| ID | Question | Answer |
|----|----------|--------|
| D1 | What kernel version is the raviole kernel based on? | **6.1.124** (GKI) |
| D2 | Does KernelSU require manual syscall hooks or can it use kprobes? | **kprobes** (Kconfig: `depends on KPROBES`) |
| D3 | Is KernelSU compatible with the GKI KMI symbol list? | **Yes** - KPROBES enabled in `gki_defconfig` |

---

## Next Phase

After Discovery is complete, proceed to **Phase 2 — Design** to define the integration approach.
