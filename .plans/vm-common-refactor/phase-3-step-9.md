# Phase 3, Step 9: Migrate Forge Build/Deploy/Test + cleanup

**Feature:** VM Common Code Refactor  
**Status:** 🔲 NOT STARTED  
**Parent:** [phase-3.md](phase-3.md)  
**Depends On:** [phase-3-step-8.md](phase-3-step-8.md)

---

## Objective

Migrate Forge Build(), Deploy(), and Test() methods to common, then delete empty files.

---

## Prerequisites

- [ ] Step 8 complete (lifecycle migrated)
- [ ] Common build/deploy/test functions proven to work with SQL

---

## Current State

**File:** `internal/vm/forge/forge.go`

```go
func (v *VM) Build() error {
    // ~65 lines
}

func (v *VM) Deploy() error {
    // ~45 lines
}
```

**File:** `internal/vm/forge/verify.go`

```go
func (v *VM) Test() error {
    // ~85 lines
}

// RemoveTailscaleRegistrations already migrated in step 7
```

---

## Target State

**File:** `internal/vm/forge/forge.go` (~80 lines total)

```go
package forge

import (
    "github.com/anthropics/sovereign/internal/vm"
    "github.com/anthropics/sovereign/internal/vm/common"
)

var forgeConfig = common.VMConfig{
    // ... defined in step 7
}

func init() {
    vm.Register("forge", &VM{})
}

type VM struct{}

func (v *VM) Name() string    { return "forge" }
func (v *VM) Build() error    { return common.BuildVM(&forgeConfig) }
func (v *VM) Deploy() error   { return common.DeployVM(&forgeConfig) }
func (v *VM) Start() error    { return common.StartVM(&forgeConfig) }
func (v *VM) Stop() error     { return common.StopVM(&forgeConfig) }
func (v *VM) Remove() error   { return common.RemoveVM(&forgeConfig) }
func (v *VM) Test() error     { return common.RunVMTests(&forgeConfig, forgeCustomTests) }

var forgeCustomTests = []common.TestFunc{
    testForgejoWebUI,
    testSSHPort,
}

func testForgejoWebUI(cfg *common.VMConfig) common.TestResult {
    // Check HTTP response on port 3000 via Tailscale
    // ~15 lines
}

func testSSHPort(cfg *common.VMConfig) common.TestResult {
    // Check port 22 accessible
    // ~10 lines
}
```

---

## Tasks

1. **Update Build() in forge.go**
   - Replace implementation with `common.BuildVM(&forgeConfig)`
   - SharedKernel=true means skip kernel check, use KernelSource
   - NeedsSecrets=false means no hooks needed
   - Delete ~65 lines

2. **Update Deploy() in forge.go**
   - Replace implementation with `common.DeployVM(&forgeConfig)`
   - Delete ~45 lines

3. **Migrate Test() from verify.go to forge.go**
   - Add `common.RunVMTests(&forgeConfig, forgeCustomTests)`
   - Define forgeCustomTests with Forge-specific tests
   - Extract testForgejoWebUI and testSSHPort functions

4. **Delete empty files**
   - Delete `lifecycle.go` (all methods moved to forge.go or delegated)
   - Delete `verify.go` (Test moved, RemoveTailscaleRegistrations is 3-line wrapper)

5. **Verify**
   - `go build ./...` succeeds
   - `sovereign build --forge` works
   - `sovereign deploy --forge` works
   - `sovereign test --forge` works

---

## Verification Commands

```bash
cd sovereign
go build ./...
# Full Forge workflow test:
# sovereign build --forge
# sovereign deploy --forge
# sovereign start --forge
# sovereign test --forge
# sovereign stop --forge
# sovereign remove --forge
```

---

## Exit Criteria

- [ ] Build() delegates to common.BuildVM
- [ ] Deploy() delegates to common.DeployVM
- [ ] Test() delegates to common.RunVMTests with custom tests
- [ ] lifecycle.go deleted
- [ ] verify.go deleted (or reduced to 3-line wrapper)
- [ ] Only forge.go remains (~80 lines)
- [ ] ~220 lines removed total
- [ ] Build succeeds
- [ ] All Forge commands work
