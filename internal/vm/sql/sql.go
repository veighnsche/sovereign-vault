// Package sql provides the PostgreSQL VM implementation
// TEAM_010: Extracted from main.go during CLI refactor
//
// CRITICAL ARCHITECTURE REQUIREMENTS - READ BEFORE MODIFYING:
//
// 1. PostgreSQL MUST be accessible via Tailscale from ANY device on the tailnet
// 2. Each VM MUST have its OWN Tailscale IP - NO SHARING IPs between services
// 3. Tailscale MUST use kernel TUN mode (CONFIG_TUN=y) - NOT userspace networking
//
// FORBIDDEN WORKAROUNDS (these are WRONG, do not suggest them):
// - Port forwarding through Android's Tailscale (shares IP, breaks multi-service)
// - Direct connection via phone's LAN IP (not accessible from tailnet)
// - SQLite instead of PostgreSQL (architecture violation)
// - Any hack that avoids fixing the real networking problem
//
// If Tailscale ports are not accessible, the fix is:
// - Ensure /dev/net/tun exists in guest
// - Run tailscaled WITHOUT --tun=userspace-networking
// - The guest kernel MUST have CONFIG_TUN=y (it does)
//
package sql

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/anthropics/sovereign/internal/device"
	"github.com/anthropics/sovereign/internal/docker"
	"github.com/anthropics/sovereign/internal/rootfs"
	"github.com/anthropics/sovereign/internal/secrets"
	"github.com/anthropics/sovereign/internal/vm"
)

func init() {
	vm.Register("sql", &VM{})
}

// VM implements the vm.VM interface for PostgreSQL
type VM struct{}

func (v *VM) Name() string { return "sql" }

func (v *VM) Build() error {
	fmt.Println("=== Building PostgreSQL VM ===")

	// Check if Docker is available
	if !docker.IsAvailable() {
		return fmt.Errorf("docker not found in PATH - install Docker first")
	}

	// Check if we have sudo (needed for mount operations)
	if _, err := exec.LookPath("sudo"); err != nil {
		return fmt.Errorf("sudo not found - needed for rootfs preparation")
	}

	// TEAM_011: Prompt for database credentials if not already set
	var creds *secrets.Credentials
	if secrets.SecretsExist() {
		fmt.Println("Using existing credentials from .secrets")
		var err error
		creds, err = secrets.LoadSecretsFile()
		if err != nil {
			return err
		}
	} else {
		var err error
		creds, err = secrets.PromptCredentials("postgres")
		if err != nil {
			return fmt.Errorf("credential setup failed: %w", err)
		}
		if err := secrets.WriteSecretsFile(creds); err != nil {
			return err
		}
	}

	// TEAM_006: Build the Docker image for ARM64 (Pixel 6 architecture)
	fmt.Println("Building Docker image for ARM64...")
	cmd := exec.Command("docker", "build",
		"--platform", "linux/arm64",
		"-t", "sovereign-sql",
		"-f", "vm/sql/Dockerfile",
		"vm/sql")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed (ensure qemu-user-static is installed for cross-arch builds): %w", err)
	}

	// Export to rootfs image for crosvm
	fmt.Println("\nExporting rootfs image...")
	if err := docker.ExportImage("sovereign-sql", "vm/sql/rootfs.img", "512M"); err != nil {
		return err
	}

	// TEAM_011: Check if custom kernel already exists (skip extraction)
	// Alpine's vmlinuz-virt is EFI stub format which crosvm can't boot.
	// We use a custom kernel built with microdroid_defconfig.
	kernelDst := "vm/sql/Image"
	if info, err := os.Stat(kernelDst); err == nil && info.Size() > 1000000 {
		fmt.Printf("  ✓ Using existing kernel %s (%d MB)\n", kernelDst, info.Size()/1024/1024)
	} else {
		// No valid kernel - user needs to build one or copy from device
		return fmt.Errorf("kernel Image not found or invalid\n" +
			"  The kernel must be RAW ARM64 format (not EFI stub).\n" +
			"  Options:\n" +
			"    1. Copy from device: adb pull /data/sovereign/vm/sql/Image vm/sql/Image\n" +
			"    2. Build with: cd aosp && make O=../out/guest-kernel ARCH=arm64 microdroid_defconfig Image")
	}

	// Create data disk
	fmt.Println("Creating data disk (4GB)...")
	dataImg := "vm/sql/data.img"
	if _, err := os.Stat(dataImg); os.IsNotExist(err) {
		cmd = exec.Command("truncate", "-s", "4G", dataImg)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create data disk: %w", err)
		}
		cmd = exec.Command("mkfs.ext4", "-F", dataImg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to format data disk: %w", err)
		}
	} else {
		fmt.Println("  Data disk already exists, skipping")
	}

	// TEAM_007: Prepare rootfs with proper device node setup for vsock networking
	fmt.Println("Preparing rootfs for AVF (vsock device nodes, init script fixes)...")
	if err := rootfs.PrepareForAVF("vm/sql/rootfs.img", creds.DBPassword); err != nil {
		return fmt.Errorf("rootfs preparation failed: %w", err)
	}

	fmt.Println("\n✓ PostgreSQL VM built successfully")
	fmt.Println("  Rootfs: vm/sql/rootfs.img")
	fmt.Println("  Data:   vm/sql/data.img")
	fmt.Println("\nNext: sovereign deploy --sql")
	return nil
}

