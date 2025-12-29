# Phase 3, Step 3: Migrate SQL Start to common

**Feature:** VM Common Code Refactor  
**Status:** 🔲 NOT STARTED  
**Parent:** [phase-3.md](phase-3.md)  
**Depends On:** [phase-3-step-2.md](phase-3-step-2.md)

---

## Objective

Replace `Start()` method and `streamBootAndWaitForPostgres()` in SQL package with calls to common lifecycle functions.

---

## Prerequisites

- [ ] Step 2 complete (Stop/Remove migration)
- [ ] `internal/vm/common/lifecycle.go` has `StartVM(cfg *VMConfig)` and `StreamBootLogs(cfg *VMConfig)` functions

---

## Current State

**File:** `internal/vm/sql/lifecycle.go`

```go
func (v *VM) Start() error {
    fmt.Println("=== Starting PostgreSQL VM ===")
    runningPid := device.GetProcessPID("crosvm.*sql")
    // ... ~40 lines
    return streamBootAndWaitForPostgres()
}

func streamBootAndWaitForPostgres() error {
    // ... ~70 lines of boot streaming logic
}
```

---

## Target State

**File:** `internal/vm/sql/lifecycle.go`

```go
func (v *VM) Start() error {
    return common.StartVM(&sqlConfig)
}

// streamBootAndWaitForPostgres removed - now in common.StreamBootLogs
```

---

## Tasks

1. **Ensure sqlConfig has required fields**
   - `ReadyMarker`: "PostgreSQL started" or "INIT COMPLETE"
   - `StartTimeout`: 90 seconds
   - `ProcessPattern`: "crosvm.*sql"

2. **Update Start() in lifecycle.go**
   - Replace implementation with `common.StartVM(&sqlConfig)`
   - Delete ~40 lines of implementation

3. **Delete streamBootAndWaitForPostgres()**
   - Remove entire function (~70 lines)
   - This logic is now in common.StreamBootLogs

4. **Verify**
   - `go build ./...` succeeds
   - `sovereign start --sql` works
   - Boot log streaming still works

---

## Notes

The common.StartVM function should:
1. Check if VM already running (using ProcessPattern)
2. Clear old console log
3. Execute start script
4. Call StreamBootLogs with ReadyMarker and Timeout

---

## Verification Commands

```bash
cd sovereign
go build ./...
# If device connected and VM deployed:
# sovereign start --sql
```

---

## Exit Criteria

- [ ] Start() delegates to common.StartVM
- [ ] streamBootAndWaitForPostgres() deleted
- [ ] ~110 lines removed from sql/lifecycle.go
- [ ] Build succeeds
- [ ] Boot streaming still works
- [ ] Behavior unchanged
