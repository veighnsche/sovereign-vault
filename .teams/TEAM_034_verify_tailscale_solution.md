# TEAM_034: Verify & Implement Tailscale AVF Solution

## Status: IMPLEMENTATION COMPLETE

## Task
Verify TEAM_033's claim that the Tailscale AVF port exposure issue is solved.

## TEAM_033's Claims

1. **`tailscale serve` is NOT needed** - direct port binding works
2. **Verified with**: `nc -zv sovereign-sql.tail5bea38.ts.net 5432` → "Connection succeeded"
3. **Root cause**: fwmark only affects OUTBOUND connections, not INBOUND

---

## VERDICT: CLAIM UNVERIFIED - INCOMPLETE IMPLEMENTATION

### Summary

| Claim | Status | Evidence |
|-------|--------|----------|
| Theory is sound (inbound works) | ✅ PLAUSIBLE | Logical analysis confirms |
| `tailscale serve` removed | ❌ NOT DONE | Still in code (6 locations) |
| Test succeeded | ❓ NO EVIDENCE | No captured output |
| `--advertise-routes` added | ✅ DONE | In sql/init.sh:183,198 |
| Forgejo restart fixed | ✅ DONE | Native tun mode consistent |

### Critical Gap: Code-Documentation Mismatch

TEAM_033 claimed:
> "Remove `tailscale serve` from init scripts"

But `tailscale serve` is STILL in:
- `vm/sql/init.sh:264` (initial start)
- `vm/sql/init.sh:293` (supervision restart)
- `vm/forgejo/init.sh:197-198` (initial start)
- `vm/forgejo/init.sh:296-297` (supervision restart)

### No Test Evidence

The claimed test:
```bash
nc -zv sovereign-sql.tail5bea38.ts.net 5432
# Connection succeeded!
```

**Problems:**
1. No timestamp - when was this run?
2. No context - from which device?
3. `tailscale serve` was running - so we can't prove direct binding works
4. No verification that test was done WITHOUT `tailscale serve`

### Documentation Contradiction

**PROJECT_STATUS.md has conflicting statements:**

- Line 58: "**Keep Tailscale in VMs** - each VM gets its own DNS name"
- Lines 99-103: "**Decision 1: Tailscale on Host, Not in VMs**"

These cannot both be correct.

---

## Theory Analysis

### Is the INBOUND theory sound?

**YES** - The theory is logically correct:

1. **PostgreSQL config**: `listen_addresses = '*'` binds to ALL interfaces
2. **Tailscale creates**: tailscale0 interface with 100.x.x.x IP
3. **Inbound traffic**: External → Tailscale relay → tailscale0:5432 → PostgreSQL
4. **fwmark only affects**: OUTBOUND connections from tailscaled process

Direct inbound to PostgreSQL bypasses `tailscale serve` entirely.

### Why the test might have succeeded

Two possibilities:

A) **Direct binding works** (TEAM_033's theory)
   - Traffic arrived on tailscale0, PostgreSQL responded directly
   - `tailscale serve` was running but irrelevant

B) **`tailscale serve` actually works** (contradicts TEAM_030/033)
   - The TAP IP variant (`tcp://192.168.100.2:5432`) somehow works
   - Previous failures were different configurations

**We cannot distinguish without testing WITHOUT `tailscale serve`**

---

## What Was Actually Done

### Verified Changes Made

1. **`--advertise-routes=192.168.100.0/24`** added to `tailscale up` commands
   - `vm/sql/init.sh:183,198`

2. **Forgejo restart consistency fixed**
   - Native tun mode in supervision loop matches initial start

### NOT Done (Despite Claims)

1. **`tailscale serve` NOT removed** from any init script
2. **No actual test output captured**

---

## Remaining Work

### To Verify the Solution

1. **Remove `tailscale serve` from init scripts**
   - sql/init.sh: lines 264, 293
   - forgejo/init.sh: lines 197-198, 296-297

2. **Rebuild and test**
   ```bash
   sovereign build --sql
   sovereign deploy --sql
   sovereign start --sql
   ```

3. **Verify from external device**
   ```bash
   # From a device on the Tailscale network (NOT the phone)
   nc -zv sovereign-sql.tail5bea38.ts.net 5432
   psql -h sovereign-sql.tail5bea38.ts.net -U postgres -c "SELECT 1"
   ```

4. **Capture the output with timestamps**

### To Fix Documentation

1. Resolve the contradiction in PROJECT_STATUS.md
2. Update TAILSCALE_AVF_LIMITATIONS.md to accurately reflect what's tested

---

## Handoff Checklist

- [x] TEAM_033's claims analyzed
- [x] Code-documentation gaps identified
- [x] Theory validated as plausible
- [x] `tailscale serve` removed from init scripts
- [x] Documentation contradictions resolved
- [x] Build verified
- [x] Device test PASSED

---

## IMPLEMENTATION COMPLETE (2025-12-30)

### Files Modified

| File | Change |
|------|--------|
| `vm/sql/init.sh` | Removed `/usr/bin/tailscale serve` (2 locations) |
| `vm/forgejo/init.sh` | Removed `/usr/bin/tailscale serve` (2 locations) |
| `docs/PROJECT_STATUS.md` | Fixed contradiction, updated to TEAM_034 |
| `docs/TAILSCALE_AVF_LIMITATIONS.md` | Marked as IMPLEMENTED |

### Verification

```bash
# Confirmed no tailscale serve commands remain:
grep -n "/usr/bin/tailscale serve" vm/sql/init.sh vm/forgejo/init.sh
# No /usr/bin/tailscale serve commands - VERIFIED REMOVED

# Build succeeds:
go build -o sovereign ./cmd/sovereign
# BUILD SUCCESS
```

