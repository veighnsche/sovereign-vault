// VM deployment operations (push files to device)
// TEAM_029: Extracted from sql/sql.go and forge/forge.go Deploy()
// TEAM_037: Added boot script deployment to fix VM killing issue
package common

import (
	"fmt"
	"os"
	"sync"

	"github.com/anthropics/sovereign/internal/device"
)

var (
	// bootScriptDeployed tracks if boot script has been deployed this session
	bootScriptDeployed bool
	bootScriptMu       sync.Mutex
)

// FreshDataDeploy forces wiping data.img and re-registering Tailscale
// TEAM_034: Package-level flag for --fresh-data CLI option
var FreshDataDeploy bool

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

	// Push data disk - BUT PRESERVE IF EXISTS (contains Tailscale state!)
	// TEAM_034: Only push data.img if it doesn't exist on device OR --fresh-data flag
	// This preserves Tailscale machine identity across redeploys
	dataImgDevice := fmt.Sprintf("%s/data.img", cfg.DevicePath)
	if FreshDataDeploy {
		// User explicitly wants a fresh start - clean up old Tailscale registrations
		fmt.Println("--fresh-data: Cleaning up old Tailscale registrations...")
		if err := RemoveTailscaleRegistrations(cfg.TailscaleHost); err != nil {
			fmt.Printf("  ⚠ Warning: %v\n", err)
		}
		fmt.Println("Pushing fresh data.img (new Tailscale identity)...")
		if err := device.PushFile(
			fmt.Sprintf("%s/data.img", cfg.LocalPath),
			dataImgDevice); err != nil {
			return err
		}
	} else if device.FileExists(dataImgDevice) {
		fmt.Println("Preserving existing data.img (contains Tailscale identity)")
	} else {
		fmt.Println("Pushing data.img (first deploy)...")
		if err := device.PushFile(
			fmt.Sprintf("%s/data.img", cfg.LocalPath),
			dataImgDevice); err != nil {
			return err
		}
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

	// TEAM_037: Deploy boot script to /data/adb/service.d/ (once per session)
	if err := DeployBootScript(); err != nil {
		fmt.Printf("  ⚠ Warning: boot script deployment failed: %v\n", err)
	}

	fmt.Printf("\n✓ %s VM deployed\n", cfg.DisplayName)
	fmt.Printf("\nNext: sovereign start --%s\n", cfg.Name)
	return nil
}

// DeployBootScript deploys the sovereign_start.sh boot script to /data/adb/service.d/
// TEAM_037: This script runs at boot via KernelSU and keeps VMs alive as its children,
// preventing Android init from killing them as orphaned processes.
func DeployBootScript() error {
	bootScriptMu.Lock()
	defer bootScriptMu.Unlock()

	// Only deploy once per session
	if bootScriptDeployed {
		return nil
	}

	localScript := "host/sovereign_start.sh"
	if _, err := os.Stat(localScript); os.IsNotExist(err) {
		return fmt.Errorf("boot script not found: %s", localScript)
	}

	// Create /data/adb/service.d/ if it doesn't exist
	serviceDir := "/data/adb/service.d"
	device.RunShellCommand(fmt.Sprintf("mkdir -p %s", serviceDir))

	// Push boot script
	destScript := serviceDir + "/sovereign_start.sh"
	fmt.Println("Deploying boot script to " + destScript + "...")
	if err := device.PushFile(localScript, destScript); err != nil {
		return fmt.Errorf("failed to push boot script: %w", err)
	}

	// Make executable
	if _, err := device.RunShellCommand(fmt.Sprintf("chmod +x %s", destScript)); err != nil {
		return fmt.Errorf("failed to chmod boot script: %w", err)
	}

	// Also copy to /data/sovereign/ for CLI access
	device.RunShellCommand("mkdir -p /data/sovereign")
	if err := device.PushFile(localScript, "/data/sovereign/sovereign_start.sh"); err != nil {
		return fmt.Errorf("failed to push boot script to /data/sovereign: %w", err)
	}
	device.RunShellCommand("chmod +x /data/sovereign/sovereign_start.sh")

	bootScriptDeployed = true
	fmt.Println("✓ Boot script deployed (VMs will auto-start at boot)")
	return nil
}
