# Phase 3, Step 1: Migrate SQL to use common.RemoveTailscaleRegistrations

**Feature:** VM Common Code Refactor  
**Status:** 🔲 NOT STARTED  
**Parent:** [phase-3.md](phase-3.md)

---

## Objective

Replace the duplicated `RemoveTailscaleRegistrations()` function in SQL package with a call to the common package.

---

## Prerequisites

- [ ] Phase 2 complete (common package exists with `tailscale.go`)
- [ ] `internal/vm/common/tailscale.go` has `RemoveTailscaleRegistrations(hostnamePrefix string)` function

---

## Current State

**File:** `internal/vm/sql/verify.go`

```go
// RemoveTailscaleRegistrations removes existing sovereign-sql Tailscale registrations
func RemoveTailscaleRegistrations() error {
    // ~110 lines of implementation
}
```

---

## Target State

**File:** `internal/vm/sql/verify.go`

```go
import "github.com/anthropics/sovereign/internal/vm/common"

// RemoveTailscaleRegistrations delegates to common package
func RemoveTailscaleRegistrations() error {
    return common.RemoveTailscaleRegistrations("sovereign-sql")
}
```

---

## Tasks

1. **Update import in `internal/vm/sql/verify.go`**
   - Add `"github.com/anthropics/sovereign/internal/vm/common"`

2. **Replace function body**
   - Keep the function signature (for backward compatibility within package)
   - Delegate to `common.RemoveTailscaleRegistrations("sovereign-sql")`

3. **Remove dead code**
   - Delete the ~100 lines of implementation
   - Keep only the 3-line wrapper

4. **Verify**
   - `go build ./...` succeeds
   - `sovereign remove --sql` still works

---

## Verification Commands

```bash
cd sovereign
go build ./...
# If device connected:
# sovereign remove --sql
```

---

## Exit Criteria

- [ ] SQL package imports common
- [ ] RemoveTailscaleRegistrations delegates to common
- [ ] ~100 lines removed from sql/verify.go
- [ ] Build succeeds
- [ ] Behavior unchanged
