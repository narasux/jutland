package manager

import (
	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"

	audioPlayer "github.com/narasux/jutland/pkg/audio/player"
	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/controller/computer"
	"github.com/narasux/jutland/pkg/mission/controller/human"
	"github.com/narasux/jutland/pkg/mission/drawer"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/hacker"
	"github.com/narasux/jutland/pkg/mission/sidebar"
	"github.com/narasux/jutland/pkg/mission/state"
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
}

// New 创建任务管理器
func New(mission string, ui *ebitenui.UI) *MissionManager {
	magnify.Init()
	return &MissionManager{
		state:          state.NewMissionState(mission),
		drawer:         drawer.NewDrawer(mission),
		sidebar:        sidebar.New(mission, ui),
		terminal:       hacker.NewTerminal(),
		instructionSet: NewInstructionSet(),
		// 目前用户一只能是人类，用户二是电脑 TODO 支持多人远程联机
		playerAlphaHandler: human.NewHandler(faction.HumanAlpha),
		playerBetaHandler:  computer.NewHandler(faction.ComputerAlpha),
		weaponFirePlayer:   audioPlayer.NewWeaponFire(),
	}
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
		m.sidebar.Update(m.state)
	} else {
		m.state.UI.SidebarConsumesCursor = false
	}

	switch status {
	case state.MissionRunning:
		if !m.state.UI.SidebarConsumesCursor {
			m.updateRallyLineClick()
			m.updateRallyPointRightClick()
		}
		m.updateGameOptions(m.state.UI.SidebarConsumesCursor)
	case state.MissionInTerminal:
		m.updateTerminal()
	case state.MissionPaused:
		m.updateGameOptions(false)
		// 暂停时也可以移动相机
		m.updateCameraPosition()
	}

	if missionStatusRunsSimulation(status) {
		m.updateCommandPhase()
		switch status {
		case state.MissionRunning, state.MissionInMap:
			m.updateCameraPosition()
		case state.MissionInBuilding:
			m.updateReinforcePoints()
		}
		m.updateSupportPhase()
		m.updateMapBlockPrewarm()
		if status == state.MissionRunning {
			if !m.state.UI.SidebarConsumesCursor {
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
	m.weaponFirePlayer.Update()
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
