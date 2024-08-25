package manager

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/action"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
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

	// 同时按下 - 和 + 键，开启终端
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyMinus) &&
		action.DetectKeyboardKeyJustPressed(ebiten.KeyEqual) {
		m.state.GameOpts.DisplayTerminal = !m.state.GameOpts.DisplayTerminal
	}

	// 按下 b 键，开启查看增援点模式
	// FIXME 移除该测试用逻辑
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyB) {
		m.state.ReinforcePoints[0].Summon("montana")
	}
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyV) {
		m.state.ReinforcePoints[0].Summon("alaska_plus")
	}
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyC) {
		m.state.ReinforcePoints[0].Summon("new_orleans")
	}
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyX) {
		m.state.ReinforcePoints[0].Summon("porter")
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
	var nextPos *obj.MapPos
	// 游戏模式 / 全屏地图模式走不同的相机位置更新模式
	if m.state.GameOpts.MapDisplayMode == state.MapDisplayModeFull {
		nextPos = m.getNextCameraPosInFullMapMode()
	} else {
		nextPos = m.getNextCameraPosInGameMode()
	}

	// 无法获取下一帧相机位置，不更新
	if nextPos == nil {
		return
	}

	// 剪掉小尾巴，避免出现黑边
	moveSpeed := m.state.Camera.BaseMoveSpeed
	rx := float64(int(nextPos.RX/moveSpeed)) * moveSpeed
	ry := float64(int(nextPos.RY/moveSpeed)) * moveSpeed
	m.state.Camera.Pos.AssignRxy(rx, ry)
}

// 计算下一帧相机位置（全屏地图模式）
// 全屏模式，鼠标点击可以移动相机位置（点击位置居中）
func (m *MissionManager) getNextCameraPosInFullMapMode() *obj.MapPos {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}

	sx, sy := ebiten.CursorPosition()
	xOffset := float64(m.state.Layout.Width-m.state.Layout.Height) / 2

	abbrMapWidth, abbrMapHeight := float64(m.state.Layout.Height), float64(m.state.Layout.Height)
	mapWidth, mapHeight := float64(m.state.MissionMD.MapCfg.Width), float64(m.state.MissionMD.MapCfg.Height)

	rx := (float64(sx) - xOffset) / abbrMapWidth * mapWidth
	ry := float64(sy) / abbrMapHeight * mapHeight

	pos := m.state.Camera.Pos.Copy()
	pos.AssignRxy(rx-float64(m.state.Camera.Width)/2, ry-float64(m.state.Camera.Height)/2)
	// 防止超出边界
	pos.EnsureBorder(m.state.CameraPosBorder())

	// 如果是全屏地图模式，且点击位置相同，则退出全屏（模拟双击效果）
	if m.state.Camera.Pos.MEqual(pos) {
		m.state.GameOpts.MapDisplayMode = state.MapDisplayModeNone
	}

	return &pos
}

// 计算下一帧相机位置（游戏模式）
// 游戏模式，可以通过 hover 鼠标在边缘上，移动相机
func (m *MissionManager) getNextCameraPosInGameMode() *obj.MapPos {
	pos := m.state.Camera.Pos.Copy()
	// TODO 支持在游戏设置内修改相机移动速度
	moveSpeed := m.state.Camera.BaseMoveSpeed

	switch action.DetectCursorHoverOnGameMap(m.state.Layout) {
	case action.HoverScreenLeft:
		pos.SubRx(moveSpeed)
	case action.HoverScreenRight:
		pos.AddRx(moveSpeed)
	case action.HoverScreenTop:
		pos.SubRy(moveSpeed)
	case action.HoverScreenBottom:
		pos.AddRy(moveSpeed)
	case action.HoverScreenTopLeft:
		pos.SubRx(moveSpeed)
		pos.SubRy(moveSpeed)
	case action.HoverScreenTopRight:
		pos.AddRx(moveSpeed)
		pos.SubRy(moveSpeed)
	case action.HoverScreenBottomLeft:
		pos.SubRx(moveSpeed)
		pos.AddRy(moveSpeed)
	case action.HoverScreenBottomRight:
		pos.AddRx(moveSpeed)
		pos.AddRy(moveSpeed)
	default:
		// DoNothing
		return nil
	}

	// 防止超出边界
	pos.EnsureBorder(m.state.CameraPosBorder())

	return &pos
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
					nextPos := m.state.Ships[m.state.SelectedShips[0]].CurPos.Copy()
					nextPos.SubMx(m.state.Camera.Width / 2)
					nextPos.SubMy(m.state.Camera.Height / 2)

					moveSpeed := m.state.Camera.BaseMoveSpeed
					rx := float64(int(nextPos.RX/moveSpeed)) * moveSpeed
					ry := float64(int(nextPos.RY/moveSpeed)) * moveSpeed
					m.state.Camera.Pos.AssignRxy(rx, ry)
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
