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
