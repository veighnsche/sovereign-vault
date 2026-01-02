// VM lifecycle operations (Start, Stop, Remove)
// TEAM_029: Extracted from sql/lifecycle.go and forge/lifecycle.go
// TEAM_037: Fixed VM killing by using daemon script that stays alive
package common

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/anthropics/sovereign/internal/device"
)

// StopVM stops a running VM and cleans up networking.
// TEAM_029: Extracted from sql/lifecycle.go Stop() and forge/lifecycle.go Stop()
// TEAM_029: Added robust error handling and timeouts for all edge cases
// TEAM_037: Also kills watchdog daemon to fully clean up
// CleanVM removes Tailscale registrations for a VM.
// TEAM_039: Added for Tailscale cleanup via CLI
func CleanVM(cfg *VMConfig) error {
	fmt.Printf("=== Cleaning %s Tailscale registrations ===\n", cfg.DisplayName)
	return RemoveTailscaleRegistrations(cfg.TailscaleHost)
}

func StopVM(cfg *VMConfig) error {
	fmt.Printf("=== Stopping %s VM ===\n", cfg.DisplayName)

	// Get PID with timeout protection (already in RunShellCommand)
	pid := device.GetProcessPID(cfg.ProcessPattern)

	if pid != "" {
		fmt.Printf("Stopping VM (PID: %s)...\n", pid)
		// Try graceful kill first
		if err := device.KillProcess(pid); err != nil {
			// Force kill if graceful fails
			device.RunShellCommand(fmt.Sprintf("kill -9 %s 2>/dev/null", pid))
		}
		// Brief wait for process to die
		time.Sleep(500 * time.Millisecond)
		// Verify it's dead, force kill if not
		if device.GetProcessPID(cfg.ProcessPattern) != "" {
			fmt.Println("Process still alive, force killing...")
			device.RunShellCommand(fmt.Sprintf("kill -9 %s 2>/dev/null", pid))
		}
	} else {
		fmt.Println("VM not running")
	}

	// TEAM_037: Kill watchdog daemon for this VM if running
	// The watchdog is a background sovereign_start.sh process monitoring this VM
	daemonPattern := fmt.Sprintf("[s]overeign_start.sh.*%s", cfg.Name)
	daemonPid, _ := device.RunShellCommand(fmt.Sprintf("pgrep -f '%s' 2>/dev/null | head -1", daemonPattern))
	if daemonPid != "" {
		daemonPid = strings.TrimSpace(daemonPid)
		if daemonPid != "" {
			fmt.Printf("Stopping watchdog daemon (PID: %s)...\n", daemonPid)
			device.RunShellCommand(fmt.Sprintf("kill %s 2>/dev/null", daemonPid))
		}
	}

	fmt.Println("Cleaning up networking...")
	cleanupNetworking(cfg)

	device.RunShellCommand(fmt.Sprintf("rm -f %s/vm.pid 2>/dev/null", cfg.DevicePath))

	fmt.Println("✓ VM stopped")
	return nil
}

// cleanupNetworking removes TAP interface and iptables rules.
// TEAM_029: Extracted from sql/lifecycle.go and forge/lifecycle.go
// TEAM_029: Each command has 2>/dev/null and runs independently to avoid blocking
func cleanupNetworking(cfg *VMConfig) {
	// Delete TAP interface - ignore errors (may not exist)
	device.RunShellCommandQuick(fmt.Sprintf("ip link del %s 2>/dev/null || true", cfg.TAPInterface))

	if cfg.TAPSubnet != "" {
		// Remove iptables rules - ignore errors (may not exist)
		device.RunShellCommandQuick(fmt.Sprintf("iptables -t nat -D POSTROUTING -s %s -o wlan0 -j MASQUERADE 2>/dev/null || true", cfg.TAPSubnet))
		device.RunShellCommandQuick(fmt.Sprintf("iptables -D FORWARD -i %s -o wlan0 -j ACCEPT 2>/dev/null || true", cfg.TAPInterface))
		device.RunShellCommandQuick(fmt.Sprintf("iptables -D FORWARD -i wlan0 -o %s -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || true", cfg.TAPInterface))
	}

	// SQL-specific cleanup (policy routing rules)
	if cfg.Name == "sql" {
		device.RunShellCommandQuick("ip rule del from all lookup main pref 1 2>/dev/null || true")
		device.RunShellCommandQuick(fmt.Sprintf("ip rule del from %s lookup wlan0 2>/dev/null || true", cfg.TAPSubnet))
		device.RunShellCommandQuick(fmt.Sprintf("ip rule del from %s lookup main 2>/dev/null || true", cfg.TAPSubnet))
	}
}

