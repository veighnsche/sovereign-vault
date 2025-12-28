# Phase 2: Structural Extraction

**Team**: TEAM_008  
**Goal**: Extract new files in parallel with existing code

---

## 2.1 Target Design

### File Layout
```
sovereign/
├── main.go          # CLI entry point only (~60 lines)
├── commands.go      # Command dispatch logic (~80 lines)
├── kernel.go        # Kernel build/deploy/test (~180 lines)
├── device.go        # ADB/fastboot utilities (~120 lines)
├── vm.go            # VM interface + registry (~60 lines)
├── vm_sql.go        # SQL VM implementation (~300 lines)
├── docker.go        # Docker export utilities (~80 lines)
├── rootfs.go        # Rootfs AVF preparation (~100 lines)
└── go.mod
```

### VM Interface Design
```go
// vm.go
type VM interface {
    Name() string
    Build() error
    Deploy() error
    Start() error
    Stop() error
    Test() error
    Prepare() error  // Optional - rootfs fixes
}

var vmRegistry = map[string]VM{
    "sql":   &SQLVM{},
    // "vault": &VaultVM{},  // Future
    // "forge": &ForgeVM{},  // Future
}

func GetVM(name string) (VM, bool)
func ListVMs() []string
```

---

## 2.2 Extraction Order

Extract in this order (dependencies first):

1. **device.go** - No dependencies, used by kernel and VMs
2. **docker.go** - No dependencies, used by VM build
3. **rootfs.go** - Depends on exec, used by VM build
4. **kernel.go** - Depends on device.go
5. **vm.go** - Interface definition
6. **vm_sql.go** - Implements VM interface, depends on device, docker, rootfs
7. **commands.go** - Dispatch logic using vm.go
8. **main.go** - Slim down to entry point

---

## 2.3 Step 1: Extract device.go

**Move these functions**:
- `waitForFastboot(timeoutSecs int) error`
- `waitForAdb(timeoutSecs int) error`
- `flashImage(partition, path string) error`
- `ensureBootloaderMode() error`
- `pushFileToDevice(localPath, remotePath string) error`

**File template**:
```go
// device.go - Android device utilities (ADB/fastboot)
package main

import (...)

// waitForFastboot waits for device to appear in fastboot mode
func waitForFastboot(timeoutSecs int) error { ... }

// waitForAdb waits for device to be available via ADB
func waitForAdb(timeoutSecs int) error { ... }

// flashImage flashes an image to a partition via fastboot
func flashImage(partition, path string) error { ... }

// ensureBootloaderMode ensures device is in bootloader mode
func ensureBootloaderMode() error { ... }

// pushFileToDevice pushes a file via ADB (through /data/local/tmp)
func pushFileToDevice(localPath, remotePath string) error { ... }
```

**Exit Criteria**: `go build` passes with device.go extracted

---

## 2.4 Step 2: Extract docker.go

**Move these functions**:
- `exportDockerImage(imageName, outputPath, size string) error`

**Exit Criteria**: `go build` passes

---

## 2.5 Step 3: Extract rootfs.go

**Move these functions**:
- `prepareRootfsForAVF(rootfsPath string) error`

**Exit Criteria**: `go build` passes

---

## 2.6 Step 4: Extract kernel.go

**Move these functions**:
- Kernel-specific parts of `cmdBuild()` → `BuildKernel() error`
- Kernel-specific parts of `cmdDeploy()` → `DeployKernel() error`
- Kernel-specific parts of `cmdTest()` → `TestKernel() error`

**Exit Criteria**: `go build` passes

---

## 2.7 Step 5: Create vm.go with interface

**Create**:
```go
// vm.go - VM interface and registry
package main

type VM interface {
    Name() string
    Build() error
    Deploy() error
    Start() error
    Stop() error
    Test() error
    Prepare() error
}

var vmRegistry = make(map[string]VM)

func RegisterVM(name string, vm VM) {
    vmRegistry[name] = vm
}

func GetVM(name string) (VM, bool) {
    vm, ok := vmRegistry[name]
    return vm, ok
}

func init() {
    RegisterVM("sql", &SQLVM{})
}
```

**Exit Criteria**: `go build` passes

---

## 2.8 Step 6: Extract vm_sql.go

**Move these functions**:
- `buildSQL()` → `(v *SQLVM) Build()`
- `deploySQL()` → `(v *SQLVM) Deploy()`
- `startSQL()` → `(v *SQLVM) Start()`
- `stopSQL()` → `(v *SQLVM) Stop()`
- `testSQL()` → `(v *SQLVM) Test()`
- `createSQLStartScript()` → helper in vm_sql.go (used by Deploy)

**Create SQLVM struct implementing VM interface**:
```go
// vm_sql.go - PostgreSQL VM implementation
package main

type SQLVM struct{}

func (v *SQLVM) Name() string { return "sql" }
func (v *SQLVM) Build() error { ... }   // from buildSQL
func (v *SQLVM) Deploy() error { ... }  // from deploySQL
func (v *SQLVM) Start() error { ... }   // from startSQL
func (v *SQLVM) Stop() error { ... }    // from stopSQL
func (v *SQLVM) Test() error { ... }    // from testSQL
func (v *SQLVM) Prepare() error { ... } // calls prepareRootfsForAVF

// createSQLStartScript is a helper for Deploy
func createSQLStartScript() error { ... }
```

**Exit Criteria**: `go build` passes

---

## 2.9 Step 7: Create commands.go

**Move command dispatch logic**:
```go
// commands.go - Command implementations
package main

func cmdBuild() error {
    if flagSQL {
        vm, _ := GetVM("sql")
        return vm.Build()
    }
    return BuildKernel()
}

// Similar for cmdDeploy, cmdTest, cmdStart, cmdStop, cmdPrepare, cmdStatus
```

**Exit Criteria**: `go build` passes

---

## 2.10 Step 8: Slim down main.go

**Keep only**:
- `var flagKernel, flagSQL bool`
- `func init()` with flag definitions
- `func main()` with command dispatch
- `func printUsage()`

**Exit Criteria**: `go build` passes, main.go < 100 lines
