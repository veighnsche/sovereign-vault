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

---

## VM Stop Hang Fix (2025-12-29)

### Problem
`sovereign stop --forge` was hanging indefinitely because ADB shell commands can block forever.

### Root Cause
`exec.Command(...).Output()` has no timeout - if the device or command hangs, the host CLI hangs too.

### Solution

1. **Added `RunShellCommand()` with 30s timeout** in `internal/device/device.go`:
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   out, err := exec.CommandContext(ctx, "adb", "shell", "su", "-c", cmd).Output()
   ```

2. **Added `RunShellCommandQuick()` with 5s timeout** for cleanup commands that should complete quickly:
   - Silently ignores timeouts (cleanup commands are best-effort)
   - Used for iptables rules, ip link deletion, etc.

3. **Cleanup commands use `|| true`** to prevent blocking on errors:
   ```go
   device.RunShellCommandQuick("ip link del vm_forge 2>/dev/null || true")
   ```

4. **Verify process death** in `StopVM()`:
   - After `kill`, wait 500ms
   - Check if process still exists
   - Force kill with `kill -9` if needed

### Files Modified
- `internal/device/device.go` - Added timeout functions
- `internal/vm/common/lifecycle.go` - Used quick timeout for cleanup

### Result
`sovereign stop --forge` now completes in ~1.2s instead of hanging forever.

### Documentation
- Added to `vm/forgejo/CHECKLIST.md` under "Common Issues"
- Added to `vm/sql/CHECKLIST.md` mistake #21