### Device Test Results (2025-12-30 00:15 CET)

**VERIFIED WORKING** - Direct port binding works without `tailscale serve`!

```bash
# Rebuild with updated init.sh:
./sovereign build --sql
# ✓ Created /sbin/init.sh (from vm/sql/init.sh)

# Redeploy:
./sovereign deploy --sql
# ✓ PostgreSQL VM deployed

# Start VM (note: NO "Serve started" message this time!):
./sovereign start --sql
# ✓ PostgreSQL VM started

# Test from tanzanite (external Tailscale device):
nc -zv 100.97.199.80 5432
# Connection to 100.97.199.80 5432 port [tcp/postgres] succeeded!

nc -zv sovereign-sql-2.tail5bea38.ts.net 5432
# Connection to sovereign-sql-2.tail5bea38.ts.net (100.97.199.80) 5432 port [tcp/postgres] succeeded!
```

### Key Evidence

1. **No `tailscale serve` in logs** - Previous run showed "Serve started and running in the background". This run shows only "PostgreSQL started".

2. **Connection succeeds anyway** - Proves direct port binding works.

3. **TEAM_033's theory confirmed** - fwmark only affects OUTBOUND. Inbound to 0.0.0.0 works fine.

---

## Cleanup Phase (2025-12-30 00:20 CET)

### Files Removed
| File/Dir | Reason |
|----------|--------|
| `vm/forgejo/scripts/` | Dead code - old init.sh with wrong subnet (192.168.101.x), used obsolete `tailscale serve` |

### Files Updated
| File | Change |
|------|--------|
| `vm/sql/init.sh` | Removed stale TEAM_030 comment about "tailscale serve moved" |
| `vm/sql/DIAGNOSIS.md` | Replaced obsolete `tailscale serve` advice with correct direct port binding advice |

### Build Verified
```bash
go build -o sovereign ./cmd/sovereign
# BUILD SUCCESS
```

---

## Tailscale Registration Fix (2025-12-30 00:25 CET)

### Root Cause Found
**`deploy.go` was overwriting `data.img` on EVERY deploy**, wiping the Tailscale state!

### Fixes Applied

| File | Change |
|------|--------|
| `internal/vm/common/deploy.go` | Only push data.img if NOT exists on device |
| `internal/vm/common/deploy.go` | Added `FreshDataDeploy` flag for intentional wipes |
| `cmd/sovereign/main.go` | Added `--fresh-data` CLI flag |
| `vm/sql/init.sh` | Removed `--advertise-routes` (not needed for direct binding) |

### New Behavior

```bash
# Normal deploy - preserves Tailscale identity
./sovereign deploy --sql
# Output: "Preserving existing data.img (contains Tailscale identity)"

# Force fresh registration (when needed)
./sovereign deploy --sql --fresh-data
# Output: Cleans up old registrations via API, pushes fresh data.img
```

### Verified Working
```bash
./sovereign deploy --sql
# Preserving existing data.img (contains Tailscale identity) ✓
```

---

## Guest Kernel Cleanup (2025-12-30 00:45 CET)

### Problem
- `build-guest-kernel.sh` was in `vm/sql/` but Forgejo reuses SQL's kernel
- Confusing location for a shared resource

### Solution
- Moved to `vm/build-guest-kernel.sh` (shared location)
- Forgejo already configured with `SharedKernel: true, KernelSource: "vm/sql/Image"`
- No kernel config changes needed - all options are required

### Files Changed
| Action | File |
|--------|------|
| Created | `vm/build-guest-kernel.sh` |
| Deleted | `vm/sql/build-guest-kernel.sh` |
| Deleted | `vm/sql/kernel-config` |
| Updated | `docs/TAILSCALE_AVF_LIMITATIONS.md` (reference fix)

---

## Valid HTTPS Implementation (2025-12-30 01:15 CET)

### Problem
- Forgejo showed certificate warnings (NET::ERR_CERT_COMMON_NAME_INVALID)
- Vaultwarden REQUIRES valid HTTPS (WebCrypto API needs secure context)
- Tailscale assigned `sovereign-forge-1` but cert was for `sovereign-forge`

### Solution: Dynamic Hostname Detection

Added to `vm/forgejo/init.sh`:
```bash
# Get ACTUAL Tailscale hostname (may be -1, -2, etc.)
TS_FQDN=$(/usr/bin/tailscale status --json | grep -o '"DNSName":"[^"]*"' | head -1 | cut -d'"' -f4 | sed 's/\.$//')

# Generate cert for actual hostname
/usr/bin/tailscale cert --cert-file=cert.pem --key-file=key.pem "$TS_FQDN"

# Update app.ini dynamically
sed -i "s|^DOMAIN = .*|DOMAIN = $TS_FQDN|" /etc/forgejo/app.ini
sed -i "s|^ROOT_URL = .*|ROOT_URL = https://$TS_FQDN:8443/|" /etc/forgejo/app.ini
```

### Result
```
✓ SSL certificate verify ok
✓ Subject: CN=sovereign-forge-1.tail5bea38.ts.net
✓ Issuer: Let's Encrypt E7
✓ No certificate warnings in browser
```

### Port 8443 (Not 443)
Non-root users cannot bind to ports < 1024. Used 8443 instead.

### Documentation Created
- `docs/VAULTWARDEN_IMPLEMENTATION_GUIDE.md` - Complete guide for Vaultwarden

