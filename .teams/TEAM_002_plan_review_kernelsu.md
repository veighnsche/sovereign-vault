# TEAM_002 — Plan Review: KernelSU Integration

**Created:** 2024-12-27
**Status:** Complete
**Task:** Review and refine KernelSU integration plan; fix version numbering issue

---

## Mission

1. Review the KernelSU integration plan per the /review-a-plan workflow
2. Identify and fix the KernelSU version numbering issue (currently defaults to 16)

---

## Critical Finding: KernelSU Version Issue

### Problem
When KernelSU is integrated via symlink (`aosp/drivers/kernelsu` → `KernelSU/kernel`), the Kbuild version detection fails because:
- It looks for `.git` at `aosp/drivers/.git` (doesn't exist)
- `$(MDIR)/../.git` check may fail in Bazel sandboxed builds
- Falls back to `KSU_VERSION=16` (hardcoded default)

### Root Cause (from `KernelSU/kernel/Kbuild:33-55`)
```makefile
ifeq ($(shell test -e $(srctree)/$(src)/../.git && echo "in-tree"),in-tree)
  # git version detection
else ifeq ($(shell test -e $(MDIR)/../.git && echo "out-of-tree"),out-of-tree)
  # git version detection
else
  ccflags-y += -DKSU_VERSION=16  # <-- FALLBACK
endif
```

### Solution Applied
Added **Step 1** to Phase 3: Patch `KernelSU/kernel/Kbuild` to use explicit version.

**Current version:** 30000 + 2245 commits = **32245**

Two approaches documented:
1. **Direct patch:** Replace `16` with `32245` in Kbuild
2. **Version file:** Create `.ksu_version` file and patch Kbuild to read it

---

## Progress Log

| Date       | Action                                      |
|------------|---------------------------------------------|
| 2024-12-27 | Team registered, beginning plan review      |
| 2024-12-27 | Identified KSU_VERSION symlink issue        |
| 2024-12-27 | Analyzed Kbuild version detection logic     |
| 2024-12-27 | Added Step 1 to Phase 3 for version fix     |
| 2024-12-27 | Updated Phase 4 TC-2 with version verification |
| 2024-12-27 | Review complete                             |

---

## Review Findings Summary

### Phase 1 — Questions and Answers Audit ✓
- `.questions/` directory is empty
- No open questions to reconcile

### Phase 2 — Scope and Complexity Check ✓
- 5 phases: appropriate for kernel work
- 7 steps in Phase 3: reasonable granularity
- 6 test cases: comprehensive coverage
- **Issue found:** Version numbering not addressed → **FIXED**

### Phase 3 — Architecture Alignment ✓
- Symlink approach (Option A) is sound
- Follows KernelSU standard integration pattern
- **Issue found:** Symlink breaks version detection → **FIXED**

### Phase 4 — Global Rules Compliance ✓
- Rule 0 (Quality): Version fix addresses quality gap
- Rule 1 (SSOT): Plan in correct location
- Rule 2 (Team Registration): TEAM_001 registered
- Rule 4 (Regression Protection): TC-6 covers regressions
- Rule 11 (TODO Tracking): Version maintenance documented

### Phase 5 — Verification and References ✓
- Verified: KernelSU version = 30000 + git_commit_count
- Current commit count: 2245
- Correct version: 32245
- Incorrect fallback: 16

### Phase 6 — Final Refinements ✓
- Added Step 1 to Phase 3 for version fix
- Updated Phase 4 TC-2 with version verification
- Added failure mode for "version shows 16"

---

## Files Modified

| File | Change |
|------|--------|
| `.planning/kernelsu-integration/phase-3.md` | **REWRITTEN:** sovereign CLI as foundation, 6 steps |
| `.planning/kernelsu-integration/phase-3-step-1.md` | **REWRITTEN:** Create sovereign CLI (THE FOUNDATION) |
| `.planning/kernelsu-integration/phase-3-step-2.md` | **REWRITTEN:** Kernel Integration (KernelSU patches) |
| `.planning/kernelsu-integration/phase-3-step-3.md` | **REWRITTEN:** VM Components Setup (sql, vault, forge) |
| `.planning/kernelsu-integration/phase-3-step-4.md` | **REWRITTEN:** Build (`sovereign build --all`) |
| `.planning/kernelsu-integration/phase-3-step-5.md` | **REWRITTEN:** Deploy (`sovereign deploy --all`) |
| `.planning/kernelsu-integration/phase-3-step-6.md` | **REWRITTEN:** Verify (`sovereign test --all`) |
| `.planning/kernelsu-integration/phase-3-step-7.md` | DELETED (consolidated) |
| `.planning/kernelsu-integration/phase-3-step-8.md` | DELETED (consolidated) |
| `.planning/kernelsu-integration/phase-4.md` | Added version verification to TC-2 |

---

## Handoff Notes

### For TEAM_001 (Implementation)

**INCREMENTAL APPROACH:** Build one thing at a time. Get it working. Then move on.

**Phase 3A: KernelSU (Steps 1-3)**
1. **Step 1:** Create sovereign CLI (kernel-only)
2. **Step 2:** KernelSU Integration (patches)
3. **Step 3:** Build, Deploy, Verify Root

**Phase 3B: PostgreSQL (Steps 4-6)**
4. **Step 4:** PostgreSQL VM Setup
5. **Step 5:** Deploy & Start VM
6. **Step 6:** Tailscale + Verify

**Phase 3C & 3D:** Vaultwarden and Forgejo (future)

**Current sovereign commands:**
```bash
# Phase 3A
sovereign build --kernel
sovereign deploy --kernel
sovereign test --kernel

# Phase 3B (added after 3A complete)
sovereign build --sql
sovereign deploy --sql
sovereign start --sql
sovereign test --sql
sovereign stop --sql
```

### Maintenance
When updating KernelSU (`git pull`), recalculate and update version:
```bash
cd KernelSU && echo $((30000 + $(git rev-list --count HEAD)))
```

---

## Checklist

- [x] Project builds cleanly (N/A - review only)
- [x] Plan files updated
- [x] Team file updated
- [x] Version fix documented
