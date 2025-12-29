// SQL VM lifecycle operations (Start, Stop, Remove)
// TEAM_022: Split from sql.go for readability
package sql

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/anthropics/sovereign/internal/device"
)

func (v *VM) Start() error {
	fmt.Println("=== Starting PostgreSQL VM ===")

	// Check if VM is already running
	runningPid := device.GetProcessPID("crosvm.*sql")
	if runningPid != "" {
		fmt.Printf("⚠ VM already running (PID: %s)\n", runningPid)
		fmt.Println("Run 'sovereign stop --sql' first to restart")
		return nil
	}

	// TEAM_022: STABILITY IS THE DEFAULT - always clean up old registrations
	// This ensures dependants who rely on sovereign-sql IP don't break.
	// --force means "skip cleanup" (dangerous)
	if !ForceDeploySkipTailscaleCheck {
		// DEFAULT: Remove old registrations to maintain stable IP
		RemoveTailscaleRegistrations()
	} else {
		fmt.Println("⚠ FORCE MODE: Skipping Tailscale cleanup (may create duplicates!)")
	}

	// Check if start script exists
	if !device.FileExists("/data/sovereign/vm/sql/start.sh") {
		return fmt.Errorf("start script not found - run 'sovereign deploy --sql' first")
	}

	// TEAM_020: Clear old console log before starting
	device.RunShellCommand("rm -f /data/sovereign/vm/sql/console.log")

	// Start the VM (start.sh backgrounds crosvm and returns immediately)
	fmt.Println("Starting VM...")
	cmd := exec.Command("adb", "shell", "su", "-c", "/data/sovereign/vm/sql/start.sh")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start script failed: %w", err)
	}

	// TEAM_020: Stream boot sequence and wait for PostgreSQL readiness
	fmt.Println("\n--- Boot Sequence ---")
	return streamBootAndWaitForPostgres()
}

// TEAM_020: Stream console.log and wait for PostgreSQL to be ready
func streamBootAndWaitForPostgres() error {
	const (
		maxWaitSeconds   = 90
		pollIntervalMs   = 500
		pgCheckAfterSecs = 15 // Start checking PostgreSQL port after kernel boot
	)

	var lastLineCount int
	var kernelBooted bool
	startTime := time.Now()

	for {
		elapsed := time.Since(startTime)
		if elapsed > maxWaitSeconds*time.Second {
			return fmt.Errorf("timeout waiting for PostgreSQL (%.0fs) - check 'adb shell cat /data/sovereign/vm/sql/console.log'", elapsed.Seconds())
		}

		// Get current console.log content
		out, _ := device.RunShellCommand(fmt.Sprintf("cat /data/sovereign/vm/sql/console.log 2>/dev/null | tail -n +%d", lastLineCount+1))
		if out != "" {
			// Print new lines
			lines := strings.Split(out, "\n")
			for _, line := range lines {
				if line != "" {
					fmt.Println(line)
					lastLineCount++

					// Check for PostgreSQL ready marker
					if strings.Contains(line, "PostgreSQL started") || strings.Contains(line, "database system is ready") {
						fmt.Println("\n✓ PostgreSQL VM started and ready")
						fmt.Println("\nNext: sovereign test --sql")
						return nil
					}

					// Check for INIT COMPLETE as fallback
					if strings.Contains(line, "INIT COMPLETE") {
						time.Sleep(2 * time.Second)
						fmt.Println("\n✓ PostgreSQL VM started")
						fmt.Println("\nNext: sovereign test --sql")
						return nil
					}

					// Track kernel boot completion
					if strings.Contains(line, "Run /sbin/simple_init") {
						kernelBooted = true
					}

					// Check for fatal errors
					if strings.Contains(line, "Kernel panic") || strings.Contains(line, "FATAL") {
						return fmt.Errorf("VM boot failed - see output above")
					}
				}
			}
		}

		// Check if crosvm is still running
		if device.GetProcessPID("crosvm.*sql") == "" {
			return fmt.Errorf("VM process died during boot - check console.log")
		}

		// TEAM_020: After kernel boots, check if PostgreSQL port is open (fallback for old rootfs)
		if kernelBooted && elapsed > pgCheckAfterSecs*time.Second {
			pgCheck, _ := device.RunShellCommand("nc -z 192.168.100.2 5432 && echo OPEN")
			if pgCheck == "OPEN" {
				fmt.Println("\n✓ PostgreSQL VM started (port 5432 responding)")
				fmt.Println("\nNext: sovereign test --sql")
				return nil
			}
		}

		time.Sleep(pollIntervalMs * time.Millisecond)
	}
}

