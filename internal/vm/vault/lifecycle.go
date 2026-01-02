// Vault VM lifecycle operations (Start, Stop, Remove)
// TEAM_034: Based on forge/lifecycle.go pattern
package vault

import (
	"github.com/anthropics/sovereign/internal/vm/common"
)

// Start delegates to common.StartVM
func (v *VM) Start() error {
	return common.StartVM(VaultConfig)
}

// Stop delegates to common.StopVM
func (v *VM) Stop() error {
	return common.StopVM(VaultConfig)
}

// Remove delegates to common.RemoveVM
func (v *VM) Remove() error {
	return common.RemoveVM(VaultConfig)
}

// TEAM_039: Clean delegates to common.CleanVM for Tailscale cleanup
func (v *VM) Clean() error {
	return common.CleanVM(VaultConfig)
}

// TEAM_041: Diagnose delegates to common.DiagnoseVM for comprehensive debugging
func (v *VM) Diagnose() error {
	return common.DiagnoseVM(VaultConfig)
}

// TEAM_041: Fix delegates to common.FixVM for automatic issue repair
func (v *VM) Fix() error {
	return common.FixVM(VaultConfig)
}
