// VM build operations (Docker, data disk, rootfs preparation)
// TEAM_029: Extracted from sql/sql.go and forge/forge.go Build()
package common

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/anthropics/sovereign/internal/docker"
	"github.com/anthropics/sovereign/internal/rootfs"
)

// BuildVM builds a VM image using Docker.
// TEAM_029: Extracted from sql/sql.go Build() and forge/forge.go Build()
func BuildVM(cfg *VMConfig, dbPassword string) error {
	fmt.Printf("=== Building %s VM ===\n", cfg.DisplayName)

	// Check if Docker is available
	if !docker.IsAvailable() {
		return fmt.Errorf("docker not found in PATH - install Docker first")
	}

	// Check if we have sudo (needed for mount operations)
	if _, err := exec.LookPath("sudo"); err != nil {
		return fmt.Errorf("sudo not found - needed for rootfs preparation")
	}

	// Run pre-build hook if defined
	if cfg.PreBuildHook != nil {
		if err := cfg.PreBuildHook(cfg); err != nil {
			return fmt.Errorf("pre-build hook failed: %w", err)
		}
	}

	// Build Docker image
	fmt.Println("Building Docker image for ARM64...")
	dockerfilePath := fmt.Sprintf("%s/Dockerfile", cfg.LocalPath)
	cmd := exec.Command("docker", "build",
		"--platform", "linux/arm64",
		"-t", cfg.DockerImage,
		"-f", dockerfilePath,
		cfg.LocalPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed (ensure qemu-user-static is installed for cross-arch builds): %w", err)
	}

	// Export to rootfs image
	fmt.Println("\nExporting rootfs image...")
	rootfsPath := fmt.Sprintf("%s/rootfs.img", cfg.LocalPath)
	if err := docker.ExportImage(cfg.DockerImage, rootfsPath, "512M"); err != nil {
		return err
	}

	// Check kernel
	kernelDst := fmt.Sprintf("%s/Image", cfg.LocalPath)
	if cfg.SharedKernel {
		// Use kernel from another VM (e.g., forge uses sql's kernel)
		if cfg.KernelSource != "" {
			if _, err := os.Stat(cfg.KernelSource); os.IsNotExist(err) {
				return fmt.Errorf("kernel Image not found at %s\n"+
					"  Build SQL VM first: sovereign build --sql", cfg.KernelSource)
			}
			fmt.Printf("  ✓ Using shared kernel from %s\n", cfg.KernelSource)
		}
	} else {
		// Check for existing kernel
		if info, err := os.Stat(kernelDst); err == nil && info.Size() > 1000000 {
			fmt.Printf("  ✓ Using existing kernel %s (%d MB)\n", kernelDst, info.Size()/1024/1024)
		} else {
			return fmt.Errorf("kernel Image not found or invalid\n" +
				"  The kernel must be RAW ARM64 format (not EFI stub).\n" +
				"  Options:\n" +
				"    1. Copy from device: adb pull /data/sovereign/vm/sql/Image vm/sql/Image\n" +
				"    2. Build with: cd aosp && make O=../out/guest-kernel ARCH=arm64 microdroid_defconfig Image")
		}
	}

	// Create data disk if needed
	dataImg := fmt.Sprintf("%s/data.img", cfg.LocalPath)
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
	fmt.Println("Preparing rootfs for AVF (vsock device nodes, init script fixes)...")
	if err := rootfs.PrepareForAVF(rootfsPath, dbPassword); err != nil {
		return fmt.Errorf("rootfs preparation failed: %w", err)
	}

	// Run post-build hook if defined
	if cfg.PostBuildHook != nil {
		if err := cfg.PostBuildHook(cfg); err != nil {
			return fmt.Errorf("post-build hook failed: %w", err)
		}
	}

	fmt.Printf("\n✓ %s VM built successfully\n", cfg.DisplayName)
	fmt.Printf("  Rootfs: %s/rootfs.img\n", cfg.LocalPath)
	fmt.Printf("  Data:   %s/data.img\n", cfg.LocalPath)
	fmt.Printf("\nNext: sovereign deploy --%s\n", cfg.Name)
	return nil
}