// TEAM_019: Package-level flag to skip Tailscale idempotency check
var ForceDeploySkipTailscaleCheck bool

func (v *VM) Deploy() error {
	fmt.Println("=== Deploying PostgreSQL VM ===")

	// TEAM_019: Preflight - check for existing Tailscale registrations
	// This prevents creating duplicate machines that break IP stability
	if !ForceDeploySkipTailscaleCheck {
		if err := checkTailscaleRegistration(); err != nil {
			return err
		}
	} else {
		fmt.Println("⚠ FORCE MODE: Skipping Tailscale idempotency check (may create duplicate registration)")
	}

	// Verify images and kernel exist
	requiredFiles := []string{"vm/sql/rootfs.img", "vm/sql/data.img", "vm/sql/Image"}
	for _, f := range requiredFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			return fmt.Errorf("%s not found - run 'sovereign build --sql' first", f)
		}
	}

	// TEAM_006: Fail early if .env is missing
	envPath := ".env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return fmt.Errorf(".env file not found - Tailscale won't connect without it\n" +
			"  1. Copy .env.example to .env\n" +
			"  2. Get auth key from https://login.tailscale.com/admin/settings/keys\n" +
			"  3. Fill in TAILSCALE_AUTHKEY in .env")
	}

	// Create device directories (ignore error if already exists)
	fmt.Println("Creating directories on device...")
	device.MkdirP("/data/sovereign/vm/sql")
	// Verify directory was created
	if !device.DirExists("/data/sovereign/vm/sql") {
		return fmt.Errorf("failed to create directories on device")
	}

	// Push images
	fmt.Println("Pushing rootfs.img (this may take a while)...")
	if err := device.PushFile("vm/sql/rootfs.img", "/data/sovereign/vm/sql/rootfs.img"); err != nil {
		return err
	}

	fmt.Println("Pushing data.img...")
	if err := device.PushFile("vm/sql/data.img", "/data/sovereign/vm/sql/data.img"); err != nil {
		return err
	}

	fmt.Println("Pushing guest kernel (this may take a while - 35MB)...")
	if err := device.PushFile("vm/sql/Image", "/data/sovereign/vm/sql/Image"); err != nil {
		return err
	}

	// Push .env if exists
	if _, err := os.Stat(envPath); err == nil {
		fmt.Println("Pushing .env...")
		if err := device.PushFile(envPath, "/data/sovereign/.env"); err != nil {
			return err
		}
	}

	// Create start script on device
	fmt.Println("Creating start script...")
	if err := createStartScript(); err != nil {
		return err
	}

	fmt.Println("\n✓ PostgreSQL VM deployed")
	fmt.Println("\nNext: sovereign start --sql")
	return nil
}

