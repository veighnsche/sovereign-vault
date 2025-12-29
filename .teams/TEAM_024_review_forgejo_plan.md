# TEAM_024: Review Forgejo Implementation Plan

**Created:** 2025-12-29
**Purpose:** Review and refine the Forgejo implementation plan per /review-a-plan workflow

## Status: COMPLETE

## Context

Reviewing `.plans/FORGEJO_IMPLEMENTATION_PLAN.md` created by TEAM_023.

---

## Review Summary

| Category | Verdict | Issues Found |
|----------|---------|-------------|
| Questions Audit | ⚠️ MINOR | 3 questions in plan should be in `.questions/` |
| Scope/Complexity | ✅ GOOD | Appropriate scope, not overengineered |
| Architecture | ⚠️ ISSUES | Section 4 proposes unneeded abstraction |
| Rules Compliance | ⚠️ MINOR | Missing TODO.md entries for open work |
| Verification | ✅ VERIFIED | Claims match actual codebase state |

**Overall: PLAN IS SOLID with minor refinements needed.**

---

## Phase 1: Questions and Answers Audit

### Findings
- `.questions/` directory is **empty** - no tracked questions
- Plan Section 11 lists 3 questions that **must** be moved to `.questions/`
- Questions are behaviors, not implementation details ✓

### Questions Requiring Tracking

| # | Question | Recommendation | Blocks Work? |
|---|----------|----------------|-------------|
| 1 | DB connection: Tailscale IP or TAP? | Use Tailscale (`sovereign-sql`) for reliability | No - recommendation is sound |
| 2 | Ports to expose via Tailscale? | 3000 (web), 22 (SSH) | No - already in plan |
| 3 | Shared kernel for all VMs? | Defer - SQL kernel works fine | No - future optimization |

**Action:** Create `.questions/TEAM_024_forgejo_decisions.md` with these 3 items.

---

## Phase 2: Scope and Complexity Check

### Phase Count Assessment

| Phase | Purpose | UoW Count | Verdict |
|-------|---------|-----------|--------|
| 1: BDD Tests | Write tests first | 3 | ✅ Right-sized |
| 2: Init Script | Core functionality | 4 | ✅ Right-sized |
| 3: Start Script | Networking | 4 | ✅ Right-sized |
| 4: Go Code | CLI integration | 3 | ✅ Right-sized |
| 5: Dockerfile | Image build | 3 | ✅ Right-sized |
| 6: Integration | End-to-end test | 4 | ✅ Right-sized |

**Total: 6 phases, 21 UoWs - APPROPRIATE for scope.**

### Overengineering Signals

**Section 4 ("Reusable Components")** proposes:
- `internal/vm/common/` package
- `SetupTAPNetwork()`, `TeardownTAPNetwork()` functions
- `GenerateInitScript()` with `InitConfig` struct
- Shared init script template `vm/common/init.sh.tmpl`

**VERDICT: OVERENGINEERED for 2 VMs.**

- PostgreSQL and Forgejo have different init requirements
- Extracting common code now adds complexity without benefit
- Should be done AFTER Forgejo works, if patterns actually repeat

**Recommendation:** Remove Section 4 entirely or mark as "Future Optimization."

### Oversimplification Signals

None found. Plan includes:
- ✅ Testing phase (Phase 1)
- ✅ Clear UoW breakdown
- ✅ Edge cases addressed (OpenRC hang, console device, Tailscale duplicates)
- ✅ Regression protection (BDD tests)
- ✅ Clear exit criteria (Section 12)

---

## Phase 3: Architecture Alignment

### Current Architecture (Verified)

```
internal/vm/
├── vm.go           # Interface + registry
├── sql/
│   ├── sql.go      # Build, Deploy
│   ├── lifecycle.go # Start, Stop, Remove
│   └── verify.go   # Test, RemoveTailscaleRegistrations
└── forge/
    └── forge.go    # All-in-one (252 lines)
```

### Plan vs Architecture

