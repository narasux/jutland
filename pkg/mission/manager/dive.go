package manager

import (
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/state"
)

// updateShipDiveStates 推进所有潜艇的下潜/上浮计时，由战斗阶段每拍调用。
func (m *MissionManager) updateShipDiveStates() {
	for _, ship := range m.state.Arena.Ships {
		ship.AdvanceDive()
	}
}

// diveToggleInstruction 按潜艇当前深度生成下潜/上浮指令。
// 过渡中或非潜艇返回 nil，避免计时器被反复重置。
func diveToggleInstruction(ms *state.MissionState, shipUid string) instr.Instruction {
	return instr.NewDiveToggle(ms.Arena.Ships[shipUid])
}
