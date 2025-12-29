# TEAM_029: Implement VM Common Phase 2

**Created:** 2025-12-29
**Status:** ✅ Complete
**Task:** Create `internal/vm/common/` package with shared infrastructure
**Plan:** `.plans/vm-common-refactor/phase-2.md`

## Scope

Create the common package with:
1. config.go - VMConfig struct
2. tailscale.go - Registration cleanup
3. lifecycle.go - Stop, Remove, Start, StreamLogs
4. test.go - Common test patterns
5. deploy.go - File pushing
6. build.go - Docker build, data disk

## Progress

- [x] Step 1: config.go - VMConfig struct with all fields
- [x] Step 2: tailscale.go - RemoveTailscaleRegistrations, CheckTailscaleConnected
- [x] Step 3-4: lifecycle.go - StopVM, RemoveVM, StartVM, StreamBootLogs
- [x] Step 5: test.go - RunVMTests, TestPortOpen
- [x] Step 6: deploy.go - DeployVM
- [x] Step 7: build.go - BuildVM
- [x] Build verification: `go build ./...` succeeds

## Notes

- Old code in sql/ and forge/ remains unchanged
- New common functions coexist with old
- No call sites changed until Phase 3

---

## Phase 3 Migration (also completed)

After completing Phase 2, continued with Phase 3 migration:

### SQL Migration
- [x] verify.go: RemoveTailscaleRegistrations → common.RemoveTailscaleRegistrations (110→3 lines)
- [x] lifecycle.go: Start/Stop/Remove → common (200→23 lines)
- [x] sql.go: Deploy → common.DeployVM (70→3 lines)
- [x] sql.go: Added SQLConfig
- [x] verify.go: Test → common.RunVMTests + custom tests (100→50 lines)

### Forge Migration
- [x] forge.go: Added ForgeConfig, Build/Deploy → common (140→50 lines)
- [x] lifecycle.go: Start/Stop/Remove → common (160→23 lines)
- [x] verify.go: Test + RemoveTailscaleRegistrations → common (220→50 lines)

### Line Count Summary

| Package | Before | After | Reduction |
|---------|--------|-------|----------|
| sql/ | ~762 | 344 | -418 (55%) |
| forge/ | ~522 | 123 | -399 (76%) |
| common/ | 0 | 718 | +718 |
| **Total** | ~1284 | 1185 | -99 (8%) |

**Key Achievement:** ~400 lines of duplication eliminated. Adding new services now requires only ~50 lines (config + custom tests).