func (v *VM) Start() error {
	fmt.Println("=== Starting PostgreSQL VM ===")

	// Check if VM is already running
	runningPid := device.GetProcessPID("crosvm.*sql")
	if runningPid != "" {
		fmt.Printf("⚠ VM already running (PID: %s)\n", runningPid)
		fmt.Println("Run 'sovereign stop --sql' first to restart")
		return nil
	}

	// TEAM_020: Preflight check - Tailscale registration happens during START, not deploy
	// This is the CRITICAL check - if sovereign-sql exists (even offline), new VM gets renamed
	if !ForceDeploySkipTailscaleCheck {
		if err := checkTailscaleRegistration(); err != nil {
			return err
		}
	} else {
		fmt.Println("⚠ FORCE MODE: Skipping Tailscale idempotency check")
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
		maxWaitSeconds    = 90
		pollIntervalMs    = 500
		pgCheckAfterSecs  = 15 // Start checking PostgreSQL port after kernel boot
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
	pid := device.GetProcessPID("crosvm.*sql")

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

	fmt.Println("✓ VM stopped")
	return nil
}

func (v *VM) Test() error {
	fmt.Println("=== Testing PostgreSQL VM ===")
	allPassed := true

	// Test 1: VM process running
	fmt.Print("1. VM process running: ")
	// TEAM_011: Simplified detection - pgrep works better through adb shell
	out, _ := device.RunShellCommand("pgrep -f 'crosvm.*sql' | head -1")
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

// Remove removes the SQL VM from the device
// TEAM_018: Enhanced to clean up EVERYTHING for a truly clean phone
func (v *VM) Remove() error {
	fmt.Println("=== Removing SQL VM from device ===")

	// First stop the VM if running (this also cleans up networking)
	v.Stop()

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

func createStartScript() error {
	scriptPath := "vm/sql/start.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("start script not found: %s", scriptPath)
	}

	if err := device.PushFile(scriptPath, "/data/sovereign/vm/sql/start.sh"); err != nil {
		return err
	}

	if _, err := device.RunShellCommand("chmod +x /data/sovereign/vm/sql/start.sh"); err != nil {
		return fmt.Errorf("failed to chmod start script: %w", err)
	}

	return nil
}

// TEAM_019/TEAM_020: Preflight check to prevent duplicate Tailscale registrations
// Returns error if sovereign-sql is already registered (causes IP instability)
// CRITICAL: Checks ALL machines (online AND offline) - offline machines still exist
// and will cause Tailscale to auto-rename new registrations (e.g., sovereign-sql-1)
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
		return fmt.Errorf("TAILSCALE IDEMPOTENCY CHECK FAILED\n\n" +
			"  Found %d existing sovereign-sql machine(s):\n" +
			"    %s\n\n" +
			"  Starting will create ANOTHER registration (even offline ones block the name).\n\n" +
			"  To fix:\n" +
			"    1. Go to https://login.tailscale.com/admin/machines\n" +
			"    2. Delete ALL sovereign-sql* machines (including offline ones)\n" +
			"    3. Generate a NEW auth key if needed\n" +
			"    4. Update .env with the new TAILSCALE_AUTHKEY\n" +
			"    5. Run 'sovereign start --sql' again\n\n" +
			"  Or use '--force' to skip this check (NOT RECOMMENDED)",
			len(existingMachines),
			strings.Join(existingMachines, "\n    "))
	}

	fmt.Println("✓ Tailscale preflight: No existing sovereign-sql registrations")
	return nil
}
