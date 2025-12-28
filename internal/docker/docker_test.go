package docker

import (
	"testing"
)

func TestIsAvailable(t *testing.T) {
	// This test checks if Docker is available on the system
	// It's informational - passes regardless of result
	available := IsAvailable()
	t.Logf("Docker available: %v", available)
}

func TestExportImageMissingDocker(t *testing.T) {
	if IsAvailable() {
		t.Skip("Docker is available, skipping missing docker test")
	}

	// When Docker is not available, ExportImage should fail
	err := ExportImage("test", "/tmp/test.img", "100M")
	if err == nil {
		t.Error("Expected error when Docker is not available")
	}
}
