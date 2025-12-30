// Package rootfs provides rootfs AVF preparation utilities
// TEAM_010: Extracted from main.go during CLI refactor
// TEAM_023: Extracted simple_init.sh to external file
package rootfs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PrepareForAVF fixes Alpine rootfs for AVF/crosvm compatibility
// This is IDEMPOTENT - safe to run multiple times
// TEAM_007: Original implementation
// TEAM_011: Added dbPassword parameter for secure credential handling
func PrepareForAVF(rootfsPath string, dbPassword string) error {
	mountDir := "/tmp/sovereign-rootfs-prep"
	os.MkdirAll(mountDir, 0755)

	// Mount rootfs
	if err := exec.Command("sudo", "mount", rootfsPath, mountDir).Run(); err != nil {
		return fmt.Errorf("mount failed: %w", err)
	}
	defer func() {
		exec.Command("sudo", "umount", mountDir).Run()
		os.RemoveAll(mountDir)
	}()

	// TEAM_020: Removed gvforwarder code - we use TAP networking now

	// Create local.d script for early device node creation
	// This runs early in boot and ensures critical device nodes exist
	localDDir := mountDir + "/etc/local.d"
	exec.Command("sudo", "mkdir", "-p", localDDir).Run()

	devNodesScript := localDDir + "/00-avf-devices.start"
	devNodesContent := `#!/bin/sh
# TEAM_020: Ensure AVF-required device nodes exist

# Console devices
[ -e /dev/console ] || mknod /dev/console c 5 1
[ -e /dev/tty ] || mknod /dev/tty c 5 0
[ -e /dev/tty0 ] || mknod /dev/tty0 c 4 0
[ -e /dev/null ] || mknod /dev/null c 1 3

# TUN device (required for Tailscale)
mkdir -p /dev/net
[ -e /dev/net/tun ] || mknod /dev/net/tun c 10 200
chmod 666 /dev/net/tun

# Set permissions
chmod 666 /dev/null /dev/tty 2>/dev/null
chmod 600 /dev/console 2>/dev/null
`
	// Write the script (idempotent - overwrites if exists)
	writeCmd := fmt.Sprintf("cat > %s << 'EOFSCRIPT'\n%sEOFSCRIPT", devNodesScript, devNodesContent)
	if err := exec.Command("sudo", "sh", "-c", writeCmd).Run(); err != nil {
		return fmt.Errorf("failed to create device nodes script: %w", err)
	}
	exec.Command("sudo", "chmod", "+x", devNodesScript).Run()
	fmt.Println("  ✓ Created /etc/local.d/00-avf-devices.start")

	// Fix 3: Ensure 'local' service is enabled in default runlevel
	localLink := mountDir + "/etc/runlevels/default/local"
	if _, err := os.Stat(localLink); os.IsNotExist(err) {
		exec.Command("sudo", "ln", "-sf", "/etc/init.d/local", localLink).Run()
		fmt.Println("  ✓ Enabled 'local' service in default runlevel")
	}

	// Fix 4: Ensure devfs runs in sysinit runlevel
	devfsLink := mountDir + "/etc/runlevels/sysinit/devfs"
	if _, err := os.Stat(devfsLink); os.IsNotExist(err) {
		exec.Command("sudo", "ln", "-sf", "/etc/init.d/devfs", devfsLink).Run()
		fmt.Println("  ✓ Enabled 'devfs' service in sysinit runlevel")
	}

	// Fix 5: Pre-create critical device nodes directly in rootfs
	// TEAM_011: The local.d script runs too late - sovereign-init needs these earlier
	devDir := mountDir + "/dev"
	devNetDir := devDir + "/net"
	exec.Command("sudo", "mkdir", "-p", devNetDir).Run()

	// Create device nodes if they don't exist
	devNodes := []struct{ path, major, minor string }{
		{devDir + "/console", "5", "1"},
		{devDir + "/null", "1", "3"},
		{devDir + "/zero", "1", "5"},
		{devDir + "/tty", "5", "0"},
		{devDir + "/random", "1", "8"},
		{devDir + "/urandom", "1", "9"},
		{devDir + "/vsock", "10", "121"},
		{devNetDir + "/tun", "10", "200"},
	}
	for _, dev := range devNodes {
		if _, err := os.Stat(dev.path); os.IsNotExist(err) {
			exec.Command("sudo", "mknod", dev.path, "c", dev.major, dev.minor).Run()
			exec.Command("sudo", "chmod", "666", dev.path).Run()
		}
	}
	fmt.Println("  ✓ Pre-created device nodes in /dev")

	// Fix 6: Create init script (OpenRC doesn't work on AVF - it hangs)
	// TEAM_023: Script extracted to vm/sql/init.sh for maintainability
	initScriptPath := mountDir + "/sbin/init.sh"

	// Find the init.sh script relative to the sovereign directory
	// TEAM_025: Now VM-agnostic - uses rootfsPath to determine which init.sh to use
	scriptPath, err := findInitScript(rootfsPath)
	if err != nil {
		return fmt.Errorf("failed to find init.sh: %w", err)
	}

	// Read the script template
	scriptBytes, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read init.sh: %w", err)
	}

	// Inject DB_PASSWORD into the script (add export line after PATH export)
	scriptContent := string(scriptBytes)
	dbPasswordExport := fmt.Sprintf("export DB_PASSWORD=\"%s\"", dbPassword)
	scriptContent = strings.Replace(scriptContent,
		"# DB_PASSWORD is injected by rootfs.go - DO NOT HARDCODE",
		dbPasswordExport,
		1)

	// Write the script to rootfs
	initCmd := fmt.Sprintf("cat > %s << 'EOFSCRIPT'\n%sEOFSCRIPT", initScriptPath, scriptContent)
	if err := exec.Command("sudo", "sh", "-c", initCmd).Run(); err != nil {
		return fmt.Errorf("failed to create init.sh: %w", err)
	}
	exec.Command("sudo", "chmod", "+x", initScriptPath).Run()
	// Also symlink to /sbin/init for kernel's default init path
	exec.Command("sudo", "ln", "-sf", "/sbin/init.sh", mountDir+"/sbin/init").Run()
	fmt.Printf("  ✓ Created /sbin/init.sh (from %s)\n", scriptPath)

	// TEAM_020: Removed dhclient wrapper - gvforwarder not used anymore

	fmt.Println("  ✓ Rootfs prepared for AVF")
	return nil
}

