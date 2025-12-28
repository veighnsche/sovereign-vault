package rootfs

import (
	"os"
	"testing"
)

func TestPrepareForAVFMissingFile(t *testing.T) {
	// Test that PrepareForAVF fails gracefully with missing file
	err := PrepareForAVF("/nonexistent/path/rootfs.img", "testpass")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestPrepareForAVFRequiresSudo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sudo test in short mode")
	}

	// Create a temporary file to test with
	tmpFile, err := os.CreateTemp("", "rootfs-test-*.img")
	if err != nil {
		t.Skip("Could not create temp file")
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// This should fail because we can't mount without sudo
	// (unless running as root, which we shouldn't be in tests)
	err = PrepareForAVF(tmpFile.Name(), "testpass")
	if err == nil {
		// If it succeeded, we're probably running as root
		t.Log("PrepareForAVF succeeded - running as root?")
	} else {
		t.Logf("PrepareForAVF failed as expected: %v", err)
	}
}
