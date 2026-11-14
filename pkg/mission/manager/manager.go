package manager

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	audioPlayer "github.com/narasux/jutland/pkg/audio/player"
	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/controller/computer"
	"github.com/narasux/jutland/pkg/mission/controller/human"
	"github.com/narasux/jutland/pkg/mission/drawer"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/hacker"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/object"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/sidebar"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/targeting"
	"github.com/narasux/jutland/pkg/mission/unitpanel"
	mapBlockImg "github.com/narasux/jutland/pkg/resources/images/mapblock"
	"github.com/narasux/jutland/pkg/utils/magnify"
)

const (
	mapBlockPrewarmMargin     = 2
	mapBlockPrewarmIdleBudget = 4
	mapBlockPrewarmZoomBudget = 24
	mapBlockPrewarmZoomTicks  = 45
	// mapBlockPrewarmTimeSlice 每拍预热的时间片上限；预算张数仍是硬上限，
	// 纹理创建变慢时靠时间片兜底，不让单帧开销失控。
	mapBlockPrewarmTimeSlice = 3 * time.Millisecond
	wheelZoomCooldownTicks   = 8
	// reportWheelRowsPerUnit 是一个标准滚轮刻度在战报列表里滚动的行数。
	reportWheelRowsPerUnit = 3
	// reportWheelMaxRowsPerEvent 限制单次滚轮事件的最大行数，避免触控板惯性一下跳太远。
	reportWheelMaxRowsPerEvent = 6
)

// MissionManager 任务管理器
type MissionManager struct {
	state                     *state.MissionState
	drawer                    *drawer.Drawer
	sidebar                   *sidebar.Panel
	terminal                  *hacker.Terminal
	instructionSet            *InstructionSet
	playerAlphaHandler        controller.InputHandler
	playerBetaHandler         controller.InputHandler
	weaponFirePlayer          *audioPlayer.WeaponFire
	pinchWheelAccum           float64
	wheelZoomCooldown         int
	reportWheelAccum          float64
	mapBlockPrewarmZoom       int
	mapBlockPrewarmBurstTicks int
	mapBlockPrewarmFocusX     int
	mapBlockPrewarmFocusY     int
	mapBlockPrewarmFocusW     int
	mapBlockPrewarmFocusH     int
	mapBlockPrewarmSettled    bool
	combatBuckets             combatIndex
	// effectSeq 命中特效的确定性种子来源：不用全局随机数，避免影响战斗随机序列
	effectSeq uint32

	simTick                  int64
	targetingPlan            targeting.Plan
	targetingResults         chan targeting.Plan
	targetingBusy            bool
	targetingDirty           bool
	targetingLastRequestTick int64
	targetingCursors         map[string]map[object.Type]int
	takeoffTypeCursors       map[string]int
}

// New 创建任务管理器，playerSide 为玩家在任务配置中选择的阵营。
func New(mission string, playerSide faction.Side) *MissionManager {
	magnify.Init()
	manager := &MissionManager{
		state:          state.NewMissionState(mission, playerSide),
		drawer:         drawer.NewDrawer(mission),
		sidebar:        sidebar.New(mission),
		terminal:       hacker.NewTerminal(),
		instructionSet: NewInstructionSet(),
		// 目前用户一只能是人类，用户二是电脑 TODO 支持多人远程联机
		playerAlphaHandler: human.NewHandler(faction.HumanAlpha),
		playerBetaHandler:  computer.NewHandler(faction.ComputerAlpha),
		weaponFirePlayer:   audioPlayer.NewWeaponFire(),
		targetingResults:   make(chan targeting.Plan, 1),
		targetingDirty:     true,
		targetingCursors:   map[string]map[object.Type]int{},
		takeoffTypeCursors: map[string]int{},
	}
	return manager
}

// Resize 将 Ebiten 的真实逻辑屏幕尺寸同步到任务布局，并刷新相机视野范围。
func (m *MissionManager) Resize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	if m.state.View.Layout.Width == width && m.state.View.Layout.Height == height {
		return
	}
	m.state.View.Layout.Width = width
	m.state.View.Layout.Height = height
	m.state.RefreshCameraSize()
}

// Draw 绘制任务图像
func (m *MissionManager) Draw(screen *ebiten.Image) {
	m.drawer.Draw(screen, m.state, m.terminal)
	m.sidebar.Draw(screen, m.state)
}

