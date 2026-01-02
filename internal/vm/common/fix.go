// Fix command - automatic issue detection and repair
// TEAM_041: Created for self-healing VM infrastructure
package common

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/anthropics/sovereign/internal/device"
)

// FixResult represents the result of a fix attempt
type FixResult struct {
	Issue   string
	Fixed   bool
	Message string
}

// FixVM attempts to automatically detect and fix common issues for a VM
func FixVM(cfg *VMConfig) error {
	fmt.Printf("=== Auto-Fix for %s VM ===\n\n", cfg.DisplayName)

	var results []FixResult

	// 1. Check and fix device connectivity
	fmt.Println("## 1. Checking device connectivity...")
	if !device.IsConnected() {
		fmt.Println("   ✗ Device not connected - cannot auto-fix")
		return fmt.Errorf("device not connected")
	}
	fmt.Println("   ✓ Device connected")

	// 2. Check and fix bridge network
	fmt.Println("\n## 2. Checking bridge network...")
	result := fixBridge()
	results = append(results, result)
	if result.Fixed {
		fmt.Printf("   ✓ Fixed: %s\n", result.Message)
	} else if result.Message != "" {
		fmt.Printf("   %s\n", result.Message)
	}

	// 3. Check and fix process killers
	fmt.Println("\n## 3. Checking Android process killers...")
	result = fixProcessKillers()
	results = append(results, result)
	if result.Fixed {
		fmt.Printf("   ✓ Fixed: %s\n", result.Message)
	} else if result.Message != "" {
		fmt.Printf("   %s\n", result.Message)
	}

	// 4. Check and fix VM process
	fmt.Println("\n## 4. Checking VM process...")
	result = fixVMProcess(cfg)
	results = append(results, result)
	if result.Fixed {
		fmt.Printf("   ✓ Fixed: %s\n", result.Message)
	} else if result.Message != "" {
		fmt.Printf("   %s\n", result.Message)
	}

	// 5. Check and fix TAP interface
	fmt.Println("\n## 5. Checking TAP interface...")
	result = fixTAP(cfg)
	results = append(results, result)
	if result.Fixed {
		fmt.Printf("   ✓ Fixed: %s\n", result.Message)
	} else if result.Message != "" {
		fmt.Printf("   %s\n", result.Message)
	}

	// 6. Check and fix stale state files
	fmt.Println("\n## 6. Checking for stale state...")
	result = fixStaleState(cfg)
	results = append(results, result)
	if result.Fixed {
		fmt.Printf("   ✓ Fixed: %s\n", result.Message)
	} else if result.Message != "" {
		fmt.Printf("   %s\n", result.Message)
	}

	// 7. For Vault/Forge - check SQL dependency
	if len(cfg.Dependencies) > 0 {
		fmt.Println("\n## 7. Checking dependencies...")
		result = fixDependencies(cfg)
		results = append(results, result)
		if result.Fixed {
			fmt.Printf("   ✓ Fixed: %s\n", result.Message)
		} else if result.Message != "" {
			fmt.Printf("   %s\n", result.Message)
		}
	}

	// 8. Check Tailscale connectivity (for Vault/Forge)
	if cfg.TailscaleHost != "" {
		fmt.Println("\n## 8. Checking Tailscale...")
		result = fixTailscale(cfg)
		results = append(results, result)
		if result.Fixed {
			fmt.Printf("   ✓ Fixed: %s\n", result.Message)
		} else if result.Message != "" {
			fmt.Printf("   %s\n", result.Message)
		}
	}

	// Summary
	fmt.Println("\n=== Fix Summary ===")
	fixedCount := 0
	for _, r := range results {
		if r.Fixed {
			fixedCount++
		}
	}

	if fixedCount > 0 {
		fmt.Printf("Fixed %d issue(s)\n", fixedCount)
	} else {
		fmt.Println("No issues needed fixing")
	}

	// Verify with test
	fmt.Println("\n## Running verification tests...")
	time.Sleep(2 * time.Second) // Give time for fixes to take effect

	// Quick connectivity check
	pid := device.GetProcessPID(cfg.ProcessPattern)
	if pid != "" {
		fmt.Printf("   ✓ VM process running (PID: %s)\n", pid)
	} else {
		fmt.Printf("   ⚠ VM process not running - may need 'sovereign start --%s'\n", cfg.Name)
	}

	if cfg.TailscaleHost != "" {
		fqdn := GetTailscaleFQDN(cfg)
		if fqdn != "" {
			cmd := exec.Command("curl", "-sk", "-o", "/dev/null", "-w", "%{http_code}",
				"--connect-timeout", "5", fmt.Sprintf("https://%s", fqdn))
			output, _ := cmd.Output()
			httpCode := strings.TrimSpace(string(output))
			if httpCode == "200" {
				fmt.Printf("   ✓ HTTPS connectivity: OK (%s)\n", fqdn)
			} else {
				fmt.Printf("   ⚠ HTTPS connectivity: HTTP %s\n", httpCode)
			}
		}
	}

	fmt.Println("\n=== Auto-Fix Complete ===")
	return nil
}

