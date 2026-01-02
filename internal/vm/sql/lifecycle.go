// SQL VM lifecycle operations (Start, Stop, Remove)
// TEAM_022: Split from sql.go for readability
// TEAM_029: Migrated to use common package
package sql

import (
	"github.com/anthropics/sovereign/internal/vm/common"
)

// TEAM_029: Start delegates to common.StartVM
func (v *VM) Start() error {
	return common.StartVM(SQLConfig)
}

// TEAM_029: Stop delegates to common.StopVM
func (v *VM) Stop() error {
	return common.StopVM(SQLConfig)
}

// TEAM_029: Remove delegates to common.RemoveVM
func (v *VM) Remove() error {
	return common.RemoveVM(SQLConfig)
}

// TEAM_039: Clean delegates to common.CleanVM for Tailscale cleanup
func (v *VM) Clean() error {
	return common.CleanVM(SQLConfig)
}

// TEAM_041: Diagnose delegates to common.DiagnoseVM for comprehensive debugging
func (v *VM) Diagnose() error {
	return common.DiagnoseVM(SQLConfig)
}

// TEAM_041: Fix delegates to common.FixVM for automatic issue repair
func (v *VM) Fix() error {
	return common.FixVM(SQLConfig)
}
