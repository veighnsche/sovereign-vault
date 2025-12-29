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
