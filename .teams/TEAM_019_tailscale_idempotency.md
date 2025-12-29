# TEAM_019: Tailscale Registration Idempotency

## ⚠️ STATUS: BUG NOT FIXED - STILL PERSISTS

**User has requested fix 10+ times. No progress has been made.**

## Problem
Every `sovereign start --sql` or `sovereign deploy --sql` creates a NEW Tailscale 
machine registration instead of reconnecting to existing one. 

Result: Multiple duplicate machines (sovereign-sql, sovereign-sql-1, sovereign-sql-2...)
with different IPs, breaking dependency stability for PostgreSQL clients.

## Impact
- Forgejo and other services depend on `sovereign-sql` having a STABLE hostname/IP
- Every restart creates a new registration with a new IP
- All dependants break when the hostname changes

## Root Cause (IDENTIFIED BUT NOT FIXED)
1. Tailscale state is stored in `/var/lib/tailscale` inside `rootfs.img`
2. Every `sovereign build --sql` creates a fresh `rootfs.img`
3. Fresh rootfs = fresh machine ID = NEW Tailscale registration
4. The authkey creates a NEW machine, it doesn't reconnect to existing

## Mitigation Attempts (ALL FAILED)

| Team | Attempt | Result |
|------|---------|--------|
| TEAM_019 | Preflight check - fail if registration exists | DIDN'T HELP |
| TEAM_020 | Made cleanup the default behavior | DIDN'T HELP |
| TEAM_022 | Added `RemoveTailscaleRegistrations()` to delete via API | DIDN'T HELP |
| TEAM_022 | Called cleanup in Deploy(), Start(), Remove() | DIDN'T HELP |
| TEAM_022 | Added TAILSCALE_API_KEY support for auto-deletion | STILL DOESN'T WORK |

## Potential Fix (NOT IMPLEMENTED)
Move `/var/lib/tailscale` to `data.img` (persistent disk) instead of `rootfs.img`
so machine identity survives rebuilds.

This would require:
1. Mount data.img to /data in init.sh
2. Symlink /var/lib/tailscale -> /data/tailscale
3. First boot: copy state from rootfs to data
4. Subsequent boots: use existing state from data

## Files Where Bug Is Documented
- `internal/vm/sql/verify.go` - RemoveTailscaleRegistrations()
- `internal/vm/sql/sql.go` - Deploy()
- `internal/vm/sql/lifecycle.go` - Start(), Remove()
- `vm/sql/init.sh` - Tailscale startup section
- `sovereign_vault.md` - Section on AI failure modes

## User's Frustration Level: EXTREME
The user has explicitly stated this bug has been requested to be fixed at least 10 times
with NO PROGRESS. Every restart creates duplicate registrations.
