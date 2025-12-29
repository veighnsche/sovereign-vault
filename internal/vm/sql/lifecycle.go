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
