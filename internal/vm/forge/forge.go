// Package forge provides Forgejo VM operations
// TEAM_012: Git forge VM with CI/CD capabilities
package forge

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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
func (v *VM) Deploy() error {
	fmt.Println("=== Deploying Forgejo VM ===")

	// Create device directories
	fmt.Println("Creating directories on device...")
	exec.Command("adb", "shell", "su", "-c", "mkdir -p /data/sovereign/forgejo/bin").Run()

	// Verify directory was created
	out, err := exec.Command("adb", "shell", "su", "-c", "[ -d /data/sovereign/forgejo ] && echo ok").Output()
	if err != nil || strings.TrimSpace(string(out)) != "ok" {
		return fmt.Errorf("failed to create directories on device")
	}

	// Push files
	fmt.Println("Pushing rootfs.img...")
	if err := device.PushFile("vm/forgejo/rootfs.img", "/data/sovereign/forgejo/rootfs.img"); err != nil {
		return err
	}

	fmt.Println("Pushing data.img...")
	if err := device.PushFile("vm/forgejo/data.img", "/data/sovereign/forgejo/data.img"); err != nil {
		return err
	}

	fmt.Println("Pushing start.sh...")
	if err := device.PushFile("vm/forgejo/start.sh", "/data/sovereign/forgejo/start.sh"); err != nil {
		return err
	}

	// Use shared kernel from sql VM
	fmt.Println("Pushing kernel (shared from sql VM)...")
	if err := device.PushFile("vm/sql/Image", "/data/sovereign/forgejo/Image"); err != nil {
		return err
	}

	// Use shared gvproxy/gvforwarder from sql VM
	if _, err := os.Stat("vm/sql/bin/gvproxy"); err == nil {
		fmt.Println("Pushing gvproxy...")
		device.PushFile("vm/sql/bin/gvproxy", "/data/sovereign/forgejo/bin/gvproxy")
		fmt.Println("Pushing gvforwarder...")
		device.PushFile("vm/sql/bin/gvforwarder", "/data/sovereign/forgejo/bin/gvforwarder")
	}

	// Make executables
	exec.Command("adb", "shell", "su", "-c", "chmod +x /data/sovereign/forgejo/start.sh").Run()
	exec.Command("adb", "shell", "su", "-c", "chmod +x /data/sovereign/forgejo/bin/*").Run()

	fmt.Println("\n✓ Forgejo VM deployed")
	fmt.Println("\nNext: sovereign start --forge")
	return nil
}

// Start starts the Forgejo VM
func (v *VM) Start() error {
	fmt.Println("=== Starting Forgejo VM ===")

	// Note: Forgejo requires sql-vm to be running
	fmt.Println("Note: Forgejo requires sql-vm to be running for database")

	// Check if start script exists
	out, err := exec.Command("adb", "shell", "su", "-c", "[ -x /data/sovereign/forgejo/start.sh ] && echo ok").Output()
	if err != nil || strings.TrimSpace(string(out)) != "ok" {
		return fmt.Errorf("start script not found - run 'sovereign deploy --forge' first")
	}

	// Start the VM
	fmt.Println("Starting VM...")
	cmd := exec.Command("adb", "shell", "su", "-c", "/data/sovereign/forgejo/start.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start script failed: %w", err)
	}

	// Wait and verify
	time.Sleep(2 * time.Second)
	out, _ = exec.Command("adb", "shell", "su", "-c", "pgrep -f 'crosvm.*forgejo'").Output()
	if strings.TrimSpace(string(out)) == "" {
		fmt.Println("\n⚠ VM process not found - check logs")
		return fmt.Errorf("VM failed to start")
	}

	fmt.Println("\n✓ Forgejo VM started")
	fmt.Println("  Check Tailscale for forge-vm to appear")
	fmt.Println("  Web UI: http://forge-vm:3000")
	return nil
}

// Stop stops the Forgejo VM
func (v *VM) Stop() error {
	fmt.Println("=== Stopping Forgejo VM ===")

	// Kill crosvm process for forge VM
	exec.Command("adb", "shell", "su", "-c", "pkill -f 'crosvm.*forgejo'").Run()
	exec.Command("adb", "shell", "su", "-c", "pkill -f 'gvproxy.*forgejo'").Run()

	fmt.Println("✓ Forgejo VM stopped")
	return nil
}

// Test tests the Forgejo VM connectivity
func (v *VM) Test() error {
	fmt.Println("=== Testing Forgejo VM ===")

	// Test 1: Check Tailscale
	fmt.Println("\n[Test 1/3] Checking Tailscale connectivity...")
	cmd := exec.Command("tailscale", "ping", "-c", "1", "forge-vm")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("forge-vm not reachable via Tailscale")
	}
	fmt.Println("  ✓ forge-vm reachable via Tailscale")

	// Test 2: Check web UI
	fmt.Println("\n[Test 2/3] Checking Forgejo web UI...")
	cmd = exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "http://forge-vm:3000")
	output, err := cmd.Output()
	if err != nil || string(output) != "200" {
		fmt.Printf("  ⚠ Web UI returned: %s (may need initial setup)\n", string(output))
	} else {
		fmt.Println("  ✓ Forgejo web UI responding")
	}

	// Test 3: Check SSH
	fmt.Println("\n[Test 3/3] Checking SSH port...")
	cmd = exec.Command("nc", "-z", "-w", "3", "forge-vm", "22")
	if err := cmd.Run(); err != nil {
		fmt.Println("  ⚠ SSH port not responding")
	} else {
		fmt.Println("  ✓ SSH port open")
	}

	fmt.Println("\n✓ Forgejo VM tests complete")
	return nil
}

// Remove removes the Forgejo VM from the device
func (v *VM) Remove() error {
	fmt.Println("=== Removing Forgejo VM from device ===")

	// First stop the VM if running
	v.Stop()

	// Remove all files from device
	fmt.Println("Removing VM files from device...")
	device.RemoveDir("/data/sovereign/forgejo")

	// Verify removal
	if device.DirExists("/data/sovereign/forgejo") {
		return fmt.Errorf("failed to remove /data/sovereign/forgejo")
	}

	fmt.Println("✓ Forgejo VM removed from device")
	fmt.Println("\nTo redeploy: sovereign deploy --forge")
	return nil
}
