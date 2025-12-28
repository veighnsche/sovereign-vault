# TEAM_001 — KernelSU Integration into Stock Kernel

**Created:** 2024-12-27
**Status:** Active
**Feature:** KernelSU integration for Sovereign Vault on Pixel 6 (raviole)

---

## Mission

Integrate KernelSU into the stock raviole kernel to enable root access for Sovereign Vault VM orchestration.

## Context

- The kernel workspace is a stock GKI-based Android kernel for Pixel 6
- KernelSU source exists at `/home/vince/Projects/android/kernel/KernelSU/`
- Target device: Google Pixel 6 (raviole), SoC: Google Tensor G1 (gs101)
- Sovereign Vault requires KernelSU for `/data/adb/service.d/` boot scripts

## Current State

- raviole_defconfig: No KernelSU configuration present
- BUILD.bazel: Uses `//common:kernel_aarch64` as base kernel
- KernelSU kernel module code available at `KernelSU/kernel/`

## Progress Log

| Date       | Action                                      |
|------------|---------------------------------------------|
| 2024-12-27 | Team registered, beginning discovery phase  |
| 2024-12-27 | Phase 1 (Discovery) complete - kernel 6.1.124, KPROBES enabled |
| 2024-12-27 | Phase 2 (Design) complete - Option A selected, recommendations applied |
| 2024-12-27 | All phase files updated with AI warnings and granular detail |
| 2024-12-27 | **READY FOR IMPLEMENTATION** - Phase 3 can begin |

## Design Decisions (Locked)

| Decision | Value | Rationale |
|----------|-------|----------|
| Integration method | Option A (symlink) | Standard KernelSU pattern |
| CONFIG_KSU | y (built-in) | Early boot requirement |
| CONFIG_KSU_DEBUG | n | Production mode |
| LOCALVERSION | "-sovereign" | Identify kernel build |
| Guest kernel | Deferred | Focus on host first |

## Handoff Notes

(To be filled on completion)

---

## Checklist Before Handoff

- [ ] Project builds cleanly
- [ ] All tests pass
- [ ] Behavioral regression tests pass
- [ ] Team file updated
- [ ] Remaining TODOs documented
