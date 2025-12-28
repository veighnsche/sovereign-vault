# TEAM_011: Build/Deploy Workflow Refactor

## Summary
Refactored the sovereign CLI to consolidate workflow and secure credentials.

## Changes Made

### 1. Removed `prepare` Command
- Merged rootfs preparation into `build` command
- Removed `Prepare()` from VM interface (`internal/vm/vm.go`)
- Removed `cmdPrepare()` from CLI (`cmd/sovereign/main.go`)
- Updated MockVM in tests

### 2. Secure Credentials Management
- Created `internal/secrets/secrets.go` module:
  - `PromptPassword()` - secure password entry (no echo)
  - `PromptCredentials()` - interactive DB credential setup
  - `WriteSecretsFile()` - writes to `.secrets` with mode 0600
  - `LoadSecretsFile()` - loads existing credentials
- Build prompts for password on first run, reuses on subsequent runs

### 3. Updated Build Process
- `sql.go:Build()` now prompts for credentials if `.secrets` doesn't exist
- Password injected into `simple_init` script via `DB_PASSWORD` env var
- `rootfs.PrepareForAVF()` now accepts password parameter

### 4. Device Package Improvements
- Added centralized device command methods:
  - `RunShellCommand()`, `GetProcessPID()`, `FileExists()`
  - `DirExists()`, `ReadFileContent()`, `RemoveDir()`
  - `MkdirP()`, `KillProcess()`, `GrepFile()`
- Updated `sql.go` to use device package instead of raw adb commands

### 5. Documentation
- Updated README.md with new workflow
- Added `.secrets` to `.gitignore`
- Added security section explaining credential handling

## New Workflow
```bash
./sovereign build --sql     # Prompts for DB password on first run
./sovereign deploy --sql    # Idempotent - creates dirs if needed
./sovereign start --sql
./sovereign test --sql
```

## Files Modified
- `internal/vm/vm.go` - Removed Prepare() from interface
- `internal/vm/sql/sql.go` - Added secrets integration, device package usage
- `internal/vm/forge/forge.go` - Removed Prepare(), updated Remove()
- `internal/vm/vm_test.go` - Removed Prepare from MockVM
- `internal/rootfs/rootfs.go` - Added dbPassword parameter
- `internal/rootfs/rootfs_test.go` - Updated test calls
- `cmd/sovereign/main.go` - Removed prepare command
- `internal/device/device.go` - Added abstraction methods
- `internal/secrets/secrets.go` - NEW: credential management
- `.gitignore` - Added .secrets
- `README.md` - Updated workflow documentation

## Handoff Checklist
- [x] Project builds cleanly
- [x] All tests pass (`go test -short ./...`)
- [x] No hardcoded credentials
- [x] Documentation updated
