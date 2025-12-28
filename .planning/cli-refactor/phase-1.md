# Phase 1: Discovery and Safeguards

**Team**: TEAM_008  
**Goal**: Understand current structure, establish behavioral contracts, ensure build passes

---

## 1.1 Refactor Summary

**What**: Split `main.go` (966 lines) into ~8 focused files  
**Why**: 
- Monolithic file is hard to navigate
- Adding vault/forge VMs will bloat it further
- No abstraction for VM operations (duplicated patterns)

**Success Criteria**:
- Each file < 300 lines
- VM operations share common interface
- Adding new VM requires only one new file
- All existing commands work identically

---

## 1.2 Behavioral Contracts

### CLI Commands (must remain identical)
```
build --kernel     → builds kernel via ../build_raviole.sh
build --sql        → builds SQL VM via Docker
deploy --kernel    → flashes kernel via fastboot
deploy --sql       → pushes VM files via adb
start --sql        → starts SQL VM via crosvm
stop --sql         → stops SQL VM
test --kernel      → tests KernelSU
test --sql         → tests SQL VM connectivity
status             → shows device/kernel/VM status
prepare --sql      → fixes rootfs for AVF
```

### Exit Codes
- 0: Success
- 1: Error (with message to stderr)

### Output Format
- Progress messages to stdout
- Errors to stderr
- Checkmarks (✓) and X marks (✗) for pass/fail

---

## 1.3 Current Architecture

```
main.go
├── CLI Layer (main, printUsage, flags)
├── Command Layer (cmdBuild, cmdDeploy, cmdTest, cmdStatus, cmdStart, cmdStop, cmdPrepare)
├── Kernel Domain
│   ├── build kernel (lines 106-130)
│   ├── deploy kernel (lines 137-216)
│   └── test kernel (lines 304-358)
├── SQL VM Domain
│   ├── buildSQL (lines 446-534)
│   ├── deploySQL (lines 690-753)
│   ├── startSQL (lines 794-834)
│   ├── stopSQL (lines 837-862)
│   └── testSQL (lines 865-965)
├── Rootfs Utilities
│   ├── prepareRootfsForAVF (lines 537-627)
│   └── exportDockerImage (lines 630-687)
└── Device Utilities
    ├── waitForFastboot (lines 219-229)
    ├── waitForAdb (lines 232-247)
    ├── flashImage (lines 250-258)
    ├── ensureBootloaderMode (lines 261-297)
    └── pushFileToDevice (lines 756-772)
```

---

## 1.4 Constraints

1. **No behavior changes** - refactor only, no new features
2. **Build must pass** after each extraction step
3. **Keep same package** - all files in `package main`
4. **Preserve all comments** - especially TEAM_XXX markers

---

## 1.5 Step 1: Verify Current Build

**Tasks**:
1. Run `go build` - must pass
2. Run `go vet` - note any warnings
3. Document current line count

**Exit Criteria**: Build passes, baseline established

---

## 1.5b Step 1b: Capture Behavioral Baselines (Rule 4)

**Tasks**:
1. Capture help output: `go run main.go help > baselines/help.txt`
2. Capture error for missing target: `go run main.go start 2>&1 | head -5 > baselines/start_no_target.txt`
3. Document expected exit codes in `baselines/exit_codes.md`

**Exit Criteria**: Baseline files created in `baselines/` directory

**Regression Check**: After each extraction step, verify:
```bash
go run main.go help | diff - baselines/help.txt
```

---

## 1.6 Step 2: Document Function Dependencies

**Tasks**:
1. List all exported functions (none expected - package main)
2. Map function call graph
3. Identify shared utilities

**Call Graph**:
```
cmdBuild → buildSQL → exportDockerImage, prepareRootfsForAVF
cmdDeploy → deploySQL → pushFileToDevice, createSQLStartScript
         → flashImage, ensureBootloaderMode, waitForFastboot, waitForAdb
cmdStart → startSQL
cmdStop → stopSQL
cmdTest → testSQL
cmdStatus → (adb commands)
cmdPrepare → prepareRootfsForAVF
```

**Shared Utilities**:
- `pushFileToDevice` - used by deploySQL, createSQLStartScript
- `waitForFastboot`, `waitForAdb` - used by kernel deploy
- `exec.Command` wrappers - ad-hoc throughout

---

## 1.7 Open Questions

1. Should device utilities be a separate package or same package?
   **Decision**: Same package (package main) for simplicity

2. Should VM interface use Go interfaces or just naming convention?
   **Decision**: Use interfaces for future extensibility

3. How to handle path constants (`../vm/sql/` etc)?
   **Decision**: Keep as constants, possibly in a `paths.go` later
