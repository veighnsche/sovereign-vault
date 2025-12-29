# Phase 3, Step 2: Migrate SQL Stop/Remove to common

**Feature:** VM Common Code Refactor  
**Status:** 🔲 NOT STARTED  
**Parent:** [phase-3.md](phase-3.md)  
**Depends On:** [phase-3-step-1.md](phase-3-step-1.md)

---

## Objective

Replace `Stop()` and `Remove()` methods in SQL package with calls to common lifecycle functions.

---

## Prerequisites

- [ ] Step 1 complete (Tailscale migration)
- [ ] `internal/vm/common/lifecycle.go` has `StopVM(cfg *VMConfig)` and `RemoveVM(cfg *VMConfig)` functions

---

## Current State

**File:** `internal/vm/sql/lifecycle.go`

```go
func (v *VM) Stop() error {
    fmt.Println("=== Stopping PostgreSQL VM ===")
    pid := device.GetProcessPID("[c]rosvm.*sql")
    // ... ~30 lines of implementation
}

func (v *VM) Remove() error {
    fmt.Println("=== Removing SQL VM from device ===")
    v.Stop()
    // ... ~25 lines of implementation
}
```

---

## Target State

**File:** `internal/vm/sql/sql.go` (add config)

```go
var sqlConfig = common.VMConfig{
    Name:          "sql",
    DisplayName:   "PostgreSQL",
    TAPInterface:  "vm_sql",
    TAPHostIP:     "192.168.100.1",
    TAPGuestIP:    "192.168.100.2",
    TAPSubnet:     "192.168.100.0/24",
    TailscaleHost: "sovereign-sql",
    DevicePath:    "/data/sovereign/vm/sql",
    // ...
}
```

**File:** `internal/vm/sql/lifecycle.go`

```go
func (v *VM) Stop() error {
    return common.StopVM(&sqlConfig)
}

func (v *VM) Remove() error {
    return common.RemoveVM(&sqlConfig)
}
```

---

## Tasks

1. **Add sqlConfig to `internal/vm/sql/sql.go`**
   - Define VMConfig with SQL-specific values
   - Export or make package-level variable

2. **Update Stop() in lifecycle.go**
   - Replace implementation with `common.StopVM(&sqlConfig)`
   - Delete ~30 lines of implementation

3. **Update Remove() in lifecycle.go**
   - Replace implementation with `common.RemoveVM(&sqlConfig)`
   - Delete ~25 lines of implementation

4. **Verify**
   - `go build ./...` succeeds
   - `sovereign stop --sql` works
   - `sovereign remove --sql` works

---

## Verification Commands

```bash
cd sovereign
go build ./...
# If device connected:
# sovereign stop --sql
# sovereign remove --sql
```

---

## Exit Criteria

- [ ] sqlConfig defined in sql.go
- [ ] Stop() delegates to common.StopVM
- [ ] Remove() delegates to common.RemoveVM
- [ ] ~55 lines removed from sql/lifecycle.go
- [ ] Build succeeds
- [ ] Behavior unchanged
