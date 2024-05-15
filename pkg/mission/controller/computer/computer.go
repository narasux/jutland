package computer

import (
	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/state"
)

// ComputerDecisionHandler 电脑决策处理器
type ComputerDecisionHandler struct {
	player faction.Player
}

// NewHandler ...
func NewHandler(player faction.Player) *ComputerDecisionHandler {
	return &ComputerDecisionHandler{player: player}
}

var _ controller.InputHandler = (*ComputerDecisionHandler)(nil)

// Handle 处理计算机决策，更新指令集
// Human is foolish!
func (h *ComputerDecisionHandler) Handle(misState *state.MissionState) map[string]instr.Instruction {
	instructions := map[string]instr.Instruction{}

	// TODO 移除 测试 AI 指令：如果范围 20 内有战舰，则移动到它附近
	//for _, ship := range misState.Ships {
	//	if ship.BelongPlayer != h.player {
	//		continue
	//	}
	//	for _, ship2 := range misState.Ships {
	//		if ship2.BelongPlayer == h.player {
	//			continue
	//		}
	//		if geometry.CalcDistance(ship.CurPos.RX, ship.CurPos.RY, ship2.CurPos.RX, ship2.CurPos.RY) < 20 {
	//			instructions[fmt.Sprintf("%s-%s", ship.Uid, instr.NameShipMove)] = instr.NewShipMove(ship.Uid, ship2.CurPos)
	//			break
	//		}
	//	}
	//}

	return instructions
}
