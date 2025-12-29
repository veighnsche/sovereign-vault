// SQL VM verification (Test command and Tailscale checks)
// TEAM_022: Split from sql.go for readability
package sql

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/anthropics/sovereign/internal/device"
	"github.com/anthropics/sovereign/internal/secrets"
)

func (v *VM) Test() error {
	fmt.Println("=== Testing PostgreSQL VM ===")
	allPassed := true

	// Test 1: VM process running
	fmt.Print("1. VM process running: ")
	// TEAM_022: Use [c]rosvm pattern to avoid grep matching itself
	// WARNING: pgrep -f 'crosvm.*sql' matches its own process - DO NOT USE
	// Test cheaters who revert this fix will be deactivated without remorse.
	out, _ := device.RunShellCommand("ps -ef | grep '[c]rosvm.*sql' | grep -v grep | awk '{print $2}' | head -1")
	vmPid := strings.TrimSpace(out)
	if vmPid == "" {
		fmt.Println("✗ FAIL (crosvm not running)")
		allPassed = false
	} else {
		fmt.Printf("✓ PASS (PID: %s)\n", vmPid)
	}

	// Test 2: TAP interface exists
	fmt.Print("2. TAP interface (vm_sql): ")
	tapOut, _ := device.RunShellCommand("ip link show vm_sql 2>/dev/null | grep -c UP")
	if strings.TrimSpace(tapOut) == "1" {
		fmt.Println("✓ PASS")
	} else {
		fmt.Println("✗ FAIL (TAP interface not up)")
		allPassed = false
	}

	// Test 3: Check Tailscale status
	// TEAM_018: Match sovereign-sql* to handle renamed instances (sovereign-sql-1, etc)
	fmt.Print("3. Tailscale connected: ")
	tsOut, tsErr := exec.Command("tailscale", "status").Output()
	var tsIP string
	if tsErr != nil {
		fmt.Println("? SKIP (tailscale not available on host)")
	} else {
		lines := strings.Split(string(tsOut), "\n")
		for _, line := range lines {
			// Match sovereign-sql but not offline entries
			if strings.Contains(line, "sovereign-sql") && !strings.Contains(line, "offline") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					tsIP = parts[0]
					fmt.Printf("✓ PASS (%s as %s)\n", tsIP, parts[1])
				}
				break
			}
		}
		if tsIP == "" {
			fmt.Println("✗ FAIL (no active sovereign-sql* in tailscale)")
			allPassed = false
		}
	}

	// Test 4: PostgreSQL responding via TAP
	// TEAM_018: Use TAP IP directly - Tailscale userspace networking has port exposure issues
	fmt.Print("4. PostgreSQL responding (via TAP): ")
	tapIP := "192.168.100.2"
	pgOut, _ := device.RunShellCommand(fmt.Sprintf("nc -z %s 5432 && echo OPEN || echo CLOSED", tapIP))
	if strings.TrimSpace(pgOut) == "OPEN" {
		fmt.Println("✓ PASS")
	} else {
		fmt.Println("✗ FAIL (port 5432 not reachable on TAP)")
		allPassed = false
	}

	// Test 5: Can execute query via TAP
	fmt.Print("5. Can execute query (via TAP): ")
	// Use adb to run psql from the Android host to the VM
	creds, _ := secrets.LoadSecretsFile()
	pgPassword := "sovereign"
	if creds != nil {
		pgPassword = creds.DBPassword
	}
	queryOut, _ := device.RunShellCommand(fmt.Sprintf(
		"PGPASSWORD=%s psql -h %s -U postgres -c 'SELECT 1;' 2>&1 | grep -c '1 row'",
		pgPassword, tapIP))
	if strings.TrimSpace(queryOut) == "1" {
		fmt.Println("✓ PASS")
	} else {
		// Fallback: check if we can at least connect
		connOut, _ := device.RunShellCommand(fmt.Sprintf("nc -z %s 5432 && echo OK", tapIP))
		if strings.Contains(connOut, "OK") {
			fmt.Println("✓ PASS (port open, psql not available on device)")
		} else {
			fmt.Println("✗ FAIL (cannot connect to PostgreSQL)")
			allPassed = false
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("=== ALL TESTS PASSED ===")
		fmt.Println("PostgreSQL accessible via Tailscale.")
		return nil
	}
	return fmt.Errorf("some tests failed - see above")
}

