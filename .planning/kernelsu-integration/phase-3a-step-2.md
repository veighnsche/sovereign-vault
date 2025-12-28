# Phase 3A, Step 2 — KernelSU Integration (Patches)

**Phase:** 3A (KernelSU)
**Step:** 2 of 3
**Team:** TEAM_001
**Status:** Pending
**Depends On:** Step 1

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** This step involves patching kernel build files. I am prone to:
> - Skipping the version patch because "16 is probably fine" (it's NOT fine - the manager will show wrong version)
> - Creating absolute symlinks that break on other machines
> - Not verifying each patch actually applied
> - Moving on before confirming the patches are correct
>
> **The rule:** Each task has a verification step. I do not proceed until verification passes.
>
> **Version 16 means failure.** If KernelSU reports version 16, the Kbuild patch was not applied. This is not acceptable.

---

## 1. Goal

Apply all kernel patches needed for KernelSU. After this step, `sovereign build --kernel` will produce a kernel with KernelSU.

---

## 2. Pre-Conditions

- [ ] Step 1 complete (`sovereign.go` exists)
- [ ] `KernelSU/kernel/` directory exists
- [ ] `aosp/drivers/Makefile` exists

---

## 3. Task 1: Fix KernelSU Version

**Problem:** Symlink breaks git-based version detection → falls back to version 16

```bash
# Get current commit count
cd KernelSU && git rev-list --count HEAD
# Example output: 2245

# Calculate version: 30000 + commit_count
# Example: 30000 + 2245 = 32245
```

**Patch `KernelSU/kernel/Kbuild` (around line 54):**

Find:
```makefile
ccflags-y += -DKSU_VERSION=16
```

Replace with:
```makefile
ccflags-y += -DKSU_VERSION=32245
```

**Verify:**
```bash
grep "KSU_VERSION=32245" KernelSU/kernel/Kbuild
# Must return the patched line
```

> 🤖 **AI Warning:** If this grep returns nothing, the patch failed. Do NOT proceed.

---

## 4. Task 2: Create Symlink

```bash
# Create relative symlink (NOT absolute)
ln -sf "$(realpath --relative-to=aosp/drivers KernelSU/kernel)" aosp/drivers/kernelsu
```

**Verify:**
```bash
ls -la aosp/drivers/kernelsu
# Expected: kernelsu -> ../../KernelSU/kernel

# Also verify it resolves correctly
ls aosp/drivers/kernelsu/Kconfig
# Must show the Kconfig file
```

> 🤖 **AI Warning:** If the symlink is absolute (`/home/...`), it will break for other users. Use relative paths.

---

## 5. Task 3: Patch drivers/Makefile

```bash
# Add KernelSU to drivers Makefile
echo 'obj-$(CONFIG_KSU)		+= kernelsu/' >> aosp/drivers/Makefile
```

**Verify:**
```bash
grep "kernelsu" aosp/drivers/Makefile
# Expected: obj-$(CONFIG_KSU)		+= kernelsu/
```

---

## 6. Task 4: Patch drivers/Kconfig

```bash
# Find the last 'endmenu' line number
LINE=$(grep -n "endmenu" aosp/drivers/Kconfig | tail -1 | cut -d: -f1)

# Insert KernelSU Kconfig source before it
sed -i "${LINE}i\source \"drivers/kernelsu/Kconfig\"" aosp/drivers/Kconfig
```

**Verify:**
```bash
grep "kernelsu" aosp/drivers/Kconfig
# Expected: source "drivers/kernelsu/Kconfig"
```

---

## 7. Task 5: Create Defconfig Fragment

**File:** `private/devices/google/raviole/kernelsu.fragment`

```
CONFIG_KSU=y
CONFIG_LOCALVERSION="-sovereign"
```

**Verify:**
```bash
cat private/devices/google/raviole/kernelsu.fragment
```

---

## 8. Task 6: Update BUILD.bazel

Add the fragment to `defconfig_fragments` in `private/devices/google/raviole/BUILD.bazel`:

Find the `defconfig_fragments` list and add:
```starlark
"kernelsu.fragment",
```

**Verify:**
```bash
grep "kernelsu.fragment" private/devices/google/raviole/BUILD.bazel
```

---

## 9. All Verifications (Run All)

```bash
echo "=== Verification Checklist ==="

echo -n "1. KSU_VERSION patched: "
grep -q "KSU_VERSION=32245" KernelSU/kernel/Kbuild && echo "✓" || echo "✗ FAIL"

echo -n "2. Symlink exists: "
[ -L aosp/drivers/kernelsu ] && echo "✓" || echo "✗ FAIL"

echo -n "3. Symlink resolves: "
[ -f aosp/drivers/kernelsu/Kconfig ] && echo "✓" || echo "✗ FAIL"

echo -n "4. Makefile patched: "
grep -q "kernelsu" aosp/drivers/Makefile && echo "✓" || echo "✗ FAIL"

echo -n "5. Kconfig patched: "
grep -q "kernelsu" aosp/drivers/Kconfig && echo "✓" || echo "✗ FAIL"

echo -n "6. Fragment exists: "
[ -f private/devices/google/raviole/kernelsu.fragment ] && echo "✓" || echo "✗ FAIL"

echo -n "7. BUILD.bazel updated: "
grep -q "kernelsu.fragment" private/devices/google/raviole/BUILD.bazel && echo "✓" || echo "✗ FAIL"

echo "=== End Checklist ==="
```

> 🤖 **AI Warning:** ALL checks must pass. If ANY check shows "✗ FAIL", fix it before proceeding.

---

## 10. Checkpoint

- [ ] Version patched (32245, not 16)
- [ ] Symlink created (relative, not absolute)
- [ ] Symlink resolves to actual files
- [ ] Makefile patched
- [ ] Kconfig patched
- [ ] Fragment created
- [ ] BUILD.bazel updated
- [ ] All verification checks pass

---

## Next Step

Proceed to **[Phase 3A, Step 3 — Build, Deploy, Verify Root](phase-3a-step-3.md)**
