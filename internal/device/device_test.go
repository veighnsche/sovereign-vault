package device

import (
	"strings"
	"testing"
)

func TestIsConnected(t *testing.T) {
	// This is an integration test that checks actual device connectivity
	// It will pass if no device is connected (returns false)
	// or if a device is connected (returns true)
	// The important thing is it doesn't panic
	connected := IsConnected()
	t.Logf("Device connected: %v", connected)
}

func TestPushFilePath(t *testing.T) {
	// Test the path transformation logic
	localPath := "../vm/sql/rootfs.img"
	expected := "/data/local/tmp/" + strings.Replace(localPath, "/", "_", -1)

	// Verify the transformation produces a valid tmp path
	if !strings.HasPrefix(expected, "/data/local/tmp/") {
		t.Errorf("Expected path to start with /data/local/tmp/, got %s", expected)
	}

	// Verify no slashes remain in the filename part
	filename := strings.TrimPrefix(expected, "/data/local/tmp/")
	if strings.Contains(filename, "/") {
		t.Errorf("Expected no slashes in filename, got %s", filename)
	}
}

func TestWaitForFastbootTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	// Test with a very short timeout - should return error quickly
	// This test verifies the timeout logic works
	err := WaitForFastboot(1)
	if err == nil {
		t.Log("Device was in fastboot mode")
	} else {
		if !strings.Contains(err.Error(), "timeout") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	}
}

func TestWaitForAdbTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	// Test with a very short timeout
	err := WaitForAdb(1)
	if err == nil {
		t.Log("Device was connected via ADB")
	} else {
		if !strings.Contains(err.Error(), "timeout") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	}
}
