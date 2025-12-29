// Package forge provides Forgejo VM operations
// TEAM_012: Git forge VM with CI/CD capabilities
// TEAM_025: Refactored to use TAP networking and correct paths
package forge

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/anthropics/sovereign/internal/device"
	"github.com/anthropics/sovereign/internal/docker"
	"github.com/anthropics/sovereign/internal/rootfs"
	"github.com/anthropics/sovereign/internal/vm"
)

func init() {
	vm.Register("forge", &VM{})
}

// VM implements the vm.VM interface for Forgejo
type VM struct{}

func (v *VM) Name() string { return "forge" }

// Build builds the Forgejo VM image
func (v *VM) Build() error {
	fmt.Println("=== Building Forgejo VM ===")

	// Check if Docker is available
	if !docker.IsAvailable() {
		return fmt.Errorf("docker not found in PATH - install Docker first")
	}

	// Build Docker image for ARM64
	fmt.Println("Building Docker image for ARM64...")
	cmd := exec.Command("docker", "build",
		"--platform", "linux/arm64",
		"-t", "sovereign-forge",
		"-f", "vm/forgejo/Dockerfile",
		"vm/forgejo")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	// Export to rootfs image
	fmt.Println("\nExporting rootfs image...")
	if err := docker.ExportImage("sovereign-forge", "vm/forgejo/rootfs.img", "512M"); err != nil {
		return err
	}

	// Check if kernel exists (shared with SQL VM)
	kernelPath := "vm/sql/Image"
	if _, err := os.Stat(kernelPath); os.IsNotExist(err) {
		return fmt.Errorf("kernel Image not found at %s\n"+
			"  Build SQL VM first: sovereign build --sql", kernelPath)
	}
	fmt.Println("  ✓ Using shared kernel from vm/sql/Image")

	// Create data disk
	dataImg := "vm/forgejo/data.img"
	if _, err := os.Stat(dataImg); os.IsNotExist(err) {
		fmt.Println("Creating data disk (4GB)...")
		cmd = exec.Command("truncate", "-s", "4G", dataImg)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create data disk: %w", err)
		}
		cmd = exec.Command("mkfs.ext4", "-F", dataImg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to format data disk: %w", err)
		}
	} else {
		fmt.Println("  Data disk already exists, skipping")
	}

	// Prepare rootfs for AVF
	// Note: Forgejo uses the SQL VM's PostgreSQL, so no DB password needed here
	fmt.Println("Preparing rootfs for AVF...")
	if err := rootfs.PrepareForAVF("vm/forgejo/rootfs.img", ""); err != nil {
		return fmt.Errorf("rootfs preparation failed: %w", err)
	}

	fmt.Println("\n✓ Forgejo VM built successfully")
	fmt.Println("  Rootfs: vm/forgejo/rootfs.img")
	fmt.Println("  Data:   vm/forgejo/data.img")
	fmt.Println("\nNext: sovereign deploy --forge")
	return nil
}

// Deploy deploys the Forgejo VM to the Android device
// TEAM_025: Fixed device path to /data/sovereign/vm/forgejo/, removed gvproxy
func (v *VM) Deploy() error {
	fmt.Println("=== Deploying Forgejo VM ===")

	// TEAM_025: Use persistent Tailscale identity (no cleanup needed)
	fmt.Println("Tailscale: Using persistent machine identity (no cleanup needed)")

	// Create device directories
	// TEAM_025: CRITICAL - path is /data/sovereign/vm/forgejo/, NOT /data/sovereign/forgejo/
	fmt.Println("Creating directories on device...")
	if err := device.MkdirP("/data/sovereign/vm/forgejo"); err != nil {
		return fmt.Errorf("failed to create directories on device: %w", err)
	}

	// Push files
	fmt.Println("Pushing rootfs.img...")
	if err := device.PushFile("vm/forgejo/rootfs.img", "/data/sovereign/vm/forgejo/rootfs.img"); err != nil {
		return err
	}

	fmt.Println("Pushing data.img...")
	if err := device.PushFile("vm/forgejo/data.img", "/data/sovereign/vm/forgejo/data.img"); err != nil {
		return err
	}

	fmt.Println("Pushing start.sh...")
	if err := device.PushFile("vm/forgejo/start.sh", "/data/sovereign/vm/forgejo/start.sh"); err != nil {
		return err
	}

	// Use shared kernel from sql VM
	fmt.Println("Pushing kernel (shared from sql VM)...")
	if err := device.PushFile("vm/sql/Image", "/data/sovereign/vm/forgejo/Image"); err != nil {
		return err
	}

	// TEAM_025: gvproxy removed - we use TAP networking now

	// Make executables
	exec.Command("adb", "shell", "su", "-c", "chmod +x /data/sovereign/vm/forgejo/start.sh").Run()

	fmt.Println("\n✓ Forgejo VM deployed")
	fmt.Println("\nNext: sovereign start --forge")
	return nil
}

// TEAM_025: Start, Stop, Test, Remove moved to lifecycle.go and verify.go