// RemoveVM removes a VM from the device (stop + cleanup + delete files).
// TEAM_029: Extracted from sql/lifecycle.go Remove() and forge/lifecycle.go Remove()
func RemoveVM(cfg *VMConfig) error {
	fmt.Printf("=== Removing %s VM from device ===\n", cfg.DisplayName)

	StopVM(cfg)

	fmt.Println("Removing Tailscale registration...")
	RemoveTailscaleRegistrations(cfg.TailscaleHost)

	if cfg.Name == "sql" {
		fmt.Println("Ensuring all networking rules are removed...")
		device.RunShellCommand(fmt.Sprintf("ip rule del from %s lookup wlan0 2>/dev/null", cfg.TAPSubnet))
		device.RunShellCommand(fmt.Sprintf("ip rule del from %s lookup main 2>/dev/null", cfg.TAPSubnet))
	}

	fmt.Println("Removing VM files from device...")
	device.RemoveDir(cfg.DevicePath)

	if device.DirExists(cfg.DevicePath) {
		return fmt.Errorf("failed to remove %s", cfg.DevicePath)
	}

	fmt.Printf("✓ %s VM removed from device\n", cfg.DisplayName)
	fmt.Printf("\nTo redeploy: sovereign deploy --%s\n", cfg.Name)
	return nil
}

// StartVM starts a VM and streams boot logs until ready.
// TEAM_029: Extracted from sql/lifecycle.go Start() and forge/lifecycle.go Start()
// TEAM_037: Uses daemon script to keep parent alive, preventing Android init from killing VMs
func StartVM(cfg *VMConfig) error {
	fmt.Printf("=== Starting %s VM ===\n", cfg.DisplayName)

	// TEAM_029: Check dependencies first (fail-fast)
	if len(cfg.Dependencies) > 0 {
		if err := CheckDependencies(cfg); err != nil {
			return err
		}
	}

	runningPid := device.GetProcessPID(cfg.ProcessPattern)
	if runningPid != "" {
		fmt.Printf("⚠ VM already running (PID: %s)\n", runningPid)
		fmt.Printf("Run 'sovereign stop --%s' first to restart\n", cfg.Name)
		return nil
	}

	fmt.Println("Tailscale: Using persistent machine identity (no cleanup needed)")

	// TEAM_037: Check for daemon script (preferred) or fall back to legacy start.sh
	daemonScript := "/data/sovereign/sovereign_start.sh"
	legacyScript := fmt.Sprintf("%s/start.sh", cfg.DevicePath)

	if !device.FileExists(daemonScript) && !device.FileExists(legacyScript) {
		return fmt.Errorf("no start script found - run 'sovereign deploy --%s' first", cfg.Name)
	}

	consoleLog := fmt.Sprintf("%s/console.log", cfg.DevicePath)

	// TEAM_041: Clean up any stale state before starting
	// Remove old console.log, socket, and pid files
	device.RunShellCommand(fmt.Sprintf("rm -f %s %s/vm.sock %s/vm.pid", consoleLog, cfg.DevicePath, cfg.DevicePath))

	// TEAM_037: Use daemon script with "start <vm>" to start a single VM
	// The daemon script stays alive in background, keeping crosvm as its child
	// This prevents Android init from killing crosvm as an orphaned process
	if device.FileExists(daemonScript) {
		fmt.Println("Starting VM via daemon (prevents Android killing)...")

		// TEAM_041: Clear old daemon log before starting
		daemonLog := fmt.Sprintf("/data/sovereign/daemon_%s.log", cfg.Name)
		device.RunShellCommand(fmt.Sprintf("rm -f %s", daemonLog))

		// TEAM_041: Run daemon in a completely detached process
		// Use SysProcAttr to create a new process group so it survives when Go exits
		startCmd := fmt.Sprintf("%s start %s", daemonScript, cfg.Name)
		cmd := exec.Command("adb", "shell", "su", "-c", startCmd)

		// Detach the process completely - new process group, new session
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true, // Create new process group
			Pgid:    0,    // Use the new process's PID as PGID
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("daemon start failed: %w", err)
		}

		fmt.Printf("Started adb process (PID: %d)\n", cmd.Process.Pid)

		// Release the process - we don't want to wait for it or kill it on exit
		if err := cmd.Process.Release(); err != nil {
			fmt.Printf("Warning: could not release process: %v\n", err)
		}

		// Give the daemon time to set up networking and start crosvm
		fmt.Println("Waiting for daemon to start VM...")
		time.Sleep(10 * time.Second)
	} else {
		// Fallback to legacy approach (will still be killed after ~90s)
		fmt.Println("⚠ Using legacy start.sh (VMs may be killed after ~90s)")
		fmt.Println("  Run 'sovereign deploy' to install the daemon script for stability")
		cmd := exec.Command("adb", "shell", "su", "-c", legacyScript)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("start script failed: %w", err)
		}
	}

	fmt.Println("\n--- Boot Sequence ---")
	return StreamBootLogs(cfg)
}

