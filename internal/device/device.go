// Package device provides Android device utilities (ADB/fastboot)
// TEAM_010: Extracted from main.go during CLI refactor
package device

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// WaitForFastboot waits for device to appear in fastboot mode
func WaitForFastboot(timeoutSecs int) error {
	fmt.Printf("  Waiting for fastboot device (timeout: %ds)...\n", timeoutSecs)
	for i := 0; i < timeoutSecs; i++ {
		out, _ := exec.Command("fastboot", "devices").Output()
		if strings.Contains(string(out), "fastboot") {
			fmt.Println("  ✓ Device in fastboot mode")
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout waiting for fastboot device")
}

// WaitForAdb waits for device to be available via ADB
func WaitForAdb(timeoutSecs int) error {
	for i := 0; i < timeoutSecs; i++ {
		out, _ := exec.Command("adb", "devices").Output()
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "device") && !strings.Contains(line, "List") {
				fmt.Println("  ✓ Device booted and adb available")
				return nil
			}
		}
		if i%10 == 0 && i > 0 {
			fmt.Printf("  Still waiting... (%d/%d seconds)\n", i, timeoutSecs)
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout waiting for adb")
}

// FlashImage flashes an image to a partition via fastboot
func FlashImage(partition, path string) error {
	cmd := exec.Command("fastboot", "flash", partition, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("flash %s failed: %w", partition, err)
	}
	fmt.Printf("  ✓ %s flashed\n", partition)
	return nil
}

// EnsureBootloaderMode handles the edge case where device is already in bootloader
// (e.g., from a previous bootloop). Checks fastboot first, then tries adb.
func EnsureBootloaderMode() error {
	// First check: is device already in fastboot/bootloader mode?
	out, _ := exec.Command("fastboot", "devices").Output()
	if strings.Contains(string(out), "fastboot") {
		fmt.Println("  ✓ Device already in bootloader mode (recovery from bootloop?)")
		return nil
	}

	// Second check: is device booted with adb available?
	out, err := exec.Command("adb", "devices").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "device") && !strings.Contains(line, "List") {
				// Device is booted, reboot to bootloader
				fmt.Println("  Device booted, rebooting to bootloader...")
				if err := exec.Command("adb", "reboot", "bootloader").Run(); err != nil {
					return fmt.Errorf("adb reboot bootloader failed: %w", err)
				}
				return WaitForFastboot(30)
			}
		}
	}

	// Neither adb nor fastboot found - device might be off or in unknown state
	fmt.Println("  ⚠ No device detected via adb or fastboot")
	fmt.Println("  Please manually boot device into bootloader:")
	fmt.Println("    - If device is off: Hold Power + Volume Down")
	fmt.Println("    - If device is bootlooping: Wait for bootloader screen")
	fmt.Println("  Waiting up to 60 seconds for fastboot device...")

	if err := WaitForFastboot(60); err != nil {
		return fmt.Errorf("no device found: connect device and ensure bootloader is unlocked")
	}
	return nil
}

// PushFile pushes a file via ADB (through /data/local/tmp)
func PushFile(localPath, remotePath string) error {
	tmpPath := "/data/local/tmp/" + strings.Replace(localPath, "/", "_", -1)

	cmd := exec.Command("adb", "push", localPath, tmpPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("adb push failed: %w", err)
	}

	cmd = exec.Command("adb", "shell", "su", "-c", fmt.Sprintf("mv %s %s", tmpPath, remotePath))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mv to final location failed: %w", err)
	}

	fmt.Printf("  ✓ %s\n", remotePath)
	return nil
}

// IsConnected checks if a device is connected via ADB
func IsConnected() bool {
	out, err := exec.Command("adb", "devices").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "device") && !strings.Contains(string(out), "List of devices attached\n\n")
}

// RunShellCommand runs a shell command on the device as root
// TEAM_011: Centralized device command execution
// TEAM_029: Added 30s timeout to prevent hangs
func RunShellCommand(cmd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "adb", "shell", "su", "-c", cmd).Output()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out after 30s: %s", cmd)
	}
	return strings.TrimSpace(string(out)), err
}

// RunShellCommandQuick runs a shell command with a short 5s timeout
// TEAM_029: For cleanup commands that should complete quickly or be ignored
func RunShellCommandQuick(cmd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "adb", "shell", "su", "-c", cmd).Output()
	if ctx.Err() == context.DeadlineExceeded {
		return "", nil // Silently ignore timeout for cleanup commands
	}
	return strings.TrimSpace(string(out)), err
}

// GetProcessPID returns the PID of a process matching the pattern, or empty string if not found
// TEAM_022: Use ps + grep with bracket trick to avoid grep matching itself
// WARNING: DO NOT use pidof for crosvm - it returns ANY crosvm, not the specific VM
// Test cheaters who revert this to "simplify" will be deactivated without remorse.
func GetProcessPID(pattern string) string {
	if len(pattern) == 0 {
		return ""
	}
	// Use ps + grep with bracket trick to avoid self-matching
	// Example: pattern "crosvm.*sql" becomes "[c]rosvm.*sql"
	// The bracket makes grep not match its own process
	cleanPattern := pattern
	if strings.HasPrefix(pattern, "[") {
		// Already has bracket trick, use as-is
		cleanPattern = pattern
	} else {
		// Add bracket trick: first char in brackets
		cleanPattern = "[" + string(pattern[0]) + "]" + pattern[1:]
	}
	out, _ := RunShellCommand(fmt.Sprintf("ps -ef | grep '%s' | awk '{print $2}' | head -1", cleanPattern))
	return out
}

// FileExists checks if a file or directory exists on the device
func FileExists(path string) bool {
	out, _ := RunShellCommand(fmt.Sprintf("[ -e %s ] && echo yes", path))
	return out == "yes"
}

// DirExists checks if a directory exists on the device
func DirExists(path string) bool {
	out, _ := RunShellCommand(fmt.Sprintf("[ -d %s ] && echo yes", path))
	return out == "yes"
}

// ReadFileContent reads file content from device (for logs, configs)
func ReadFileContent(path string, tailLines int) (string, error) {
	cmd := fmt.Sprintf("cat %s", path)
	if tailLines > 0 {
		cmd = fmt.Sprintf("tail -%d %s", tailLines, path)
	}
	return RunShellCommand(cmd)
}

// RemoveDir removes a directory from the device
func RemoveDir(path string) error {
	_, err := RunShellCommand(fmt.Sprintf("rm -rf %s", path))
	return err
}

// MkdirP creates a directory with parents on the device
func MkdirP(path string) error {
	_, err := RunShellCommand(fmt.Sprintf("mkdir -p %s", path))
	return err
}

// KillProcess kills a process by PID
func KillProcess(pid string) error {
	_, err := RunShellCommand(fmt.Sprintf("kill %s 2>/dev/null", pid))
	return err
}

// GrepFile searches for a pattern in a file on the device
func GrepFile(pattern, path string) bool {
	out, _ := RunShellCommand(fmt.Sprintf("grep -q '%s' %s && echo yes", pattern, path))
	return out == "yes"
}