// DrawPreview 绘制开始任务前的战场预览，不激活任务侧栏。
func (m *MissionManager) DrawPreview(screen *ebiten.Image) {
	m.drawer.Draw(screen, m.state, m.terminal)
}

// BattleReportLayout 返回当前屏幕尺寸下的战报面板布局。
// pkg/game 与绘制层共用这份布局，保证按钮命中区域和画面一致。
func (m *MissionManager) BattleReportLayout(withButtons bool) state.BattleReportLayout {
	return state.CalcBattleReportLayout(
		m.state.View.Layout.Width,
		m.state.View.Layout.Height,
		withButtons,
	)
}

// DrawBattleReport 绘制结算 / 暂停实时战报。
func (m *MissionManager) DrawBattleReport(screen *ebiten.Image, final bool) {
	m.drawer.DrawBattleReport(screen, m.state, m.BattleReportLayout(final), final)
}

// BattleReportPage 返回战报当前页。
func (m *MissionManager) BattleReportPage() state.BattleReportPage {
	return m.state.UI.BattleReportPage
}

// ShiftBattleReportPage 在总览 / 舰艇 / 飞机 / 详情之间循环切换。
func (m *MissionManager) ShiftBattleReportPage(delta int) {
	m.SetBattleReportPage(m.state.UI.BattleReportPage.Shift(delta))
}

// SetBattleReportPage 切换到指定战报页。
func (m *MissionManager) SetBattleReportPage(page state.BattleReportPage) {
	if !page.Valid() {
		return
	}
	if page != m.state.UI.BattleReportPage {
		m.state.UI.BattleReportEventScroll = 0
		m.state.UI.BattleReportShipScroll = [2]int{}
		m.state.UI.BattleReportScrollDrag = state.BattleReportScrollDragNone
		m.state.UI.BattleReportScrollDragOffset = 0
		m.reportWheelAccum = 0
	}
	m.state.UI.BattleReportPage = page
}

// finalizeBattleStats 在任务进入胜败状态后收尾本局统计。
// 必须在 Game 切换模式前完成，保证结算页读到最终值。
func (m *MissionManager) finalizeBattleStats() {
	status := m.state.Core.MissionStatus
	if status != state.MissionSuccess && status != state.MissionFailed {
		return
	}
	if m.state.Stats.Finalized {
		return
	}
	reason := state.BattleEndVictory
	if status == state.MissionFailed {
		reason = state.BattleEndDefeat
		if m.state.Core.ConfirmQuitMission {
			reason = state.BattleEndAborted
		}
	}
	m.state.FinalizeBattleStats(reason)
	m.state.UI.BattleReportPage = state.BattleReportPageOverview
	m.state.UI.BattleReportEventScroll = 0
	m.state.UI.BattleReportEventFilter = state.BattleReportEventFilterAll
	m.state.UI.BattleReportShipScroll = [2]int{}
	m.state.UI.BattleReportScrollDrag = state.BattleReportScrollDragNone
	m.state.UI.BattleReportScrollDragOffset = 0
	m.reportWheelAccum = 0
}

// WarmupMapBlocks 分帧预热地图块缓存，并报告当前视野是否已经就绪。
func (m *MissionManager) WarmupMapBlocks() bool {
	return m.updateMapBlockPrewarm()
}