// fixBridge ensures the VM bridge network is properly configured
func fixBridge() FixResult {
	bridgeOut, _ := device.RunShellCommand("ip addr show vm_bridge 2>/dev/null")

	if bridgeOut == "" {
		// Bridge doesn't exist - create it
		device.RunShellCommand("ip link add vm_bridge type bridge")
		device.RunShellCommand("ip addr add 192.168.100.1/24 dev vm_bridge")
		device.RunShellCommand("ip link set vm_bridge up")
		return FixResult{Issue: "bridge", Fixed: true, Message: "Created vm_bridge with 192.168.100.1/24"}
	}

	if !strings.Contains(bridgeOut, "192.168.100.1") {
		// Bridge exists but wrong IP
		device.RunShellCommand("ip addr add 192.168.100.1/24 dev vm_bridge 2>/dev/null")
		return FixResult{Issue: "bridge", Fixed: true, Message: "Added IP 192.168.100.1/24 to vm_bridge"}
	}

	if !strings.Contains(bridgeOut, "UP") {
		device.RunShellCommand("ip link set vm_bridge up")
		return FixResult{Issue: "bridge", Fixed: true, Message: "Brought vm_bridge UP"}
	}

	return FixResult{Issue: "bridge", Fixed: false, Message: "✓ Bridge OK"}
}

// fixProcessKillers disables Android's phantom process killer
func fixProcessKillers() FixResult {
	// Check current setting
	out, _ := device.RunShellCommand("device_config get activity_manager max_phantom_processes 2>/dev/null")

	if strings.TrimSpace(out) != "2147483647" {
		device.RunShellCommand("device_config set_sync_disabled_for_tests persistent")
		device.RunShellCommand("device_config put activity_manager max_phantom_processes 2147483647")
		device.RunShellCommand("settings put global settings_enable_monitor_phantom_procs false")
		return FixResult{Issue: "process_killers", Fixed: true, Message: "Disabled phantom process killer"}
	}

	return FixResult{Issue: "process_killers", Fixed: false, Message: "✓ Process killers already disabled"}
}

