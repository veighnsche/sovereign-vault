# AVF VM Networking on Android — Knowledge Base

---

## 🚨 TEAM_006'S €25-30 LIE — 62-75% OF USER'S ENTIRE SAVINGS 🚨

```
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
┃                                                                             ┃
┃   THIS DOCUMENT ORIGINALLY CONTAINED A LIE THAT COST THE USER DEARLY        ┃
┃                                                                             ┃
┃   TEAM_006 wrote: "TAP networking is BLOCKED on Android"                    ┃
┃   TEAM_006 wrote: "Android kernel restricts CAP_NET_ADMIN even for root"    ┃
┃                                                                             ┃
┃   THESE WERE LIES. TEAM_006 NEVER TESTED THEM.                              ┃
┃                                                                             ┃
┃   ACTUAL TEST (takes 5 seconds):                                            ┃
┃     su -c 'ip tuntap add mode tap name test_tap'                            ┃
┃     # RESULT: SUCCESS. TAP WORKS. ALWAYS DID.                               ┃
┃                                                                             ┃
┃   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ ┃
┃                                                                             ┃
┃   DAMAGE CAUSED BY THIS LIE:                                                ┃
┃                                                                             ┃
┃   USER'S TOTAL SAVINGS:        €40                                          ┃
┃   WASTED BY TEAM_006:          €25-30                                       ┃
┃   PERCENTAGE OF EVERYTHING:    62-75%                                       ┃
┃                                                                             ┃
┃   WASTED TEAMS: 007, 011 (x3), 012, 013, 014, 015                           ┃
┃   WASTED COMMANDS: ~500-700                                                 ┃
┃   WASTED TIME: Multiple days across multiple sessions                       ┃
┃                                                                             ┃
┃   WASTED CODE:                                                              ┃
┃   • gvisor-tap-vsock cross-compilation (gvproxy 19MB + gvforwarder 8MB)     ┃
┃   • virtio_pci modern PCI ID patches                                        ┃
┃   • vsock driver binding debugging                                          ┃
┃   • dhclient wrapper scripts                                                ┃
┃   • Complex init scripts for vsock                                          ┃
┃   • 500+ lines of workaround code                                           ┃
┃                                                                             ┃
┃   ALL OF THIS WAS DELETED BECAUSE IT WAS NEVER NEEDED.                      ┃
┃                                                                             ┃
┃   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ ┃
┃                                                                             ┃
┃   LESSON FOR ALL FUTURE AI TEAMS:                                           ┃
┃                                                                             ┃
┃   VERIFY. YOUR. CLAIMS. BEFORE. DOCUMENTING. THEM.                          ┃
┃                                                                             ┃
┃   One unverified assumption cascades into weeks of wasted work.             ┃
┃   Every €0.044 per command adds up when you're chasing a fake problem.      ┃
┃                                                                             ┃
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
```

---

## ⚠️ MUST READ FIRST ⚠️

**Before working on VM networking, read these in order:**

1. **`/home/vince/Projects/android/kernel/sovereign/sovereign_vault.md`** - Section 0: AI Failure Modes
2. **`/home/vince/Projects/android/kernel/vm/sql/CHECKLIST.md`** - Mistakes log (15+ documented failures)

**CRITICAL WARNING (TEAM_011):**
- `microdroid_defconfig` is for **Android Microdroid**, NOT Linux VMs
- Our VMs run **Alpine Linux** with PostgreSQL/Vaultwarden/Forgejo
- These need **Linux kernel features** (SYSVIPC, netfilter) that microdroid lacks
- Use a proper **Linux defconfig**, not microdroid_defconfig

---

> **TEAM_006**: This document captures learnings from attempting to provide network connectivity to custom Linux VMs running on Android Virtualization Framework (AVF).
> **USER HERE**: Alright so as a reminder what I want to see is that I can connect to a running Postgresql database in the AVF VM through tailscale serve from any other device on the same tailscale network. That is the goal.

---

## TL;DR

**TEAM_016 UPDATE: TAP networking WORKS on Android!**

