# Phase 3A, Step 1 — sovereign.go: The Foundation

**Phase:** 3A (KernelSU)
**Step:** 1 of 3
**Team:** TEAM_001
**Status:** Pending

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** I am about to create `sovereign.go`. This is not "just a helper script." This is THE foundation of the entire project. Every operation flows through this file.
>
> **My failure modes:**
> - Writing a throwaway script that "we'll improve later" (we won't)
> - Hardcoding paths that only work on my imaginary system
> - Skipping error handling because "it's just a prototype"
> - Not testing if it actually compiles
>
> **The rule:** `sovereign.go` must be production-quality from day one. It grows with the project.

---

## 1. Goal

Create `sovereign.go` — **the single orchestrator** for all Sovereign Vault operations.

---

## 2. Why sovereign.go is Fundamental

`sovereign.go` is not just a helper script. It is:

1. **The single entry point** for all build, deploy, and test operations
2. **Architecture-aware** — knows about the 3-VM design (database, vault, forge)
3. **Incrementally extensible** — starts with `--kernel`, grows to `--sql`, `--vault`, `--forge`
4. **The source of truth** for what the system can do

Every operation in Sovereign Vault flows through `sovereign.go`. No ad-hoc scripts.

---

## 3. Pre-Conditions

- [ ] Go 1.21+ installed (`go version`)
- [ ] Working directory is project root

---

## 4. Implementation

**File:** `sovereign.go` (project root)

```go
// sovereign.go - Sovereign Vault Management CLI
// TEAM_001: Phase 3A - Kernel-only version
// This file is THE FOUNDATION. All operations flow through here.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var flagKernel bool

func init() {
	flag.BoolVar(&flagKernel, "kernel", false, "Target kernel/KernelSU")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	flag.CommandLine.Parse(os.Args[2:])

	// Default to kernel for Phase 3A
	if !flagKernel {
		flagKernel = true
	}

	var err error
	switch command {
	case "build":
		err = cmdBuild()
	case "deploy":
		err = cmdDeploy()
	case "test":
		err = cmdTest()
	case "status":
		err = cmdStatus()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Sovereign Vault CLI

Usage: go run sovereign.go <command> [flags]

Commands:
  build     Build kernel with KernelSU
  deploy    Flash kernel to device
  test      Test KernelSU root access
  status    Show kernel/KernelSU status

Flags:
  --kernel  Target kernel operations (default for Phase 3A)

Examples:
  go run sovereign.go build --kernel
  go run sovereign.go deploy --kernel
  go run sovereign.go test --kernel
  go run sovereign.go status

Future flags (Phase 3B+):
  --sql     Target PostgreSQL VM
  --vault   Target Vaultwarden VM
  --forge   Target Forgejo VM`)
}

