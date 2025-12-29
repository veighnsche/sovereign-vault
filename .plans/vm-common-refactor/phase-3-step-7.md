# Phase 3, Step 7: Migrate Forge config + Tailscale

**Feature:** VM Common Code Refactor  
**Status:** 🔲 NOT STARTED  
**Parent:** [phase-3.md](phase-3.md)  
**Depends On:** [phase-3-step-6.md](phase-3-step-6.md)

---

## Objective

Create Forge VMConfig and migrate RemoveTailscaleRegistrations to common package.

---

## Prerequisites

- [ ] Steps 1-6 complete (SQL fully migrated)
- [ ] Common package proven to work with SQL

---

## Current State

**File:** `internal/vm/forge/verify.go`

```go
// RemoveTailscaleRegistrations removes existing sovereign-forge Tailscale registrations
func RemoveTailscaleRegistrations() error {
    // ~110 lines - nearly identical to sql/verify.go
}
```

---

## Target State

**File:** `internal/vm/forge/forge.go` (add config)

```go
var forgeConfig = common.VMConfig{
    Name:          "forge",
    DisplayName:   "Forgejo",
    TAPInterface:  "vm_forge",
    TAPHostIP:     "192.168.101.1",
    TAPGuestIP:    "192.168.101.2",
    TAPSubnet:     "192.168.101.0/24",
    TailscaleHost: "sovereign-forge",
    DevicePath:    "/data/sovereign/vm/forgejo",
    LocalPath:     "vm/forgejo",
    ServicePorts:  []int{3000, 22},
    ReadyMarker:   "INIT COMPLETE",
    StartTimeout:  120,
    DockerImage:   "sovereign-forge",
    SharedKernel:  true,
    KernelSource:  "vm/sql/Image",
    NeedsSecrets:  false,
}
```

**File:** `internal/vm/forge/verify.go`

```go
// RemoveTailscaleRegistrations delegates to common package
func RemoveTailscaleRegistrations() error {
    return common.RemoveTailscaleRegistrations("sovereign-forge")
}
```

---

## Tasks

1. **Add forgeConfig to `internal/vm/forge/forge.go`**
   - Define VMConfig with Forge-specific values (see Target State)
   - Different subnet (192.168.101.x) than SQL (192.168.100.x)
   - SharedKernel: true, KernelSource: "vm/sql/Image"

2. **Replace RemoveTailscaleRegistrations() in verify.go**
   - Delegate to `common.RemoveTailscaleRegistrations("sovereign-forge")`
   - Delete ~110 lines of duplicated implementation

3. **Verify**
   - `go build ./...` succeeds
   - `sovereign remove --forge` still cleans up Tailscale

---

## Verification Commands

```bash
cd sovereign
go build ./...
# If device connected:
# sovereign remove --forge
```

---

## Exit Criteria

- [ ] forgeConfig defined with all Forge-specific values
- [ ] RemoveTailscaleRegistrations delegates to common
- [ ] ~110 lines removed from forge/verify.go
- [ ] Build succeeds
- [ ] Behavior unchanged
