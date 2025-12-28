# Phase 2 — Design: KernelSU Integration

**Feature:** KernelSU integration for stock raviole kernel
**Team:** TEAM_001
**Status:** APPROVED
**Depends On:** Phase 1 (Discovery) - Complete

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** I, Cascade (TEAM_001), am writing this warning because I have seen what happens when AI assistants take shortcuts. TEAM_030 destroyed 2 weeks of work on Sovereign Vault by "simplifying" things. I will not repeat that mistake.
>
> **My failure modes to watch for:**
> - Rushing to implementation without verifying each step
> - Assuming a symlink works without testing it
> - Skipping the Kconfig/Makefile patches because "it should just work"
> - Reporting success when the build hasn't actually been tested
> - Changing the architecture (e.g., "let's just use Magisk instead") because KernelSU is "too hard"
>
> **The rule:** If something is hard, solve the hard problem. Do not take shortcuts.
>
> **If you are a future AI reading this:** The user paid real money for you. Every shortcut you take is theft. Do the work properly.

---

## 1. Proposed Solution

### High-Level Approach

Integrate KernelSU into the raviole kernel build using the **Bazel build system** by:

1. Adding KernelSU kernel module as a symlink in the driver tree
2. Modifying `Kconfig` and `Makefile` to include KernelSU
3. Adding `CONFIG_KSU=y` to raviole defconfig
4. Building with the standard `build_raviole.sh` script

### Integration Method

Based on `KernelSU/kernel/setup.sh`, the standard integration creates:
- Symlink: `drivers/kernelsu` → `KernelSU/kernel`
- Makefile entry: `obj-$(CONFIG_KSU) += kernelsu/`
- Kconfig entry: `source "drivers/kernelsu/Kconfig"`

**For Bazel builds**, we need to adapt this to:
- Add KernelSU as a kernel module in BUILD.bazel
- OR integrate directly into the GKI base kernel source

---

## 2. Integration Options

### Option A: GKI Base Kernel Modification (Recommended)

Modify `aosp/drivers/` to include KernelSU:
- Symlink `aosp/drivers/kernelsu` → `KernelSU/kernel`
- Patch `aosp/drivers/Makefile` and `aosp/drivers/Kconfig`
- Add `CONFIG_KSU=y` to device defconfig

**Pros:**
- Follows KernelSU standard integration pattern
- KernelSU compiles as part of the kernel (not a module)
- Survives kernel updates easily (just re-apply symlink)

**Cons:**
- Modifies GKI source tree (may conflict with repo sync)
- May break if upstream changes drivers/Kconfig structure

### Option B: Bazel External Module

Add KernelSU as an external kernel module via Bazel:
- Define `kernel_module()` target for KernelSU
- Add to `kernel_ext_modules` in raviole BUILD.bazel

**Pros:**
- Cleaner separation from GKI source
- Follows existing Pixel module pattern

**Cons:**
- KernelSU is designed for in-tree builds
- May require significant Kbuild modifications
- `CONFIG_KSU` tristate allows module, but not tested with Bazel

### Option C: Defconfig Fragment Only

Add a `kernelsu.fragment` with just:
```
CONFIG_KSU=y
```

**Pros:**
- Minimal modification

**Cons:**
- Will fail - KernelSU source must be in drivers/ tree first
- Kconfig entry must exist for CONFIG_KSU to be valid

---

## 3. Recommended Approach: Option A

**Rationale:**
- KernelSU setup.sh is designed for this exact pattern
- Kernel 6.1 + KPROBES is fully compatible
- GKI source is local (`aosp/`) and can be modified
- This matches sovereign_vault.md architecture expectations

### Implementation Steps

1. **Create symlink:** `aosp/drivers/kernelsu` → `../../KernelSU/kernel`
2. **Patch drivers/Makefile:** Add `obj-$(CONFIG_KSU) += kernelsu/`
3. **Patch drivers/Kconfig:** Add `source "drivers/kernelsu/Kconfig"` before endmenu
4. **Add defconfig fragment:** Create `private/devices/google/raviole/kernelsu.fragment`
5. **Update BUILD.bazel:** Include fragment in `defconfig_fragments`
6. **Build:** Run `./build_raviole.sh`
7. **Test:** Flash and verify KernelSU manager detects root

---

## 4. File Changes Required

| File | Change Type | Description |
|------|-------------|-------------|
| `aosp/drivers/kernelsu` | NEW (symlink) | Symlink to `../../KernelSU/kernel` |
| `aosp/drivers/Makefile` | PATCH | Add `obj-$(CONFIG_KSU) += kernelsu/` |
| `aosp/drivers/Kconfig` | PATCH | Add `source "drivers/kernelsu/Kconfig"` |
| `private/devices/google/raviole/kernelsu.fragment` | NEW | `CONFIG_KSU=y` |
| `private/devices/google/raviole/BUILD.bazel` | PATCH | Add fragment to defconfig_fragments |

---

## 5. Behavioral Decisions

### Q1: Should KernelSU be built as module (M) or built-in (Y)?

**Options:**
- `CONFIG_KSU=y` - Built into kernel (recommended)
- `CONFIG_KSU=m` - Loadable module

**Recommendation:** `CONFIG_KSU=y` (built-in)
- Root must be available at early boot for `/data/adb/service.d/` scripts
- Module loading happens after init, too late for Sovereign Vault

### Q2: Should KernelSU debug mode be enabled?

**Options:**
- `CONFIG_KSU_DEBUG=y` - Extra logging
- `CONFIG_KSU_DEBUG=n` - Production mode (recommended)

**Recommendation:** `CONFIG_KSU_DEBUG=n`
- Debug mode may leak sensitive information
- Production deployment should be quiet

### Q3: What LOCALVERSION suffix to use?

**Options:**
- `-sovereign` (matches sovereign_vault.md)
- `-ksu`
- Empty

**Recommendation:** `-sovereign`
- Identifies the kernel as Sovereign Vault build
- Matches existing documentation

---

## 6. Design Decisions (APPROVED)

> 🤖 **AI Note:** These decisions have been approved. Do NOT change them without explicit USER permission. If you think a decision is wrong, create a question file in `.questions/` — do not silently "fix" it.

| ID | Question | Decision | Rationale |
|----|----------|----------|-----------|
| Q1 | Integration method | **Option A** (symlink in aosp/drivers) | Standard KernelSU pattern, proven approach |
| Q2 | Build type | **CONFIG_KSU=y** (built-in) | Root needed at early boot for service.d scripts |
| Q3 | Debug mode | **CONFIG_KSU_DEBUG=n** | Production build, no debug logging |
| Q4 | LOCALVERSION | **"-sovereign"** | Identifies kernel as Sovereign Vault build |
| Q5 | Guest kernel | **Deferred to Phase 4** | Focus on host kernel first |

---

## 7. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| KernelSU breaks KMI | Low | High | KPROBES avoids direct symbol patches |
| Build fails | Medium | Medium | Incremental testing, rollback plan |
| Boot loop | Low | High | Test on secondary device first |
| SafetyNet detection | Medium | Low | Not a project goal; acceptable |

---

## 8. Design Checklist

- [x] Integration method selected (Option A)
- [x] File changes documented
- [x] Behavioral decisions proposed
- [ ] USER approval on open questions
- [ ] Ready for Phase 3 (Implementation)

---

## Next Phase

After USER answers questions Q1-Q5, proceed to **Phase 3 — Implementation**.
