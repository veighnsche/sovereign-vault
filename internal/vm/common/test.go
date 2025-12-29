// Common test patterns for VMs
// TEAM_029: Extracted from sql/verify.go and forge/verify.go
package common

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/anthropics/sovereign/internal/device"
)

// RunVMTests runs the common VM tests plus any custom tests.
// TEAM_029: Extracted from sql/verify.go Test() and forge/verify.go Test()
func RunVMTests(cfg *VMConfig, customTests []TestFunc) error {
	fmt.Printf("=== Testing %s VM ===\n", cfg.DisplayName)
	allPassed := true
	testNum := 1

	// Test 1: VM process running
	fmt.Printf("%d. VM process running: ", testNum)
	testNum++
	out, _ := device.RunShellCommand(fmt.Sprintf("ps -ef | grep '%s' | grep -v grep | awk '{print $2}' | head -1", cfg.ProcessPattern))
	vmPid := strings.TrimSpace(out)
	if vmPid == "" {
		fmt.Println("✗ FAIL (crosvm not running)")
		allPassed = false
	} else {
		fmt.Printf("✓ PASS (PID: %s)\n", vmPid)
	}

	// Test 2: TAP interface exists
	fmt.Printf("%d. TAP interface (%s): ", testNum, cfg.TAPInterface)
	testNum++
	tapOut, _ := device.RunShellCommand(fmt.Sprintf("ip link show %s 2>/dev/null | grep -c UP", cfg.TAPInterface))
	if strings.TrimSpace(tapOut) == "1" {
		fmt.Println("✓ PASS")
	} else {
		fmt.Println("✗ FAIL (TAP interface not up)")
		allPassed = false
	}

	// Test 3: Tailscale connected
	fmt.Printf("%d. Tailscale connected: ", testNum)
	testNum++
	tsOut, tsErr := exec.Command("tailscale", "status").Output()
	if tsErr != nil {
		fmt.Println("? SKIP (tailscale not available on host)")
	} else {
		var tsIP string
		lines := strings.Split(string(tsOut), "\n")
		for _, line := range lines {
			if strings.Contains(line, cfg.TailscaleHost) && !strings.Contains(line, "offline") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					tsIP = parts[0]
					fmt.Printf("✓ PASS (%s as %s)\n", tsIP, parts[1])
				}
				break
			}
		}
		if tsIP == "" {
			fmt.Printf("✗ FAIL (no active %s* in tailscale)\n", cfg.TailscaleHost)
			allPassed = false
		}
	}

	// Run custom tests
	for _, testFn := range customTests {
		result := testFn(cfg)
		fmt.Printf("%d. %s: ", testNum, result.Name)
		testNum++
		if result.Passed {
			if result.Message != "" {
				fmt.Printf("✓ PASS (%s)\n", result.Message)
			} else {
				fmt.Println("✓ PASS")
			}
		} else {
			fmt.Printf("✗ FAIL (%s)\n", result.Message)
			allPassed = false
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("=== ALL TESTS PASSED ===")
		return nil
	}
	return fmt.Errorf("some tests failed - see above")
}

// TestPortOpen checks if a port is accessible on the TAP interface.
// TEAM_029: Helper for service-specific tests
func TestPortOpen(cfg *VMConfig, port int) TestResult {
	out, _ := device.RunShellCommand(fmt.Sprintf("nc -z %s %d && echo OPEN || echo CLOSED", cfg.TAPGuestIP, port))
	if strings.TrimSpace(out) == "OPEN" {
		return TestResult{
			Name:    fmt.Sprintf("Port %d responding (via TAP)", port),
			Passed:  true,
			Message: "",
		}
	}
	return TestResult{
		Name:    fmt.Sprintf("Port %d responding (via TAP)", port),
		Passed:  false,
		Message: fmt.Sprintf("port %d not reachable on TAP", port),
	}
}
