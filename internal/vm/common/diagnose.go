// Diagnose command - comprehensive debugging for VMs
// TEAM_041: Created for better troubleshooting
package common

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/anthropics/sovereign/internal/device"
)

// DiagnoseVM runs comprehensive diagnostics for a VM
func DiagnoseVM(cfg *VMConfig) error {
	fmt.Printf("=== Diagnosing %s VM ===\n\n", cfg.DisplayName)

	// 1. Process Status
	fmt.Println("## 1. Process Status")
	pid := device.GetProcessPID(cfg.ProcessPattern)
	if pid != "" {
		fmt.Printf("   ✓ VM process running (PID: %s)\n", pid)
		// Get process details
		out, _ := device.RunShellCommand(fmt.Sprintf("ps -p %s -o pid,ppid,etime,args 2>/dev/null | tail -1", pid))
		if out != "" {
			fmt.Printf("   Details: %s\n", out)
		}
	} else {
		fmt.Printf("   ✗ VM process NOT running\n")
		fmt.Printf("   Pattern used: %s\n", cfg.ProcessPattern)
	}

	// 2. TAP Interface
	fmt.Println("\n## 2. TAP Interface")
	tapOut, _ := device.RunShellCommand(fmt.Sprintf("ip link show %s 2>/dev/null", cfg.TAPInterface))
	if tapOut != "" {
		if strings.Contains(tapOut, "UP") && strings.Contains(tapOut, "LOWER_UP") {
			fmt.Printf("   ✓ TAP %s is UP\n", cfg.TAPInterface)
		} else if strings.Contains(tapOut, "NO-CARRIER") {
			fmt.Printf("   ⚠ TAP %s has NO-CARRIER (VM may not be connected)\n", cfg.TAPInterface)
		} else {
			fmt.Printf("   ⚠ TAP %s exists but state unclear\n", cfg.TAPInterface)
		}
		// Show TAP details
		lines := strings.Split(tapOut, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				fmt.Printf("   %s\n", strings.TrimSpace(line))
			}
		}
	} else {
		fmt.Printf("   ✗ TAP %s does NOT exist\n", cfg.TAPInterface)
	}

	// 3. Bridge Status
	fmt.Println("\n## 3. Bridge Network")
	bridgeOut, _ := device.RunShellCommand("ip link show vm_bridge 2>/dev/null")
	if bridgeOut != "" {
		fmt.Printf("   ✓ Bridge vm_bridge exists\n")
	} else {
		fmt.Printf("   ✗ Bridge vm_bridge does NOT exist\n")
	}
	// Show bridge members
	brctl, _ := device.RunShellCommand("cat /sys/class/net/vm_bridge/brif/*/ifindex 2>/dev/null | wc -l")
	if brctl != "" && brctl != "0" {
		fmt.Printf("   Bridge has %s attached interfaces\n", strings.TrimSpace(brctl))
	}

	// 4. Port Connectivity (TAP)
	fmt.Println("\n## 4. Port Connectivity (TAP)")
	for _, port := range cfg.ServicePorts {
		// Test via nc from host
		testCmd := fmt.Sprintf("timeout 2 nc -zv %s %d 2>&1", cfg.TAPGuestIP, port)
		out, _ := device.RunShellCommand(testCmd)
		if strings.Contains(out, "succeeded") || strings.Contains(out, "open") {
			fmt.Printf("   ✓ Port %d on %s: OPEN\n", port, cfg.TAPGuestIP)
		} else {
			fmt.Printf("   ✗ Port %d on %s: CLOSED/UNREACHABLE\n", port, cfg.TAPGuestIP)
		}
	}

	// 5. Tailscale Status
	fmt.Println("\n## 5. Tailscale Status")
	tsOut, err := exec.Command("tailscale", "status").Output()
	if err != nil {
		fmt.Printf("   ✗ Cannot get tailscale status: %v\n", err)
	} else {
		lines := strings.Split(string(tsOut), "\n")
		found := false
		for _, line := range lines {
			if strings.Contains(line, cfg.TailscaleHost) {
				found = true
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					ip := parts[0]
					hostname := parts[1]
					online := !strings.Contains(line, "offline")
					if online {
						fmt.Printf("   ✓ %s (%s) - ONLINE\n", hostname, ip)
					} else {
						fmt.Printf("   ⚠ %s (%s) - OFFLINE\n", hostname, ip)
					}
				}
			}
		}
		if !found {
			fmt.Printf("   ✗ No Tailscale entry matching '%s'\n", cfg.TailscaleHost)
		}
	}

	// 6. HTTPS Connectivity (if Tailscale)
	if cfg.TailscaleHost != "" {
		fmt.Println("\n## 6. HTTPS Connectivity")
		fqdn := GetTailscaleFQDN(cfg)
		if fqdn == "" {
			fmt.Printf("   ✗ Cannot determine Tailscale FQDN\n")
		} else {
			url := fmt.Sprintf("https://%s", fqdn)
			fmt.Printf("   Testing: %s\n", url)

			// Do actual HTTP request with verbose output
			start := time.Now()
			cmd := exec.Command("curl", "-sk", "-o", "/dev/null", "-w",
				"HTTP %{http_code}, Time: %{time_total}s, IP: %{remote_ip}",
				"--connect-timeout", "10", url)
			output, err := cmd.CombinedOutput()
			elapsed := time.Since(start)

			if err != nil {
				fmt.Printf("   ✗ Request failed: %v\n", err)
				fmt.Printf("   Output: %s\n", strings.TrimSpace(string(output)))
			} else {
				result := strings.TrimSpace(string(output))
				if strings.Contains(result, "HTTP 200") {
					fmt.Printf("   ✓ %s (%.2fs)\n", result, elapsed.Seconds())
				} else {
					fmt.Printf("   ⚠ %s (%.2fs)\n", result, elapsed.Seconds())
				}
			}

			// Also test /api endpoint for Vaultwarden
			if cfg.Name == "vault" {
				apiUrl := fmt.Sprintf("https://%s/api/config", fqdn)
				cmd = exec.Command("curl", "-sk", "-o", "/dev/null", "-w", "%{http_code}",
					"--connect-timeout", "5", apiUrl)
				output, _ = cmd.Output()
				httpCode := strings.TrimSpace(string(output))
				if httpCode == "200" {
					fmt.Printf("   ✓ API endpoint: HTTP %s\n", httpCode)
				} else {
					fmt.Printf("   ⚠ API endpoint: HTTP %s\n", httpCode)
				}
			}
		}
	}

	// 7. Console Log (last 10 lines)
	fmt.Println("\n## 7. Recent Console Output")
	consoleLog := fmt.Sprintf("%s/console.log", cfg.DevicePath)
	logOut, _ := device.RunShellCommand(fmt.Sprintf("tail -10 %s 2>/dev/null", consoleLog))
	if logOut != "" {
		lines := strings.Split(logOut, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				// Truncate long lines
				if len(line) > 100 {
					line = line[:100] + "..."
				}
				fmt.Printf("   %s\n", line)
			}
		}
	} else {
		fmt.Printf("   (no console.log found)\n")
	}

	// 8. Error Detection
	fmt.Println("\n## 8. Error Detection")
	errOut, _ := device.RunShellCommand(fmt.Sprintf("grep -iE '(error|fatal|failed|panic)' %s 2>/dev/null | tail -5", consoleLog))
	if errOut != "" && strings.TrimSpace(errOut) != "" {
		fmt.Printf("   ⚠ Errors found in console.log:\n")
		lines := strings.Split(errOut, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				if len(line) > 100 {
					line = line[:100] + "..."
				}
				fmt.Printf("   %s\n", line)
			}
		}
	} else {
		fmt.Printf("   ✓ No obvious errors in console.log\n")
	}

	// 9. Recommendations
	fmt.Println("\n## 9. Recommendations")
	if pid == "" {
		fmt.Printf("   → Run: sovereign start --%s\n", cfg.Name)
	}
	if tapOut == "" || strings.Contains(tapOut, "NO-CARRIER") {
		fmt.Printf("   → TAP issue: Try sovereign stop --%s && sovereign start --%s\n", cfg.Name, cfg.Name)
	}
	if errOut != "" && strings.Contains(errOut, "password authentication failed") {
		fmt.Printf("   → Database password mismatch: Restart SQL VM first, then restart this VM\n")
	}

	fmt.Println()
	fmt.Println("=== Diagnosis Complete ===")
	return nil
}

