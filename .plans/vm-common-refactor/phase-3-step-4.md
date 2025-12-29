# Phase 3, Step 4: Migrate SQL Deploy to common

**Feature:** VM Common Code Refactor  
**Status:** 🔲 NOT STARTED  
**Parent:** [phase-3.md](phase-3.md)  
**Depends On:** [phase-3-step-3.md](phase-3-step-3.md)

---

## Objective

Replace `Deploy()` method in SQL package with a call to common deploy function.

---

## Prerequisites

- [ ] Step 3 complete (Start migration)
- [ ] `internal/vm/common/deploy.go` has `DeployVM(cfg *VMConfig)` function

---

## Current State

**File:** `internal/vm/sql/sql.go`

```go
func (v *VM) Deploy() error {
    fmt.Println("=== Deploying PostgreSQL VM ===")
    
    // Verify images exist
    requiredFiles := []string{"vm/sql/rootfs.img", "vm/sql/data.img", "vm/sql/Image"}
    // ... file checks
    
    // Create directories
    device.MkdirP("/data/sovereign/vm/sql")
    
    // Push files
    device.PushFile("vm/sql/rootfs.img", "/data/sovereign/vm/sql/rootfs.img")
    // ... more pushes
    
    // Create start script
    createStartScript()
    
    // ~70 lines total
}
```

---

## Target State

**File:** `internal/vm/sql/sql.go`

```go
func (v *VM) Deploy() error {
    return common.DeployVM(&sqlConfig)
}
```

---

## Tasks

1. **Ensure sqlConfig has required deploy fields**
   - `LocalPath`: "vm/sql"
   - `DevicePath`: "/data/sovereign/vm/sql"
   - `RequiredFiles`: []string{"rootfs.img", "data.img", "Image"}
   - `StartScript`: "start.sh"

2. **Update Deploy() in sql.go**
   - Replace implementation with `common.DeployVM(&sqlConfig)`
   - Delete ~70 lines of implementation

3. **Delete createStartScript()**
   - Remove entirely - logic replaced by generic common.DeployVM
   - common.DeployVM will derive path from `cfg.LocalPath + "/start.sh"`
   - Push and chmod handled generically using config values

4. **Verify**
   - `go build ./...` succeeds
   - `sovereign deploy --sql` works
   - All files pushed correctly

---

## Notes

The common.DeployVM function should:
1. Check required files exist locally
2. Create device directories
3. Push all required files
4. Push .env if exists
5. chmod start script

---

## Verification Commands

```bash
cd sovereign
go build ./...
# If device connected:
# sovereign deploy --sql
```

---

## Exit Criteria

- [ ] Deploy() delegates to common.DeployVM
- [ ] createStartScript() removed or moved to common
- [ ] ~70 lines removed from sql/sql.go
- [ ] Build succeeds
- [ ] File pushing works correctly
- [ ] Behavior unchanged