// fixVMProcess checks if VM is running and restarts if dead
func fixVMProcess(cfg *VMConfig) FixResult {
	pid := device.GetProcessPID(cfg.ProcessPattern)

	if pid != "" {
		return FixResult{Issue: "vm_process", Fixed: false, Message: fmt.Sprintf("✓ VM running (PID: %s)", pid)}
	}

	// VM not running - check if there's a console.log with errors
	consoleLog := fmt.Sprintf("%s/console.log", cfg.DevicePath)
	errOut, _ := device.RunShellCommand(fmt.Sprintf("grep -iE '(FATAL|panic|Killed)' %s 2>/dev/null | tail -1", consoleLog))

	if strings.Contains(errOut, "password authentication failed") {
		return FixResult{
			Issue:   "vm_process",
			Fixed:   false,
			Message: "⚠ VM died with password auth error - restart SQL first, then this VM",
		}
	}

	// Try to start the VM via daemon
	daemonScript := "/data/sovereign/sovereign_start.sh"
	exists, _ := device.RunShellCommand(fmt.Sprintf("[ -f %s ] && echo yes", daemonScript))
	if strings.TrimSpace(exists) != "yes" {
		return FixResult{
			Issue:   "vm_process",
			Fixed:   false,
			Message: "⚠ Daemon script not deployed - run 'sovereign deploy' first",
		}
	}

	// Clean stale state and start
	device.RunShellCommand(fmt.Sprintf("rm -f %s/vm.sock %s/vm.pid %s/console.log",
		cfg.DevicePath, cfg.DevicePath, cfg.DevicePath))

	startCmd := fmt.Sprintf("%s start %s", daemonScript, cfg.Name)
	device.RunShellCommand(startCmd)

	// Wait for startup
	time.Sleep(5 * time.Second)

	// Verify
	newPid := device.GetProcessPID(cfg.ProcessPattern)
	if newPid != "" {
		return FixResult{Issue: "vm_process", Fixed: true, Message: fmt.Sprintf("Started VM (PID: %s)", newPid)}
	}

	return FixResult{Issue: "vm_process", Fixed: false, Message: "⚠ Failed to start VM - check 'sovereign diagnose'"}
}

// fixTAP ensures TAP interface is properly configured
func fixTAP(cfg *VMConfig) FixResult {
	tapOut, _ := device.RunShellCommand(fmt.Sprintf("ip link show %s 2>/dev/null", cfg.TAPInterface))

	if tapOut == "" {
		// TAP doesn't exist - will be created when VM starts
		return FixResult{Issue: "tap", Fixed: false, Message: "TAP will be created when VM starts"}
	}

	if strings.Contains(tapOut, "NO-CARRIER") {
		// TAP has no carrier - VM probably died, TAP needs recreation
		// This will be fixed when VM restarts
		return FixResult{Issue: "tap", Fixed: false, Message: "⚠ TAP shows NO-CARRIER - VM not connected"}
	}

	if !strings.Contains(tapOut, "UP") {
		device.RunShellCommand(fmt.Sprintf("ip link set %s up", cfg.TAPInterface))
		return FixResult{Issue: "tap", Fixed: true, Message: fmt.Sprintf("Brought %s UP", cfg.TAPInterface)}
	}

	// Check if attached to bridge
	if !strings.Contains(tapOut, "master vm_bridge") {
		device.RunShellCommand(fmt.Sprintf("ip link set %s master vm_bridge", cfg.TAPInterface))
		return FixResult{Issue: "tap", Fixed: true, Message: fmt.Sprintf("Attached %s to vm_bridge", cfg.TAPInterface)}
	}

	return FixResult{Issue: "tap", Fixed: false, Message: "✓ TAP OK"}
}

// fixStaleState cleans up stale socket/pid files
func fixStaleState(cfg *VMConfig) FixResult {
	pid := device.GetProcessPID(cfg.ProcessPattern)

	// If VM is running, don't clean state
	if pid != "" {
		return FixResult{Issue: "stale_state", Fixed: false, Message: "✓ VM running, state OK"}
	}

	// VM not running - check for stale files
	sockExists, _ := device.RunShellCommand(fmt.Sprintf("[ -f %s/vm.sock ] && echo yes", cfg.DevicePath))
	pidExists, _ := device.RunShellCommand(fmt.Sprintf("[ -f %s/vm.pid ] && echo yes", cfg.DevicePath))

	if strings.TrimSpace(sockExists) == "yes" || strings.TrimSpace(pidExists) == "yes" {
		device.RunShellCommand(fmt.Sprintf("rm -f %s/vm.sock %s/vm.pid", cfg.DevicePath, cfg.DevicePath))
		return FixResult{Issue: "stale_state", Fixed: true, Message: "Removed stale socket/pid files"}
	}

	return FixResult{Issue: "stale_state", Fixed: false, Message: "✓ No stale state"}
}

