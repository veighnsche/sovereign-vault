// Package kernel provides kernel build/deploy/test operations
// TEAM_010: Extracted from main.go during CLI refactor
package kernel

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/anthropics/sovereign/internal/device"
)

// getKernelRoot returns the absolute path to the kernel root directory
// TEAM_012: Extracted to helper function for reuse across commands
func getKernelRoot() string {
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to CWD
		cwd, _ := os.Getwd()
		if strings.HasSuffix(cwd, "/sovereign") {
			return strings.TrimSuffix(cwd, "/sovereign")
		}
		return cwd + "/.."
	}

	// Go up from sovereign binary to kernel root
	kernelRoot := strings.TrimSuffix(exePath, "/sovereign")
	kernelRoot = strings.TrimSuffix(kernelRoot, "/cmd/sovereign")
	kernelRoot = strings.TrimSuffix(kernelRoot, "/sovereign")

	// If running via go run, use working directory heuristic
	if strings.Contains(exePath, "go-build") {
		cwd, _ := os.Getwd()
		if strings.HasSuffix(cwd, "/sovereign") {
			return strings.TrimSuffix(cwd, "/sovereign")
		}
		return cwd + "/.."
	}

	return kernelRoot
}

// Build builds the kernel with KernelSU
func Build() error {
	fmt.Println("=== Building Kernel with KernelSU ===")

	// TEAM_012: Use helper function for absolute paths
	kernelRoot := getKernelRoot()
	buildScript := kernelRoot + "/build_raviole.sh"
	fmt.Printf("Running: %s\n", buildScript)

	if _, err := os.Stat(buildScript); os.IsNotExist(err) {
		return fmt.Errorf("build script not found: %s\n"+
			"  Make sure you're in the sovereign/ directory or the script exists", buildScript)
	}

	cmd := exec.Command(buildScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = kernelRoot

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Verify CONFIG_KSU=y in output
	configPath := kernelRoot + "/out/raviole/dist/.config"
	if _, err := os.Stat(configPath); err == nil {
		configData, _ := os.ReadFile(configPath)
		if !strings.Contains(string(configData), "CONFIG_KSU=y") {
			return fmt.Errorf("CONFIG_KSU=y not found in %s - KernelSU not enabled!", configPath)
		}
		fmt.Println("✓ CONFIG_KSU=y confirmed")
	}

	fmt.Println("\n✓ Kernel build complete")
	fmt.Printf("Output: %s/out/raviole/dist/boot.img\n", kernelRoot)
	return nil
}

// Deploy deploys the kernel to a Pixel 6 device
func Deploy() error {
	fmt.Println("=== Deploying Kernel (Pixel 6 / Raviole) ===")
	// TEAM_004: Follow official Google flash sequence from:
	// https://source.android.com/docs/setup/build/building-pixel-kernels

	// TEAM_012: Use absolute paths
	kernelRoot := getKernelRoot()
	distDir := kernelRoot + "/out/raviole/dist"

	// Verify all required images exist
	requiredImages := []string{"boot.img", "dtbo.img", "dtb.img", "initramfs.img", "vendor_dlkm.img"}
	fmt.Println("Checking build artifacts...")
	for _, img := range requiredImages {
		path := distDir + "/" + img
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("%s not found: %s\nRun 'sovereign build' first", img, path)
		}
		fmt.Printf("  ✓ %s\n", img)
	}

	// Step 1: Get device into bootloader mode
	fmt.Println("\n[Step 1/6] Getting device into bootloader mode...")
	if err := device.EnsureBootloaderMode(); err != nil {
		return err
	}

	// Step 2: Flash boot.img
	fmt.Println("\n[Step 2/6] Flashing boot.img...")
	if err := device.FlashImage("boot", distDir+"/boot.img"); err != nil {
		return err
	}

	// Step 3: Flash dtbo.img
	fmt.Println("\n[Step 3/6] Flashing dtbo.img...")
	if err := device.FlashImage("dtbo", distDir+"/dtbo.img"); err != nil {
		return err
	}

	// Step 4: Flash initramfs with dtb (special command per Google docs)
	fmt.Println("\n[Step 4/6] Flashing initramfs with dtb...")
	cmd := exec.Command("fastboot", "flash", "--dtb", distDir+"/dtb.img",
		"vendor_boot:dlkm", distDir+"/initramfs.img")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("flash initramfs failed: %w", err)
	}
	fmt.Println("  ✓ initramfs flashed")

	// Step 5: Reboot to fastboot mode for vendor_dlkm
	fmt.Println("\n[Step 5/6] Rebooting to fastboot mode...")
	if err := exec.Command("fastboot", "reboot", "fastboot").Run(); err != nil {
		return fmt.Errorf("reboot to fastboot failed: %w", err)
	}

	if err := device.WaitForFastboot(30); err != nil {
		return err
	}

	// Step 6: Flash vendor_dlkm
	fmt.Println("\n[Step 6/6] Flashing vendor_dlkm.img...")
	if err := device.FlashImage("vendor_dlkm", distDir+"/vendor_dlkm.img"); err != nil {
		return err
	}

	// Final reboot
	fmt.Println("\nRebooting device...")
	if err := exec.Command("fastboot", "reboot").Run(); err != nil {
		return fmt.Errorf("final reboot failed: %w", err)
	}

	// Wait for device to boot and adb to become available
	fmt.Println("\nWaiting for device to boot (up to 120 seconds)...")
	if err := device.WaitForAdb(120); err != nil {
		return fmt.Errorf("BOOTLOOP DETECTED: Device did not boot within 120 seconds.\n" +
			"Recovery: Hold Power + Volume Down to enter bootloader, then flash stock image")
	}

	fmt.Println("\n✓ Kernel deployed successfully!")
	fmt.Println("Run 'sovereign test' to verify KernelSU")
	return nil
}

