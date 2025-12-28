# TEAM_013: PostgreSQL Tailscale Reachability Investigation

**Created:** 2024-12-28
**Status:** BLOCKED - virtio_vsock driver not binding
**Goal:** Get PostgreSQL to appear as a Tailscale machine that other machines can reach

---

## Root Cause Analysis

### What's FIXED ✓
1. **CONFIG_SYSVIPC=y** - Verified in `sovereign_guest.fragment` and `out/guest-kernel/.config`
2. **Kernel rebuilt** - New Image deployed to device (md5: 382290828799abe630583fb64e2ba7d0)
3. **CONFIG_NETFILTER=y** - For Tailscale iptables support
4. **CONFIG_VIRTIO_VSOCKETS=y** - In kernel config

### What's BLOCKING ✗
**virtio_vsock driver not binding to PCI device**

The vsock PCI device (1af4:1053) is enumerated by the kernel:
```
[0.795404] pci 0000:00:04.0: [1af4:1053] type 00 class 0x028000
```

But virtio_vsock driver does NOT probe it. Only virtio_blk binds:
```
[0.824368] virtio_blk virtio0: 2/0/0 default/read/poll queues
```

This breaks the entire networking chain:
- gvforwarder in VM → can't connect to host gvproxy → no tap0 → no network → no Tailscale

### Why TAP Doesn't Work
TAP interfaces are **blocked on Android** (kernel restricts CAP_NET_ADMIN even for root).
Confirmed by testing: TAP interface creates on host, but crosvm can't bridge traffic.

---

## Attempted Solutions

| Approach | Result |
|----------|--------|
| `--vsock cid=10` | PCI device exists, driver doesn't bind |
| `--cid 10` | Same result |
| TAP networking `--net tap-name=` | Interface creates, no traffic passes |
| Clean kernel rebuild | Issue persists |

---

## Hypothesis: Kernel Build Issue

The kernel config has `CONFIG_VIRTIO_VSOCKETS=y` and symbols exist in System.map, but:
- Other virtio drivers (balloon, console) also don't bind
- Only virtio_blk binds successfully
- May indicate incomplete kernel rebuild or missing dependency

---

## Next Steps (For Future Teams)

1. **Try full kernel clean rebuild**: `make mrproper` then full rebuild
2. **Check virtio_pci initialization**: Debug why only virtio0 (block) is created
3. **Compare with working kernel**: TEAM_012 got vsock working - what kernel did they use?
4. **Alternative**: Try crosvm with slirp networking if available

---

## Files Modified

- `vm/sql/start.sh` - Updated for gvisor-tap-vsock with --cid flag
- `internal/rootfs/rootfs.go` - Updated simple_init for gvforwarder networking

---

## Handoff Checklist

- [x] Kernel has CONFIG_SYSVIPC=y (verified)
- [x] Kernel has CONFIG_NETFILTER=y (verified)  
- [x] Kernel deployed to device (verified by md5)
- [ ] virtio_vsock driver binding - **BLOCKED**
- [ ] gvforwarder connects to gvproxy - **BLOCKED**
- [ ] PostgreSQL starts - **BLOCKED** (needs networking first)
- [ ] Tailscale reachable - **BLOCKED**