func cmdBuild() error {
	fmt.Println("=== Building Kernel with KernelSU ===")
	fmt.Println("Running: ./build_raviole.sh")
	
	cmd := exec.Command("./build_raviole.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = "."
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	
	// Verify CONFIG_KSU=y in output
	configPath := "out/raviole/dist/.config"
	if _, err := os.Stat(configPath); err == nil {
		configData, _ := os.ReadFile(configPath)
		if !strings.Contains(string(configData), "CONFIG_KSU=y") {
			return fmt.Errorf("CONFIG_KSU=y not found in %s - KernelSU not enabled!", configPath)
		}
		fmt.Println("✓ CONFIG_KSU=y confirmed")
	}
	
	fmt.Println("\n✓ Kernel build complete")
	fmt.Println("Output: out/raviole/dist/boot.img")
	return nil
}

func cmdDeploy() error {
	fmt.Println("=== Deploying Kernel ===")
	
	bootImg := "out/raviole/dist/boot.img"
	if _, err := os.Stat(bootImg); os.IsNotExist(err) {
		return fmt.Errorf("boot.img not found: %s\nRun 'sovereign build' first", bootImg)
	}
	
	fmt.Println("Step 1: Rebooting to bootloader...")
	if err := exec.Command("adb", "reboot", "bootloader").Run(); err != nil {
		return fmt.Errorf("adb reboot failed: %w", err)
	}
	
	fmt.Println("Waiting for fastboot...")
	if err := exec.Command("fastboot", "wait-for-device").Run(); err != nil {
		return fmt.Errorf("fastboot wait failed: %w", err)
	}
	
	fmt.Println("Step 2: Flashing boot.img...")
	cmd := exec.Command("fastboot", "flash", "boot", bootImg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("flash failed: %w", err)
	}
	
	fmt.Println("Step 3: Rebooting...")
	if err := exec.Command("fastboot", "reboot").Run(); err != nil {
		return fmt.Errorf("reboot failed: %w", err)
	}
	
	fmt.Println("\n✓ Kernel deployed")
	fmt.Println("Wait for device to boot, then run: go run sovereign.go test")
	return nil
}

func cmdTest() error {
	fmt.Println("=== Testing Kernel/KernelSU ===\n")
	allPassed := true
	
	// Test 1: Kernel version
	fmt.Print("1. Kernel version contains 'sovereign': ")
	out, err := exec.Command("adb", "shell", "cat", "/proc/version").Output()
	if err != nil {
		fmt.Println("✗ FAIL (cannot read)")
		allPassed = false
	} else if !strings.Contains(string(out), "sovereign") {
		fmt.Println("✗ FAIL")
		fmt.Printf("   Got: %s\n", strings.TrimSpace(string(out)))
		allPassed = false
	} else {
		fmt.Println("✓ PASS")
	}
	
	// Test 2: Root access
	fmt.Print("2. Root access via su: ")
	out, err = exec.Command("adb", "shell", "su", "-c", "id").Output()
	if err != nil {
		fmt.Println("✗ FAIL (su not working)")
		allPassed = false
	} else if !strings.Contains(string(out), "uid=0") {
		fmt.Println("✗ FAIL (not root)")
		fmt.Printf("   Got: %s\n", strings.TrimSpace(string(out)))
		allPassed = false
	} else {
		fmt.Println("✓ PASS")
	}
	
	// Test 3: KernelSU version
	fmt.Print("3. KernelSU version (not 16): ")
	out, err = exec.Command("adb", "shell", "su", "-v").Output()
	if err != nil {
		fmt.Println("✗ FAIL (cannot get version)")
		allPassed = false
	} else {
		version := strings.TrimSpace(string(out))
		if version == "16" || version == "" {
			fmt.Println("✗ FAIL (version is 16 - Kbuild patch not applied)")
			allPassed = false
		} else {
			fmt.Printf("✓ PASS (version: %s)\n", version)
		}
	}
	
	fmt.Println()
	if allPassed {
		fmt.Println("=== ALL TESTS PASSED ===")
		fmt.Println("Phase 3A complete! Root access working.")
		fmt.Println("\nNext: go run sovereign.go build --sql")
		return nil
	}
	return fmt.Errorf("some tests failed - see above")
}

func cmdStatus() error {
	fmt.Println("=== Sovereign Status ===\n")
	
	// Check device connection
	fmt.Print("Device connected: ")
	out, err := exec.Command("adb", "devices").Output()
	if err != nil || !strings.Contains(string(out), "device") {
		fmt.Println("✗ No")
		return nil
	}
	fmt.Println("✓ Yes")
	
	// Check kernel version
	fmt.Print("Kernel: ")
	out, err = exec.Command("adb", "shell", "uname", "-r").Output()
	if err != nil {
		fmt.Println("Unknown")
	} else {
		version := strings.TrimSpace(string(out))
		if strings.Contains(version, "sovereign") {
			fmt.Printf("%s ✓\n", version)
		} else {
			fmt.Printf("%s (not sovereign kernel)\n", version)
		}
	}
	
	// Check KernelSU
	fmt.Print("KernelSU: ")
	out, err = exec.Command("adb", "shell", "su", "-v").Output()
	if err != nil {
		fmt.Println("Not installed")
	} else {
		version := strings.TrimSpace(string(out))
		if version == "16" {
			fmt.Printf("Version %s ⚠️  (Kbuild patch needed)\n", version)
		} else {
			fmt.Printf("Version %s ✓\n", version)
		}
	}
	
	return nil
}
```

---

## 5. Verification

```bash
# Check Go is installed
go version

# Verify it compiles
go build -o /dev/null sovereign.go

# Test help
go run sovereign.go help

# Test status (before kernel is built)
go run sovereign.go status
```

---

## 6. Checkpoint

- [ ] `sovereign.go` created in project root
- [ ] Compiles without errors
- [ ] `go run sovereign.go help` shows usage
- [ ] `go run sovereign.go status` runs (even if device not connected)

---

## Next Step

Proceed to **[Phase 3A, Step 2 — KernelSU Integration](phase-3a-step-2.md)**
