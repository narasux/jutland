package manager

import (
	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/controller/computer"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// activePlayerHandler 返回本拍驱动玩家侧的处理器。
// 托管开关由 Hacker 终端秘籍写入，这里只按开关选实现。
func (m *MissionManager) activePlayerHandler() controller.InputHandler {
	if m.state.Core.PlayerAIControl && m.playerAlphaAuto != nil {
		return m.playerAlphaAuto
	}
	return m.playerAlphaHandler
}

// syncPlayerAIControl 把秘籍写入的控制权开关落到处理器上。
// 开启时新建玩家侧 AI 并恢复舰队的控制能力；关闭时丢弃处理器，
// 但保留 InstructionSet 里已经在执行的航路，玩家可用新指令覆盖。
func (m *MissionManager) syncPlayerAIControl() {
	if !m.state.Core.PlayerAIControl {
		m.playerAlphaAuto = nil
		return
	}
	if m.playerAlphaAuto != nil {
		return
	}
	m.playerAlphaAuto = computer.NewHandler(m.state.Player.CurPlayer)
	m.preparePlayerFleetForAI()
}

// preparePlayerFleetForAI 在全权托管前恢复玩家侧单位的控制能力。
// 玩家可能手动关过武器、飞机或机场，AI 不会自己重新打开；不恢复会让整队哑火。
func (m *MissionManager) preparePlayerFleetForAI() {
	for _, ship := range m.state.Arena.OrderedShips() {
		if ship == nil || ship.CurHP <= 0 || ship.BelongPlayer != m.state.Player.CurPlayer {
			continue
		}
		ship.EnableWeapon(objUnit.WeaponTypeAll)
		ship.Aircraft.Disable = false
		// 清掉玩家手动锁定的目标，让 AI 从干净状态重新择敌。
		ship.AttackTarget = ""
	}
	for _, airfield := range m.state.Arena.OrderedAirfields() {
		if airfield == nil || airfield.BelongPlayer != m.state.Player.CurPlayer {
			continue
		}
		airfield.Disabled = false
		airfield.Aircraft.Disable = false
	}
}