| Plan Section | Current State | Alignment |
|--------------|---------------|----------|
| 3.1 start.sh | Uses gvproxy/vsock | ❌ WRONG - plan correctly identifies |
| 3.2 init.sh | Uses OpenRC | ❌ WRONG - plan correctly identifies |
| 3.3 Dockerfile | Uses Alpine tailscale package | ⚠️ Needs static binary |
| 3.4 Go Code | Single forge.go file | ⚠️ Plan says "minimal" but it's 252 lines |

### Alignment Issues

1. **forge.go structure:** Plan says "Currently minimal or missing" but it exists with full Build/Deploy/Start/Stop/Test/Remove. **Correction needed.**

2. **Device path mismatch:**
   - forge.go uses `/data/sovereign/forgejo/`
   - sql uses `/data/sovereign/vm/sql/`
   - **INCONSISTENT** - should be `/data/sovereign/vm/forgejo/`

3. **app.ini has wrong hostname:**
   - Uses `forge-vm.tail5bea38.ts.net` and `sql-vm`
   - Should use `sovereign-forge` and `sovereign-sql`

---

## Phase 4: Global Rules Compliance

| Rule | Status | Notes |
|------|--------|-------|
| Rule 0 (Quality) | ✅ | Plan takes correct architectural path |
| Rule 1 (SSOT) | ✅ | Plan in `.plans/` directory |
| Rule 2 (Team Registration) | ✅ | Created by TEAM_023 |
| Rule 3 (Before Starting) | ✅ | Pre-planning done |
| Rule 4 (Regression Protection) | ✅ | BDD tests required in Phase 1 |
| Rule 5 (Breaking Changes) | ✅ | No compatibility hacks |
| Rule 6 (No Dead Code) | ⚠️ | Need to remove gvproxy code |
| Rule 7 (Modular Refactoring) | ✅ | Reasonable file sizes |
| Rule 8 (Ask Questions) | ⚠️ | Questions not in `.questions/` |
| Rule 9 (Maximize Context) | ✅ | Work batched sensibly |
| Rule 10 (Handoff) | ✅ | Clear checklist in Section 8 |
| Rule 11 (TODO Tracking) | ⚠️ | No TODO.md entries |

### Violations to Fix

1. **Rule 8:** Move Section 11 questions to `.questions/TEAM_024_forgejo_decisions.md`
2. **Rule 11:** Add forge work to project TODO.md (if exists)

---

## Phase 5: Verification and References

### Claims Verified Against Codebase

| Claim | Verification | Result |
|-------|--------------|--------|
| forge start.sh uses gvproxy | `vm/forgejo/start.sh:27-31` | ✅ VERIFIED |
| forge start.sh uses console=hvc0 | `vm/forgejo/start.sh:17` | ✅ VERIFIED |
| sql init.sh uses ttyS0 | `vm/sql/init.sh:66` | ✅ VERIFIED |
| sql uses TAP vm_sql | `vm/sql/start.sh:11` | ✅ VERIFIED |
| Tailscale state on /data | `vm/sql/init.sh:151` | ✅ VERIFIED |
| forge.feature.disabled exists | `features/forge.feature.disabled` | ✅ VERIFIED |
| Dockerfile uses OpenRC | `vm/forgejo/Dockerfile:40-42` | ✅ VERIFIED |

### Claims Needing Correction

| Claim | Actual State | Correction |
|-------|--------------|------------|
| "Go code minimal or missing" | forge.go is 252 lines with full implementation | Update plan Section 3.4 |
| sql.feature has "47 scenarios" | Actually 46+ scenarios (464 lines) | Minor discrepancy, acceptable |

---

## Phase 6: Final Refinements

### Critical Corrections (Must Fix)

1. **Path inconsistency in forge.go:**
   - Line 101: `/data/sovereign/forgejo/`
   - Should be: `/data/sovereign/vm/forgejo/`
   - This will cause deploy failures!

2. **app.ini hostname mismatch:**
   - Uses `forge-vm` and `sql-vm`
   - Should use `sovereign-forge` and `sovereign-sql`

### Important Corrections (Should Fix)