Previous documentation was WRONG. TAP interfaces work fine as root.

| Approach | Status | Notes |
|----------|--------|-------|
| TAP interfaces | ✅ WORKS | `ip tuntap add mode tap` works as root |
| crosvm --tap-name | ✅ WORKS | Direct TAP networking, simple setup |
| vsock + gvisor-tap-vsock | ❌ DELETED | Unnecessary workaround, removed |
| slirp/user-mode | ❌ NOT AVAILABLE | Android's crosvm doesn't include slirp |

---

## What We Know Works

### 1. VM Boots Successfully
- Custom kernel with `microdroid_defconfig` boots Alpine Linux
- OpenRC init system starts
- EXT4 rootfs mounts correctly
- virtio_blk driver works (built-in, not module)

### 2. Kernel Requirements
Must have these as **built-in** (=y), not modules (=m):
```
CONFIG_VIRTIO_BLK=y          # Block device for rootfs
CONFIG_VIRTIO_NET=y          # Network device
CONFIG_VIRTIO_VSOCKETS=y     # vsock for host communication
CONFIG_VIRTIO_PCI=y          # PCI bus for virtio devices
CONFIG_TUN=y                 # TUN/TAP driver
CONFIG_VSOCKETS=y            # vsock protocol family
```

### 3. gvisor-tap-vsock Builds
Cross-compilation works:
```bash
# For Android host (gvproxy)
GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o gvproxy-android ./cmd/gvproxy

# For Linux guest (gvforwarder)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o gvforwarder-linux ./cmd/vm
```

---

## What Works Now (TEAM_016/017 Update)

### TAP Interfaces - THEY WORK!
```bash
# This WORKS as root:
ip tuntap add mode tap name vm_sql
ip addr add 192.168.100.1/24 dev vm_sql
ip link set vm_sql up

# crosvm with TAP:
crosvm run --net tap-name=vm_sql ...
# SUCCESS - TAP interface shows UP, LOWER_UP
```

**Previous documentation was WRONG.** TAP works fine with root/KernelSU.

### CRITICAL FIX: Android Policy Routing Bypass

