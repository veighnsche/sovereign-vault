// Forge VM lifecycle operations (Start, Stop, Remove)
// TEAM_025: Split from forge.go following sql/lifecycle.go pattern
package forge

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/anthropics/sovereign/internal/device"
)

// Start starts the Forgejo VM
// TEAM_025: Refactored to use TAP networking and correct paths
func (v *VM) Start() error {
	fmt.Println("=== Starting Forgejo VM ===")

	// Check if VM is already running
	// TEAM_025: Use [c]rosvm pattern to avoid grep matching itself
	runningPid := device.GetProcessPID("[c]rosvm.*forge")
	if runningPid != "" {
		fmt.Printf("⚠ VM already running (PID: %s)\n", runningPid)
		fmt.Println("Run 'sovereign stop --forge' first to restart")
		return nil
	}

	// TEAM_025: Use persistent Tailscale identity (no cleanup needed)
	fmt.Println("Tailscale: Using persistent machine identity (no cleanup needed)")
	fmt.Println("Note: Forgejo requires SQL VM for database")

	// Check if start script exists
	if !device.FileExists("/data/sovereign/vm/forgejo/start.sh") {
		return fmt.Errorf("start script not found - run 'sovereign deploy --forge' first")
	}

	// TEAM_025: Clear old console log before starting
	device.RunShellCommand("rm -f /data/sovereign/vm/forgejo/console.log")

	// Start the VM (start.sh backgrounds crosvm and returns immediately)
	fmt.Println("Starting VM...")
	cmd := exec.Command("adb", "shell", "su", "-c", "/data/sovereign/vm/forgejo/start.sh")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start script failed: %w", err)
	}

	// TEAM_025: Stream boot sequence and wait for Forgejo readiness
	fmt.Println("\n--- Boot Sequence ---")
	return streamBootAndWaitForForgejo()
}

// streamBootAndWaitForForgejo streams console.log and waits for Forgejo to be ready
// TEAM_025: Modeled after sql/lifecycle.go streamBootAndWaitForPostgres
func streamBootAndWaitForForgejo() error {
	const (
		maxWaitSeconds = 120 // Forgejo takes longer to start than PostgreSQL
		pollIntervalMs = 500
	)

	var lastLineCount int
	startTime := time.Now()

	for {
		elapsed := time.Since(startTime)
		if elapsed > maxWaitSeconds*time.Second {
			return fmt.Errorf("timeout waiting for Forgejo (%.0fs) - check 'adb shell cat /data/sovereign/vm/forgejo/console.log'", elapsed.Seconds())
		}

		// Get current console.log content
		out, _ := device.RunShellCommand(fmt.Sprintf("cat /data/sovereign/vm/forgejo/console.log 2>/dev/null | tail -n +%d", lastLineCount+1))
		if out != "" {
			// Print new lines
			lines := strings.Split(out, "\n")
			for _, line := range lines {
				if line != "" {
					fmt.Println(line)
					lastLineCount++

					// Check for INIT COMPLETE marker
					if strings.Contains(line, "INIT COMPLETE") {
						time.Sleep(2 * time.Second)
						fmt.Println("\n✓ Forgejo VM started")
						fmt.Println("\nNext: sovereign test --forge")
						return nil
					}

					// Check for fatal errors
					if strings.Contains(line, "Kernel panic") || strings.Contains(line, "FATAL") {
						return fmt.Errorf("VM boot failed - see output above")
					}
				}
			}
		}

		// Check if crosvm is still running
		if device.GetProcessPID("[c]rosvm.*forge") == "" {
			return fmt.Errorf("VM process died during boot - check console.log")
		}

		time.Sleep(pollIntervalMs * time.Millisecond)
	}
}

// Stop stops the Forgejo VM
// TEAM_025: Added TAP networking cleanup
func (v *VM) Stop() error {
	fmt.Println("=== Stopping Forgejo VM ===")

	// TEAM_025: Use [c]rosvm pattern to avoid grep matching itself
	pid := device.GetProcessPID("[c]rosvm.*forge")

	if pid != "" {
		fmt.Printf("Stopping VM (PID: %s)...\n", pid)
		if err := device.KillProcess(pid); err != nil {
			device.RunShellCommand(fmt.Sprintf("kill -9 %s", pid))
		}
	} else {
		fmt.Println("VM not running")
	}

	// TEAM_025: Clean up TAP networking
	fmt.Println("Cleaning up networking...")
	device.RunShellCommand("ip link del vm_forge 2>/dev/null")
	device.RunShellCommand("iptables -t nat -D POSTROUTING -s 192.168.101.0/24 -o wlan0 -j MASQUERADE 2>/dev/null")
	device.RunShellCommand("iptables -D FORWARD -i vm_forge -o wlan0 -j ACCEPT 2>/dev/null")
	device.RunShellCommand("iptables -D FORWARD -i wlan0 -o vm_forge -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null")

	// Clean up pid file
	device.RunShellCommand("rm -f /data/sovereign/vm/forgejo/vm.pid 2>/dev/null")

	fmt.Println("✓ VM stopped")
	return nil
}

// Remove removes the Forgejo VM from the device
// TEAM_025: Also removes Tailscale registration
func (v *VM) Remove() error {
	fmt.Println("=== Removing Forgejo VM from device ===")

	// First stop the VM if running (this also cleans up networking)
	v.Stop()

	// TEAM_025: Remove Tailscale registration to prevent duplicates on next deploy
	fmt.Println("Removing Tailscale registration...")
	RemoveTailscaleRegistrations()

	// Remove all VM files from device
	fmt.Println("Removing VM files from device...")
	device.RemoveDir("/data/sovereign/vm/forgejo")

	// Verify removal
	if device.DirExists("/data/sovereign/vm/forgejo") {
		return fmt.Errorf("failed to remove /data/sovereign/vm/forgejo")
	}

	fmt.Println("✓ Forgejo VM removed from device")
	fmt.Println("\nTo redeploy: sovereign deploy --forge")
	return nil
}