// Update 更新一帧任务状态
func (m *MissionManager) Update() (state.MissionStatus, error) {
	status := m.state.Core.MissionStatus
	// 声呐覆盖轮廓是按住 V 的临时覆盖层：默认不画，松开或离开运行状态立即隐藏。
	m.state.UI.ShowSonarRanges = status == state.MissionRunning &&
		ebiten.IsKeyPressed(ebiten.KeyV)
	if status == state.MissionRunning {
		actions := m.sidebar.Update(m.state)
		m.state.UI.UIConsumesCursor = m.sidebar.ConsumesCursor(m.state)
		m.handleUnitPanelActions(actions)
	} else {
		m.state.UI.UIConsumesCursor = false
	}

	switch status {
	case state.MissionRunning:
		if !m.state.UI.UIConsumesCursor {
			m.updateRallyLineClick()
			m.updateRallyPointRightClick()
			m.updateAirfieldSelection()
		}
		m.updateGameOptions(m.state.UI.UIConsumesCursor)
	case state.MissionInTerminal:
		m.updateTerminal()
	case state.MissionPaused:
		// 实时战报是整屏 UI：滚轮交给列表，不能再缩放或边缘移动背后的战场。
		reportOpen := m.state.UI.PauseStats
		m.updateGameOptions(reportOpen)
		if !reportOpen {
			m.updateCameraPosition()
		}
	}

	if missionStatusRunsSimulation(status) {
		m.simTick++
		m.state.Core.SimTick = m.simTick
		objUnit.SetSimTick(m.simTick)
		m.updateCommandPhase()
		switch status {
		case state.MissionRunning:
			// 是否响应鼠标边缘滚动由 getNextCameraPosInGameMode 内部判断；
			// 这里必须每帧调用，否则鼠标停在 UI 面板上时进行中的相机定位会卡住。
			m.updateCameraPosition()
		case state.MissionInMap:
			m.updateCameraPosition()
		case state.MissionInBuilding:
			m.updateReinforcePoints()
		}
		m.updateSupportPhase()
		m.updateMapBlockPrewarm()
		if status == state.MissionRunning {
			if !m.state.UI.UIConsumesCursor {
				m.updateSelectedShips()
			} else {
				m.state.Interaction.IsAreaSelecting = false
			}
			m.updateShipGroups()
		}
		m.updateCombatPhase()
	}

	m.updateMissionStatus()
	m.finalizeBattleStats()

	return m.state.Core.MissionStatus, nil
}

// handleUnitPanelActions 将 UI 动作转换为相机操作或游戏指令。
func (m *MissionManager) handleUnitPanelActions(actions []unitpanel.Action) {
	for _, action := range actions {
		switch action.Kind {
		case unitpanel.ActionFocusShip:
			if ship := m.state.Arena.Ships[action.FocusUid]; ship != nil {
				m.state.Interaction.FocusedShipUid = ship.Uid
			}
		case unitpanel.ActionCenterTarget:
			if target := m.state.Arena.Ships[action.TargetUid]; target != nil {
				m.centerCameraOn(target.CurPos)
			}
		case unitpanel.ActionToggleWeapon:
			for _, shipUid := range action.ShipUids {
				if action.Enable {
					m.instructionSet.Add(instr.NewEnableWeapon(shipUid, action.WeaponType))
				} else {
					m.instructionSet.Add(instr.NewDisableWeapon(shipUid, action.WeaponType))
				}
			}
		case unitpanel.ActionToggleAircraft:
			for _, shipUid := range action.ShipUids {
				if action.Enable {
					m.instructionSet.Add(instr.NewEnableAircraft(shipUid))
				} else {
					m.instructionSet.Add(instr.NewDisableAircraft(shipUid))
				}
			}
		case unitpanel.ActionToggleDive:
			for _, shipUid := range action.ShipUids {
				if instruction := diveToggleInstruction(m.state, shipUid); instruction != nil {
					m.instructionSet.Add(instruction)
				}
			}
		case unitpanel.ActionToggleAirfield:
			if af := m.state.Arena.Airfields[action.AirfieldUid]; af != nil {
				af.Disabled = !af.Disabled
			}
		case unitpanel.ActionSetProducing:
			if af := m.state.Arena.Airfields[action.AirfieldUid]; af != nil {
				af.CurProducing = action.PlaneName
			}
		}
	}
}

// missionStatusRunsSimulation 判断当前任务状态是否需要继续推进战斗模拟
func missionStatusRunsSimulation(status state.MissionStatus) bool {
	switch status {
	case state.MissionRunning, state.MissionInMap, state.MissionInBuilding:
		return true
	default:
		return false
	}
}

// updateCommandPhase 更新玩家和电脑指令并执行已就绪指令
func (m *MissionManager) updateCommandPhase() {
	m.updateInstructions()
	m.executeInstructions()
}

// updateSupportPhase 更新标识、建筑和辅助单位效果
func (m *MissionManager) updateSupportPhase() {
	m.updateGameMarks()
	m.updateBuildings()
	m.updateHospitalShipHealing()
}

