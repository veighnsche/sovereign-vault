# Phase 3, Step 5: Migrate SQL Build to common (with hooks)

**Feature:** VM Common Code Refactor  
**Status:** 🔲 NOT STARTED  
**Parent:** [phase-3.md](phase-3.md)  
**Depends On:** [phase-3-step-4.md](phase-3-step-4.md)

---

## Objective

Replace `Build()` method in SQL package with a call to common build function, using hooks for SQL-specific logic (secrets prompt).

---

## Prerequisites

- [ ] Step 4 complete (Deploy migration)
- [ ] `internal/vm/common/build.go` has `BuildVM(cfg *VMConfig)` function
- [ ] VMConfig supports `PreBuildHook` and `PostBuildHook` function fields

---

## Current State

**File:** `internal/vm/sql/sql.go`

```go
func (v *VM) Build() error {
    fmt.Println("=== Building PostgreSQL VM ===")
    
    // Check Docker
    if !docker.IsAvailable() { ... }
    
    // SQL-SPECIFIC: Prompt for credentials
    if secrets.SecretsExist() {
        creds, _ = secrets.LoadSecretsFile()
    } else {
        creds, _ = secrets.PromptCredentials("postgres")
        secrets.WriteSecretsFile(creds)
    }
    
    // Docker build
    cmd := exec.Command("docker", "build", ...)
    
    // Export rootfs
    docker.ExportImage(...)
    
    // Check kernel
    // Create data disk
    // Prepare rootfs with DB password
    
    // ~100 lines total
}
```

---

## Target State

**File:** `internal/vm/sql/sql.go`

```go
var sqlConfig = common.VMConfig{
    // ... other fields
    NeedsSecrets:  true,
    PreBuildHook:  sqlPreBuild,
    PostBuildHook: nil,
}

// sqlPreBuild handles SQL-specific credential prompting
func sqlPreBuild(cfg *common.VMConfig) (*secrets.Credentials, error) {
    if secrets.SecretsExist() {
        return secrets.LoadSecretsFile()
    }
    creds, err := secrets.PromptCredentials("postgres")
    if err != nil {
        return nil, err
    }
    return creds, secrets.WriteSecretsFile(creds)
}

func (v *VM) Build() error {
    return common.BuildVM(&sqlConfig)
}
```

---

## Tasks

1. **Define sqlPreBuild hook function**
   - Extract secrets prompting logic
   - Return credentials for rootfs preparation

2. **Add hook fields to sqlConfig**
   - `NeedsSecrets: true`
   - `PreBuildHook: sqlPreBuild`

3. **Update Build() in sql.go**
   - Replace implementation with `common.BuildVM(&sqlConfig)`
   - Delete ~80 lines of generic build logic
   - Keep ~15 lines in hook function

4. **Verify**
   - `go build ./...` succeeds
   - `sovereign build --sql` works
   - Credential prompting still works
   - Rootfs prepared with DB password

---

## Design Decision: Hooks vs Subclassing

Using hooks (function fields) instead of interface subclassing because:
- Simpler mental model
- SQL is the only VM that needs secrets
- Avoids complex inheritance hierarchy

---

## Verification Commands

```bash
cd sovereign
go build ./...
# Test build (may need to remove existing images first):
# sovereign build --sql
```

---

## Exit Criteria

- [ ] Build() delegates to common.BuildVM
- [ ] sqlPreBuild hook handles credentials
- [ ] ~80 lines removed from sql/sql.go
- [ ] Build succeeds
- [ ] Credential prompting works
- [ ] Rootfs prepared correctly
