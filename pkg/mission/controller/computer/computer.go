package computer

import (
	"fmt"
	"math/rand"

	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/geometry"
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

	// TODO 移除 测试 AI 指令：如果本身静止，范围 20 内有敌方战舰，且正在移动，则随机移动
	for _, ship := range misState.Ships {
		if ship.BelongPlayer != h.player {
			continue
		}
		for _, enemy := range misState.Ships {
			if enemy.BelongPlayer == h.player {
				continue
			}
			distance := geometry.CalcDistance(ship.CurPos.RX, ship.CurPos.RY, enemy.CurPos.RX, enemy.CurPos.RY)
			if distance < 20 && ship.CurSpeed == 0 && enemy.CurSpeed != 0 {
				x, y := rand.Intn(11)-5, rand.Intn(11)-5
				instructions[fmt.Sprintf("%s-%s", ship.Uid, instr.NameShipMove)] = instr.NewShipMove(
					ship.Uid, obj.NewMapPos(
						misState.Ships[ship.Uid].CurPos.MX+x,
						misState.Ships[ship.Uid].CurPos.MY+y,
					),
				)
				break
			}
		}
	}

	// TODO 移除 测试 AI 指令：扫描所有增援点，只要可用，就随机召唤增援
	for _, rp := range misState.ReinforcePoints {
		if rp.BelongPlayer != h.player {
			continue
		}
		if len(rp.OncomingShips) < rp.MaxOncomingShip {
			instructions[fmt.Sprintf("%s-%s", rp.Uid, instr.NameShipSummon)] = instr.NewShipSummon(rp.Uid, "")
		}
	}

	return instructions
}
