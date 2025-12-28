# Phase 4: Cleanup and Hardening

**Team**: TEAM_008  
**Goal**: Remove dead code, tighten encapsulation, prepare for future VMs

---

## 4.1 Dead Code Removal

After migration, scan for:
- [ ] Unused imports in each file
- [ ] Unused functions
- [ ] Commented-out code blocks
- [ ] Duplicate helper functions

**Tool**: `go vet`, `staticcheck`, manual review

---

## 4.2 File Size Verification

Target: Each file < 300 lines

| File | Target Lines | Responsibility |
|------|-------------|----------------|
| main.go | ~60 | Entry point, flags, dispatch |
| commands.go | ~80 | Command implementations |
| kernel.go | ~180 | Kernel build/deploy/test |
| device.go | ~120 | ADB/fastboot utilities |
| vm.go | ~60 | VM interface + registry |
| vm_sql.go | ~300 | SQL VM implementation |
| docker.go | ~80 | Docker export utilities |
| rootfs.go | ~100 | Rootfs AVF preparation |

---

## 4.3 ~~Future VM Template~~ REMOVED

**REMOVED per Rule 6 (No Dead Code)**: Template files are dead code by definition. The README documentation (section 4.4) is sufficient for explaining how to add new VMs. Do NOT create `vm_template.go`.

---

## 4.4 Documentation Update

Update README.md with new structure:

```markdown
## Project Structure

```
sovereign/
├── main.go          # CLI entry point
├── commands.go      # Command dispatch
├── kernel.go        # Kernel operations
├── device.go        # ADB/fastboot utilities
├── vm.go            # VM interface
├── vm_sql.go        # PostgreSQL VM
├── docker.go        # Docker utilities
├── rootfs.go        # Rootfs preparation
└── go.mod
```

## Adding a New VM

1. Copy `vm_template.go` to `vm_<name>.go`
2. Implement the `VM` interface
3. Register in `init()`: `RegisterVM("name", &NameVM{})`
4. Add flag in `main.go`: `flag.BoolVar(&flagName, "name", false, "...")`
5. Add dispatch in `commands.go`
```

---

## 4.5 Lint Fixes

Address existing lint warnings:
- [ ] Line 304: Remove redundant newline in fmt.Println
- [ ] Line 362: Remove redundant newline in fmt.Println

---

## 4.6 Final Verification

**Build verification**:
```bash
go build -o sovereign .
go vet ./...
```

**Functional verification**:
```bash
./sovereign help
./sovereign status
./sovereign build --sql  # (if Docker available)
```

---

## 4.7 Handoff Checklist

- [ ] All files < 300 lines
- [ ] `go build` passes
- [ ] `go vet` passes with no warnings
- [ ] All commands work as before (verify with baselines)
- [ ] README updated with new structure
- [ ] Behavioral regression test passes: `go run main.go help | diff - baselines/help.txt`
- [ ] Team file updated with completion status
