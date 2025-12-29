# VM Common Code Refactor Plan

**Feature:** Extract shared VM infrastructure for trivial service additions  
**Status:** 🔲 NOT STARTED  
**Created:** 2025-12-29 by TEAM_026  
**SSOT:** This directory (`.plans/vm-common-refactor/`)

---

## Executive Summary

**Problem:** SQL and Forge VMs share ~300+ lines of duplicated code across Build, Deploy, Start, Stop, Remove, and Test operations. Adding a new service (e.g., Vaultwarden) requires copying and modifying all this code.

**Solution:** Extract common VM infrastructure into `internal/vm/common/` package with configuration-driven VM creation.

**Goal:** Adding a new service should require:
1. A `Dockerfile` and `init.sh` for the service
2. A small config struct (~20 lines)
3. Zero lifecycle code duplication

---

## Phase Index

| Phase | File | Status | Description |
|-------|------|--------|-------------|
| Discovery | [phase-1.md](phase-1.md) | 🔲 Not Started | Map current code, lock in tests |
| Extraction | [phase-2.md](phase-2.md) | 🔲 Not Started | Create common package |
| Migration | [phase-3.md](phase-3.md) | 🔲 Not Started | Move SQL/Forge to common |
| Cleanup | [phase-4.md](phase-4.md) | 🔲 Not Started | Remove dead code |
| Hardening | [phase-5.md](phase-5.md) | 🔲 Not Started | Final verification |

---

## Current Architecture

```
internal/vm/
├── vm.go              # VM interface + registry
├── sql/
│   ├── sql.go         # Build(), Deploy()
│   ├── lifecycle.go   # Start(), Stop(), Remove()
│   └── verify.go      # Test(), RemoveTailscaleRegistrations()
└── forge/
    ├── forge.go       # Build(), Deploy()
    ├── lifecycle.go   # Start(), Stop(), Remove()
    └── verify.go      # Test(), RemoveTailscaleRegistrations()
```

## Target Architecture

```
internal/vm/
├── vm.go              # VM interface + registry (unchanged)
├── common/
│   ├── config.go      # VMConfig struct
│   ├── build.go       # Docker build, data disk, rootfs prep
│   ├── deploy.go      # File pushing, directory creation
│   ├── lifecycle.go   # Start, Stop, Remove (config-driven)
│   ├── tailscale.go   # Registration cleanup, status checks
│   └── test.go        # Generic test framework
├── sql/
│   └── sql.go         # ~50 lines: config + service-specific tests
└── forge/
    └── forge.go       # ~50 lines: config + service-specific tests
```

---

## Duplication Analysis

| Function | SQL Lines | Forge Lines | Shared Pattern |
|----------|-----------|-------------|----------------|
| Build() | 97 | 65 | Docker build, data disk, rootfs prep |
| Deploy() | 68 | 43 | Dir creation, file push, chmod |
| Start() | 43 | 51 | Check running, run script, stream logs |
| Stop() | 31 | 29 | Kill process, TAP cleanup, iptables |
| Remove() | 28 | 23 | Stop, Tailscale cleanup, rm dir |
| RemoveTailscaleRegistrations() | 112 | 111 | Nearly identical |
| Test() (common parts) | ~50 | ~50 | VM process, TAP, Tailscale checks |

**Total duplicated:** ~300+ lines

---

## VMConfig Design

```go
type VMConfig struct {
    // Identity
    Name          string   // "sql", "forge", "vault"
    DisplayName   string   // "PostgreSQL", "Forgejo", "Vaultwarden"
    
    // Networking
    TAPInterface  string   // "vm_sql", "vm_forge", "vm_vault"
    TAPHostIP     string   // "192.168.100.1", "192.168.101.1", "192.168.102.1"
    TAPGuestIP    string   // "192.168.100.2", "192.168.101.2", "192.168.102.2"
    TAPSubnet     string   // "192.168.100.0/24", etc.
    
    // Tailscale
    TailscaleHost string   // "sovereign-sql", "sovereign-forge"
    
    // Paths
    DevicePath    string   // "/data/sovereign/vm/sql"
    LocalPath     string   // "vm/sql"
    
    // Service
    ServicePorts  []int    // [5432], [3000, 22], [80]
    ReadyMarker   string   // "PostgreSQL started", "INIT COMPLETE"
    StartTimeout  int      // seconds: 90, 120
    
    // Build
    DockerImage   string   // "sovereign-sql", "sovereign-forge"
    SharedKernel  bool     // true for forge (uses sql's kernel)
    NeedsSecrets  bool     // true for sql (prompts for DB password)
}
```

---

## Success Criteria

1. **Zero duplication:** Common patterns extracted to `internal/vm/common/`
2. **Simple addition:** New service = config struct + Dockerfile + init.sh
3. **Backward compatible:** `sovereign build/deploy/start/stop/test/remove --sql` unchanged
4. **All tests pass:** Existing BDD scenarios still work
5. **Build compiles:** No regressions

---

## Constraints

- **No behavior changes:** All CLI commands must work identically
- **No breaking changes:** VM interface stays the same
- **Incremental migration:** SQL and Forge can coexist during refactor
- **Tests first:** Lock in behavior before refactoring

---

## Open Questions

1. Should shared kernel be a separate `internal/vm/kernel/` package?
2. Should we add a generic `--vm <name>` flag instead of `--sql`, `--forge`?
3. How to handle service-specific Build() logic (e.g., SQL needs secrets)?

---

## References

- SQL VM: `internal/vm/sql/`
- Forge VM: `internal/vm/forge/`
- VM interface: `internal/vm/vm.go`
- Device helpers: `internal/device/device.go`