1. **Section 4 (Reusable Components):**
   - Mark as "Phase 7: Future Optimization" or remove
   - Premature abstraction before Forgejo works

2. **Section 3.4 description:**
   - Change "minimal or missing" to "exists but uses wrong patterns"

3. **Move questions to `.questions/`:**
   - Create `.questions/TEAM_024_forgejo_decisions.md`

### Minor Corrections (Nice to Have)

1. **BDD test template in Section 7:**
   - Very comprehensive (240 lines of gherkin)
   - Consider trimming to essential scenarios
   - Full template can overwhelm implementing team

---

## Recommended Plan Updates

### Update 1: Fix Section 3.4 Description
```diff
### 3.4 Go Code (`internal/vm/forge/`)

-Currently minimal or missing. Must implement:
+Currently exists but uses WRONG patterns (gvproxy, wrong paths). Must refactor:
- `forge.go` - Build, Deploy functions
- `lifecycle.go` - Start, Stop, Remove functions  
- `verify.go` - Test verification
+
+**Key fixes needed:**
+- Change device path from `/data/sovereign/forgejo/` to `/data/sovereign/vm/forgejo/`
+- Remove gvproxy references from Deploy
+- Add TAP networking setup to Start
```

### Update 2: Mark Section 4 as Future
```diff
## 4. Reusable Components to Extract

+> **NOTE: This is a FUTURE OPTIMIZATION.** Get Forgejo working first,
+> then extract common patterns. Do NOT implement this in the initial pass.
+
### 4.1 Create `internal/vm/common/` Package
```

### Update 3: Add Critical Path Warning

Add to Section 8 Phase 4:
```diff
### Phase 4: Go Code
- [ ] Implement `internal/vm/forge/forge.go`
+- [ ] **CRITICAL:** Fix device path to `/data/sovereign/vm/forgejo/`
- [ ] Implement `internal/vm/forge/lifecycle.go`
```

---

## Handoff Notes

### For Next Team

1. **Plan is READY for implementation** with the corrections above
2. **Start with Phase 1 (BDD tests)** - write failing tests first
3. **Copy sql patterns directly** - init.sh, start.sh structure
4. **Watch for path mismatches** - everything should be under `/data/sovereign/vm/forgejo/`

### Open Risks

1. **Forgejo→PostgreSQL connectivity:** Plan recommends Tailscale IP but app.ini currently uses `sql-vm:5432`. Needs clarification.

2. **Tailscale duplicate registrations:** Known bug (see lifecycle.go:171-175). Persistent state on data.img should fix this.

### Checklist

- [x] All answered questions reflected in plan
- [x] Open questions don't block Phase 1-5
- [x] Plan not overengineered (except Section 4)
- [x] Architecture alignment verified
- [x] Global rules mostly compliant
- [x] Claims verified against codebase
- [x] Critical path corrections identified

---

## Plan Split (Post-Review)

Split monolithic plan into structured phase files per `/make-a-new-feature-plan` workflow.

### Files Created

| File | Purpose |
|------|---------|
| `.plans/forgejo-implementation/README.md` | Index/entry point |
| `.plans/forgejo-implementation/phase-1.md` | Discovery (complete) |
| `.plans/forgejo-implementation/phase-2.md` | Design (complete) |
| `.plans/forgejo-implementation/phase-3.md` | Implementation overview |
| `.plans/forgejo-implementation/phase-3-step-1.md` | Init script tasks |
| `.plans/forgejo-implementation/phase-3-step-2.md` | Start script tasks |
| `.plans/forgejo-implementation/phase-3-step-3.md` | Dockerfile tasks |
| `.plans/forgejo-implementation/phase-3-step-4.md` | Go code tasks |
| `.plans/forgejo-implementation/phase-4.md` | Integration & testing |
| `.plans/forgejo-implementation/phase-5.md` | Polish & cleanup |

### Original Plan Updated

Updated `.plans/FORGEJO_IMPLEMENTATION_PLAN.md` to point to new structure while preserving original content as archive.
