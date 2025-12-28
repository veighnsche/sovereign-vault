# TEAM_009: Plan Review - CLI Refactor

## Task
Review and refine the CLI refactor plan in `.planning/cli-refactor/`

## Review Status
- [x] Phase 1: Questions and Answers Audit
- [x] Phase 2: Scope and Complexity Check
- [x] Phase 3: Architecture Alignment
- [x] Phase 4: Global Rules Compliance
- [x] Phase 5: Verification and References
- [x] Phase 6: Final Refinements

## Summary

The CLI refactor plan is **solid overall** with good structural design. The VM interface abstraction is well-designed and the file split is logical. However, several issues were identified and corrected.

## Issues Found and Fixed

### Critical (Fixed)
| Issue | Fix Applied |
|-------|-------------|
| No behavioral regression tests (Rule 4 violation) | Added Step 1b to Phase 1 with baseline capture |
| Speculative `--vault`/`--forge` flags | Removed from Phase 3 scope |
| Template file is dead code (Rule 6 violation) | Removed from Phase 4 scope |

### Important (Fixed)
| Issue | Fix Applied |
|-------|-------------|
| `createSQLStartScript` missing from extraction list | Added to Phase 2.8 |
| Handoff checklist missing regression verification | Updated Phase 4.7 |

### Minor (Not Fixed - Implementation Decision)
| Issue | Recommendation |
|-------|----------------|
| `commands.go` naming | Consider keeping dispatch in main.go (~60 lines) |
| `Prepare()` forces no-op implementations | Consider optional `Preparable` interface |

## Files Modified
- `.planning/cli-refactor/phase-1.md` - Added behavioral baseline step
- `.planning/cli-refactor/phase-2.md` - Added createSQLStartScript to extraction
- `.planning/cli-refactor/phase-3.md` - Removed speculative future flags
- `.planning/cli-refactor/phase-4.md` - Removed template file, updated checklist

## Verdict

**APPROVED with corrections applied.** Plan is ready for implementation.

## Handoff Notes for Implementation Team

1. **Start with Phase 1** - Capture baselines BEFORE any code changes
2. **Build after each extraction** - `go build` must pass after every step
3. **Verify regression** - `go run main.go help | diff - baselines/help.txt`
4. **Scope discipline** - Only refactor existing behavior, no new features
