package manager

import (
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
	wheelZoomCooldownTicks    = 8
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
	mapBlockPrewarmZoom       int
	mapBlockPrewarmBurstTicks int
	mapBlockPrewarmFocusX     int
	mapBlockPrewarmFocusY     int
	mapBlockPrewarmFocusW     int
	mapBlockPrewarmFocusH     int

	simTick                  int64
	targetingPlan            targeting.Plan
	targetingResults         chan targeting.Plan
	targetingBusy            bool
	targetingDirty           bool
	targetingLastRequestTick int64
	targetingCursors         map[string]map[object.Type]int
	takeoffTypeCursors       map[string]int
}

// New 创建任务管理器
func New(mission string) *MissionManager {
	magnify.Init()
	manager := &MissionManager{
		state:          state.NewMissionState(mission),
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

// WarmupMapBlocks 分帧预热地图块缓存，并报告当前视野是否已经就绪。
func (m *MissionManager) WarmupMapBlocks() bool {
	return m.updateMapBlockPrewarm()
}

// Update 更新一帧任务状态
func (m *MissionManager) Update() (state.MissionStatus, error) {
	status := m.state.Core.MissionStatus
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
		m.updateGameOptions(false)
		// 暂停时也可以移动相机
		m.updateCameraPosition()
	}

	if missionStatusRunsSimulation(status) {
		m.simTick++
		m.updateCommandPhase()
		switch status {
		case state.MissionRunning:
			if !m.state.UI.UIConsumesCursor {
				m.updateCameraPosition()
			}
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
func (m *MissionManager) updateMapBlockPrewarm() bool {
	camera := m.state.View.Camera
	zoom := state.NormalizeZoom(m.state.UI.GameOpts.Zoom)
	if m.mapBlockPrewarmZoom != zoom ||
		m.mapBlockPrewarmFocusX != camera.Pos.MX ||
		m.mapBlockPrewarmFocusY != camera.Pos.MY ||
		m.mapBlockPrewarmFocusW != camera.Width ||
		m.mapBlockPrewarmFocusH != camera.Height {
		mapBlockImg.SceneBlockCache.ResetPrewarmQueue()
		m.mapBlockPrewarmZoom = zoom
		m.mapBlockPrewarmFocusX = camera.Pos.MX
		m.mapBlockPrewarmFocusY = camera.Pos.MY
		m.mapBlockPrewarmFocusW = camera.Width
		m.mapBlockPrewarmFocusH = camera.Height
		m.mapBlockPrewarmBurstTicks = mapBlockPrewarmZoomTicks
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
	processed := mapBlockImg.SceneBlockCache.StepPrewarm(budget)
	if mapBlockImg.SceneBlockCache.HasMissingAround(
		camera.Pos.MX, camera.Pos.MY,
		camera.Width, camera.Height,
		zoom,
		mapBlockPrewarmMargin,
	) {
		return false
	}

	remainingBudget := budget - processed
	if remainingBudget <= 0 {
		return true
	}
	adjacentZooms := getAdjacentZooms(zoom)
	if len(adjacentZooms) == 0 {
		return true
	}
	mapBlockImg.SceneBlockCache.SchedulePrewarmAround(
		camera.Pos.MX, camera.Pos.MY,
		camera.Width, camera.Height,
		adjacentZooms,
		mapBlockPrewarmMargin,
	)
	mapBlockImg.SceneBlockCache.StepPrewarm(remainingBudget)
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

// updateCombatPhase 更新武器开火、弹药、尾流和单位消亡状态
func (m *MissionManager) updateCombatPhase() {
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