// updateMapBlockPrewarm 分帧预热相机附近场景地图块的缩放缓存，返回当前缩放是否就绪。
// 相机平移不重置队列：平移只是补充新视野的任务，已有排队任务（含相邻档位）
// 继续消化；只有缩放档位真正变化时才丢弃其他档位的任务并触发一轮爆发。
func (m *MissionManager) updateMapBlockPrewarm() bool {
	camera := m.state.View.Camera
	zoom := state.NormalizeZoom(m.state.UI.GameOpts.Zoom)
	zoomChanged := m.mapBlockPrewarmZoom != zoom
	focusChanged := zoomChanged ||
		m.mapBlockPrewarmFocusX != camera.Pos.MX ||
		m.mapBlockPrewarmFocusY != camera.Pos.MY ||
		m.mapBlockPrewarmFocusW != camera.Width ||
		m.mapBlockPrewarmFocusH != camera.Height
	if zoomChanged {
		mapBlockImg.SceneBlockCache.ResetPrewarmQueue(zoom)
	}
	if focusChanged {
		m.mapBlockPrewarmFocusX = camera.Pos.MX
		m.mapBlockPrewarmFocusY = camera.Pos.MY
		m.mapBlockPrewarmFocusW = camera.Width
		m.mapBlockPrewarmFocusH = camera.Height
		if zoomChanged {
			m.mapBlockPrewarmZoom = zoom
			m.mapBlockPrewarmBurstTicks = mapBlockPrewarmZoomTicks
		}
		m.mapBlockPrewarmSettled = false
	} else if m.mapBlockPrewarmSettled {
		return true
	}

	budget := mapBlockPrewarmIdleBudget
	if m.mapBlockPrewarmBurstTicks > 0 {
		budget = mapBlockPrewarmZoomBudget
		m.mapBlockPrewarmBurstTicks--
	}

	mapBlockImg.SceneBlockCache.SchedulePrewarmAround(
		camera.Pos.MX, camera.Pos.MY,
		camera.Width, camera.Height,
		[]int{zoom},
		mapBlockPrewarmMargin,
	)
	processed := mapBlockImg.SceneBlockCache.StepPrewarm(budget, mapBlockPrewarmTimeSlice)
	if mapBlockImg.SceneBlockCache.HasMissingAround(
		camera.Pos.MX, camera.Pos.MY,
		camera.Width, camera.Height,
		zoom,
		mapBlockPrewarmMargin,
	) {
		m.mapBlockPrewarmSettled = false
		return false
	}

	remainingBudget := budget - processed
	if remainingBudget <= 0 {
		// 当前视野已经齐，但这一拍没有余量去排相邻缩放，下一拍再排。
		return true
	}
	adjacentZooms := getAdjacentZooms(zoom)
	if len(adjacentZooms) == 0 {
		m.mapBlockPrewarmSettled = mapBlockImg.SceneBlockCache.PrewarmQueueLen() == 0
		return true
	}
	mapBlockImg.SceneBlockCache.SchedulePrewarmAround(
		camera.Pos.MX, camera.Pos.MY,
		camera.Width, camera.Height,
		adjacentZooms,
		mapBlockPrewarmMargin,
	)
	mapBlockImg.SceneBlockCache.StepPrewarm(remainingBudget, mapBlockPrewarmTimeSlice)
	m.mapBlockPrewarmSettled = mapBlockImg.SceneBlockCache.PrewarmQueueLen() == 0
	return true
}

// getAdjacentZooms 返回当前 zoom 的相邻档位，当前视野热好后再作为下一层预热目标。
func getAdjacentZooms(zoom int) []int {
	zoom = state.NormalizeZoom(zoom)
	for idx, availableZoom := range state.AvailableZooms {
		if availableZoom != zoom {
			continue
		}

		zooms := []int{}
		if idx > 0 {
			zooms = append(zooms, state.AvailableZooms[idx-1])
		}
		if idx+1 < len(state.AvailableZooms) {
			zooms = append(zooms, state.AvailableZooms[idx+1])
		}
		return zooms
	}
	return nil
}

func (m *MissionManager) rebuildCombatBuckets() {
	m.combatBuckets.rebuild(m.state.Arena.Ships, m.state.Arena.Planes)
}

// updateCombatPhase 更新武器开火、弹药、尾流和单位消亡状态
func (m *MissionManager) updateCombatPhase() {
	m.updateShipDiveStates()
	// 声呐探测独立于迷雾，必须在刷新视野之前跑。
	m.updateSubmarineDetection()
	m.updateVision()
	m.updateAutoScout()
	m.updateTargetPlanning()
	m.weaponFirePlayer.Update()
	m.updateAirfieldAlertLaunch()
	m.updateShipWeaponFire()
	m.updatePlaneAttackOrReturn()
	m.updatePlaneWeaponFire()
	m.updateObjectTrails()
	m.updateShotBullets()
	m.updateShipAnimations()
	m.updateExplosions()
	m.updateMissionShips()
	m.updateMissionPlanes()
}