// BuildGuestKernel builds the guest VM kernel with sovereign_guest.fragment
// TEAM_011: Codified from manual commands - NO MORE AD-HOC KERNEL BUILDS
func BuildGuestKernel() error {
	fmt.Println("=== Building Guest VM Kernel ===")

	kernelRoot := getKernelRoot()
	aospDir := kernelRoot + "/aosp"
	outDir := kernelRoot + "/out/guest-kernel"
	fragmentPath := kernelRoot + "/private/devices/google/raviole/sovereign_guest.fragment"
	clangBin := kernelRoot + "/prebuilts/clang/host/linux-x86/clang-r487747c/bin"

	// Verify paths exist
	if _, err := os.Stat(aospDir); os.IsNotExist(err) {
		return fmt.Errorf("aosp directory not found: %s", aospDir)
	}
	if _, err := os.Stat(fragmentPath); os.IsNotExist(err) {
		return fmt.Errorf("sovereign_guest.fragment not found: %s", fragmentPath)
	}
	if _, err := os.Stat(clangBin); os.IsNotExist(err) {
		return fmt.Errorf("clang toolchain not found: %s", clangBin)
	}

	// Setup environment
	// TEAM_011 FIX: Use AOSP prebuilt OpenSSL/BoringSSL (has engine.h for PKCS#11)
	// This is the CORRECT fix instead of disabling security features
	opensslInclude := kernelRoot + "/prebuilts/kernel-build-tools/linux-x86/include"
	opensslLib := kernelRoot + "/prebuilts/kernel-build-tools/linux-x86/lib64"

	env := os.Environ()
	env = append(env, fmt.Sprintf("PATH=%s:%s", clangBin, os.Getenv("PATH")))
	env = append(env, fmt.Sprintf("HOSTCFLAGS=-I%s", opensslInclude))
	env = append(env, fmt.Sprintf("HOSTLDFLAGS=-L%s -Wl,-rpath,%s", opensslLib, opensslLib))
	env = append(env, fmt.Sprintf("LD_LIBRARY_PATH=%s", opensslLib))

	// Create output directory
	os.MkdirAll(outDir, 0755)

	// Step 1: Generate base config with olddefconfig (non-interactive!)
	fmt.Println("[1/4] Generating base config (olddefconfig)...")
	defconfigCmd := exec.Command("make", "O="+outDir, "ARCH=arm64", "LLVM=1", "olddefconfig")
	defconfigCmd.Dir = aospDir
	defconfigCmd.Env = env
	defconfigCmd.Stdout = os.Stdout
	defconfigCmd.Stderr = os.Stderr

	// First run defconfig to create initial .config if it doesn't exist
	if _, err := os.Stat(outDir + "/.config"); os.IsNotExist(err) {
		fmt.Println("  No existing config, running defconfig first...")
		initCmd := exec.Command("make", "O="+outDir, "ARCH=arm64", "LLVM=1", "defconfig")
		initCmd.Dir = aospDir
		initCmd.Env = env
		// Pipe yes to answer all prompts with default
		initCmd.Stdin = strings.NewReader(strings.Repeat("\\n", 1000))
		if err := initCmd.Run(); err != nil {
			return fmt.Errorf("initial defconfig failed: %w", err)
		}
	}

	if err := defconfigCmd.Run(); err != nil {
		return fmt.Errorf("olddefconfig failed: %w", err)
	}

	// Step 2: Merge sovereign_guest.fragment
	fmt.Println("[2/4] Merging sovereign_guest.fragment...")
	mergeScript := aospDir + "/scripts/kconfig/merge_config.sh"
	mergeCmd := exec.Command(mergeScript, "-O", outDir, outDir+"/.config", fragmentPath)
	mergeCmd.Dir = aospDir
	mergeCmd.Env = env
	mergeCmd.Stdout = os.Stdout
	mergeCmd.Stderr = os.Stderr
	if err := mergeCmd.Run(); err != nil {
		return fmt.Errorf("config merge failed: %w", err)
	}

	// Verify critical configs are set
	fmt.Println("[3/4] Verifying critical configs...")
	configData, err := os.ReadFile(outDir + "/.config")
	if err != nil {
		return fmt.Errorf("cannot read config: %w", err)
	}
	criticalConfigs := []string{"CONFIG_TUN=y", "CONFIG_SYSVIPC=y", "CONFIG_NETFILTER=y", "CONFIG_VIRTIO_VSOCKETS=y"}
	for _, cfg := range criticalConfigs {
		if !strings.Contains(string(configData), cfg) {
			return fmt.Errorf("CRITICAL: %s not found in guest kernel config!", cfg)
		}
		fmt.Printf("  ✓ %s\n", cfg)
	}

	// Step 3: Build the kernel Image
	fmt.Println("[4/4] Building kernel Image (this takes a few minutes)...")
	nproc, _ := exec.Command("nproc").Output()
	jobs := strings.TrimSpace(string(nproc))
	if jobs == "" {
		jobs = "4"
	}

	buildCmd := exec.Command("make", "O="+outDir, "ARCH=arm64", "LLVM=1", "-j"+jobs, "Image")
	buildCmd.Dir = aospDir
	buildCmd.Env = env
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("kernel build failed: %w", err)
	}

	// Copy to vm/sql/Image
	srcImage := outDir + "/arch/arm64/boot/Image"
	dstImage := kernelRoot + "/sovereign/vm/sql/Image"

	fmt.Printf("Copying kernel to %s...\n", dstImage)
	cpCmd := exec.Command("cp", srcImage, dstImage)
	if err := cpCmd.Run(); err != nil {
		return fmt.Errorf("failed to copy kernel: %w", err)
	}

	fmt.Println("\n✓ Guest kernel built successfully")
	fmt.Printf("  Output: %s\n", dstImage)
	fmt.Println("\nNext: sovereign build --sql")
	return nil
}

