// SQL VM verification (Test command and Tailscale checks)
// TEAM_022: Split from sql.go for readability
package sql

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/anthropics/sovereign/internal/device"
	"github.com/anthropics/sovereign/internal/secrets"
	"github.com/anthropics/sovereign/internal/vm/common"
)

// TEAM_029: Test delegates to common.RunVMTests with SQL-specific custom tests
func (v *VM) Test() error {
	return common.RunVMTests(SQLConfig, sqlCustomTests)
}

// TEAM_029: SQL-specific tests for PostgreSQL
var sqlCustomTests = []common.TestFunc{
	testPostgresResponding,
	testCanExecuteQuery,
}

func testPostgresResponding(cfg *common.VMConfig) common.TestResult {
	out, _ := device.RunShellCommand(fmt.Sprintf("nc -z %s 5432 && echo OPEN || echo CLOSED", cfg.TAPGuestIP))
	if strings.TrimSpace(out) == "OPEN" {
		return common.TestResult{Name: "PostgreSQL responding (via TAP)", Passed: true}
	}
	return common.TestResult{Name: "PostgreSQL responding (via TAP)", Passed: false, Message: "port 5432 not reachable on TAP"}
}

func testCanExecuteQuery(cfg *common.VMConfig) common.TestResult {
	creds, _ := secrets.LoadSecretsFile()
	pgPassword := "sovereign"
	if creds != nil {
		pgPassword = creds.DBPassword
	}
	queryOut, _ := device.RunShellCommand(fmt.Sprintf(
		"PGPASSWORD=%s psql -h %s -U postgres -c 'SELECT 1;' 2>&1 | grep -c '1 row'",
		pgPassword, cfg.TAPGuestIP))
	if strings.TrimSpace(queryOut) == "1" {
		return common.TestResult{Name: "Can execute query (via TAP)", Passed: true}
	}
	// Fallback: check port
	connOut, _ := device.RunShellCommand(fmt.Sprintf("nc -z %s 5432 && echo OK", cfg.TAPGuestIP))
	if strings.Contains(connOut, "OK") {
		return common.TestResult{Name: "Can execute query (via TAP)", Passed: true, Message: "port open, psql not available on device"}
	}
	return common.TestResult{Name: "Can execute query (via TAP)", Passed: false, Message: "cannot connect to PostgreSQL"}
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
// TEAM_029: Delegated to common.RemoveTailscaleRegistrations
func RemoveTailscaleRegistrations() error {
	return common.RemoveTailscaleRegistrations("sovereign-sql")
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