// fixDependencies checks if required services are running
func fixDependencies(cfg *VMConfig) FixResult {
	for _, dep := range cfg.Dependencies {
		// Check if dependency is reachable via TAP IP
		testCmd := fmt.Sprintf("timeout 2 nc -z %s %d 2>/dev/null && echo OK || echo FAIL",
			dep.TAPIP, dep.Port)
		out, _ := device.RunShellCommand(testCmd)

		if strings.TrimSpace(out) != "OK" {
			return FixResult{
				Issue: "dependencies",
				Fixed: false,
				Message: fmt.Sprintf("⚠ %s not reachable at %s:%d - start SQL VM first",
					dep.Name, dep.TAPIP, dep.Port),
			}
		}
	}

	return FixResult{Issue: "dependencies", Fixed: false, Message: "✓ Dependencies OK"}
}

// fixTailscale checks Tailscale connectivity
func fixTailscale(cfg *VMConfig) FixResult {
	tsOut, err := exec.Command("tailscale", "status").Output()
	if err != nil {
		return FixResult{Issue: "tailscale", Fixed: false, Message: "⚠ Cannot get tailscale status on host"}
	}

	lines := strings.Split(string(tsOut), "\n")
	for _, line := range lines {
		if strings.Contains(line, cfg.TailscaleHost) {
			if strings.Contains(line, "offline") {
				return FixResult{
					Issue:   "tailscale",
					Fixed:   false,
					Message: "⚠ VM registered but offline - restart VM",
				}
			}
			return FixResult{Issue: "tailscale", Fixed: false, Message: "✓ Tailscale connected"}
		}
	}

	return FixResult{
		Issue:   "tailscale",
		Fixed:   false,
		Message: "⚠ VM not registered with Tailscale - check auth key in .env",
	}
}

// FixAll attempts to fix common infrastructure issues
func FixAll() error {
	fmt.Println("=== Auto-Fix All Infrastructure ===")
	fmt.Println()

	if !device.IsConnected() {
		return fmt.Errorf("device not connected")
	}

	// Fix bridge
	fmt.Println("## Fixing bridge network...")
	fixBridge()

	// Fix process killers
	fmt.Println("## Disabling process killers...")
	fixProcessKillers()

	// Enable IP forwarding
	fmt.Println("## Enabling IP forwarding...")
	device.RunShellCommand("echo 1 > /proc/sys/net/ipv4/ip_forward")

	// Fix routing
	fmt.Println("## Fixing routing...")
	device.RunShellCommand("ip rule del from all lookup main pref 1 2>/dev/null; ip rule add from all lookup main pref 1")

	// Fix NAT
	fmt.Println("## Setting up NAT...")
	device.RunShellCommand("iptables -t nat -D POSTROUTING -s 192.168.100.0/24 -o wlan0 -j MASQUERADE 2>/dev/null")
	device.RunShellCommand("iptables -t nat -A POSTROUTING -s 192.168.100.0/24 -o wlan0 -j MASQUERADE")

	// Fix forwarding rules
	fmt.Println("## Setting up forwarding...")
	device.RunShellCommand("iptables -D FORWARD -i vm_bridge -o wlan0 -j ACCEPT 2>/dev/null")
	device.RunShellCommand("iptables -I FORWARD 1 -i vm_bridge -o wlan0 -j ACCEPT")
	device.RunShellCommand("iptables -D FORWARD -i wlan0 -o vm_bridge -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null")
	device.RunShellCommand("iptables -I FORWARD 2 -i wlan0 -o vm_bridge -m state --state RELATED,ESTABLISHED -j ACCEPT")

	fmt.Println("\n=== Infrastructure Fix Complete ===")
	fmt.Println("Run 'sovereign fix --sql' then 'sovereign fix --vault' to fix individual VMs")
	return nil
}
