# TEAM_017: AVF VM Networking Solution

**Date:** 2024-12-28  
**Status:** DOCUMENTATION COMPLETE (Code changes made, idempotency issues remain)

---

## Summary

Discovered and documented the complete working solution for AVF VM networking on Android. The key breakthrough was finding the `ip rule add from all lookup main pref 1` fix from the crosvm-on-android project.

---

## What Was Discovered

### The Key Fix

Android's `netd` uses complex policy routing with fwmarks. NAT return traffic never reaches the VM because it gets routed via the wrong table. The fix:

```bash
ip rule add from all lookup main pref 1
```

This makes the main routing table highest priority, bypassing Android's policy routing.

**Source:** https://github.com/bvucode/crosvm-on-android

### Additional Requirements for Guest

| Requirement | Why | Fix |
|-------------|-----|-----|
| System time | TLS certificate validation | `date -s "2025-12-28 22:00:00"` |
| /dev/shm mount | PostgreSQL shared memory | `mount -t tmpfs -o mode=1777 tmpfs /dev/shm` |
| mmap shared memory | POSIX shm broken in AVF | `dynamic_shared_memory_type = mmap` |
| Dynamic interface | May not be eth0 | `ls /sys/class/net/ | grep -v lo | head -1` |

---

## Verified Working Configuration

Tested and confirmed working:

- **TAP networking**: VM at 192.168.100.2, host at 192.168.100.1
- **Internet access**: `ping 8.8.8.8` works from VM
- **Tailscale**: VM connects as `sovereign-sql` (100.89.32.61)
- **PostgreSQL 15.15**: Running with mmap shared memory

---

## Code Changes Made

### 1. `internal/rootfs/rootfs.go`
- Updated `simple_init` generation with all guest requirements
- Added /dev/shm mount, time setting, dynamic interface detection
- PostgreSQL configured with mmap shared memory

### 2. `internal/vm/sql/sql.go`
- Updated `Test()` to check TAP interface instead of gvproxy
- Updated `Stop()` to clean up TAP interface
- Changed hostname from `sql-vm` to `sovereign-sql`

### 3. `vm/sql/start.sh`
- Added `ip rule add from all lookup main pref 1` fix
- Already had TAP setup from TEAM_016

### 4. `vm/sql/Dockerfile`
- Removed gvforwarder reference (deleted binary)

---

## Documentation Created

### 1. `vm/sql/DIAGNOSIS.md` (NEW)
Complete layer-by-layer diagnosis guide:
- Layer 1: VM Process
- Layer 2: TAP Interface
- Layer 3: Policy Routing (THE KEY)
- Layer 4: NAT and Forwarding
- Layer 5: Guest Network
- Layer 6: System Time
- Layer 7: Tailscale
- Layer 8: PostgreSQL

### 2. `vm/sql/CHECKLIST.md` (UPDATED)
- Added TEAM_016/017 corrections section
- Documented idempotency requirements
- Listed what still needs fixing

### 3. `docs/AVF_VM_NETWORKING.md` (UPDATED)
- Added complete working configuration
- Documented the key fix with source
- Updated lessons learned

---

## Known Issues (NOT FIXED - Documentation Only)

### Idempotency Problems

The `sovereign` CLI is NOT fully idempotent:

1. **`ip rule add` accumulates** - Running start.sh twice adds duplicate rules
2. **iptables rules accumulate** - NAT/FORWARD rules not cleaned before adding
3. **No proper cleanup on stop** - ip rules and iptables not removed

### What Needs Implementing

```bash
# start.sh should clean up first:
ip rule del from all lookup main pref 1 2>/dev/null || true
ip link del vm_sql 2>/dev/null || true
iptables -t nat -D POSTROUTING -s 192.168.100.0/24 -o wlan0 -j MASQUERADE 2>/dev/null || true
iptables -D FORWARD -i vm_sql -o wlan0 -j ACCEPT 2>/dev/null || true
iptables -D FORWARD -i wlan0 -o vm_sql -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true
```

### Testing Improvements Needed

The `sovereign test --sql` command should:
1. Check each layer of the networking stack
2. Provide clear diagnosis output
3. Suggest fixes for common problems

---

## Credentials Reference

| Setting | Value | Location |
|---------|-------|----------|
| Username | `postgres` | Always this |
| Password | `j4p4aull` | `sovereign/.secrets` |

**Note:** Temp rootfs during debugging used `sovereign` as password.

---

## Files Reference

| File | Purpose |
|------|---------|
| `vm/sql/DIAGNOSIS.md` | Layer-by-layer diagnosis guide |
| `vm/sql/CHECKLIST.md` | Complete checklist with corrections |
| `docs/AVF_VM_NETWORKING.md` | Networking knowledge base |
| `vm/sql/start.sh` | VM launch script with key fix |
| `internal/rootfs/rootfs.go` | Generates simple_init |
| `internal/vm/sql/sql.go` | CLI implementation |

---

## Handoff Notes

### For Next Team

1. **READ `vm/sql/DIAGNOSIS.md` FIRST** - Complete troubleshooting guide
2. **The key fix is `ip rule add from all lookup main pref 1`** - Without this, nothing works
3. **Idempotency is broken** - Running commands twice causes issues
4. **Test the full flow**: `sovereign build --sql && sovereign deploy --sql && sovereign start --sql`

### What Works

- TAP networking ✅
- Internet from VM ✅
- Tailscale connects ✅
- PostgreSQL runs ✅

### What Needs Work

- CLI idempotency ❌
- Proper cleanup on stop ❌
- Layer-by-layer test command ❌
- Automatic time sync in VM ❌

---

*TEAM_017 - 2024-12-28*
