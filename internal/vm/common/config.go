// Package common provides shared infrastructure for VM lifecycle operations.
// TEAM_029: Created during vm-common-refactor Phase 2
//
// This package enables trivial addition of new services by providing
// configuration-driven VM management. Each service defines a VMConfig,
// and the common functions handle Build, Deploy, Start, Stop, Test, Remove.
package common

// ServiceDependency defines a dependency on another service.
// TEAM_029: Used to fail-fast if required services are unavailable.
type ServiceDependency struct {
	Name          string // "sql" - the service name
	TailscaleHost string // "sovereign-sql" - Tailscale hostname to check
	TAPIP         string // "192.168.100.2" - TAP IP for local VM-to-VM (optional)
	Port          int    // 5432 - port to verify connectivity
	Description   string // "PostgreSQL database" - for error messages
}

// VMConfig defines the configuration for a VM service.
// Each service (sql, forge, vault, etc.) creates one of these.
type VMConfig struct {
	// Identity
	Name        string // "sql", "forge", "vault"
	DisplayName string // "PostgreSQL", "Forgejo", "Vaultwarden"

	// Networking - all VMs on shared bridge (192.168.100.0/24)
	// TEAM_033: Fixed comments - all VMs use same subnet for VM-to-VM communication
	TAPInterface string // "vm_sql", "vm_forge", "vm_vault"
	TAPHostIP    string // "192.168.100.1" (gateway for all VMs)
	TAPGuestIP   string // "192.168.100.2" (SQL), "192.168.100.3" (Forge)
	TAPSubnet    string // "192.168.100.0/24" (shared by all VMs)

	// Tailscale
	TailscaleHost string // "sovereign-sql", "sovereign-forge"

	// Paths
	DevicePath string // "/data/sovereign/vm/sql"
	LocalPath  string // "vm/sql"

	// Service
	ServicePorts []int  // [5432], [3000, 22], [80]
	ReadyMarker  string // "PostgreSQL started", "INIT COMPLETE"
	StartTimeout int    // seconds: 90, 120

	// Build options
	DockerImage  string // "sovereign-sql", "sovereign-forge"
	SharedKernel bool   // false for sql, true for forge (uses sql's kernel)
	KernelSource string // "vm/sql/Image" - where to get kernel if SharedKernel
	NeedsSecrets bool   // true for sql (prompts for DB password)

	// Process detection pattern for pgrep/grep
	// TEAM_029: Use [c]rosvm pattern to avoid grep matching itself
	ProcessPattern string // "[c]rosvm.*sql", "[c]rosvm.*forge"

	// Dependencies - other services this VM requires
	// TEAM_029: Checked before Start() to fail fast
	Dependencies []ServiceDependency

	// Hooks for service-specific logic (optional)
	PreBuildHook  func(*VMConfig) error
	PostBuildHook func(*VMConfig) error
}

// TestResult represents the outcome of a single test.
type TestResult struct {
	Name    string
	Passed  bool
	Message string
}

// TestFunc is a custom test function that services can provide.
type TestFunc func(cfg *VMConfig) TestResult
