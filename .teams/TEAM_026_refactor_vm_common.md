# TEAM_026: Refactor VM Common Code

**Created:** 2025-12-29
**Status:** ✅ Plan Complete (ready for implementation)
**Task:** Extract shared VM code to make adding new services trivial
**Plan:** `.plans/vm-common-refactor/`

## Motivation

Currently SQL and Forge VMs have ~300+ lines of duplicated code:
- Build (Docker, data disk creation)
- Deploy (file pushing, chmod)
- Start/Stop/Remove lifecycle
- Tailscale registration cleanup
- Test patterns

**Goal:** Make adding a new service (e.g., Vaultwarden) as simple as:
```go
var VaulwardenVM = common.NewVM(common.VMConfig{
    Name:          "vault",
    TailscaleHost: "sovereign-vault",
    TAPInterface:  "vm_vault",
    TAPHostIP:     "192.168.102.1",
    ServicePorts:  []int{80},
    // ...
})
```

## Scope

1. Create `internal/vm/common/` package
2. Extract shared lifecycle code
3. Extract Tailscale utilities
4. Extract build/deploy helpers
5. Migrate SQL and Forge to use common code
6. Remove duplication

## References

- Current SQL VM: `internal/vm/sql/`
- Current Forge VM: `internal/vm/forge/`
- Duplication analysis: ~300 lines shared patterns

## Progress

- [ ] Phase 1: Discovery and Safeguards
- [ ] Phase 2: Structural Extraction
- [ ] Phase 3: Migration
- [ ] Phase 4: Cleanup
- [ ] Phase 5: Hardening and Handoff
