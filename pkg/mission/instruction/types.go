/*
工作流：用户操作 / AI 决策 -> 指令 —> 指令集 -> 战舰/建筑行为
*/
package instruction

import "github.com/narasux/jutland/pkg/mission/state"

const (
	NameEnableWeapon  = "EnableWeapon"
	NameDisableWeapon = "DisableWeapon"
	NameShipMove      = "ShipMove"
	NameShipMovePath  = "ShipMovePath"
	NameShipSummon    = "ShipSummon"
)

// Instruction 指令
type Instruction interface {
	// Exec 指令执行
	Exec(*state.MissionState) error
	// Executed 是否已执行完成
	Executed() bool
	// Uid 获取指令 UID
	Uid() string
	// String 指令描述
	String() string
}
