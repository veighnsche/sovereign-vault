# TEAM_005 — Bypass Prebuilt GKI for KernelSU

**Created:** 2024-12-28
**Status:** Complete
**Task:** Trace build system source code to find all prebuilt GKI gates

---

## Mission

Build flags in `build_raviole.sh` are being ignored. Trace through the ACTUAL source code to find:
1. Where prebuilt GKI selection happens
2. ALL flags/conditions that trigger prebuilt usage
3. The correct way to force source build

---

## Progress Log

| Date       | Action                                      |
|------------|---------------------------------------------|
| 2024-12-28 | Team registered, investigating build system |
| 2024-12-28 | ROOT CAUSE FOUND - bazel.py wrapper bug |
| 2024-12-28 | FIX APPLIED - Using explicit label syntax |

---

## ROOT CAUSE ANALYSIS

### The Bug: `bazel.py` Wrapper String Interception

**Location:** `build/kernel/kleaf/bazel.py` lines 174-177 and 267-273

```python
# Line 174-177: Parser definition
group.add_argument(
    "--use_prebuilt_gki",
    metavar="BUILD_NUMBER",  # <-- Treated as STRING, not boolean!
    help="Use prebuilt GKI downloaded from ci.android.com")

# Line 267-273: Processing  
if self.known_args.use_prebuilt_gki:  # <-- "false" is TRUTHY string!
    self.transformed_command_args.append("--use_prebuilt_gki")  # <-- NO VALUE!
```

**What happens with `--use_prebuilt_gki=false`:**
1. Wrapper parses it as STRING `"false"` (not boolean False)
2. `if "false":` evaluates to `True` (non-empty string is truthy!)
3. Appends `--use_prebuilt_gki` WITHOUT the `=false` suffix
4. Bazel sees this as `--use_prebuilt_gki=true` (default for boolean flags)

### The Alias Chain

1. `//common:kernel_aarch64` → alias to `kernel_aarch64_download_or_build`
2. `kernel_aarch64_download_or_build` uses `select()` on `use_prebuilt_gki_set`
3. `use_prebuilt_gki_set` matches when `//build/kernel/kleaf:use_prebuilt_gki=true`

### File Trace

| File | Role |
|------|------|
| `build/kernel/kleaf/bazel.py:267-273` | BUG: Intercepts flag, strips `=false` |
| `build/kernel/kleaf/bazelrc/flags.bazelrc:63` | Alias: `--use_prebuilt_gki` → `//build/kernel/kleaf:use_prebuilt_gki` |
| `device.bazelrc:14` | Default: `--use_prebuilt_gki=true` |
| `build/kernel/kleaf/common_kernels.bzl:995-1000` | Select logic for download_or_build |
| `private/devices/google/common/kleaf/common.BUILD.bazel:58-64` | Alias kernel_aarch64 → download_or_build |

---

## THE FIX

### Solution: Explicit Label Syntax

Use `--//path:flag=value` format which **bypasses the bazel.py wrapper entirely**:

```bash
# WRONG (intercepted by wrapper):
--use_prebuilt_gki=false

# CORRECT (bypasses wrapper):
--//build/kernel/kleaf:use_prebuilt_gki=false
--//build/kernel/kleaf:use_signed_prebuilts=false  
--//private/devices/google/common:download_prebuilt_gki_fips140=false
```

### File Changed

**`build_raviole.sh`** - Updated to use explicit label syntax for all 3 GKI gates.

---

## Verification

After this fix, run:
```bash
./build_raviole.sh
```

The build should now compile GKI kernel from `aosp/` source instead of downloading prebuilts.

To verify source build is happening, check build output for:
- Compilation of `aosp/` kernel sources
- NO download of `gki_prebuilts` artifacts
- `CONFIG_KSU=y` in final `.config`

---

## Handoff Notes

- Root cause was in Python wrapper, not Bazel/Starlark code
- The wrapper's argparse treats the flag as a string argument, not boolean
- Alternative fix: Use `--config=use_source_tree_aosp` (but that changes kernel_package too)
- Explicit label syntax is cleanest solution - targets flags directly

---