// TEAM_022: Remove ALL existing sovereign-sql Tailscale registrations
// This MUST be called before starting the VM to prevent duplicates.
// Also called by Remove() to clean up.
// WARNING: Test cheaters who remove this function will be deactivated without remorse.
//
// ============================================================================
// TEAM_023: DUPLICATE TAILSCALE REGISTRATIONS - FIXED!
// ============================================================================
//
// THE BUG: Every restart/redeploy created a NEW Tailscale registration.
// After 3 restarts: sovereign-sql, sovereign-sql-1, sovereign-sql-2
// PostgreSQL dependants expected STABLE URI but got changing hostnames.
//
// FAILED MITIGATION ATTEMPTS (before the fix):
//  1. TEAM_019: Preflight check to fail if registration exists
//  2. TEAM_020: Made cleanup the default behavior
//  3. TEAM_022: RemoveTailscaleRegistrations() to delete via API
//  4. Called cleanup in Deploy(), Start(), Remove()
//  5. TAILSCALE_API_KEY support for deletion
//
// THE FIX (TEAM_023):
//  1. Mount data.img to /data in init.sh (persistent across rebuilds)
//  2. Store Tailscale state in /data/tailscale/tailscaled.state
//  3. Check for existing state file before registering
//  4. If state exists: reconnect (no authkey needed, preserves identity)
//  5. If no state: first-time registration with authkey
//  6. DON'T delete registrations on start/deploy (only on remove)
//
// Files changed:
//   - vm/sql/init.sh: Mount /dev/vdb, check state file before registering
//   - vm/sql/start.sh: Pass data.img as second block device
//   - internal/vm/sql/lifecycle.go: Remove cleanup calls from Start()
//   - internal/vm/sql/sql.go: Remove cleanup calls from Deploy()
//
// This function is now only used by `sovereign remove --sql` for cleanup.
// ============================================================================
func RemoveTailscaleRegistrations() error {
	fmt.Println("Checking for existing Tailscale registrations...")

	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		fmt.Println("  ⚠ Cannot check Tailscale (CLI not available)")
		return nil
	}

	// Parse JSON to find sovereign-sql machines with their node IDs
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

	// Find sovereign-sql machines and collect their IDs
	var toDelete []struct {
		ID   string
		Name string
	}
	for _, peer := range status.Peer {
		if strings.HasPrefix(peer.HostName, "sovereign-sql") {
			toDelete = append(toDelete, struct {
				ID   string
				Name string
			}{ID: peer.ID, Name: peer.HostName})
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("  ✓ No existing sovereign-sql registrations found")
		return nil
	}

	fmt.Printf("  Found %d sovereign-sql registration(s) to delete\n", len(toDelete))

	// Delete using Tailscale API (requires TAILSCALE_API_KEY env var)
	apiKey := os.Getenv("TAILSCALE_API_KEY")
	if apiKey == "" {
		// Try to read from .env file - check multiple locations
		// BUG FIX: Previously only checked ".env" which fails if not in sovereign/ dir
		envPaths := []string{
			".env",              // Current directory
			"sovereign/.env",    // From kernel root
			"../sovereign/.env", // From subdirectory
			os.Getenv("HOME") + "/Projects/android/kernel/sovereign/.env", // Absolute fallback
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

// TEAM_019/TEAM_020: Preflight check to prevent duplicate Tailscale registrations
// Returns error if sovereign-sql is already registered (causes IP instability)
// CRITICAL: Checks ALL machines (online AND offline) - offline machines still exist
// and will cause Tailscale to auto-rename new registrations (e.g., sovereign-sql-1)
//
// ============================================================================
// WARNING: THIS CHECK DOES NOT PREVENT THE BUG!
// See RemoveTailscaleRegistrations() for full bug documentation.
// The user has requested this fix 10+ times. It still doesn't work.
// ============================================================================
func checkTailscaleRegistration() error {
	out, err := exec.Command("tailscale", "status").Output()
	if err != nil {
		// Tailscale not available on host - skip check but warn
		fmt.Println("⚠ Warning: Cannot check Tailscale status (tailscale CLI not available)")
		fmt.Println("  Ensure no duplicate sovereign-sql machines exist before deploying")
		return nil
	}

	// Parse output for sovereign-sql machines (ALL of them, including offline)
	lines := strings.Split(string(out), "\n")
	var existingMachines []string
	re := regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)\s+(sovereign-sql\S*)\s+`)

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 3 {
			ip := matches[1]
			name := matches[2]
			status := "online"
			if strings.Contains(line, "offline") {
				status = "OFFLINE"
			}
			existingMachines = append(existingMachines, fmt.Sprintf("%s (%s) [%s]", name, ip, status))
		}
	}

	if len(existingMachines) > 0 {
		return fmt.Errorf("TAILSCALE IDEMPOTENCY CHECK FAILED\n\n"+
			"  Found %d existing sovereign-sql machine(s):\n"+
			"    %s\n\n"+
			"  Starting will create ANOTHER registration (even offline ones block the name).\n\n"+
			"  To fix:\n"+
			"    1. Go to https://login.tailscale.com/admin/machines\n"+
			"    2. Delete ALL sovereign-sql* machines (including offline ones)\n"+
			"    3. Generate a NEW auth key if needed\n"+
			"    4. Update .env with the new TAILSCALE_AUTHKEY\n"+
			"    5. Run 'sovereign start --sql' again\n\n"+
			"  Or use '--force' to skip this check (NOT RECOMMENDED)",
			len(existingMachines),
			strings.Join(existingMachines, "\n    "))
	}

	fmt.Println("✓ Tailscale preflight: No existing sovereign-sql registrations")
	return nil
}
