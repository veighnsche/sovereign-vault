# Phase 1: Discovery

**Feature:** Forgejo VM Implementation  
**Status:** ✅ COMPLETE (discovery done by TEAM_023)  
**Parent:** `.plans/forgejo-implementation/`

---

## 1. Feature Summary

**Problem:** We have a working PostgreSQL VM but no Git hosting capability on the Android device.

**Solution:** Implement Forgejo VM using the same proven patterns as PostgreSQL VM (TAP networking, custom init.sh, persistent Tailscale identity).

**Who Benefits:** Developers who want self-hosted git hosting on their Android device with Tailscale connectivity.

---

## 2. Success Criteria

The Forgejo implementation is complete when:

1. All BDD tests pass
2. `sovereign build --forge` creates rootfs.img and data.img
3. `sovereign deploy --forge` pushes files to device
4. `sovereign start --forge` starts VM with TAP networking
5. Forgejo web UI accessible via Tailscale
6. Git operations work via SSH through Tailscale
7. Restart preserves Tailscale identity (no duplicates)
8. Forgejo connects to PostgreSQL on SQL VM
9. Both VMs run simultaneously without conflict

---

## 3. Current State Analysis

### What PostgreSQL Has (Working)

| Component | Implementation | Status |
|-----------|----------------|--------|
| Networking | TAP interface `vm_sql` at 192.168.100.x | ✅ Working |
| Init System | Custom `/sbin/init.sh` (OpenRC hangs) | ✅ Working |
| Console | `console=ttyS0` with `--serial type=stdout` | ✅ Working |
| Data Disk | `/dev/vdb` mounted to `/data` (persistent) | ✅ Working |
| Tailscale | State persisted on data disk, reconnects on restart | ✅ Working |
| Port Exposure | `tailscale serve --tcp 5432 5432` | ✅ Working |
| BDD Tests | 47 scenarios in `sql.feature` | ✅ All passing |

### What Forgejo Has (Broken/Stale)

| Component | Implementation | Status |
|-----------|----------------|--------|
| Networking | Tries to use gvproxy/vsock | ❌ Wrong approach |
| Init System | Uses OpenRC via `/sbin/init` | ❌ Will hang |
| Console | `console=hvc0` with virtio-console | ❌ Wrong device |
| Data Disk | Passed to crosvm but not mounted in init | ⚠️ Incomplete |
| Tailscale | Uses Alpine package, registers every time | ❌ Duplicates |
| Port Exposure | Uses `tailscale serve --https=443` | ⚠️ Needs review |
| BDD Tests | 85 lines in `forge.feature.disabled` | ❌ Stale/disabled |

---

## 4. Codebase Reconnaissance

### Files to Modify

| File | Current State | Required Change |
|------|---------------|-----------------|
| `vm/forgejo/start.sh` | Uses gvproxy/vsock | Rewrite with TAP networking |
| `vm/forgejo/scripts/init.sh` | OpenRC service script | Replace with standalone init |
| `vm/forgejo/Dockerfile` | Uses Alpine tailscale package | Use static binary |
| `internal/vm/forge/forge.go` | Wrong device paths | Fix paths, refactor |
| `vm/forgejo/config/app.ini` | Wrong hostnames | Update to sovereign-* |
| `features/forge.feature.disabled` | Stale tests | Expand and re-enable |

### Files to Reference (Gold Standard)

| File | Purpose |
|------|---------|
| `vm/sql/init.sh` | **THE GOLD STANDARD** - copy this pattern |
| `vm/sql/start.sh` | TAP networking setup |
| `internal/vm/sql/lifecycle.go` | Start/Stop/Remove with persistent Tailscale |
| `features/sql.feature` | BDD test patterns |

---

## 5. Constraints

- **Must use TAP networking** - gvproxy/vsock doesn't work in AVF
- **Must use custom init.sh** - OpenRC hangs in crosvm
- **Must use ttyS0 console** - crosvm --serial captures serial, not virtio hvc0
- **Must persist Tailscale state on data.img** - prevents duplicate registrations
- **Must connect to PostgreSQL** - Forgejo requires a database backend

---

## 6. Critical Lessons Learned

1. **OpenRC Hangs:** Use custom init.sh, NOT `/sbin/init`
2. **Console Mismatch:** Use `console=ttyS0` with `--serial type=stdout`
3. **Tailscale Duplicates:** Persist state on data.img, not rootfs.img
4. **Android Policy Routing:** `ip rule add from all lookup main pref 1` is CRITICAL
5. **gvproxy Doesn't Work:** Use TAP networking with NAT

---

## Next Phase

→ [Phase 2: Design](phase-2.md)
