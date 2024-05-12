/*
工作流：用户操作 / AI 决策 -> 指令 —> 指令集 -> 战舰/建筑行为
*/
package instruction

import "github.com/narasux/jutland/pkg/mission/state"

const (
	NameEnableWeapon  = "EnableWeapon"
	NameDisableWeapon = "DisableWeapon"
	NameShipMove      = "ShipMove"
)

// Instruction 指令
type Instruction interface {
	// Exec 指令执行
	Exec(*state.MissionState) error
	// IsExecuted 是否已执行完成
	IsExecuted() bool
	// GetObjUid 获取指令关联的对象 UID
	GetObjUid() string
	// Name 指令名称
	Name() string
	// String 指令描述
	String() string
}