// Test tests KernelSU installation
func Test() error {
	fmt.Println("=== Testing Kernel/KernelSU ===")
	allPassed := true

	// Test 1: Kernel version
	fmt.Print("1. Kernel version contains 'sovereign': ")
	out, err := exec.Command("adb", "shell", "cat", "/proc/version").Output()
	if err != nil {
		fmt.Println("✗ FAIL (cannot read)")
		allPassed = false
	} else if !strings.Contains(string(out), "sovereign") {
		fmt.Println("✗ FAIL")
		fmt.Printf("   Got: %s\n", strings.TrimSpace(string(out)))
		allPassed = false
	} else {
		fmt.Println("✓ PASS")
	}

	// Test 2: Root access
	fmt.Print("2. Root access via su: ")
	out, err = exec.Command("adb", "shell", "su", "-c", "id").Output()
	if err != nil {
		fmt.Println("✗ FAIL (su not working)")
		allPassed = false
	} else if !strings.Contains(string(out), "uid=0") {
		fmt.Println("✗ FAIL (not root)")
		fmt.Printf("   Got: %s\n", strings.TrimSpace(string(out)))
		allPassed = false
	} else {
		fmt.Println("✓ PASS")
	}

	// Test 3: KernelSU version
	fmt.Print("3. KernelSU version (not 16): ")
	out, err = exec.Command("adb", "shell", "su", "-v").Output()
	if err != nil {
		fmt.Println("✗ FAIL (cannot get version)")
		allPassed = false
	} else {
		version := strings.TrimSpace(string(out))
		if version == "16" || version == "" {
			fmt.Println("✗ FAIL (version is 16 - Kbuild patch not applied)")
			allPassed = false
		} else {
			fmt.Printf("✓ PASS (version: %s)\n", version)
		}
	}

	fmt.Println()
	if allPassed {
		fmt.Println("=== ALL TESTS PASSED ===")
		fmt.Println("Root access working.")
		return nil
	}
	return fmt.Errorf("some tests failed - see above")
}
