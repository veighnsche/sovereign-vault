# Phase 4: Cleanup

**Feature:** VM Common Code Refactor  
**Status:** 🔲 NOT STARTED  
**Parent:** [VM_COMMON_REFACTOR_PLAN.md](VM_COMMON_REFACTOR_PLAN.md)  
**Depends On:** [Phase 3](phase-3.md)

---

## Purpose

Remove dead code, temporary adapters, and tighten encapsulation.

---

## Dead Code Removal (Rule 6)

### SQL Package - Expected Deletions

| File | Functions to Delete | Lines |
|------|---------------------|-------|
| lifecycle.go | `streamBootAndWaitForPostgres()` | ~70 |
| verify.go | `RemoveTailscaleRegistrations()` | ~110 |
| verify.go | `checkTailscaleRegistration()` | ~45 |

**Expected reduction:** ~225 lines

### Forge Package - Expected Deletions

| File | Functions to Delete | Lines |
|------|---------------------|-------|
| lifecycle.go | `streamBootAndWaitForForgejo()` | ~50 |
| verify.go | `RemoveTailscaleRegistrations()` | ~110 |

**Expected reduction:** ~160 lines

### Total Expected Reduction

~385 lines of duplicated code removed.

---

## Temporary Adapter Removal

No temporary adapters expected if Phase 3 follows clean break strategy.

---

## Encapsulation Tightening

### Make Functions Private

After migration, these become internal to common package:
- `streamBootLogs` (not exported, called by StartVM)
- `cleanupNetworking` (not exported, called by StopVM)
- `findAPIKey` (not exported, called by RemoveTailscaleRegistrations)

### Remove Unnecessary Exports

SQL and Forge packages should only export:
- `VM` struct
- `init()` function (for registration)
- Possibly `ForceDeploySkipTailscaleCheck` flag

---

## File Size Check (Rule 7)

Target sizes after refactor:

| File | Expected Lines | Status |
|------|----------------|--------|
| common/config.go | ~80 | ✓ Good |
| common/tailscale.go | ~120 | ✓ Good |
| common/lifecycle.go | ~200 | ✓ Good |
| common/build.go | ~150 | ✓ Good |
| common/deploy.go | ~100 | ✓ Good |
| common/test.go | ~150 | ✓ Good |
| sql/sql.go | ~100 | ✓ Good (down from 234) |
| forge/forge.go | ~80 | ✓ Good (down from 142) |

All files under 500 lines (ideal).

---

## Steps

| Step | File | Description |
|------|------|-------------|
| 1 | [phase-4-step-1.md](phase-4-step-1.md) | Delete dead code from SQL package |
| 2 | [phase-4-step-2.md](phase-4-step-2.md) | Delete dead code from Forge package |
| 3 | [phase-4-step-3.md](phase-4-step-3.md) | Make internal functions private |
| 4 | [phase-4-step-4.md](phase-4-step-4.md) | Consolidate SQL/Forge to single files |

---

## Exit Criteria

- [ ] No dead code in SQL package
- [ ] No dead code in Forge package
- [ ] All files < 500 lines
- [ ] No unnecessary exports
- [ ] `go build ./...` succeeds
- [ ] All tests pass
- [ ] Ready for Phase 5 (hardening)
