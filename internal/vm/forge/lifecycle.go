// Forge VM lifecycle operations (Start, Stop, Remove)
// TEAM_025: Split from forge.go following sql/lifecycle.go pattern
// TEAM_029: Migrated to use common package
package forge

import (
	"github.com/anthropics/sovereign/internal/vm/common"
)

// TEAM_029: Start delegates to common.StartVM
func (v *VM) Start() error {
	return common.StartVM(ForgeConfig)
}

// TEAM_029: Stop delegates to common.StopVM
func (v *VM) Stop() error {
	return common.StopVM(ForgeConfig)
}

// TEAM_029: Remove delegates to common.RemoveVM
func (v *VM) Remove() error {
	return common.RemoveVM(ForgeConfig)
}

// TEAM_039: Clean delegates to common.CleanVM for Tailscale cleanup
func (v *VM) Clean() error {
	return common.CleanVM(ForgeConfig)
}

// TEAM_041: Diagnose delegates to common.DiagnoseVM for comprehensive debugging
func (v *VM) Diagnose() error {
	return common.DiagnoseVM(ForgeConfig)
}

// TEAM_041: Fix delegates to common.FixVM for automatic issue repair
func (v *VM) Fix() error {
	return common.FixVM(ForgeConfig)
}
