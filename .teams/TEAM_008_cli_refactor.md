# TEAM_008: CLI Refactor Plan

## Task
Refactor main.go (966 lines) into smaller, more readable files with a future outlook for adding Vault and Forge VMs.

## Current State Analysis

### File: main.go (966 lines)
Contains mixed responsibilities:

| Lines | Responsibility | Functions |
|-------|---------------|-----------|
| 1-100 | CLI dispatch & flags | main, printUsage, init |
| 102-131 | Kernel build | cmdBuild (kernel path) |
| 133-217 | Kernel deploy | cmdDeploy (kernel path) |
| 219-298 | Device utilities | waitForFastboot, waitForAdb, flashImage, ensureBootloaderMode |
| 300-358 | Kernel test | cmdTest (kernel path) |
| 360-415 | Status | cmdStatus |
| 419-443 | VM dispatch | cmdPrepare, cmdStart, cmdStop |
| 446-534 | SQL build | buildSQL |
| 537-627 | AVF prep | prepareRootfsForAVF |
| 629-688 | Docker export | exportDockerImage |
| 690-792 | SQL deploy | deploySQL, pushFileToDevice, createSQLStartScript |
| 794-862 | SQL start/stop | startSQL, stopSQL |
| 864-965 | SQL test | testSQL |

### Pain Points
1. **Monolithic** - 966 lines in one file
2. **Mixed concerns** - kernel, SQL VM, device utils all tangled
3. **Hard to extend** - adding vault/forge means more if/else chains
4. **No abstraction** - each VM has its own functions with duplicated patterns

## Target Structure

```
sovereign/
├── main.go           # CLI entry, dispatch only (~80 lines)
├── cmd.go            # Command implementations (build, deploy, etc.)
├── kernel.go         # Kernel-specific operations
├── vm.go             # VM interface + common VM operations
├── vm_sql.go         # SQL VM implementation
├── vm_vault.go       # (future) Vault VM implementation
├── vm_forge.go       # (future) Forge VM implementation
├── device.go         # ADB/fastboot device utilities
├── docker.go         # Docker image export utilities
├── rootfs.go         # Rootfs preparation (AVF fixes)
└── go.mod
```

## Refactor Strategy

### Phase 1: Discovery and Safeguards
- Map current behavior
- Ensure build passes before/after each step

### Phase 2: Structural Extraction
- Extract device utilities → device.go
- Extract docker utilities → docker.go
- Extract rootfs prep → rootfs.go
- Extract kernel operations → kernel.go
- Create VM interface → vm.go
- Extract SQL VM → vm_sql.go

### Phase 3: Migration
- Update main.go to use new structure
- Clean up cmd dispatch

### Phase 4: Cleanup
- Remove dead code
- Finalize interfaces for vault/forge extensibility

## Progress
- [x] Phase 1 created
- [x] Phase 2 created
- [x] Phase 3 created
- [x] Phase 4 created

## Checklist
- [x] All phases documented
- [ ] Build passes after plan review
