# Phase 3, Step 6: Migrate SQL Test to common framework

**Feature:** VM Common Code Refactor  
**Status:** 🔲 NOT STARTED  
**Parent:** [phase-3.md](phase-3.md)  
**Depends On:** [phase-3-step-5.md](phase-3-step-5.md)

---

## Objective

Replace `Test()` method in SQL package with common test framework, keeping SQL-specific tests as custom test functions.

---

## Prerequisites

- [ ] Step 5 complete (Build migration)
- [ ] `internal/vm/common/test.go` has test framework

---

## Current State

**File:** `internal/vm/sql/verify.go`

```go
func (v *VM) Test() error {
    fmt.Println("=== Testing PostgreSQL VM ===")
    allPassed := true

    // Test 1: VM process running (COMMON)
    // Test 2: TAP interface (COMMON)
    // Test 3: Tailscale connected (COMMON)
    // Test 4: PostgreSQL responding (SQL-SPECIFIC)
    // Test 5: Can execute query (SQL-SPECIFIC)

    // ~100 lines total
}
```

---

## Target State

**File:** `internal/vm/sql/verify.go`

```go
func (v *VM) Test() error {
    return common.RunVMTests(&sqlConfig, sqlCustomTests)
}

var sqlCustomTests = []common.TestFunc{
    testPostgresResponding,
    testCanExecuteQuery,
}

func testPostgresResponding(cfg *common.VMConfig) common.TestResult {
    // ~15 lines - check port 5432 via TAP
}

func testCanExecuteQuery(cfg *common.VMConfig) common.TestResult {
    // ~20 lines - execute SELECT 1 via psql
}
```

---

## Common Tests (in common/test.go)

```go
type TestResult struct {
    Name    string
    Passed  bool
    Message string
}

type TestFunc func(cfg *VMConfig) TestResult

func RunVMTests(cfg *VMConfig, customTests []TestFunc) error {
    // 1. Test VM process running
    // 2. Test TAP interface UP
    // 3. Test Tailscale connected
    // 4. Run custom tests
    // 5. Print summary
}
```

---

## Tasks

1. **Define SQL-specific test functions**
   - `testPostgresResponding`: Check port 5432 on TAP IP
   - `testCanExecuteQuery`: Run SELECT 1 via psql

2. **Add custom tests to sqlConfig or pass separately**
   - `CustomTests: []common.TestFunc{...}`

3. **Update Test() in verify.go**
   - Replace with `common.RunVMTests(&sqlConfig, sqlCustomTests)`
   - Delete ~60 lines of common test logic
   - Keep ~35 lines in custom test functions

4. **Verify**
   - `go build ./...` succeeds
   - `sovereign test --sql` works
   - All 5 tests still run
   - Output format unchanged

---

## Verification Commands

```bash
cd sovereign
go build ./...
# If VM running:
# sovereign test --sql
```

---

## Exit Criteria

- [ ] Test() delegates to common.RunVMTests
- [ ] SQL-specific tests extracted to functions
- [ ] ~60 lines removed from sql/verify.go
- [ ] Build succeeds
- [ ] All tests run correctly
- [ ] Output unchanged
