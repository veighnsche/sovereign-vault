# Phase 1: Discovery and Safeguards

**Feature:** VM Common Code Refactor  
**Status:** 🔲 NOT STARTED  
**Parent:** [VM_COMMON_REFACTOR_PLAN.md](VM_COMMON_REFACTOR_PLAN.md)

---

## Purpose

Understand the current codebase and lock in regression protection before refactoring.

---

## Refactor Summary

Extract ~300 lines of duplicated code from `internal/vm/sql/` and `internal/vm/forge/` into a shared `internal/vm/common/` package.

**Pain Points:**
- Copy-paste code between SQL and Forge VMs
- Bug fixes need to be applied in multiple places
- Adding new services requires duplicating boilerplate
- Tailscale cleanup code is nearly identical (111 vs 112 lines)

---

## Success Criteria

| Criterion | Metric |
|-----------|--------|
| Tests locked | All existing tests pass before refactor |
| Build compiles | `go build ./...` succeeds |
| Behavior documented | Public API surface mapped |
| Duplication measured | Line counts documented per function |

---

## Behavioral Contracts

### Public API (must remain stable)

```go
// internal/vm/vm.go - VM interface
type VM interface {
    Name() string
    Build() error
    Deploy() error
    Start() error
    Stop() error
    Test() error
    Remove() error
}
```

### CLI Commands (behavior must not change)

| Command | Expected Behavior |
|---------|-------------------|
| `sovereign build --sql` | Build PostgreSQL VM |
| `sovereign build --forge` | Build Forgejo VM |
| `sovereign deploy --sql` | Push SQL VM to device |
| `sovereign deploy --forge` | Push Forge VM to device |
| `sovereign start --sql` | Start SQL VM, stream logs |
| `sovereign start --forge` | Start Forge VM, stream logs |
| `sovereign stop --sql` | Stop SQL VM, cleanup networking |
| `sovereign stop --forge` | Stop Forge VM, cleanup networking |
| `sovereign test --sql` | Run SQL VM health checks |
| `sovereign test --forge` | Run Forge VM health checks |
| `sovereign remove --sql` | Remove SQL VM, cleanup Tailscale |
| `sovereign remove --forge` | Remove Forge VM, cleanup Tailscale |

---

## Golden/Regression Tests

### Existing BDD Scenarios
- `features/sql.feature` - 50+ scenarios for SQL VM
- `features/forge.feature` - 50+ scenarios for Forge VM

### Build Verification
```bash
go build ./...           # Must succeed
go test ./...            # Must pass
```

---

## Current Architecture Notes

### Dependency Graph
```
cmd/sovereign/main.go
    └── internal/vm/
        ├── vm.go (interface + registry)
        ├── sql/
        │   ├── sql.go → device, docker, rootfs, secrets
        │   ├── lifecycle.go → device
        │   └── verify.go → device, secrets
        └── forge/
            ├── forge.go → device, docker, rootfs
            ├── lifecycle.go → device
            └── verify.go → device
```

### Shared Dependencies
- `internal/device/` - ADB shell commands, file operations
- `internal/docker/` - Docker build, export
- `internal/rootfs/` - AVF preparation
- `internal/secrets/` - Credential management (SQL only)

### Known Couplings
1. Forge depends on SQL kernel: `vm/sql/Image`
2. Forge depends on SQL database: `sovereign-sql:5432`
3. Both use same TAP networking pattern (different subnets)
4. Both use same Tailscale registration cleanup logic

---

## Constraints

1. **No behavior changes during refactor**
2. **Incremental migration** - old and new can coexist
3. **VM interface unchanged** - same 7 methods
4. **CLI unchanged** - same flags and output

---

## Steps

| Step | File | Description |
|------|------|-------------|
| 1 | [phase-1-step-1.md](phase-1-step-1.md) | Map current code and create duplication inventory |
| 2 | [phase-1-step-2.md](phase-1-step-2.md) | Verify all tests pass (lock baseline) |
| 3 | [phase-1-step-3.md](phase-1-step-3.md) | Document public API contracts |

---

## Exit Criteria

- [ ] Duplication inventory complete (line counts per function)
- [ ] `go build ./...` succeeds
- [ ] All existing tests documented
- [ ] API contracts documented
- [ ] Ready for Phase 2
