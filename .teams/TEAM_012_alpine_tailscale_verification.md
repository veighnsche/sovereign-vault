# TEAM_012: Alpine VM Tailscale Reachability Verification

**Created:** 2024-12-28
**Status:** ✅ SUCCESS
**Goal:** Verify and ensure Alpine-based VM on AVF is reachable through Tailscale serve

---

## End Goal Achieved

**Alpine-based VM on AVF is now reachable via Tailscale!**

```
$ ping -c 3 100.107.100.83
PING 100.107.100.83 (100.107.100.83) 56(84) bytes of data.
64 bytes from 100.107.100.83: icmp_seq=1 ttl=64 time=1457 ms
64 bytes from 100.107.100.83: icmp_seq=2 ttl=64 time=418 ms
64 bytes from 100.107.100.83: icmp_seq=3 ttl=64 time=141 ms
--- 3 packets transmitted, 3 received, 0% packet loss ---
```

## Issues Found & Fixed

### Issue 1: Console device (ttyS0 vs hvc0)
- **Problem:** Kernel uses `console=ttyS0` but crosvm serial probe fails
- **Fix:** Use `console=hvc0` (virtio console) instead

### Issue 2: OpenRC hangs during boot
- **Problem:** OpenRC didn't complete sysinit, sovereign-init never ran
- **Fix:** Bypassed OpenRC with simple shell init script (`init=/sbin/simple_init`)

### Issue 3: gvforwarder DHCP failure
- **Problem:** gvforwarder calls `dhclient` which doesn't exist in Alpine (uses `udhcpc`)
- **Fix:** Created `/usr/bin/dhclient` wrapper script that calls `udhcpc`

### Issue 4: PATH not set in init
- **Problem:** gvforwarder couldn't find dhclient because PATH wasn't set
- **Fix:** Added `export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"` to simple_init

### Issue 5: Socket cleanup
- **Problem:** Old vm.sock/gvproxy.sock cause "Address already in use" errors
- **Fix:** Clean up sockets before starting VM

## Files Modified

| File | Changes |
|------|---------|
| `vm/sql/start.sh` | Changed console=ttyS0 to hvc0, added vm.sock control socket |
| `vm/sql/CHECKLIST.md` | Added mistakes #16-19 and lessons #16-19 |
| Rootfs `/sbin/simple_init` | New init script bypassing OpenRC |
| Rootfs `/usr/bin/dhclient` | Wrapper script calling udhcpc |

## Current Working State

| Component | Status |
|-----------|--------|
| VM boots Alpine | ✅ |
| gvforwarder → gvproxy | ✅ (stable, no reconnects) |
| tap0 network | ✅ (192.168.127.2) |
| Tailscale connected | ✅ (sql-vm-3 @ 100.107.100.83) |
| Reachable via Tailscale | ✅ (ping works) |
| PostgreSQL | ❌ (blocked on kernel SYSVIPC - separate issue) |

## Next Steps for Future Teams

1. **Fix OpenRC** - Investigate why it hangs, or make simple_init the permanent solution
2. **Rebuild kernel** - Add CONFIG_SYSVIPC=y for PostgreSQL support
3. **Integrate fixes into sovereign CLI** - The rootfs modifications should be done by the build system

## Handoff Checklist

- [x] Project builds cleanly
- [x] Team file updated
- [x] CHECKLIST.md updated with new mistakes (#16-19)
- [x] VM is reachable via Tailscale
- [ ] PostgreSQL working (BLOCKED - kernel needs SYSVIPC)

