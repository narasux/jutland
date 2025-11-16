package computer

import (
	"math/rand"

	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	obj "github.com/narasux/jutland/pkg/mission/object"
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
func (h *ComputerDecisionHandler) Handle(
	curInstructions map[string]instr.Instruction, misState *state.MissionState,
) map[string]instr.Instruction {
	instructions := map[string]instr.Instruction{}

	// AI 指令：扫描所有增援点，只要可用，就随机召唤增援
	reinforcePointUid := ""
	for _, rp := range misState.ReinforcePoints {
		if rp.BelongPlayer != h.player {
			continue
		}
		reinforcePointUid = rp.Uid
		if len(rp.OncomingShips) < rp.MaxOncomingShip {
			summonInstr := instr.NewShipSummon(rp.Uid, "")
			instructions[summonInstr.Uid()] = summonInstr
		}
	}

	var ships, enemyShips []*obj.BattleShip
	for _, s := range misState.Ships {
		if s.BelongPlayer == h.player {
			ships = append(ships, s)
		} else {
			enemyShips = append(enemyShips, s)
		}
	}

	// AI 战舰够多，就开始莽一波
	isAttackMode := lo.Ternary(len(ships) >= 20, true, false)

	for _, ship := range ships {
		if isAttackMode && len(enemyShips) != 0 {
			instrUid := instr.GenInstrUid(instr.NameShipMovePath, ship.Uid)
			// 如果战舰已经在移动了，则跳过
			if _, ok := curInstructions[instrUid]; !ok {
				// 进攻模式，随机选一个敌人冲上去
				enemy := enemyShips[rand.Intn(len(enemyShips))]
				instructions[instrUid] = instr.NewShipMovePath(ship.Uid, ship.CurPos, enemy.CurPos)
			}
		} else if ship.CurPos.OnBorder(
			float64(misState.MissionMD.MapCfg.Width-2),
			float64(misState.MissionMD.MapCfg.Height-2),
		) && reinforcePointUid != "" {
			// 如果到了边界，且存在增援点，则往自己的增援点的集结点走
			moveInstr := instr.NewShipMove(
				ship.Uid, misState.ReinforcePoints[reinforcePointUid].RallyPos,
			)
			instructions[moveInstr.Uid()] = moveInstr
		} else {
			// 防御模式，如果附近有敌人在移动，则自己在附近随机移动
			for _, enemy := range enemyShips {
				if ship.CurPos.Distance(enemy.CurPos) < 20 && ship.CurSpeed == 0 && enemy.CurSpeed != 0 {
					x, y := rand.Intn(11)-5, rand.Intn(11)-5
					moveInstr := instr.NewShipMove(
						ship.Uid, obj.NewMapPos(
							misState.Ships[ship.Uid].CurPos.MX+x,
							misState.Ships[ship.Uid].CurPos.MY+y,
						),
					)
					instructions[moveInstr.Uid()] = moveInstr
					break
				}
			}
		}
	}

	return instructions
}
