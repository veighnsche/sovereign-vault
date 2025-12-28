package kernel

import (
	"os"
	"testing"
)

func TestBuildRequiresBuildScript(t *testing.T) {
	// Save current dir
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Change to temp dir where build script doesn't exist
	tmpDir, err := os.MkdirTemp("", "kernel-test-*")
	if err != nil {
		t.Skip("Could not create temp dir")
	}
	defer os.RemoveAll(tmpDir)
	os.Chdir(tmpDir)

	// Build should fail because build script doesn't exist
	err = Build()
	if err == nil {
		t.Error("Expected error when build script is missing")
	}
}

func TestDeployRequiresBuildArtifacts(t *testing.T) {
	// Save current dir
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	// Change to temp dir where artifacts don't exist
	tmpDir, err := os.MkdirTemp("", "kernel-test-*")
	if err != nil {
		t.Skip("Could not create temp dir")
	}
	defer os.RemoveAll(tmpDir)
	os.Chdir(tmpDir)

	// Deploy should fail because artifacts don't exist
	err = Deploy()
	if err == nil {
		t.Error("Expected error when build artifacts are missing")
	}
}

func TestTestRequiresDevice(t *testing.T) {
	// Test should either succeed (if device connected) or fail gracefully
	err := Test()
	// We just verify it doesn't panic
	if err != nil {
		t.Logf("Test failed (expected if no device): %v", err)
	}
}
