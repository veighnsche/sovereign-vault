// Package preflight provides prerequisite checks before running sovereign commands
// TEAM_036: Created to avoid wasting time on missing dependencies
package preflight

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CheckResult represents the result of a single preflight check
type CheckResult struct {
	Name     string
	Required bool // true = must pass, false = warning only
	Passed   bool
	Message  string
}

// Results holds all preflight check results
type Results struct {
	Checks  []CheckResult
	Passed  bool
	Command string
}

// commandExists checks if a command is available in PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// dockerRunning checks if Docker daemon is running
func dockerRunning() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// adbConnected checks if a device is connected via adb
func adbConnected() bool {
	cmd := exec.Command("adb", "get-state")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "device"
}

// getKernelDir returns the kernel directory (parent of sovereign-vault)
func getKernelDir() string {
	exe, err := os.Executable()
	if err != nil {
		// Fallback to current directory traversal
		cwd, _ := os.Getwd()
		return filepath.Dir(cwd)
	}
	// Go up from sovereign-vault to kernel dir
	return filepath.Dir(filepath.Dir(filepath.Dir(exe)))
}

// CheckForCommand checks common prerequisites needed for any command
func CheckForCommand(command string, vmNames []string) *Results {
	results := &Results{
		Command: command,
		Passed:  true,
	}

	kernelDir := getKernelDir()

	// Always check adb for device operations
	needsDevice := command == "deploy" || command == "start" || command == "stop" ||
		command == "test" || command == "remove" || command == "status"

	needsDocker := command == "build"
	needsKernelBuild := false // For future kernel builds

	// Check: adb command available
	adbCheck := CheckResult{
		Name:     "adb",
		Required: needsDevice,
	}
	if commandExists("adb") {
		adbCheck.Passed = true
		adbCheck.Message = "adb found in PATH"
	} else {
		adbCheck.Passed = false
		adbCheck.Message = "adb not found - install Android SDK platform-tools"
	}
	results.Checks = append(results.Checks, adbCheck)

	// Check: adb device connected (only if adb exists and device operations needed)
	if needsDevice && adbCheck.Passed {
		deviceCheck := CheckResult{
			Name:     "adb device",
			Required: true,
		}
		if adbConnected() {
			deviceCheck.Passed = true
			deviceCheck.Message = "device connected"
		} else {
			deviceCheck.Passed = false
			deviceCheck.Message = "no device connected - run 'adb devices' to check"
		}
		results.Checks = append(results.Checks, deviceCheck)
	}

	// Check: Docker available and running (for build command)
	if needsDocker {
		dockerCmdCheck := CheckResult{
			Name:     "docker",
			Required: true,
		}
		if commandExists("docker") {
			dockerCmdCheck.Passed = true
			dockerCmdCheck.Message = "docker found in PATH"
		} else {
			dockerCmdCheck.Passed = false
			dockerCmdCheck.Message = "docker not found - install Docker (https://docs.docker.com/get-docker/)"
		}
		results.Checks = append(results.Checks, dockerCmdCheck)

		// Only check daemon if docker command exists
		if dockerCmdCheck.Passed {
			dockerDaemonCheck := CheckResult{
				Name:     "docker daemon",
				Required: true,
			}
			if dockerRunning() {
				dockerDaemonCheck.Passed = true
				dockerDaemonCheck.Message = "docker daemon running"
			} else {
				dockerDaemonCheck.Passed = false
				dockerDaemonCheck.Message = "docker daemon not running - start with 'sudo systemctl start docker' or 'sudo dockerd'"
			}
			results.Checks = append(results.Checks, dockerDaemonCheck)
		}

		// Check: qemu-user-static for cross-arch builds
		qemuCheck := CheckResult{
			Name:     "qemu-user-static",
			Required: false, // Warning only - may work without it on ARM hosts
		}
		if fileExists("/usr/bin/qemu-aarch64-static") || fileExists("/usr/bin/qemu-aarch64") {
			qemuCheck.Passed = true
			qemuCheck.Message = "qemu-user-static found (cross-arch builds supported)"
		} else {
			// Check if we're on ARM64 natively
			cmd := exec.Command("uname", "-m")
			output, _ := cmd.Output()
			if strings.Contains(string(output), "aarch64") {
				qemuCheck.Passed = true
				qemuCheck.Message = "native ARM64 host (qemu not needed)"
			} else {
				qemuCheck.Passed = false
				qemuCheck.Message = "qemu-user-static not found - install for cross-arch builds: sudo apt install qemu-user-static"
			}
		}
		results.Checks = append(results.Checks, qemuCheck)
	}

	// Check: .env file exists (for build/deploy)
	if command == "build" || command == "deploy" {
		envCheck := CheckResult{
			Name:     ".env file",
			Required: true,
		}
		if fileExists(".env") {
			envCheck.Passed = true
			envCheck.Message = ".env file found"
		} else if fileExists(filepath.Join(kernelDir, "sovereign-vault", ".env")) {
			envCheck.Passed = true
			envCheck.Message = ".env file found in sovereign-vault/"
		} else {
			envCheck.Passed = false
			envCheck.Message = ".env file not found - copy from .env.example and configure"
		}
		results.Checks = append(results.Checks, envCheck)
	}

	// Check: Kernel build prerequisites (for future kernel builds)
	if needsKernelBuild {
		clangPath := filepath.Join(kernelDir, "prebuilts/clang/host/linux-x86/clang-r487747c/bin/clang")
		clangCheck := CheckResult{
			Name:     "clang toolchain",
			Required: true,
		}
		if fileExists(clangPath) {
			clangCheck.Passed = true
			clangCheck.Message = "clang toolchain found"
		} else {
			clangCheck.Passed = false
			clangCheck.Message = fmt.Sprintf("clang not found at %s", clangPath)
		}
		results.Checks = append(results.Checks, clangCheck)

		opensslPath := filepath.Join(kernelDir, "prebuilts/kernel-build-tools/linux-x86/include/openssl")
		opensslCheck := CheckResult{
			Name:     "OpenSSL headers",
			Required: true,
		}
		if fileExists(opensslPath) {
			opensslCheck.Passed = true
			opensslCheck.Message = "OpenSSL headers found in kernel-build-tools"
		} else {
			opensslCheck.Passed = false
			opensslCheck.Message = "OpenSSL headers not found - check prebuilts/kernel-build-tools"
		}
		results.Checks = append(results.Checks, opensslCheck)
	}

	// Check: VM-specific prerequisites
	for _, vmName := range vmNames {
		// Check if rootfs exists for deploy/start (must have been built)
		if command == "deploy" || command == "start" {
			rootfsPath := fmt.Sprintf("vm/%s/rootfs.img", vmName)
			if vmName == "forge" {
				rootfsPath = "vm/forgejo/rootfs.img"
			}
			rootfsCheck := CheckResult{
				Name:     fmt.Sprintf("%s rootfs.img", vmName),
				Required: command == "deploy",
			}
			if fileExists(rootfsPath) {
				rootfsCheck.Passed = true
				rootfsCheck.Message = fmt.Sprintf("%s exists", rootfsPath)
			} else {
				rootfsCheck.Passed = false
				rootfsCheck.Message = fmt.Sprintf("%s not found - run 'sovereign build --%s' first", rootfsPath, vmName)
			}
			results.Checks = append(results.Checks, rootfsCheck)
		}
	}

	// Determine overall pass/fail
	for _, check := range results.Checks {
		if check.Required && !check.Passed {
			results.Passed = false
		}
	}

	return results
}

// Print outputs the preflight results in a formatted way
func (r *Results) Print() {
	fmt.Println("=== Preflight Checks ===")

	for _, check := range r.Checks {
		status := "✓"
		if !check.Passed {
			if check.Required {
				status = "✗"
			} else {
				status = "⚠"
			}
		}

		reqTag := ""
		if !check.Required && !check.Passed {
			reqTag = " (optional)"
		}

		fmt.Printf("%s %s: %s%s\n", status, check.Name, check.Message, reqTag)
	}

	fmt.Println()
	if r.Passed {
		fmt.Println("✓ All required checks passed")
	} else {
		fmt.Println("✗ Some required checks failed - fix issues above before proceeding")
	}
}

// RunChecks performs preflight checks and returns true if all required checks pass
func RunChecks(command string, vmNames []string) bool {
	results := CheckForCommand(command, vmNames)
	results.Print()
	fmt.Println()
	return results.Passed
}
