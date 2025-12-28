# Simplification Analysis: Are We Fixing Workarounds?

## TEAM_015 Critical Review

**Date:** 2024-12-28

---

## 1. The Actual Goal

**What do we ACTUALLY want?**
- Run PostgreSQL in a secure, isolated environment on an Android device
- Make it reachable via Tailscale from other machines
- Data persistence and crash recovery

**That's it.** Everything else is implementation detail.

---

## 2. The Current Workaround Chain

```
GOAL: PostgreSQL + Tailscale on Android
│
├── Decision: Use AVF (Android Virtualization Framework)
│   └── WHY: Security isolation, kernel-level separation
│
├── Problem: AVF's Microdroid is Android-based, not Linux
│   └── WORKAROUND: Build custom Alpine Linux VM
│
├── Problem: microdroid_defconfig lacks Linux features (SYSVIPC, etc.)
│   └── WORKAROUND: Build custom kernel with sovereign_guest.fragment
│
├── Problem: TAP networking blocked on Android (no CAP_NET_ADMIN)
│   └── WORKAROUND: Use vsock instead of TAP
│
├── Problem: AOSP kernel virtio_pci lacks modern PCI IDs
│   └── WORKAROUND: Patch virtio_pci_common.c with 0x1040-0x105a
│
├── Problem: vsock needs userspace networking stack
│   └── WORKAROUND: gvisor-tap-vsock (gvproxy + gvforwarder)
│
└── Current Status: Still debugging vsock connectivity
```

**Workaround depth: 5 layers**

---

## 3. Questions We Haven't Asked

### Q1: Why AVF at all?

**Alternatives to AVF:**
| Approach | Pros | Cons |
|----------|------|------|
| Termux + proot | Simple, no kernel work | No real isolation, same kernel |
| Termux + Docker (rootless) | Containers, easier | Needs kernel support, may not work |
| Cloud VM | Zero device work | Requires internet, monthly cost, defeats "sovereign" goal |
| Native Android app | Simplest | No PostgreSQL port exists |
| UserLAnd app | Ready-made Linux env | Limited, no real isolation |

### Q2: Why custom kernel?

**Alternatives to custom kernel:**
| Approach | Pros | Cons |
|----------|------|------|
| Use stock GKI kernel | No kernel builds | May lack features |
| Use prebuilt kernel from another project | Tested, stable | May not fit our needs |
| Upstream the virtio_pci patch | Fixes root cause | Takes time, needs maintainer approval |

### Q3: Why vsock?

**Alternatives to vsock for VM networking:**
| Approach | Pros | Cons |
|----------|------|------|
| Fix TAP networking | Direct, standard | May be impossible on Android |
| virtio-net with different backend | Standard approach | Same TAP problem |
| Port forwarding via crosvm | Built-in | Limited, may not support Tailscale |
| Serial console + SLIP/PPP | Old school, works | Slow, hacky |
| USB gadget networking | Hardware path | Complex, device-specific |

### Q4: Why Alpine Linux?

**Alternatives to Alpine:**
| Approach | Pros | Cons |
|----------|------|------|
| Microdroid + Android services | Native AVF support | PostgreSQL doesn't run on Android |
| Debian/Ubuntu minimal | More packages | Larger rootfs |
| Buildroot custom | Minimal, tailored | More build work |
| NixOS | Reproducible | Complex |

---

## 4. The "Other Direction" - What Haven't We Tried?

### Direction A: Fix the ROOT cause (TAP networking)

Instead of working around TAP being blocked, investigate:
1. **WHY** is TAP blocked on Android?
2. Is it a kernel config? SELinux? Capability check?
3. Can we enable it with a kernel config change?
4. Would that be ONE fix instead of FIVE workarounds?

### Direction B: Use crosvm's built-in port forwarding

crosvm has `--host-ip` and `--netmask` options. Have we tried:
```bash
crosvm run --host-ip 192.168.1.1 --netmask 255.255.255.0 --mac aa:bb:cc:dd:ee:ff ...
```

### Direction C: Question the PostgreSQL requirement

- Does it HAVE to be PostgreSQL?
- Would SQLite (which works natively) suffice?
- Could we use a PostgreSQL-compatible API over SQLite?

### Direction D: Question the on-device requirement

- Does PostgreSQL HAVE to run ON the phone?
- Could it run on a home server with Tailscale?
- The phone just connects as a client?

### Direction E: Use a different VM technology

- Has anyone tried QEMU instead of crosvm?
- What about Firecracker?
- What about Kata Containers?

---

## 5. Effort Analysis

| Approach | Estimated Effort | Risk | Simplicity |
|----------|-----------------|------|------------|
| Continue current path (vsock fixes) | Days more debugging | High (more unknowns) | Low |
| Fix TAP networking at root | Unknown (need investigation) | Medium | High if it works |
| Use crosvm port forwarding | Hours to test | Low | High |
| SQLite instead of PostgreSQL | Days (app changes) | Low | Very High |
| Off-device PostgreSQL | Hours | Very Low | Very High |
| QEMU instead of crosvm | Days | Medium | Medium |

---

## 6. Recommendation

**STOP. Before spending more time on vsock debugging:**

1. **Investigate TAP networking block** (Direction A)
   - 30 minutes to understand WHY it's blocked
   - If fixable with kernel config, that's ONE change vs FIVE workarounds

2. **Test crosvm port forwarding** (Direction B)
   - 30 minutes to test basic connectivity
   - May eliminate need for vsock entirely

3. **Question architecture** (Directions C, D)
   - Is on-device PostgreSQL the right choice?
   - Consider the simplest path to the actual goal

---

## 7. Action Items

- [ ] Investigate why TAP is blocked (kernel log? SELinux? capability check?)
- [ ] Test crosvm `--host-ip` option without vsock
- [ ] Document actual requirements (does it HAVE to be PostgreSQL? On-device?)
- [ ] Compare effort: fixing root cause vs continuing workarounds

---

## 8. The Key Question

> "If TAP networking worked, would we need ANY of this complexity?"

If the answer is "no", then we should fix TAP, not pile more workarounds on vsock.