// DiagnoseAll runs diagnostics on all VMs and infrastructure
func DiagnoseAll() error {
	fmt.Println("=== Sovereign Vault System Diagnosis ===")
	fmt.Println()

	// 1. Host connectivity
	fmt.Println("## Host System")
	if device.IsConnected() {
		fmt.Println("   ✓ ADB connected to device")
	} else {
		fmt.Println("   ✗ ADB NOT connected")
		return fmt.Errorf("no device connected")
	}

	// 2. crosvm availability
	crosvmOut, _ := device.RunShellCommand("ls -la /apex/com.android.virt/bin/crosvm 2>/dev/null")
	if crosvmOut != "" {
		fmt.Println("   ✓ crosvm binary found")
	} else {
		fmt.Println("   ✗ crosvm binary NOT found - AVF may not be enabled")
	}

	// 3. Daemon status
	daemonOut, _ := device.RunShellCommand("pgrep -f sovereign_start.sh 2>/dev/null | wc -l")
	daemonCount := strings.TrimSpace(daemonOut)
	if daemonCount != "0" && daemonCount != "" {
		fmt.Printf("   ✓ Daemon processes running: %s\n", daemonCount)
	} else {
		fmt.Println("   ⚠ No daemon processes running")
	}

	// 4. Bridge network
	bridgeOut, _ := device.RunShellCommand("ip addr show vm_bridge 2>/dev/null")
	if strings.Contains(bridgeOut, "192.168.100.1") {
		fmt.Println("   ✓ Bridge network configured (192.168.100.1)")
	} else {
		fmt.Println("   ⚠ Bridge network not configured")
	}

	// 5. Running VMs
	fmt.Println("\n## Running VMs")
	vmOut, _ := device.RunShellCommand("ps -ef | grep '[c]rosvm' | wc -l")
	vmCount := strings.TrimSpace(vmOut)
	fmt.Printf("   crosvm processes: %s\n", vmCount)

	// List each VM
	for _, vmName := range []string{"sql", "forge", "vault"} {
		pattern := fmt.Sprintf("[c]rosvm.*vm/%s/", vmName)
		pid := device.GetProcessPID(pattern)
		if pid != "" {
			fmt.Printf("   ✓ %s: running (PID %s)\n", vmName, pid)
		} else {
			fmt.Printf("   ✗ %s: not running\n", vmName)
		}
	}

	// 6. Tailscale status
	fmt.Println("\n## Tailscale Connections")
	tsOut, err := exec.Command("tailscale", "status").Output()
	if err != nil {
		fmt.Printf("   ✗ Cannot get tailscale status: %v\n", err)
	} else {
		lines := strings.Split(string(tsOut), "\n")
		for _, line := range lines {
			if strings.Contains(line, "sovereign-") {
				fmt.Printf("   %s\n", strings.TrimSpace(line))
			}
		}
	}

	fmt.Println()
	fmt.Println("=== Use 'sovereign diagnose --<vm>' for detailed VM diagnosis ===")
	return nil
}
