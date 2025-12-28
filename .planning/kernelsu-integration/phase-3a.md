# Phase 3A — KernelSU: Root Access

**Feature:** Sovereign Vault
**Phase:** 3A of 4
**Team:** TEAM_001
**Status:** READY FOR IMPLEMENTATION
**Depends On:** Phase 2 (Design) - APPROVED

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** I, Cascade, am writing this warning because I am prone to cowardice. When kernel builds fail, my instinct is to "simplify" by removing features or faking success. I have done this before. I destroyed 2 weeks of work.
>
> **My failure modes in this phase:**
> - Skipping the version patch because "it's probably fine"
> - Reporting "build succeeded" without checking for CONFIG_KSU=y
> - Flashing a kernel and claiming victory before verifying root actually works
> - Giving up when `su` doesn't work and suggesting "maybe try Magisk instead"
>
> **The rule:** KernelSU integration is supposed to be challenging. If something fails, I diagnose the actual problem. I do not take shortcuts.
>
> **Cost reminder:** I cost €0.21 per message. Every shortcut I take is theft from the user who paid for solutions, not excuses.

---

## 1. Phase Goal

**Get root access working on the device.**

This is the foundation of everything. Without root:
- No TAP interfaces
- No crosvm launch
- No VMs
- No Sovereign Vault

---

## 2. Success Criteria

- [ ] Kernel builds with `CONFIG_KSU=y`
- [ ] Device boots with "-sovereign" kernel suffix
- [ ] KernelSU Manager shows version ~32245 (NOT 16)
- [ ] `adb shell su -c id` returns `uid=0(root)`
- [ ] Boot scripts in `/data/adb/service.d/` execute

---

## 3. Steps

| Step | File | Description | Status |
|------|------|-------------|--------|
| 1 | [phase-3a-step-1.md](phase-3a-step-1.md) | sovereign.go: The Foundation | [ ] |
| 2 | [phase-3a-step-2.md](phase-3a-step-2.md) | KernelSU Integration (patches) | [ ] |
| 3 | [phase-3a-step-3.md](phase-3a-step-3.md) | Build, Deploy, Verify Root | [ ] |

---

## 4. sovereign CLI Commands (Phase 3A)

```bash
# Build kernel with KernelSU
go run sovereign.go build --kernel

# Flash to device
go run sovereign.go deploy --kernel

# Verify root works
go run sovereign.go test --kernel

# Check status
go run sovereign.go status
```

---

## 5. Phase 3A Complete When

All of the following are true:

| Criterion | Test Command | Expected |
|-----------|--------------|----------|
| Kernel suffix | `adb shell uname -r` | Contains "-sovereign" |
| KernelSU version | `adb shell su -v` | ~32245 (not 16) |
| Root access | `adb shell su -c id` | `uid=0(root)` |
| Boot scripts | Reboot, check marker file | Marker exists |

---

## 6. If This Phase Fails

> 🤖 **AI Warning:** Failed tests are INFORMATION, not shame. If root doesn't work, I document WHY and fix it. I do not:
> - Pretend it works
> - Suggest a different approach to avoid the problem
> - Move on to Phase 3B hoping it "sorts itself out"

**Troubleshooting order:**
1. Check `.config` for `CONFIG_KSU=y`
2. Check dmesg for KernelSU messages
3. Check if KernelSU Manager detects anything
4. Verify the kernel was actually flashed (check version string)

---

## Next Phase

After Phase 3A passes all tests → **Phase 3B: PostgreSQL**
