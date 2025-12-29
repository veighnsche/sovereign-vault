# Phase 2: Structural Extraction

**Feature:** VM Common Code Refactor  
**Status:** ✅ COMPLETE (TEAM_029)  
**Parent:** [VM_COMMON_REFACTOR_PLAN.md](VM_COMMON_REFACTOR_PLAN.md)  
**Depends On:** [Phase 1](phase-1.md)

---

## Purpose

Create the `internal/vm/common/` package with shared infrastructure. Old and new code coexist during this phase.

---

## Target Design

### New Package Structure

```
internal/vm/common/
├── config.go      # VMConfig struct + NewVM constructor
├── build.go       # BuildDockerImage, CreateDataDisk, PrepareRootfs
├── deploy.go      # DeployVM, PushFiles, CreateStartScript
├── lifecycle.go   # StartVM, StopVM, RemoveVM, StreamBootLogs
├── tailscale.go   # RemoveTailscaleRegistrations, CheckTailscaleConnected
└── test.go        # RunVMTests, TestVMProcess, TestTAPInterface, TestTailscale
```

### VMConfig Struct

```go
type VMConfig struct {
    // Identity
    Name          string   // "sql", "forge"
    DisplayName   string   // "PostgreSQL", "Forgejo"
    
    // Networking
    TAPInterface  string   // "vm_sql", "vm_forge"
    TAPHostIP     string   // "192.168.100.1"
    TAPGuestIP    string   // "192.168.100.2"
    TAPSubnet     string   // "192.168.100.0/24"
    
    // Tailscale
    TailscaleHost string   // "sovereign-sql"
    
    // Paths
    DevicePath    string   // "/data/sovereign/vm/sql"
    LocalPath     string   // "vm/sql"
    
    // Service
    ServicePorts  []int    // [5432]
    ReadyMarker   string   // "PostgreSQL started"
    StartTimeout  int      // 90
    
    // Build options
    DockerImage   string   // "sovereign-sql"
    SharedKernel  bool     // false (true for forge)
    KernelSource  string   // "vm/sql/Image" (where to get kernel)
    NeedsSecrets  bool     // true (prompts for DB password)
    
    // Hooks for service-specific logic
    PreBuildHook  func() error
    PostBuildHook func() error
    CustomTests   []TestFunc
}
```

---

## Extraction Strategy

### Order of Extraction

1. **config.go** - Define VMConfig struct (no dependencies)
2. **tailscale.go** - Extract RemoveTailscaleRegistrations (most duplicated)
3. **lifecycle.go** - Extract Stop, Remove (simpler lifecycle ops)
4. **test.go** - Extract common test patterns
5. **deploy.go** - Extract file pushing logic
6. **build.go** - Extract Docker build, data disk creation

### Coexistence Strategy

During extraction:
- New functions in `common/` are created
- Old functions in `sql/` and `forge/` remain unchanged
- No call sites are changed yet
- Both old and new compile simultaneously

---

## Modular Refactoring Rules

Per Rule 7:
- Each module owns its state
- Private fields, intentional public APIs
- No deep relative imports
- File sizes < 500 lines ideal, < 1000 max

---

## Steps

| Step | File | Description |
|------|------|-------------|
| 1 | [phase-2-step-1.md](phase-2-step-1.md) | Create config.go with VMConfig struct |
| 2 | [phase-2-step-2.md](phase-2-step-2.md) | Extract tailscale.go (registration cleanup) |
| 3 | [phase-2-step-3.md](phase-2-step-3.md) | Extract lifecycle.go (Stop, Remove) |
| 4 | [phase-2-step-4.md](phase-2-step-4.md) | Extract lifecycle.go (Start, StreamLogs) |
| 5 | [phase-2-step-5.md](phase-2-step-5.md) | Extract test.go (common test patterns) |
| 6 | [phase-2-step-6.md](phase-2-step-6.md) | Extract deploy.go (file pushing) |
| 7 | [phase-2-step-7.md](phase-2-step-7.md) | Extract build.go (Docker, data disk) |

---

## Exit Criteria

- [ ] `internal/vm/common/` package created with all files
- [ ] `go build ./...` succeeds
- [ ] Old SQL/Forge code unchanged and still works
- [ ] New common functions tested in isolation
- [ ] Ready for Phase 3 (migration)
