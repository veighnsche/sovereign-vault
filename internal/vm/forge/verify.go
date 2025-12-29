// Forge VM verification (Test command and Tailscale checks)
// TEAM_025: Split from forge.go following sql/verify.go pattern
package forge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/anthropics/sovereign/internal/device"
)

// Test tests the Forgejo VM connectivity
// TEAM_025: Refactored to test TAP interface and correct Tailscale hostname
func (v *VM) Test() error {
	fmt.Println("=== Testing Forgejo VM ===")
	allPassed := true

	// Test 1: VM process running
	fmt.Print("1. VM process running: ")
	// TEAM_025: Use [c]rosvm pattern to avoid grep matching itself
	out, _ := device.RunShellCommand("ps -ef | grep '[c]rosvm.*forge' | grep -v grep | awk '{print $2}' | head -1")
	vmPid := strings.TrimSpace(out)
	if vmPid == "" {
		fmt.Println("✗ FAIL (crosvm not running)")
		allPassed = false
	} else {
		fmt.Printf("✓ PASS (PID: %s)\n", vmPid)
	}

	// Test 2: TAP interface exists
	fmt.Print("2. TAP interface (vm_forge): ")
	tapOut, _ := device.RunShellCommand("ip link show vm_forge 2>/dev/null | grep -c UP")
	if strings.TrimSpace(tapOut) == "1" {
		fmt.Println("✓ PASS")
	} else {
		fmt.Println("✗ FAIL (TAP interface not up)")
		allPassed = false
	}

	// Test 3: Check Tailscale status
	// TEAM_025: Check for sovereign-forge hostname
	fmt.Print("3. Tailscale connected: ")
	tsOut, tsErr := exec.Command("tailscale", "status").Output()
	var tsIP string
	if tsErr != nil {
		fmt.Println("? SKIP (tailscale not available on host)")
	} else {
		lines := strings.Split(string(tsOut), "\n")
		for _, line := range lines {
			// Match sovereign-forge but not offline entries
			if strings.Contains(line, "sovereign-forge") && !strings.Contains(line, "offline") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					tsIP = parts[0]
					fmt.Printf("✓ PASS (%s as %s)\n", tsIP, parts[1])
				}
				break
			}
		}
		if tsIP == "" {
			fmt.Println("✗ FAIL (no active sovereign-forge* in tailscale)")
			allPassed = false
		}
	}

	// Test 4: Forgejo web UI responding via Tailscale
	fmt.Print("4. Forgejo web UI (via Tailscale): ")
	cmd := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
		"--connect-timeout", "5", "http://sovereign-forge:3000")
	output, _ := cmd.Output()
	httpCode := strings.TrimSpace(string(output))
	if httpCode == "200" {
		fmt.Println("✓ PASS")
	} else if httpCode == "302" || httpCode == "303" {
		fmt.Println("✓ PASS (redirect - needs initial setup)")
	} else {
		fmt.Printf("✗ FAIL (HTTP %s)\n", httpCode)
		allPassed = false
	}

	// Test 5: SSH port accessible
	fmt.Print("5. SSH port (via Tailscale): ")
	cmd = exec.Command("nc", "-z", "-w", "3", "sovereign-forge", "22")
	if err := cmd.Run(); err != nil {
		fmt.Println("⚠ WARN (SSH port not responding)")
		// Not a failure - SSH might be disabled
	} else {
		fmt.Println("✓ PASS")
	}

	fmt.Println()
	if allPassed {
		fmt.Println("=== ALL TESTS PASSED ===")
		fmt.Println("Forgejo accessible via Tailscale at http://sovereign-forge:3000")
		return nil
	}
	return fmt.Errorf("some tests failed - see above")
}

// RemoveTailscaleRegistrations removes existing sovereign-forge Tailscale registrations
// TEAM_025: Modeled after sql/verify.go RemoveTailscaleRegistrations
func RemoveTailscaleRegistrations() error {
	fmt.Println("Checking for existing Tailscale registrations...")

	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		fmt.Println("  ⚠ Cannot check Tailscale (CLI not available)")
		return nil
	}

	// Parse JSON to find sovereign-forge machines with their node IDs
	var status struct {
		Peer map[string]struct {
			HostName string `json:"HostName"`
			ID       string `json:"ID"`
			Online   bool   `json:"Online"`
		} `json:"Peer"`
	}

	if err := json.Unmarshal(out, &status); err != nil {
		fmt.Printf("  ⚠ Cannot parse Tailscale status: %v\n", err)
		return nil
	}

	// Find sovereign-forge machines and collect their IDs
	var toDelete []struct {
		ID   string
		Name string
	}
	for _, peer := range status.Peer {
		if strings.HasPrefix(peer.HostName, "sovereign-forge") {
			toDelete = append(toDelete, struct {
				ID   string
				Name string
			}{ID: peer.ID, Name: peer.HostName})
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("  ✓ No existing sovereign-forge registrations found")
		return nil
	}

	fmt.Printf("  Found %d sovereign-forge registration(s) to delete\n", len(toDelete))

	// Delete using Tailscale API (requires TAILSCALE_API_KEY env var)
	apiKey := os.Getenv("TAILSCALE_API_KEY")
	if apiKey == "" {
		// Try to read from .env file
		envPaths := []string{
			".env",
			"sovereign/.env",
			"../sovereign/.env",
			os.Getenv("HOME") + "/Projects/android/kernel/sovereign/.env",
		}
		for _, envPath := range envPaths {
			if envData, err := os.ReadFile(envPath); err == nil {
				for _, line := range strings.Split(string(envData), "\n") {
					if strings.HasPrefix(line, "TAILSCALE_API_KEY=") {
						apiKey = strings.TrimPrefix(line, "TAILSCALE_API_KEY=")
						apiKey = strings.Trim(apiKey, "\"'")
						fmt.Printf("  Found API key in %s\n", envPath)
						break
					}
				}
				if apiKey != "" {
					break
				}
			}
		}
	}

	if apiKey == "" {
		fmt.Println("  ⚠ TAILSCALE_API_KEY not set - cannot auto-delete")
		fmt.Println("  Please delete manually at: https://login.tailscale.com/admin/machines")
		for _, d := range toDelete {
			fmt.Printf("    - %s (ID: %s)\n", d.Name, d.ID)
		}
		return fmt.Errorf("found %d existing registrations - delete manually or set TAILSCALE_API_KEY", len(toDelete))
	}

	// Delete each machine via API
	client := &http.Client{Timeout: 10 * time.Second}
	var deleted int
	for _, d := range toDelete {
		url := fmt.Sprintf("https://api.tailscale.com/api/v2/device/%s", d.ID)
		req, _ := http.NewRequest("DELETE", url, nil)
		req.SetBasicAuth(apiKey, "")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  ⚠ Failed to delete %s: %v\n", d.Name, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == 200 || resp.StatusCode == 204 {
			fmt.Printf("  ✓ Deleted %s\n", d.Name)
			deleted++
		} else {
			fmt.Printf("  ⚠ Failed to delete %s: HTTP %d\n", d.Name, resp.StatusCode)
		}
	}

	if deleted == len(toDelete) {
		fmt.Printf("  ✓ Successfully deleted all %d registration(s)\n", deleted)
	} else {
		fmt.Printf("  ⚠ Deleted %d of %d registration(s)\n", deleted, len(toDelete))
	}

	return nil
}
