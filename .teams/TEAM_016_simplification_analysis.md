# TEAM_016: Simplification Analysis - Are We Fixing the Wrong Problem?

**Created:** 2024-12-28
**Status:** IN PROGRESS
**Trigger:** User questioned whether we're fixing workarounds instead of root cause

---

## The Critical Question

> "ARE WE SPENDING MASSIVE TIME ON MAKING A WORKAROUND TO WORK WHILE GOING THE OTHER DIRECTION HAS NEVER BEEN CONSIDERED?"

**Answer: YES. We need to investigate.**

---

## Current Workaround Stack (5 layers deep)

```
GOAL: PostgreSQL + Tailscale on Android phone
│
├── [Layer 1] AVF/crosvm VM
│   └── WHY: Need isolation
│
├── [Layer 2] Custom Alpine Linux rootfs
│   └── WHY: Microdroid is Android-based, PostgreSQL needs Linux
│
├── [Layer 3] Custom kernel with sovereign_guest.fragment
│   └── WHY: microdroid_defconfig lacks SYSVIPC, NETFILTER, etc.
│
├── [Layer 4] Patched virtio_pci with modern PCI IDs
│   └── WHY: Upstream AOSP kernel lacks 0x1040-0x107f support
│
├── [Layer 5] gvisor-tap-vsock over vsock
│   └── WHY: TAP networking "blocked" on Android
│
└── STATUS: Still debugging vsock after weeks of work
```

---

## The Root Cause We Haven't Fixed

**TAP networking is "blocked" on Android.**

But what does "blocked" actually mean?
- Is it a kernel config?
- Is it SELinux?
- Is it Android's capability management?
- Is it specific to crosvm's execution context?

**WE DON'T KNOW. We assumed it was unfixable and built 5 workarounds.**

---

## Investigation Plan

### Phase 1: Understand WHY TAP is blocked (30 min)

1. **Check kernel config:**
   ```
   CONFIG_TUN=y        # Is TUN/TAP even enabled?
   CONFIG_MACVLAN=y    # Alternative?
   CONFIG_VETH=y       # Virtual ethernet pairs?
   ```

2. **Check capability requirements:**
   - What capabilities does crosvm have?
   - What capabilities does it NEED?
   - Can we grant CAP_NET_ADMIN?

3. **Check SELinux:**
   - What SELinux context is crosvm running in?
   - Is there an avc denial for TAP creation?
   - Can we add a policy exception?

4. **Check crosvm itself:**
   - Does crosvm even TRY to create TAP correctly?
   - Is there a crosvm config option we're missing?

### Phase 2: Test Direct TAP (30 min)

If Phase 1 reveals TAP CAN be enabled:

1. Grant crosvm proper capabilities
2. Adjust SELinux if needed
3. Test TAP networking directly
4. Compare complexity: TAP fix vs vsock stack

### Phase 3: Decision

| If TAP Can Be Fixed | If TAP Cannot Be Fixed |
|---------------------|----------------------|
| Delete vsock work | Continue vsock debugging |
| Use standard networking | Document why TAP is impossible |
| Massive simplification | Accept the complexity |

---

## Alternative Directions to Consider

### Direction A: Fix TAP at root
- ONE fix instead of FIVE workarounds
- Standard Linux networking, no gvisor-tap-vsock

### Direction B: Use crosvm port forwarding
- `--host-ip`, `--netmask` options
- Built into crosvm, no extra software

### Direction C: Question the architecture
- Does PostgreSQL HAVE to be on-device?
- Could run on home server, phone just connects

### Direction D: Question PostgreSQL
- SQLite works natively on Android
- Is PostgreSQL a hard requirement?

---

## Action Items - COMPLETED

1. [x] `adb shell getenforce` → **Enforcing** (SELinux on)
2. [x] `adb shell cat /proc/config.gz | gunzip | grep TUN` → **CONFIG_TUN=y** (TAP compiled in!)
3. [x] `adb shell dmesg | grep -i denied` → No TAP-related denials
4. [x] Test TAP creation manually → **IT WORKS!**

## 🚨 CRITICAL DISCOVERY 🚨

**TAP NETWORKING WORKS ON ANDROID!**

```bash
# This SUCCEEDS:
su -c 'ip tuntap add mode tap name test_tap'

# TAP interface is created:
54: test_tap: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN
    link/ether 92:2d:55:1f:04:b3 brd ff:ff:ff:ff:ff:ff
```