func (v *VM) Stop() error {
	fmt.Println("=== Stopping PostgreSQL VM ===")

	// TEAM_020: Use pgrep like Start() and Test() - the for-loop was unreliable
	// TEAM_022: Use [c]rosvm pattern to avoid grep matching itself
	pid := device.GetProcessPID("[c]rosvm.*sql")

	if pid != "" {
		fmt.Printf("Stopping VM (PID: %s)...\n", pid)
		if err := device.KillProcess(pid); err != nil {
			device.RunShellCommand(fmt.Sprintf("kill -9 %s", pid))
		}
	} else {
		fmt.Println("VM not running")
	}

	// TEAM_018: ALWAYS clean up networking, even if VM wasn't running
	fmt.Println("Cleaning up networking...")
	device.RunShellCommand("ip link del vm_sql 2>/dev/null")
	device.RunShellCommand("ip rule del from all lookup main pref 1 2>/dev/null")
	device.RunShellCommand("ip rule del from 192.168.100.0/24 lookup wlan0 2>/dev/null")
	device.RunShellCommand("ip rule del from 192.168.100.0/24 lookup main 2>/dev/null")
	device.RunShellCommand("iptables -t nat -D POSTROUTING -s 192.168.100.0/24 -o wlan0 -j MASQUERADE 2>/dev/null")
	device.RunShellCommand("iptables -D FORWARD -i vm_sql -o wlan0 -j ACCEPT 2>/dev/null")
	device.RunShellCommand("iptables -D FORWARD -i wlan0 -o vm_sql -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null")

	// TEAM_022: Clean up pid file created by start.sh
	device.RunShellCommand("rm -f /data/sovereign/vm/sql/vm.pid 2>/dev/null")

	fmt.Println("✓ VM stopped")
	return nil
}

// Remove removes the SQL VM from the device
// TEAM_018: Enhanced to clean up EVERYTHING for a truly clean phone
// TEAM_022: Also removes Tailscale registration to prevent duplicates
func (v *VM) Remove() error {
	fmt.Println("=== Removing SQL VM from device ===")

	// First stop the VM if running (this also cleans up networking)
	v.Stop()

	// TEAM_022: Remove Tailscale registration to prevent duplicates on next deploy
	// THIS IS CRITICAL - without this, every deploy creates a new registration!
	fmt.Println("Removing Tailscale registration...")
	RemoveTailscaleRegistrations()

	// Extra cleanup for any leftover networking rules
	fmt.Println("Ensuring all networking rules are removed...")
	device.RunShellCommand("ip rule del from 192.168.100.0/24 lookup wlan0 2>/dev/null")
	device.RunShellCommand("ip rule del from 192.168.100.0/24 lookup main 2>/dev/null")

	// Remove all VM files from device
	fmt.Println("Removing VM files from device...")
	device.RemoveDir("/data/sovereign/vm/sql")

	// Verify removal
	if device.DirExists("/data/sovereign/vm/sql") {
		return fmt.Errorf("failed to remove /data/sovereign/vm/sql")
	}

	fmt.Println("✓ SQL VM removed from device")
	fmt.Println("\nTo redeploy: sovereign deploy --sql")
	return nil
}
