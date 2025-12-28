// Package docker provides Docker image export utilities
// TEAM_010: Extracted from main.go during CLI refactor
package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ExportImage exports a Docker image to an ext4 rootfs image
func ExportImage(imageName, outputPath, size string) error {
	// Create container from image
	out, err := exec.Command("docker", "create", imageName).Output()
	if err != nil {
		return fmt.Errorf("docker create failed: %w", err)
	}
	containerID := strings.TrimSpace(string(out))
	defer exec.Command("docker", "rm", containerID).Run()

	// Export to tarball
	tarPath := strings.TrimSuffix(outputPath, ".img") + ".tar"
	tarFile, err := os.Create(tarPath)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %w", err)
	}

	cmd := exec.Command("docker", "export", containerID)
	cmd.Stdout = tarFile
	if err := cmd.Run(); err != nil {
		tarFile.Close()
		os.Remove(tarPath)
		return fmt.Errorf("docker export failed: %w", err)
	}
	tarFile.Close()

	// Create ext4 image
	if err := exec.Command("truncate", "-s", size, outputPath).Run(); err != nil {
		return fmt.Errorf("truncate failed: %w", err)
	}
	if err := exec.Command("mkfs.ext4", "-F", outputPath).Run(); err != nil {
		return fmt.Errorf("mkfs.ext4 failed: %w", err)
	}

	// Mount and extract (requires sudo)
	mountDir := "/tmp/sovereign-docker-mount"
	os.MkdirAll(mountDir, 0755)

	if err := exec.Command("sudo", "mount", outputPath, mountDir).Run(); err != nil {
		return fmt.Errorf("mount failed (need sudo): %w", err)
	}

	cmd = exec.Command("sudo", "tar", "-xf", tarPath, "-C", mountDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		exec.Command("sudo", "umount", mountDir).Run()
		return fmt.Errorf("tar extract failed: %w", err)
	}

	if err := exec.Command("sudo", "umount", mountDir).Run(); err != nil {
		return fmt.Errorf("umount failed: %w", err)
	}

	os.Remove(tarPath)
	os.Remove(mountDir)

	fmt.Printf("  ✓ Exported to %s\n", outputPath)
	return nil
}

// IsAvailable checks if Docker is available
func IsAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}
