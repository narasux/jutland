package manager

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/action"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	obj "github.com/narasux/jutland/pkg/mission/object"
)

// 更新游戏选项
func (m *MissionManager) updateGameOptions() {
	// 按下 d 键，全局展示 / 不展示所有战舰状态
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyD) {
		m.state.GameOpts.ForceDisplayState = !m.state.GameOpts.ForceDisplayState
	}

	// 按下 n 键，全局展示 / 不展示所有伤害数字
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyN) {
		m.state.GameOpts.DisplayDamageNumber = !m.state.GameOpts.DisplayDamageNumber
	}

	// 按下 m 键，切换地图展示模式
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyM) {
		m.state.GameOpts.MapDisplayMode = (m.state.GameOpts.MapDisplayMode + 1) % 2
	}
}

// 更新指令集合
func (m *MissionManager) updateInstructions() {
	// 已经执行完的指令，就不再需要
	m.instructions = lo.PickBy(m.instructions, func(key string, instruction instr.Instruction) bool {
		return !instruction.IsExecuted()
	})
	// 逐个读取各个用户的输入，更新指令
	m.instructions = lo.Assign(m.instructions, m.playerAlphaHandler.Handle(m.state))
	m.instructions = lo.Assign(m.instructions, m.playerBetaHandler.Handle(m.state))
}

// 计算下一帧相机位置
func (m *MissionManager) updateCameraPosition() {
	if m.state.GameOpts.UserInputBlocked() {
		return
	}

	s := m.state
	switch action.DetectCursorHoverOnGameMap(s.Layout) {
	case action.HoverScreenLeft:
		s.Camera.Pos.MX -= 1
	case action.HoverScreenRight:
		s.Camera.Pos.MX += 1
	case action.HoverScreenTop:
		s.Camera.Pos.MY -= 1
	case action.HoverScreenBottom:
		s.Camera.Pos.MY += 1
	case action.HoverScreenTopLeft:
		s.Camera.Pos.MX -= 1
		s.Camera.Pos.MY -= 1
	case action.HoverScreenTopRight:
		s.Camera.Pos.MX += 1
		s.Camera.Pos.MY -= 1
	case action.HoverScreenBottomLeft:
		s.Camera.Pos.MX -= 1
		s.Camera.Pos.MY += 1
	case action.HoverScreenBottomRight:
		s.Camera.Pos.MX += 1
		s.Camera.Pos.MY += 1
	default:
		// DoNothing
	}

	// 防止超出边界
	s.Camera.Pos.AssignMxy(
		lo.Max([]int{s.Camera.Pos.MX, 0}),
		lo.Max([]int{s.Camera.Pos.MY, 0}),
	)
	s.Camera.Pos.AssignMxy(
		lo.Min([]int{s.Camera.Pos.MX, s.MissionMD.MapCfg.Width - s.Camera.Width - 1}),
		lo.Min([]int{s.Camera.Pos.MY, s.MissionMD.MapCfg.Height - s.Camera.Height - 1}),
	)
}

// 更新选择的战舰列表
func (m *MissionManager) updateSelectedShips() {
	if m.state.GameOpts.UserInputBlocked() {
		return
	}
	// 选择一个区域中的所有战舰
	if area := action.DetectCursorSelectArea(m.state); area != nil {
		m.state.SelectedShips = []string{}
		for _, ship := range m.state.Ships {
			// 被鼠标划区区域选中的我方战舰
			if ship.BelongPlayer == m.state.CurPlayer && area.Contain(ship.CurPos) {
				m.state.SelectedShips = append(m.state.SelectedShips, ship.Uid)
			}
		}
	}
	// 正在分组中，不可用
	if !m.state.IsGrouping {
		// 通过分组选中战舰
		groupID := action.GetGroupIDByPressedKey()
		if groupID != obj.GroupIDNone {
			shipInGroup := lo.Filter(lo.Values(m.state.Ships), func(ship *obj.BattleShip, _ int) bool {
				return ship.BelongPlayer == m.state.CurPlayer && ship.GroupID == groupID
			})
			m.state.SelectedShips = lo.Map(shipInGroup, func(ship *obj.BattleShip, _ int) string {
				return ship.Uid
			})

			// 如果当前选中的分组不是当前按键的分组，则更新记录
			if m.state.SelectedGroupID != groupID {
				m.state.SelectedGroupID = groupID
			} else {
				// 如果当前选中的分组再次被选中，移动相机中心位置到当前分组的第一艘战舰处
				if len(m.state.SelectedShips) > 0 {
					m.state.Camera.Pos = m.state.Ships[m.state.SelectedShips[0]].CurPos.Copy()
					m.state.Camera.Pos.SubMx(m.state.Camera.Width / 2)
					m.state.Camera.Pos.SubMy(m.state.Camera.Height / 2)
				}
			}
		}
	}

	// 检查选中的战舰，如果已经被摧毁，则要去掉
	m.state.SelectedShips = lo.Filter(m.state.SelectedShips, func(uid string, _ int) bool {
		ship, ok := m.state.Ships[uid]
		return ok && ship != nil && ship.CurHP > 0
	})
	// 没有战舰被选中，应该重置 SelectedGroupID
	if m.state.SelectedGroupID != obj.GroupIDNone && len(m.state.SelectedShips) == 0 {
		m.state.SelectedGroupID = obj.GroupIDNone
	}
}

// 更新舰队编组状态（左 Ctrl + 0-9 编组）
func (m *MissionManager) updateShipGroups() {
	if m.state.GameOpts.UserInputBlocked() {
		return
	}
	// 按下左边的 ctrl 键：进入 / 退出编组模式
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyControlLeft) {
		m.state.IsGrouping = !m.state.IsGrouping
	}
	// 设置编组后，如果松开 ctrl，则退出编组模式
	if action.DetectKeyboardKeyJustReleased(ebiten.KeyControlLeft) {
		m.state.IsGrouping = false
	}
	// 没有在编组模式，直接返回
	if !m.state.IsGrouping {
		return
	}
	groupID := action.GetGroupIDByPressedKey()
	// 没有设置合法的编组
	if groupID == obj.GroupIDNone {
		return
	}
	// 重新编组，只有当前选中的拥有这个编组
	for _, ship := range m.state.Ships {
		if ship.GroupID == groupID {
			ship.GroupID = obj.GroupIDNone
		}
	}
	for _, shipUid := range m.state.SelectedShips {
		m.state.Ships[shipUid].GroupID = groupID
	}
}
