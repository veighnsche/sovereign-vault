# TEAM_023: Forgejo Implementation Handoff

## Session Summary

**Date:** 2025-12-29
**Duration:** Extended session
**Objective:** Document Forgejo implementation plan after PostgreSQL success

---

## What We Accomplished

### 1. Fixed the Tailscale Duplicate Registration Bug

**The Bug:** Every restart/redeploy created a NEW Tailscale registration:
- `sovereign-sql` → `sovereign-sql-1` → `sovereign-sql-2` → ...

**The Fix (4 files changed):**

| File | Change |
|------|--------|
| `vm/sql/init.sh` | Mount `/dev/vdb` to `/data`, check state file before registering |
| `vm/sql/start.sh` | Pass `data.img` as second block device |
| `internal/vm/sql/lifecycle.go` | Remove cleanup calls from Start() |
| `internal/vm/sql/sql.go` | Remove cleanup calls from Deploy() |

**How it works now:**
- First boot: "No saved state, first-time registration..." → uses authkey
- Subsequent restarts: "Found persistent state, reconnecting..." → reuses identity

### 2. Fixed .env API Key Path Bug

The code was only reading `.env` from current directory. Fixed to check multiple paths.

### 3. Created Comprehensive Forgejo Implementation Plan

Created `.plans/FORGEJO_IMPLEMENTATION_PLAN.md` with:
- Gap analysis between PostgreSQL and Forgejo
- Complete BDD test template (40+ scenarios)
- Reusable component extraction plan
- Network subnet allocation scheme
- Implementation checklist
- Lessons learned

---

## Files Modified This Session

| File | Change |
|------|--------|
| `internal/vm/sql/verify.go` | Updated bug documentation, fixed .env path |
| `internal/vm/sql/lifecycle.go` | Removed Tailscale cleanup from Start(), added persistent identity |
| `internal/vm/sql/sql.go` | Removed Tailscale cleanup from Deploy() |
| `vm/sql/init.sh` | Added data disk mounting, persistent Tailscale state |
| `vm/sql/start.sh` | Added data.img as second block device |
| `.teams/TEAM_019_tailscale_idempotency.md` | Updated to reflect bug is FIXED |
| `.plans/FORGEJO_IMPLEMENTATION_PLAN.md` | Created comprehensive plan |

---

## For the Next Team

### Your Mission

Implement Forgejo VM using the same patterns as PostgreSQL. **Write BDD tests first.**

### Start Here

1. Read `.plans/FORGEJO_IMPLEMENTATION_PLAN.md` - it has everything
2. Rename `features/forge.feature.disabled` to `forge.feature`
3. Copy the BDD scenarios from the plan into the feature file
4. Implement step definitions
5. Follow the implementation checklist

### Key Patterns to Copy from PostgreSQL

1. **TAP networking** - NOT gvproxy/vsock
2. **Custom init.sh** - NOT OpenRC
3. **console=ttyS0** - NOT console=hvc0
4. **Persistent Tailscale state** on data.img
5. **Data disk mounting** at boot

### Don't Make These Mistakes Again

1. OpenRC hangs in AVF - use custom init.sh
2. hvc0 doesn't work with `--serial type=stdout` - use ttyS0
3. Tailscale state in rootfs = duplicates - put it on data.img
4. gvproxy/vsock doesn't work - use TAP networking
5. `ip rule add from all lookup main pref 1` is REQUIRED for NAT

---

## Test Results

After the Tailscale fix:
```
$ ./sovereign stop --sql && ./sovereign start --sql
Tailscale: Using persistent machine identity (no cleanup needed)
Mounting persistent data disk /dev/vdb -> /data
Tailscale: Found persistent state, reconnecting...

$ tailscale status | grep sovereign
100.68.204.88   sovereign-sql   ← ONLY ONE registration after multiple restarts!
```

---

## Remaining Work

- [ ] Implement Forgejo following the plan
- [ ] Extract reusable components to `internal/vm/common/`
- [ ] Create shared init.sh template
- [ ] Test multi-VM operation (SQL + Forge together)

---

## Handoff Checklist

- [x] Project builds cleanly
- [x] All SQL BDD tests pass (47/47)
- [x] Tailscale duplicate bug fixed
- [x] Comprehensive Forgejo plan written
- [x] Team file created
- [x] Documentation updated
