// Package forge provides Forgejo VM operations
// TEAM_012: Git forge VM with CI/CD capabilities
// TEAM_025: Refactored to use TAP networking and correct paths
// TEAM_029: Migrated to use common package
package forge

import (
	"github.com/anthropics/sovereign/internal/vm"
	"github.com/anthropics/sovereign/internal/vm/common"
)

// TEAM_029: ForgeConfig defines the configuration for the Forgejo VM
// TEAM_033: Fixed subnet to match actual implementation - all VMs on same bridge (192.168.100.0/24)
var ForgeConfig = &common.VMConfig{
	Name:           "forge",
	DisplayName:    "Forgejo",
	TAPInterface:   "vm_forge",
	TAPHostIP:      "192.168.100.1",
	TAPGuestIP:     "192.168.100.3",
	TAPSubnet:      "192.168.100.0/24",
	TailscaleHost:  "sovereign-forge",
	DevicePath:     "/data/sovereign/vm/forgejo",
	LocalPath:      "vm/forgejo",
	ServicePorts:   []int{3000, 22},
	ReadyMarker:    "INIT COMPLETE",
	StartTimeout:   120,
	DockerImage:    "sovereign-forge",
	SharedKernel:   true,
	KernelSource:   "vm/sql/Image",
	NeedsSecrets:   false,
	ProcessPattern: "[c]rosvm.*vm/forgejo/", // TEAM_036: Match path, not 'forge' (SQL cmdline has forgejo.db_password)
	// TEAM_029: Forgejo requires PostgreSQL for its database
	Dependencies: []common.ServiceDependency{
		common.PostgreSQLDependency,
	},
}

func init() {
	vm.Register("forge", &VM{})
}

// VM implements the vm.VM interface for Forgejo
type VM struct{}

func (v *VM) Name() string { return "forge" }

// TEAM_029: Build delegates to common.BuildVM
func (v *VM) Build() error {
	return common.BuildVM(ForgeConfig, "")
}

// TEAM_029: Deploy delegates to common.DeployVM
func (v *VM) Deploy() error {
	return common.DeployVM(ForgeConfig)
}
