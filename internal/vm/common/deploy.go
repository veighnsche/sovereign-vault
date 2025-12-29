// VM deployment operations (push files to device)
// TEAM_029: Extracted from sql/sql.go and forge/forge.go Deploy()
package common

import (
	"fmt"
	"os"

	"github.com/anthropics/sovereign/internal/device"
)

// DeployVM deploys a VM to the Android device.
// TEAM_029: Extracted from sql/sql.go Deploy() and forge/forge.go Deploy()
func DeployVM(cfg *VMConfig) error {
	fmt.Printf("=== Deploying %s VM ===\n", cfg.DisplayName)

	fmt.Println("Tailscale: Using persistent machine identity (no cleanup needed)")

	// Verify required files exist locally
	requiredFiles := []string{"rootfs.img", "data.img"}
	if !cfg.SharedKernel {
		requiredFiles = append(requiredFiles, "Image")
	}

	for _, f := range requiredFiles {
		path := fmt.Sprintf("%s/%s", cfg.LocalPath, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("%s not found - run 'sovereign build --%s' first", path, cfg.Name)
		}
	}

	// Check for .env file
	envPath := ".env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return fmt.Errorf(".env file not found - Tailscale won't connect without it\n" +
			"  1. Copy .env.example to .env\n" +
			"  2. Get auth key from https://login.tailscale.com/admin/settings/keys\n" +
			"  3. Fill in TAILSCALE_AUTHKEY in .env")
	}

	// Create device directories
	fmt.Println("Creating directories on device...")
	device.MkdirP(cfg.DevicePath)
	if !device.DirExists(cfg.DevicePath) {
		return fmt.Errorf("failed to create directories on device")
	}

	// Push rootfs
	fmt.Println("Pushing rootfs.img (this may take a while)...")
	if err := device.PushFile(
		fmt.Sprintf("%s/rootfs.img", cfg.LocalPath),
		fmt.Sprintf("%s/rootfs.img", cfg.DevicePath)); err != nil {
		return err
	}

	// Push data disk
	fmt.Println("Pushing data.img...")
	if err := device.PushFile(
		fmt.Sprintf("%s/data.img", cfg.LocalPath),
		fmt.Sprintf("%s/data.img", cfg.DevicePath)); err != nil {
		return err
	}

	// Push kernel
	if cfg.SharedKernel && cfg.KernelSource != "" {
		fmt.Println("Pushing kernel (shared from sql VM)...")
		if err := device.PushFile(cfg.KernelSource, fmt.Sprintf("%s/Image", cfg.DevicePath)); err != nil {
			return err
		}
	} else {
		fmt.Println("Pushing guest kernel (this may take a while - 35MB)...")
		if err := device.PushFile(
			fmt.Sprintf("%s/Image", cfg.LocalPath),
			fmt.Sprintf("%s/Image", cfg.DevicePath)); err != nil {
			return err
		}
	}

	// Push .env
	if _, err := os.Stat(envPath); err == nil {
		fmt.Println("Pushing .env...")
		if err := device.PushFile(envPath, "/data/sovereign/.env"); err != nil {
			return err
		}
	}

	// Push and chmod start script
	fmt.Println("Creating start script...")
	startScriptLocal := fmt.Sprintf("%s/start.sh", cfg.LocalPath)
	startScriptDevice := fmt.Sprintf("%s/start.sh", cfg.DevicePath)

	if _, err := os.Stat(startScriptLocal); os.IsNotExist(err) {
		return fmt.Errorf("start script not found: %s", startScriptLocal)
	}

	if err := device.PushFile(startScriptLocal, startScriptDevice); err != nil {
		return err
	}

	if _, err := device.RunShellCommand(fmt.Sprintf("chmod +x %s", startScriptDevice)); err != nil {
		return fmt.Errorf("failed to chmod start script: %w", err)
	}

	fmt.Printf("\n✓ %s VM deployed\n", cfg.DisplayName)
	fmt.Printf("\nNext: sovereign start --%s\n", cfg.Name)
	return nil
}
