# TEAM_027: Review Phase 3 VM Common Refactor

**Created:** 2025-12-29
**Status:** ✅ Complete
**Task:** Review phase-3 of VM common refactor plan
**Plan:** `.plans/vm-common-refactor/phase-3.md`

## Scope

Phase 3 covers migration of SQL and Forge VMs to use the common package.

## Files Reviewed

- phase-3.md (parent)
- phase-3-step-1.md through phase-3-step-7.md
- VM_COMMON_REFACTOR_PLAN.md (overall plan)
- Actual source code: internal/vm/sql/*.go, internal/vm/forge/*.go

## Review Findings

### Critical Issues
None - plan is fundamentally sound.

### Important Issues

1. **Step 7 too large** - Bundles all 6 Forge migrations into one step (~440 lines). Should split into 2-3 atomic steps.

2. **No baseline lock-in** - Exit criteria say "All BDD scenarios pass" but no step to capture baseline before migration.

3. **Ambiguous createStartScript handling** - Step 4 says "Move to common or keep as helper" - should clarify it becomes generic in common.DeployVM.

### Minor Issues

4. Open questions in parent plan should move to `.questions/` file
5. No TODO.md update step mentioned
6. `.env` file check missing from Deploy prerequisites (SQL code has it, plan omits)

## Verdict

**APPROVED WITH CHANGES** - Apply the 3 important fixes before implementation.

## Changes Applied

1. **Split step-7 into 3 atomic steps:**
   - step-7: Forge config + Tailscale (focused)
   - step-8: Forge lifecycle (Start/Stop/Remove)
   - step-9: Forge Build/Deploy/Test + cleanup

2. **Added baseline verification to phase-3.md prerequisites:**
   - Run `godog` and verify all BDD scenarios pass
   - Save passing scenario list as baseline

3. **Clarified createStartScript handling in step-4:**
   - Remove entirely - logic replaced by generic common.DeployVM
   - Uses `cfg.LocalPath + "/start.sh"` pattern

## Handoff Checklist

- [x] All answered questions reflected in plan
- [x] Open questions don't block Phase 3 work
- [x] Scope is appropriate (not over/under engineered)
- [x] Architecture aligns with existing codebase
- [x] Global rules compliance verified
- [x] Claims verified against actual code
- [x] Changes applied to plan files
- [x] Team file updated