**crosvm supports TAP directly:**
```
--tap-name    name of a configured persistent TAP interface
--host-ip     IP address to assign to host tap interface  
--netmask     netmask for VM subnet
```

## Conclusion

**WE HAVE BEEN BUILDING 5 LAYERS OF WORKAROUNDS FOR A PROBLEM THAT DOESN'T EXIST.**

The documentation saying "TAP is blocked on Android" was WRONG or outdated.

---

## SIMPLIFICATION PLAN

### What We Can DELETE:

| Component | Status | Reason |
|-----------|--------|--------|
| gvisor-tap-vsock | DELETE | Not needed with TAP |
| gvproxy | DELETE | Not needed with TAP |
| gvforwarder | DELETE | Not needed with TAP |
| vsock networking code | DELETE | Not needed with TAP |
| virtio_vsock debugging | STOP | Not needed with TAP |

### What We KEEP:

| Component | Status | Reason |
|-----------|--------|--------|
| Custom kernel | KEEP | Still need SYSVIPC, NETFILTER |
| virtio_pci patch | KEEP | Still need modern PCI IDs for virtio-net |
| Alpine rootfs | KEEP | Still need Linux for PostgreSQL |

### New Simplified Architecture:

```
BEFORE (5 workarounds):
  crosvm → vsock → gvproxy → gvforwarder → tap0 → network

AFTER (direct):
  crosvm --tap-name=vm_tap → vm_tap → network
```

### Implementation Steps:

1. **Create TAP interface on host:**
   ```bash
   ip tuntap add mode tap name vm_tap
   ip addr add 192.168.100.1/24 dev vm_tap
   ip link set vm_tap up
   ```

2. **Update start.sh to use TAP:**
   ```bash
   crosvm run \
     --tap-name=vm_tap \
     --mem 1024 \
     --rwdisk rootfs.img \
     "${KERNEL}"
   ```

3. **Configure VM networking:**
   - Static IP: 192.168.100.2/24
   - Gateway: 192.168.100.1
   - DNS: As needed

4. **Enable NAT/forwarding on host:**
   ```bash
   iptables -t nat -A POSTROUTING -s 192.168.100.0/24 -j MASQUERADE
   echo 1 > /proc/sys/net/ipv4/ip_forward
   ```

5. **Install Tailscale in VM normally** (now has real network)

### Estimated Effort:

| Task | Time |
|------|------|
| Create TAP setup script | 30 min |
| Update start.sh | 15 min |
| Configure VM networking | 30 min |
| Test connectivity | 30 min |
| **TOTAL** | **~2 hours** |

vs. continuing vsock debugging: **Days/weeks more**

---

## Cleanup Completed

### Files/Directories DELETED:
- `sovereign/vm/sql/bin/gvproxy-android` (19MB)
- `sovereign/vm/sql/bin/gvforwarder-linux` (8MB)
- `sovereign/vm/sql/gvisor-tap-vsock/` (entire directory)

### Files MODIFIED:
- `sovereign/vm/sql/start.sh` - Rewritten to use TAP networking
- `sovereign/docs/AVF_VM_NETWORKING.md` - Updated to show TAP works
- `aosp/drivers/virtio/virtio_pci_common.c` - Removed debug printk
- `aosp/drivers/virtio/virtio.c` - Removed debug printk

### What Remains (KEEP):
- Custom kernel with virtio_pci modern IDs patch (still needed for virtio-net)
- Alpine rootfs (still needed for PostgreSQL)
- TAP-based networking (the simple, correct approach)

## Status: COMPLETE ✓

### TAP Networking Verified Working

```
$ ping -I vm_sql -c 3 192.168.100.2
64 bytes from 192.168.100.2: icmp_seq=1 ttl=64 time=2.65 ms
64 bytes from 192.168.100.2: icmp_seq=2 ttl=64 time=1.99 ms
64 bytes from 192.168.100.2: icmp_seq=3 ttl=64 time=1.74 ms
3 packets transmitted, 3 received, 0% packet loss
```

### Note: Android Policy Routing
Android has complex policy routing rules. Use `-I vm_sql` to ensure 
traffic goes through the TAP interface, or from within the VM, traffic 
will route correctly.

### What Was Accomplished
1. Proved TAP networking works on Android (TEAM_006 was wrong)
2. Deleted 27MB of unnecessary workarounds (gvisor-tap-vsock)
3. Simplified start.sh to use `--net tap-name=vm_sql`
4. Fixed corrupted rootfs with e2fsck
5. Updated simple_init to dynamically find network interface
6. **VM boots and responds to pings at 192.168.100.2**
