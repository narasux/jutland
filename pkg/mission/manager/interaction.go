package manager

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/action"
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/state"
)

// 更新游戏选项
func (m *MissionManager) updateGameOptions(skipCursorInput bool) {
	_, wheelY := ebiten.Wheel()
	if !skipCursorInput && wheelY != 0 {
		sx, sy := ebiten.CursorPosition()
		if wheelY > 0 {
			m.state.StepZoomAtScreenPoint(1, sx, sy)
		} else {
			m.state.StepZoomAtScreenPoint(-1, sx, sy)
		}
	}
}

// 更新指令集合
func (m *MissionManager) updateInstructions() {
	// 已经执行完的指令，就不再需要
	m.instructionSet.RemoveExecuted()
	// 逐个读取各个用户的输入，更新指令
	m.instructionSet.Assign(m.playerAlphaHandler.Handle(m.instructionSet.Items(), m.state))
	m.instructionSet.Assign(m.playerBetaHandler.Handle(m.instructionSet.Items(), m.state))
}

// 计算下一帧相机位置
func (m *MissionManager) updateCameraPosition() {
	var nextPos *objPos.MapPos
	// 游戏模式 / 全屏地图模式走不同的相机位置更新模式
	if m.state.Core.MissionStatus == state.MissionInMap {
		nextPos = m.getNextCameraPosInFullMapMode()
		// 如果是全屏地图模式，且点击位置相同，则退出全屏（模拟双击效果）
		if nextPos != nil && m.state.View.Camera.Pos.MEqual(*nextPos) {
			m.state.Core.MissionStatus = state.MissionRunning
		}
	} else {
		nextPos = m.getNextCameraPosInGameMode()
	}

	// 无法获取下一帧相机位置，不更新
	if nextPos == nil {
		return
	}

	// 剪掉小尾巴，避免出现黑边
	moveSpeed := m.state.View.Camera.BaseMoveSpeed
	rx := float64(int(nextPos.RX/moveSpeed)) * moveSpeed
	ry := float64(int(nextPos.RY/moveSpeed)) * moveSpeed
	m.state.View.Camera.Pos.AssignRxy(rx, ry)
}

// 计算下一帧相机位置（全屏地图模式）
// 全屏模式，鼠标点击可以移动相机位置（点击位置居中）
func (m *MissionManager) getNextCameraPosInFullMapMode() *objPos.MapPos {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return nil
	}

	sx, sy := ebiten.CursorPosition()
	xOffset := float64(m.state.View.Layout.Width-m.state.View.Layout.Height) / 2

	abbrMapWidth, abbrMapHeight := float64(m.state.View.Layout.Height), float64(m.state.View.Layout.Height)
	mapWidth, mapHeight := float64(m.state.Core.MissionMD.MapCfg.Width), float64(m.state.Core.MissionMD.MapCfg.Height)

	rx := (float64(sx) - xOffset) / abbrMapWidth * mapWidth
	ry := float64(sy) / abbrMapHeight * mapHeight

	pos := m.state.View.Camera.Pos.Copy()
	pos.AssignRxy(rx-float64(m.state.View.Camera.Width)/2, ry-float64(m.state.View.Camera.Height)/2)
	// 防止超出边界
	pos.EnsureBorder(m.state.CameraPosBorder())

	return &pos
}

