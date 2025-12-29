// Package sql provides the PostgreSQL VM implementation
// TEAM_010: Extracted from main.go during CLI refactor
//
// CRITICAL ARCHITECTURE REQUIREMENTS - READ BEFORE MODIFYING:
//
// 1. PostgreSQL MUST be accessible via Tailscale from ANY device on the tailnet
// 2. Each VM MUST have its OWN Tailscale IP - NO SHARING IPs between services
// 3. Tailscale MUST use kernel TUN mode (CONFIG_TUN=y) - NOT userspace networking
//
// FORBIDDEN WORKAROUNDS (these are WRONG, do not suggest them):
// - Port forwarding through Android's Tailscale (shares IP, breaks multi-service)
// - Direct connection via phone's LAN IP (not accessible from tailnet)
// - SQLite instead of PostgreSQL (architecture violation)
// - Any hack that avoids fixing the real networking problem
//
// If Tailscale ports are not accessible, the fix is:
// - Ensure /dev/net/tun exists in guest
// - Run tailscaled WITHOUT --tun=userspace-networking
// - The guest kernel MUST have CONFIG_TUN=y (it does)
package sql

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/anthropics/sovereign/internal/docker"
	"github.com/anthropics/sovereign/internal/rootfs"
	"github.com/anthropics/sovereign/internal/secrets"
	"github.com/anthropics/sovereign/internal/vm"
	"github.com/anthropics/sovereign/internal/vm/common"
)

// ForceDeploySkipTailscaleCheck skips Tailscale idempotency check when true
// TEAM_019: Package-level flag for --force CLI option
var ForceDeploySkipTailscaleCheck bool

// TEAM_029: SQLConfig defines the configuration for the PostgreSQL VM
var SQLConfig = &common.VMConfig{
	Name:           "sql",
	DisplayName:    "PostgreSQL",
	TAPInterface:   "vm_sql",
	TAPHostIP:      "192.168.100.1",
	TAPGuestIP:     "192.168.100.2",
	TAPSubnet:      "192.168.100.0/24",
	TailscaleHost:  "sovereign-sql",
	DevicePath:     "/data/sovereign/vm/sql",
	LocalPath:      "vm/sql",
	ServicePorts:   []int{5432},
	ReadyMarker:    "PostgreSQL started",
	StartTimeout:   90,
	DockerImage:    "sovereign-sql",
	SharedKernel:   false,
	KernelSource:   "",
	NeedsSecrets:   true,
	ProcessPattern: "[c]rosvm.*sql",
}

func init() {
	vm.Register("sql", &VM{})
}

// VM implements the vm.VM interface for PostgreSQL
type VM struct{}

func (v *VM) Name() string { return "sql" }

func (v *VM) Build() error {
	fmt.Println("=== Building PostgreSQL VM ===")

	// Check if Docker is available
	if !docker.IsAvailable() {
		return fmt.Errorf("docker not found in PATH - install Docker first")
	}

	// Check if we have sudo (needed for mount operations)
	if _, err := exec.LookPath("sudo"); err != nil {
		return fmt.Errorf("sudo not found - needed for rootfs preparation")
	}

	// TEAM_011: Prompt for database credentials if not already set
	var creds *secrets.Credentials
	if secrets.SecretsExist() {
		fmt.Println("Using existing credentials from .secrets")
		var err error
		creds, err = secrets.LoadSecretsFile()
		if err != nil {
			return err
		}
	} else {
		var err error
		creds, err = secrets.PromptCredentials("postgres")
		if err != nil {
			return fmt.Errorf("credential setup failed: %w", err)
		}
		if err := secrets.WriteSecretsFile(creds); err != nil {
			return err
		}
	}

	// TEAM_006: Build the Docker image for ARM64 (Pixel 6 architecture)
	fmt.Println("Building Docker image for ARM64...")
	cmd := exec.Command("docker", "build",
		"--platform", "linux/arm64",
		"-t", "sovereign-sql",
		"-f", "vm/sql/Dockerfile",
		"vm/sql")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed (ensure qemu-user-static is installed for cross-arch builds): %w", err)
	}

	// Export to rootfs image for crosvm
	fmt.Println("\nExporting rootfs image...")
	if err := docker.ExportImage("sovereign-sql", "vm/sql/rootfs.img", "512M"); err != nil {
		return err
	}

	// TEAM_011: Check if custom kernel already exists (skip extraction)
	// Alpine's vmlinuz-virt is EFI stub format which crosvm can't boot.
	// We use a custom kernel built with microdroid_defconfig.
	kernelDst := "vm/sql/Image"
	if info, err := os.Stat(kernelDst); err == nil && info.Size() > 1000000 {
		fmt.Printf("  ✓ Using existing kernel %s (%d MB)\n", kernelDst, info.Size()/1024/1024)
	} else {
		// No valid kernel - user needs to build one or copy from device
		return fmt.Errorf("kernel Image not found or invalid\n" +
			"  The kernel must be RAW ARM64 format (not EFI stub).\n" +
			"  Options:\n" +
			"    1. Copy from device: adb pull /data/sovereign/vm/sql/Image vm/sql/Image\n" +
			"    2. Build with: cd aosp && make O=../out/guest-kernel ARCH=arm64 microdroid_defconfig Image")
	}

	// Create data disk
	fmt.Println("Creating data disk (4GB)...")
	dataImg := "vm/sql/data.img"
	if _, err := os.Stat(dataImg); os.IsNotExist(err) {
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

	// TEAM_007: Prepare rootfs with proper device node setup for vsock networking
	fmt.Println("Preparing rootfs for AVF (vsock device nodes, init script fixes)...")
	if err := rootfs.PrepareForAVF("vm/sql/rootfs.img", creds.DBPassword); err != nil {
		return fmt.Errorf("rootfs preparation failed: %w", err)
	}

	fmt.Println("\n✓ PostgreSQL VM built successfully")
	fmt.Println("  Rootfs: vm/sql/rootfs.img")
	fmt.Println("  Data:   vm/sql/data.img")
	fmt.Println("\nNext: sovereign deploy --sql")
	return nil
}

// TEAM_029: Deploy delegates to common.DeployVM
func (v *VM) Deploy() error {
	return common.DeployVM(SQLConfig)
}
