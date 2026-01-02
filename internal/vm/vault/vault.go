// Package vault provides Vaultwarden VM operations
// TEAM_034: Password manager VM with Tailscale TLS
package vault

import (
	"github.com/anthropics/sovereign/internal/vm"
	"github.com/anthropics/sovereign/internal/vm/common"
)

// VaultConfig defines the configuration for the Vaultwarden VM
// TEAM_034: Based on working Forgejo pattern
var VaultConfig = &common.VMConfig{
	Name:           "vault",
	DisplayName:    "Vaultwarden",
	TAPInterface:   "vm_vault",
	TAPHostIP:      "192.168.100.1",
	TAPGuestIP:     "192.168.100.4", // SQL=.2, Forge=.3, Vault=.4
	TAPSubnet:      "192.168.100.0/24",
	TailscaleHost:  "sovereign-vault",
	DevicePath:     "/data/sovereign/vm/vault",
	LocalPath:      "vm/vault",
	ServicePorts:   []int{443, 80, 3012}, // TEAM_035: Added WebSocket port for browser extension sync
	ReadyMarker:    "INIT COMPLETE",
	StartTimeout:   120,
	DockerImage:    "sovereign-vault",
	SharedKernel:   true,
	KernelSource:   "vm/sql/Image",
	NeedsSecrets:   true,                  // Vaultwarden needs database password
	ProcessPattern: "[c]rosvm.*vm/vault/", // TEAM_036: Match path, not 'vault' (SQL cmdline has vaultwarden.db_password)
	// Vaultwarden requires PostgreSQL for its database
	Dependencies: []common.ServiceDependency{
		common.PostgreSQLDependency,
	},
}

func init() {
	vm.Register("vault", &VM{})
}

// VM implements the vm.VM interface for Vaultwarden
type VM struct{}

func (v *VM) Name() string { return "vault" }

// Build delegates to common.BuildVM
func (v *VM) Build() error {
	return common.BuildVM(VaultConfig, "")
}

// Deploy delegates to common.DeployVM
func (v *VM) Deploy() error {
	return common.DeployVM(VaultConfig)
}