// 计算下一帧相机位置（游戏模式）
// 游戏模式，可以通过 hover 鼠标在边缘上，移动相机
func (m *MissionManager) getNextCameraPosInGameMode() *objPos.MapPos {
	pos := m.state.View.Camera.Pos.Copy()
	// TODO 支持在游戏设置内修改相机移动速度
	moveSpeed := m.state.View.Camera.BaseMoveSpeed

	switch action.DetectCursorHoverOnGameMap(m.state.View.Layout) {
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
	if m.state.Core.MissionStatus != state.MissionRunning {
		return
	}
	// 选择一个区域中的所有战舰
	if area := action.DetectCursorSelectArea(m.state); area != nil {
		m.state.Interaction.SelectedShips = []string{}
		for _, ship := range m.state.Arena.Ships {
			// 被鼠标划区区域选中的我方战舰
			if ship.BelongPlayer == m.state.Player.CurPlayer && area.Contain(ship.CurPos) {
				m.state.Interaction.SelectedShips = append(m.state.Interaction.SelectedShips, ship.Uid)
			}
		}
	}
	// 正在分组中，不可用
	if !m.state.Interaction.IsGrouping {
		// 通过分组选中战舰
		groupID := action.GetGroupIDByPressedKey()
		if groupID != object.GroupIDNone {
			m.state.Interaction.SelectedShips = m.state.Interaction.SelectedShips[:0]
			for _, ship := range m.state.Arena.Ships {
				if ship.BelongPlayer == m.state.Player.CurPlayer && ship.GroupID == groupID {
					m.state.Interaction.SelectedShips = append(m.state.Interaction.SelectedShips, ship.Uid)
				}
			}

			// 如果当前选中的分组不是当前按键的分组，则更新记录
			if m.state.Interaction.SelectedGroupID != groupID {
				m.state.Interaction.SelectedGroupID = groupID
			} else {
				// 如果当前选中的分组再次被选中，移动相机中心位置到当前分组的第一艘战舰处
				if len(m.state.Interaction.SelectedShips) > 0 {
					nextPos := m.state.Arena.Ships[m.state.Interaction.SelectedShips[0]].CurPos.Copy()
					nextPos.SubMx(m.state.View.Camera.Width / 2)
					nextPos.SubMy(m.state.View.Camera.Height / 2)

					moveSpeed := m.state.View.Camera.BaseMoveSpeed
					rx := float64(int(nextPos.RX/moveSpeed)) * moveSpeed
					ry := float64(int(nextPos.RY/moveSpeed)) * moveSpeed
					m.state.View.Camera.Pos.AssignRxy(rx, ry)
				}
			}
		}
	}

	// 检查选中的战舰，如果已经被摧毁，则要去掉
	selectedShips := m.state.Interaction.SelectedShips[:0]
	for _, uid := range m.state.Interaction.SelectedShips {
		ship, ok := m.state.Arena.Ships[uid]
		if ok && ship != nil && ship.CurHP > 0 {
			selectedShips = append(selectedShips, uid)
		}
	}
	m.state.Interaction.SelectedShips = selectedShips
	// 没有战舰被选中，应该重置 SelectedGroupID
	if m.state.Interaction.SelectedGroupID != object.GroupIDNone && len(m.state.Interaction.SelectedShips) == 0 {
		m.state.Interaction.SelectedGroupID = object.GroupIDNone
	}
}

// 更新舰队编组状态（左 Ctrl + 0-9 编组）
func (m *MissionManager) updateShipGroups() {
	if m.state.Core.MissionStatus != state.MissionRunning {
		return
	}
	// 按下左边的 ctrl 键：进入 / 退出编组模式
	if inpututil.IsKeyJustPressed(ebiten.KeyControlLeft) {
		m.state.Interaction.IsGrouping = !m.state.Interaction.IsGrouping
	}
	// 设置编组后，如果松开 ctrl，则退出编组模式
	if inpututil.IsKeyJustReleased(ebiten.KeyControlLeft) {
		m.state.Interaction.IsGrouping = false
	}
	// 没有在编组模式，直接返回
	if !m.state.Interaction.IsGrouping {
		return
	}
	groupID := action.GetGroupIDByPressedKey()
	// 没有设置合法的编组
	if groupID == object.GroupIDNone {
		return
	}
	// 重新编组，只有当前选中的拥有这个编组
	for _, ship := range m.state.Arena.Ships {
		if ship.GroupID == groupID {
			ship.GroupID = object.GroupIDNone
		}
	}
	for _, shipUid := range m.state.Interaction.SelectedShips {
		m.state.Arena.Ships[shipUid].GroupID = groupID
	}
}

// 更新终端
func (m *MissionManager) updateTerminal() {
	m.terminal.Update(m.state)
}

// 更新被选中的待增援战舰
func (m *MissionManager) updateReinforcePoints() {
	if m.state.Core.MissionStatus != state.MissionInBuilding {
		return
	}

	reinforcePointUIDs := []string{}
	for uid, rp := range m.state.Arena.ReinforcePoints {
		if rp.BelongPlayer == m.state.Player.CurPlayer {
			reinforcePointUIDs = append(reinforcePointUIDs, uid)
		}
	}
	if len(reinforcePointUIDs) == 0 {
		return
	}

	slices.Sort(reinforcePointUIDs)
	rpIndex := lo.IndexOf(reinforcePointUIDs, m.state.Interaction.SelectedReinforcePointUid)

	// 上下方向键选择增援点
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		rpIndex++
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		rpIndex--
	}
	uidCount := len(reinforcePointUIDs)
	rpIndex = (rpIndex + uidCount) % uidCount

	rpUID := reinforcePointUIDs[rpIndex]
	m.state.Interaction.SelectedReinforcePointUid = rpUID

	rp := m.state.Arena.ReinforcePoints[rpUID]

	// 左右方向键选择战舰
	shipIndex := rp.CurSelectedShipIndex
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		shipIndex--
	} else if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		shipIndex++
	}
	shipCount := len(rp.ProvidedShipNames)
	shipIndex = (shipIndex + shipCount) % shipCount

	rp.CurSelectedShipIndex = shipIndex

	// 确定增援的战舰
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		rp.Summon(rp.ProvidedShipNames[shipIndex])
	}

	// 退格键取消最后增援的战舰
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(rp.OncomingShips) > 0 {
			rp.OncomingShips = rp.OncomingShips[:len(rp.OncomingShips)-1]
		}
	}

	// 缩略地图点击设集结点
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		sx, sy := ebiten.CursorPosition()
		w, h := float64(m.state.View.Layout.Width), float64(m.state.View.Layout.Height)
		margin, topGap := 18.0, 18.0
		topY := 48.0
		topH := h * 0.55
		previewW := w*0.6 - margin - topGap/2
		mapW := w - previewW - 2*margin - topGap
		mapX := margin + previewW + topGap
		mapY := topY
		if float64(sx) >= mapX && float64(sx) <= mapX+mapW &&
			float64(sy) >= mapY && float64(sy) <= mapY+topH {
			mapWidth := float64(m.state.Core.MissionMD.MapCfg.Width)
			mapHeight := float64(m.state.Core.MissionMD.MapCfg.Height)
			mx := int((float64(sx) - mapX) / mapW * mapWidth)
			my := int((float64(sy) - mapY) / topH * mapHeight)
			mx = max(min(mx, m.state.Core.MissionMD.MapCfg.Width-1), 0)
			my = max(min(my, m.state.Core.MissionMD.MapCfg.Height-1), 0)
			if m.state.Core.MissionMD.MapCfg.Map.IsLand(mx, my) {
				m.state.UI.RallySetFailedTick = 60
			} else {
				rp.SetRallyPos(objPos.New(mx, my))
			}
		}
	}
}

// 更新集结线显示（游戏模式下点击己方增援点）
func (m *MissionManager) updateRallyLineClick() {
	if m.state.Core.MissionStatus != state.MissionRunning {
		return
	}

	pos := action.DetectMouseButtonClickOnMap(m.state, ebiten.MouseButtonLeft)
	if pos == nil {
		return
	}

	for _, rp := range m.state.Arena.ReinforcePoints {
		if rp.BelongPlayer != m.state.Player.CurPlayer {
			continue
		}
		if rp.Pos.Distance(*pos) < 3 {
			m.state.UI.ShowRallyLinePointUid = rp.Uid
			return
		}
	}
	m.state.UI.ShowRallyLinePointUid = ""
}