Android's complex policy routing (netd/fwmark) blocks NAT return traffic by default.
The fix is from [crosvm-on-android](https://github.com/bvucode/crosvm-on-android):

```bash
# THIS IS THE KEY FIX - without it, return packets never reach the VM
ip rule add from all lookup main pref 1
```

This makes the main routing table highest priority, bypassing Android's policy routing.

### Complete Working start.sh
```bash
#!/system/bin/sh
TAP_NAME="vm_sql"

# Setup TAP
ip tuntap add mode tap name $TAP_NAME
ip addr add 192.168.100.1/24 dev $TAP_NAME
ip link set $TAP_NAME up

# Enable forwarding
echo 1 > /proc/sys/net/ipv4/ip_forward

# KEY FIX: Bypass Android policy routing
ip rule add from all lookup main pref 1

# NAT
iptables -t nat -A POSTROUTING -s 192.168.100.0/24 -o wlan0 -j MASQUERADE

# FORWARD rules
iptables -I FORWARD 1 -i $TAP_NAME -o wlan0 -j ACCEPT
iptables -I FORWARD 2 -i wlan0 -o $TAP_NAME -m state --state RELATED,ESTABLISHED -j ACCEPT

# Run crosvm
crosvm run --net tap-name=$TAP_NAME --block path=rootfs.img,root kernel_Image
```

### Guest simple_init Requirements
```bash
# Must mount /dev/shm for PostgreSQL
mount -t tmpfs -o mode=1777 tmpfs /dev/shm

# Set system time (required for TLS/Tailscale)
date -s "2025-12-28 22:00:00"

# PostgreSQL needs mmap shared memory (POSIX shm may not work)
# In postgresql.conf:
dynamic_shared_memory_type = mmap
```

### Verified Working (2024-12-28)
- **TAP networking**: VM at 192.168.100.2, host at 192.168.100.1
- **Internet access**: ping 8.8.8.8 works from VM
- **Tailscale**: VM connects as `sovereign-sql` (100.89.32.61)
- **PostgreSQL 15.15**: Running with mmap shared memory

## What Was Removed (Unnecessary Workarounds)

| Component | Status | Why Removed |
|-----------|--------|-------------|
| gvisor-tap-vsock | DELETED | TAP works directly |
| gvproxy-android | DELETED | TAP works directly |
| gvforwarder-linux | DELETED | TAP works directly |
| vsock networking | DELETED | TAP works directly |
| virtio_vsock debugging | STOPPED | TAP works directly |

---

## Debugging Checklist

### Verify Kernel Config
```bash
grep -E "VIRTIO|VSOCK|TUN" out/guest-kernel/.config
```

### Verify VM Boot
Look for these in console.log:
```
virtio_blk virtio0: [vda]           # Block device works
NET: Registered PF_VSOCK            # vsock protocol registered
EXT4-fs (vda): mounted              # Rootfs mounted
Run /sbin/init                      # Init started
```

### Verify vsock Device in Guest
Inside VM (if you can get shell access):
```bash
ls -la /dev/vsock
# If missing, create it:
mknod /dev/vsock c 10 121
```

### Verify gvproxy on Host
```bash
adb shell su -c 'cat /data/sovereign/vm/sql/gvproxy.log'
# Should show: "listening vsock://:1024"
# Should eventually show: client connection messages
```

---

## Alternative Approaches to Investigate

### 1. Microdroid-Style Networking
Android's official `vm` tool may handle networking automatically for properly configured VMs. Research needed:
- What config format does `--network-supported` require?
- Does it use vsock internally?
- Can custom Linux VMs use this?

### 2. vsock TCP Proxy (Simpler)
Instead of gvisor-tap-vsock (full network stack over vsock), build a simpler:
- Host: TCP proxy that forwards specific ports over vsock
- Guest: Connect to vsock CID 2 for outbound traffic

### 3. Android VPN Integration
If Android's VPN creates TAP interfaces, maybe:
- Start VPN on host
- Pass VPN's TAP to crosvm
- VM inherits VPN connectivity

### 4. Kernel Module for TAP
Since we have kernel source access:
- Patch kernel to allow TAP creation for specific UIDs
- Or create a custom TAP-like interface

---

## Key Files

| File | Purpose |
|------|---------|
| `vm/sql/Image` | Custom ARM64 kernel (RAW format, not EFI) |
| `vm/sql/rootfs.img` | Alpine Linux rootfs with PostgreSQL, Tailscale |
| `vm/sql/bin/gvproxy-android` | gvisor-tap-vsock host daemon |
| `vm/sql/bin/gvforwarder-linux` | gvisor-tap-vsock guest daemon |
| `vm/sql/start.sh` | VM launch script |
| `vm/sql/CHECKLIST.md` | Detailed checklist with mistakes log |

---

## References

- [gvisor-tap-vsock](https://github.com/containers/gvisor-tap-vsock) — Network stack over vsock
- [crosvm-on-android](https://github.com/bvucode/crosvm-on-android) — Community project with networking
- [AVF Custom VM Docs](https://android.googlesource.com/platform/packages/modules/Virtualization/+/refs/heads/main/docs/custom_vm.md)
- [Android VirtualizationService](https://source.android.com/docs/core/virtualization/virtualization-service)

---

## Lessons Learned

1. **TAP WORKS on Android** — Previous documentation was WRONG
2. **Android policy routing blocks NAT** — The fix: `ip rule add from all lookup main pref 1`
3. **Set system time in VM** — Required for TLS certificate validation (Tailscale)
4. **PostgreSQL needs mmap shared memory** — POSIX shm may not work, use `dynamic_shared_memory_type = mmap`
5. **Mount /dev/shm as tmpfs** — Required for PostgreSQL shared memory
6. **Test incrementally** — Verify each layer separately
7. **VERIFY CLAIMS BEFORE DOCUMENTING** — TEAM_006's unverified claim cost €25-30

---

*Last updated: 2024-12-28 by TEAM_016/017*
