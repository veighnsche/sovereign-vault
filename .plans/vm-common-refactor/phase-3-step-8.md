# Phase 3, Step 8: Migrate Forge lifecycle (Start/Stop/Remove)

**Feature:** VM Common Code Refactor  
**Status:** 🔲 NOT STARTED  
**Parent:** [phase-3.md](phase-3.md)  
**Depends On:** [phase-3-step-7.md](phase-3-step-7.md)

---

## Objective

Migrate Forge Start(), Stop(), and Remove() methods to use common lifecycle functions.

---

## Prerequisites

- [ ] Step 7 complete (forgeConfig defined)
- [ ] Common lifecycle functions proven to work with SQL

---

## Current State

**File:** `internal/vm/forge/lifecycle.go` (~160 lines)

```go
func (v *VM) Start() error {
    fmt.Println("=== Starting Forgejo VM ===")
    // ... ~50 lines
    return streamBootAndWaitForForgejo()
}

func streamBootAndWaitForForgejo() error {
    // ... ~50 lines
}

func (v *VM) Stop() error {
    // ... ~30 lines
}

func (v *VM) Remove() error {
    // ... ~25 lines
}
```

---

## Target State

**File:** `internal/vm/forge/lifecycle.go` (can be deleted)

```go
func (v *VM) Start() error {
    return common.StartVM(&forgeConfig)
}

func (v *VM) Stop() error {
    return common.StopVM(&forgeConfig)
}

func (v *VM) Remove() error {
    return common.RemoveVM(&forgeConfig)
}
```

Or move these 3 one-liners to forge.go and delete lifecycle.go entirely.

---

## Tasks

1. **Update Start() in lifecycle.go**
   - Replace implementation with `common.StartVM(&forgeConfig)`
   - Delete streamBootAndWaitForForgejo() (~50 lines)

2. **Update Stop() in lifecycle.go**
   - Replace implementation with `common.StopVM(&forgeConfig)`
   - Delete ~30 lines of implementation

3. **Update Remove() in lifecycle.go**
   - Replace implementation with `common.RemoveVM(&forgeConfig)`
   - Delete ~25 lines of implementation

4. **Consider consolidation**
   - Move the 3 one-liner methods to forge.go
   - Delete lifecycle.go entirely

5. **Verify**
   - `go build ./...` succeeds
   - `sovereign start --forge` works
   - `sovereign stop --forge` works
   - `sovereign remove --forge` works

---

## Verification Commands

```bash
cd sovereign
go build ./...
# If device connected:
# sovereign start --forge
# sovereign stop --forge
# sovereign remove --forge
```

---

## Exit Criteria

- [ ] Start() delegates to common.StartVM
- [ ] Stop() delegates to common.StopVM
- [ ] Remove() delegates to common.RemoveVM
- [ ] streamBootAndWaitForForgejo() deleted
- [ ] ~155 lines removed from forge/lifecycle.go
- [ ] Build succeeds
- [ ] Behavior unchanged