// StreamBootLogs streams console.log and waits for the ready marker.
// TEAM_029: Extracted from sql/lifecycle.go streamBootAndWaitForPostgres()
// and forge/lifecycle.go streamBootAndWaitForForgejo()
// TEAM_041: Added startup grace period - daemon script takes time to set up networking
func StreamBootLogs(cfg *VMConfig) error {
	timeout := cfg.StartTimeout
	if timeout == 0 {
		timeout = 90
	}

	var lastLineCount int
	startTime := time.Now()
	consoleLog := fmt.Sprintf("%s/console.log", cfg.DevicePath)

	// TEAM_041: Grace period before checking if process died
	// The daemon script runs disable_process_killers and setup_networking before starting crosvm
	// This takes 5-10 seconds, so don't declare death until after grace period
	const startupGracePeriod = 15 * time.Second
	processEverSeen := false

	for {
		elapsed := time.Since(startTime)
		if elapsed > time.Duration(timeout)*time.Second {
			return fmt.Errorf("timeout waiting for %s (%.0fs) - check 'adb shell cat %s'",
				cfg.DisplayName, elapsed.Seconds(), consoleLog)
		}

		out, _ := device.RunShellCommand(fmt.Sprintf("cat %s 2>/dev/null | tail -n +%d", consoleLog, lastLineCount+1))
		if out != "" {
			lines := strings.Split(out, "\n")
			for _, line := range lines {
				if line != "" {
					fmt.Println(line)
					lastLineCount++

					if cfg.ReadyMarker != "" && strings.Contains(line, cfg.ReadyMarker) {
						time.Sleep(2 * time.Second)
						fmt.Printf("\n✓ %s VM started\n", cfg.DisplayName)
						fmt.Printf("\nNext: sovereign test --%s\n", cfg.Name)
						return nil
					}

					if strings.Contains(line, "INIT COMPLETE") {
						time.Sleep(2 * time.Second)
						fmt.Printf("\n✓ %s VM started\n", cfg.DisplayName)
						fmt.Printf("\nNext: sovereign test --%s\n", cfg.Name)
						return nil
					}

					if strings.Contains(line, "Kernel panic") || strings.Contains(line, "FATAL") {
						return fmt.Errorf("VM boot failed - see output above")
					}
				}
			}
		}

		// TEAM_041: Check if process is running, but respect grace period
		pid := device.GetProcessPID(cfg.ProcessPattern)
		if pid != "" {
			processEverSeen = true
		} else if processEverSeen || elapsed > startupGracePeriod {
			// Only declare death if we saw the process before and it's gone,
			// or if grace period passed and process never appeared
			return fmt.Errorf("VM process died during boot - check console.log")
		}

		time.Sleep(500 * time.Millisecond)
	}
}
