# Phase 3: Migration

**Team**: TEAM_008  
**Goal**: Move call sites to new structure, remove old code paths

---

## 3.1 Migration Strategy

After Phase 2, we have:
- New files with extracted functions
- Old code still in main.go (duplicated)

Phase 3 removes the old code and ensures all paths use new structure.

---

## 3.2 Step 1: Verify Flag Handling Unchanged

**IMPORTANT**: This refactor does NOT add new flags. Future VMs (--vault, --forge) are OUT OF SCOPE.

**Current (keep as-is)**:
```go
var flagKernel bool
var flagSQL bool
```

**Rationale**: Adding speculative flags violates Rule 0 (Quality Over Speed) - we only refactor existing behavior, not add future features.

---

## 3.3 Step 2: Update commands.go to use VM interface

**Before** (in main.go):
```go
func cmdBuild() error {
    if flagSQL {
        return buildSQL()
    }
    // kernel code inline
}
```

**After** (in commands.go):
```go
func cmdBuild() error {
    if flagSQL {
        vm, _ := GetVM("sql")
        return vm.Build()
    }
    return BuildKernel()
}
```

**Note**: Only existing flags (--kernel, --sql) are handled. Future VMs will add their own flag handling when implemented.

---

## 3.4 Step 3: Remove Old Functions from main.go

Delete from main.go:
- [ ] `buildSQL()`
- [ ] `deploySQL()`
- [ ] `startSQL()`
- [ ] `stopSQL()`
- [ ] `testSQL()`
- [ ] `prepareRootfsForAVF()`
- [ ] `exportDockerImage()`
- [ ] `waitForFastboot()`
- [ ] `waitForAdb()`
- [ ] `flashImage()`
- [ ] `ensureBootloaderMode()`
- [ ] `pushFileToDevice()`
- [ ] `createSQLStartScript()`

**Exit Criteria**: main.go contains only:
- Package declaration
- Imports
- Flag variables
- `init()` for flags
- `main()` for dispatch
- `printUsage()`

---

## 3.5 Step 4: Verify printUsage() Unchanged

**Keep existing help output** - do NOT add "coming soon" flags.

The current `printUsage()` already documents `--kernel` and `--sql`. No changes needed unless refactor breaks output format.

**Verification**:
```bash
go run main.go help | diff - baselines/help.txt
```

---

## 3.6 Step 5: Verify All Commands Work

**Test matrix**:
| Command | Flag | Expected |
|---------|------|----------|
| build | --kernel | Runs build_raviole.sh |
| build | --sql | Builds Docker image |
| deploy | --kernel | Flashes via fastboot |
| deploy | --sql | Pushes via adb |
| start | --sql | Starts crosvm |
| stop | --sql | Kills crosvm |
| test | --kernel | Tests KernelSU |
| test | --sql | Tests PostgreSQL |
| status | (none) | Shows device info |
| status | --sql | Shows VM status |
| prepare | --sql | Fixes rootfs |

**Exit Criteria**: All commands produce same output as before refactor