// findInitScript locates the init.sh script based on the rootfs path
// TEAM_025: Made VM-agnostic to support both SQL and Forge VMs
// TEAM_035: Added vault VM support
func findInitScript(rootfsPath string) (string, error) {
	// Determine VM type from rootfs path
	vmType := "sql" // default
	if strings.Contains(rootfsPath, "forgejo") {
		vmType = "forgejo"
	} else if strings.Contains(rootfsPath, "vault") {
		vmType = "vault"
	}

	// Try common locations relative to working directory
	candidates := []string{
		fmt.Sprintf("vm/%s/init.sh", vmType),           // Running from sovereign/
		fmt.Sprintf("sovereign/vm/%s/init.sh", vmType), // Running from kernel/
		fmt.Sprintf("../vm/%s/init.sh", vmType),        // Running from sovereign/internal/
		fmt.Sprintf("../../vm/%s/init.sh", vmType),     // Running from sovereign/internal/rootfs/
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return filepath.Abs(path)
		}
	}

	// Try to find it by looking for go.mod and navigating from there
	cwd, _ := os.Getwd()
	dir := cwd
	for i := 0; i < 5; i++ { // Walk up max 5 levels
		goMod := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goMod); err == nil {
			// Found sovereign root
			scriptPath := filepath.Join(dir, "vm", vmType, "init.sh")
			if _, err := os.Stat(scriptPath); err == nil {
				return scriptPath, nil
			}
		}
		dir = filepath.Dir(dir)
	}

	return "", fmt.Errorf("init.sh not found for %s VM - ensure you're running from the sovereign directory", vmType)
}
