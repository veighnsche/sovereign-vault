# Phase 5: Hardening and Handoff

**Feature:** VM Common Code Refactor  
**Status:** 🔲 NOT STARTED  
**Parent:** [VM_COMMON_REFACTOR_PLAN.md](VM_COMMON_REFACTOR_PLAN.md)  
**Depends On:** [Phase 4](phase-4.md)

---

## Purpose

Final verification, documentation, and proof that adding new services is trivial.

---

## Final Verification Checklist

### Build & Test
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `go vet ./...` clean

### CLI Behavior Unchanged
- [ ] `sovereign build --sql` works
- [ ] `sovereign build --forge` works
- [ ] `sovereign deploy --sql` works
- [ ] `sovereign deploy --forge` works
- [ ] `sovereign start --sql` works
- [ ] `sovereign start --forge` works
- [ ] `sovereign stop --sql` works
- [ ] `sovereign stop --forge` works
- [ ] `sovereign test --sql` works
- [ ] `sovereign test --forge` works
- [ ] `sovereign remove --sql` works
- [ ] `sovereign remove --forge` works

### BDD Scenarios
- [ ] All `features/sql.feature` scenarios documented
- [ ] All `features/forge.feature` scenarios documented

---

## Documentation Updates

### Update README.md
- Document `internal/vm/common/` package
- Explain VMConfig struct
- Add "Adding a New Service" section

### Create SERVICE_TEMPLATE.md
Document the minimal files needed for a new service:
1. `vm/<service>/Dockerfile`
2. `vm/<service>/init.sh`
3. `vm/<service>/config/` (optional)
4. Go config in `internal/vm/<service>/`

---

## Proof: Add Example Service Skeleton

Create a minimal Vaultwarden skeleton to prove the pattern works:

```go
// internal/vm/vault/vault.go
package vault

import "github.com/anthropics/sovereign/internal/vm/common"

var config = common.VMConfig{
    Name:          "vault",
    DisplayName:   "Vaultwarden",
    TAPInterface:  "vm_vault",
    TAPHostIP:     "192.168.102.1",
    TAPGuestIP:    "192.168.102.2",
    TAPSubnet:     "192.168.102.0/24",
    TailscaleHost: "sovereign-vault",
    DevicePath:    "/data/sovereign/vm/vault",
    LocalPath:     "vm/vault",
    ServicePorts:  []int{80},
    ReadyMarker:   "Vaultwarden started",
    StartTimeout:  60,
    DockerImage:   "sovereign-vault",
    SharedKernel:  true,
    KernelSource:  "vm/sql/Image",
}

func init() {
    vm.Register("vault", common.NewVM(config))
}
```

**Lines of code:** ~25

Compare to current SQL/Forge: ~400+ lines each.

---

## Handoff Notes

### For Future Teams

1. **Adding a new service:**
   - Create `vm/<service>/Dockerfile`
   - Create `vm/<service>/init.sh`
   - Create `internal/vm/<service>/<service>.go` with config
   - Add CLI flag in `cmd/sovereign/main.go`
   - Add BDD scenarios in `features/<service>.feature`

2. **Modifying common behavior:**
   - All VMs affected - test all of them
   - Check VMConfig for service-specific overrides

3. **Debugging:**
   - Common lifecycle code in `internal/vm/common/`
   - Service-specific tests in `internal/vm/<service>/`

---

## Steps

| Step | File | Description |
|------|------|-------------|
| 1 | [phase-5-step-1.md](phase-5-step-1.md) | Run full verification checklist |
| 2 | [phase-5-step-2.md](phase-5-step-2.md) | Update documentation |
| 3 | [phase-5-step-3.md](phase-5-step-3.md) | Create Vaultwarden skeleton as proof |
| 4 | [phase-5-step-4.md](phase-5-step-4.md) | Update team file with completion |

---

## Exit Criteria

- [ ] All verification checks pass
- [ ] Documentation updated
- [ ] Vaultwarden skeleton works (or at least compiles)
- [ ] Team file updated with completion notes
- [ ] Refactor complete
