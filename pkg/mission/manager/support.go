package manager

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/samber/lo"

	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// 更新游戏标识
func (m *MissionManager) updateGameMarks() {
	for markID, mark := range m.state.UI.GameMarks {
		mark.Life--

		// 检查游戏标识，如果生命值为 0，则删除
		if mark.Life <= 0 {
			delete(m.state.UI.GameMarks, markID)
		}
	}

	// 集结点设置失败提示倒计时
	if m.state.UI.RallySetFailedTick > 0 {
		m.state.UI.RallySetFailedTick--
	}
}

// 更新建筑物
func (m *MissionManager) updateBuildings() {
	// 增援点当然算是建筑物！
	for _, rp := range m.state.Arena.ReinforcePoints {
		if ship := rp.Update(
			m.state.Arena.ShipUidGenerators[rp.BelongPlayer],
			// FIXME 目前电脑玩家先不限制金钱
			lo.Ternary(rp.BelongPlayer == m.state.Player.CurPlayer, m.state.Player.CurFunds, 50000),
		); ship != nil {
			m.state.Arena.Ships[ship.Uid] = ship
			if rp.BelongPlayer == m.state.Player.CurPlayer {
				m.state.Player.CurFunds -= ship.FundsCost
			}
			// 战舰移动到集结点 & 随机散开 [-3, 3] 的范围（通过 ShipMove 指令实现）
			x, y := rand.Intn(7)-3, rand.Intn(7)-3
			targetPos := objPos.New(rp.RallyPos.MX+x, rp.RallyPos.MY+y)
			m.instructionSet.Add(instr.NewShipMove(ship.Uid, targetPos))
		}
	}

	// 油井当然算是建筑物！
	fontSize := float64(24)
	for _, op := range m.state.Arena.OilPlatforms {
		text := fmt.Sprintf("+%d $", op.Yield)
		for _, ship := range m.state.Arena.Ships {
			if ship.Type != objUnit.ShipTypeCargo || ship.BelongPlayer != m.state.Player.CurPlayer {
				continue
			}

			// TODO 目前是只要货轮在油井附近就给钱，后续考虑开采 & 存储 & 货轮容量的设计
			if ship.CurPos.Near(op.Pos, float64(op.Radius)) {
				op.AddShip(ship)
			} else {
				op.RemoveShip(ship.Uid)
			}
		}

		for uid, ship := range op.LoadingOilShips {
			// 如果货轮不在了，需要及时移除掉
			cargo, ok := m.state.Arena.Ships[uid]
			if !ok {
				op.RemoveShip(uid)
			} else if cargo.BelongPlayer == m.state.Player.CurPlayer && ship.Update() {
				m.state.Player.CurFunds += int64(ship.FundYield)
				mark := objMark.NewText(cargo.CurPos, text, fontSize, colorx.Gold, 50)
				m.state.UI.GameMarks[mark.ID] = mark
			}
		}
	}
}

// 更新医疗船治疗逻辑
// 医疗船自动治疗范围内同阵营战舰（含自身），显示绿色浮动文字
// 注意：治疗间隔不受 config.G.SpeedMultiplier 影响，使用 time.Now() 而非帧计数
func (m *MissionManager) updateHospitalShipHealing() {
	now := time.Now().UnixMilli()
	for _, ship := range m.state.Arena.Ships {
		// 只有存活的医疗船才能治疗
		if ship.Type != objUnit.ShipTypeHospital || ship.CurHP <= 0 {
			continue
		}
		// 检查距上次治疗是否 ≥ 5000ms（5 秒固定间隔）
		if now-ship.LastHealAt < 5000 {
			continue
		}
		// 遍历同阵营战舰目标
		for _, target := range m.state.Arena.Ships {
			// 目标必须是同阵营、存活且未满血
			if target.BelongPlayer != ship.BelongPlayer || target.CurHP <= 0 || target.CurHP >= target.TotalHP {
				continue
			}
			// 检查目标是否在治疗范围内
			if !ship.CurPos.Near(target.CurPos, objUnit.HospitalShipEffectRange) {
				continue
			}
			// 恢复 HP（不超过上限）
			healAmount := ship.Length * ship.Width / 6
			target.CurHP = min(target.TotalHP, target.CurHP+healAmount)
			// 创建绿色浮动治疗文字
			text := fmt.Sprintf("+ %d HP", int(healAmount))
			mark := objMark.NewText(target.CurPos, text, 20, colorx.Green, 50)
			m.state.UI.GameMarks[mark.ID] = mark
		}
		// 治疗完成后更新时间戳
		ship.LastHealAt = now
	}
}
