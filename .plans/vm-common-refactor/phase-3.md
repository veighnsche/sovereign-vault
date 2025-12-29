# Phase 3: Migration

**Feature:** VM Common Code Refactor  
**Status:** ✅ COMPLETE (TEAM_029)  
**Parent:** [VM_COMMON_REFACTOR_PLAN.md](VM_COMMON_REFACTOR_PLAN.md)  
**Depends On:** [Phase 2](phase-2.md)

---

## Purpose

Migrate SQL and Forge VMs to use the common package. Remove duplicated code from individual VM packages.

---

## Migration Strategy

### Order of Migration

1. **SQL VM first** (more complex, validates design)
2. **Forge VM second** (confirms pattern works for simpler case)

### Per-VM Migration Order

For each VM:
1. Replace RemoveTailscaleRegistrations with common.RemoveTailscaleRegistrations
2. Replace Stop() with common.StopVM
3. Replace Remove() with common.RemoveVM
4. Replace Start() with common.StartVM
5. Replace Deploy() with common.DeployVM
6. Replace Build() with common.BuildVM (or hooks)
7. Replace Test() with common test framework + custom tests

### Breaking Changes Strategy

Per Rule 5 (prefer clean breaks):
- Move function → let compiler fail → fix call sites
- No compatibility shims
- No temporary re-exports

---

## Call Site Inventory

### SQL VM Call Sites

| Function | Called From | Priority |
|----------|-------------|----------|
| `sql.RemoveTailscaleRegistrations()` | lifecycle.go Remove() | High |
| `sql.(*VM).Stop()` | lifecycle.go, Remove() | High |
| `sql.(*VM).Remove()` | cmd/sovereign/main.go | High |
| `sql.(*VM).Start()` | cmd/sovereign/main.go | High |
| `sql.(*VM).Deploy()` | cmd/sovereign/main.go | Medium |
| `sql.(*VM).Build()` | cmd/sovereign/main.go | Medium |
| `sql.(*VM).Test()` | cmd/sovereign/main.go | Medium |

### Forge VM Call Sites

Same pattern as SQL, all called from cmd/sovereign/main.go via VM interface.

---

## Rollback Plan

If migration causes issues:
1. Git revert the migration commit
2. Old code is still present until Phase 4
3. Fix issues in common package
4. Re-attempt migration

---

## Prerequisites

Before starting Phase 3:

- [ ] Phase 2 complete (common package exists)
- [ ] Run `godog` and verify all BDD scenarios pass
- [ ] Save passing scenario list as baseline for comparison

---

## Steps

| Step | File | Description |
|------|------|-------------|
| 1 | [phase-3-step-1.md](phase-3-step-1.md) | Migrate SQL to use common.RemoveTailscaleRegistrations |
| 2 | [phase-3-step-2.md](phase-3-step-2.md) | Migrate SQL Stop/Remove to common |
| 3 | [phase-3-step-3.md](phase-3-step-3.md) | Migrate SQL Start to common |
| 4 | [phase-3-step-4.md](phase-3-step-4.md) | Migrate SQL Deploy to common |
| 5 | [phase-3-step-5.md](phase-3-step-5.md) | Migrate SQL Build to common (with hooks) |
| 6 | [phase-3-step-6.md](phase-3-step-6.md) | Migrate SQL Test to common framework |
| 7 | [phase-3-step-7.md](phase-3-step-7.md) | Migrate Forge config + Tailscale |
| 8 | [phase-3-step-8.md](phase-3-step-8.md) | Migrate Forge lifecycle (Start/Stop/Remove) |
| 9 | [phase-3-step-9.md](phase-3-step-9.md) | Migrate Forge Build/Deploy/Test + cleanup |

---

## Exit Criteria

- [ ] SQL VM uses common package for all lifecycle operations
- [ ] Forge VM uses common package for all lifecycle operations
- [ ] `go build ./...` succeeds
- [ ] All BDD scenarios pass
- [ ] CLI behavior unchanged
- [ ] Ready for Phase 4 (cleanup)
