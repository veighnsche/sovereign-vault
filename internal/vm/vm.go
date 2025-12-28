// Package vm provides the VM interface and registry
// TEAM_010: Created during CLI refactor for VM abstraction
package vm

import "sync"

// VM defines the interface for virtual machine operations
// TEAM_011: Removed Prepare() - merged into Build() for simpler workflow
type VM interface {
	Name() string
	Build() error  // Build VM image (includes rootfs preparation)
	Deploy() error // Deploy to device (idempotent - creates dirs if needed)
	Start() error  // Start the VM
	Stop() error   // Stop the VM
	Test() error   // Test VM connectivity
	Remove() error // Remove VM from device
}

var (
	registry = make(map[string]VM)
	mu       sync.RWMutex
)

// Register registers a VM implementation
func Register(name string, vm VM) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = vm
}

// Get returns a VM by name
func Get(name string) (VM, bool) {
	mu.RLock()
	defer mu.RUnlock()
	vm, ok := registry[name]
	return vm, ok
}

// List returns all registered VM names
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
