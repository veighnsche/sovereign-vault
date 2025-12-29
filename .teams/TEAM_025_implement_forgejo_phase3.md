# TEAM_025: Implement Forgejo Phase 3

**Created:** 2025-12-29
**Status:** ✅ COMPLETE
**Task:** Implement Phase 3 of Forgejo VM plan (Implementation)
**Plan:** `.plans/forgejo-implementation/phase-3.md`

## Scope

Implementing all 4 steps:
1. Step 1: Create `vm/forgejo/init.sh`
2. Step 2: Rewrite `vm/forgejo/start.sh` 
3. Step 3: Update `vm/forgejo/Dockerfile`
4. Step 4: Refactor `internal/vm/forge/*.go`

## Progress

- [x] Step 1: init.sh - Created 230-line init script modeled after sql/init.sh
- [x] Step 2: start.sh - Rewrote to use TAP networking (94 lines)
- [x] Step 3: Dockerfile - Updated to use static Tailscale, removed OpenRC
- [x] Step 4: Go code refactor - Split into forge.go, lifecycle.go, verify.go
- [x] Fix app.ini db host (sql-vm → sovereign-sql)
- [x] Fix rootfs.go to be VM-agnostic (supports both SQL and Forge)
- [x] Verify build compiles

## Files Changed

| File | Change |
|------|--------|
| `vm/forgejo/init.sh` | NEW - 230 lines, TAP 192.168.101.2, persistent Tailscale |
| `vm/forgejo/start.sh` | REWRITE - TAP vm_forge, console=ttyS0, no gvproxy |
| `vm/forgejo/Dockerfile` | UPDATE - static Tailscale 1.78.3, no OpenRC, no CMD |
| `internal/vm/forge/forge.go` | REFACTOR - fixed paths, removed gvproxy |
| `internal/vm/forge/lifecycle.go` | NEW - Start/Stop/Remove with TAP cleanup |
| `internal/vm/forge/verify.go` | NEW - Test + RemoveTailscaleRegistrations |
| `internal/rootfs/rootfs.go` | UPDATE - VM-agnostic init script selection |
| `vm/forgejo/config/app.ini` | FIX - HOST=sovereign-sql:5432 |

## Verification

All checks passed:
- ✓ Uses TAP IP 192.168.101.2 (guest)
- ✓ Uses hostname sovereign-forge
- ✓ Uses TAP interface vm_forge
- ✓ Uses console=ttyS0
- ✓ Uses init=/sbin/init.sh
- ✓ No gvproxy references
- ✓ No OpenRC in Dockerfile
- ✓ Build compiles cleanly

## Additional Work

### Re-enabled forge.feature BDD file
- Renamed `features/forge.feature.disabled` → `features/forge.feature`
- Rewrote with 50+ scenarios mirroring `sql.feature` structure
- Covers: BUILD, DEPLOY, START, STOP, TEST, REMOVE, MULTI-VM behaviors
- Key Forgejo-specific scenarios:
  - Database dependency on sovereign-sql
  - Ports 3000 (web) + 22 (SSH)
  - TAP interface vm_forge at 192.168.101.x

## Next Steps

→ Phase 4: Integration & Testing (`.plans/forgejo-implementation/phase-4.md`)

## Notes

- Reference: SQL VM files are the gold standard
- TAP network: 192.168.101.x (not 192.168.100.x which is SQL)
- Hostname: sovereign-forge (not forge-vm)
- Device path: /data/sovereign/vm/forgejo/ (not /data/sovereign/forgejo/)
