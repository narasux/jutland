package manager

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/controller/computer"
	"github.com/narasux/jutland/pkg/mission/controller/human"
	"github.com/narasux/jutland/pkg/mission/drawer"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/hacker"
	"github.com/narasux/jutland/pkg/mission/sidebar"
	"github.com/narasux/jutland/pkg/mission/state"
)

// MissionManager 任务管理器
type MissionManager struct {
	state              *state.MissionState
	drawer             *drawer.Drawer
	sidebar            *sidebar.Panel
	terminal           *hacker.Terminal
	instructionSet     *InstructionSet
	playerAlphaHandler controller.InputHandler
	playerBetaHandler  controller.InputHandler
}

// New 创建任务管理器
func New(mission string) *MissionManager {
	return &MissionManager{
		state:          state.NewMissionState(mission),
		drawer:         drawer.NewDrawer(mission),
		sidebar:        sidebar.New(mission),
		terminal:       hacker.NewTerminal(),
		instructionSet: NewInstructionSet(),
		// 目前用户一只能是人类，用户二是电脑 TODO 支持多人远程联机
		playerAlphaHandler: human.NewHandler(faction.HumanAlpha),
		playerBetaHandler:  computer.NewHandler(faction.ComputerAlpha),
	}
}

// Draw 绘制任务图像
func (m *MissionManager) Draw(screen *ebiten.Image) {
	m.drawer.Draw(screen, m.state, m.terminal)
	m.sidebar.Draw(screen, m.state)
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

// updateCombatPhase 更新武器开火、弹药、尾流和单位消亡状态
func (m *MissionManager) updateCombatPhase() {
	m.updateShipWeaponFire()
	m.updatePlaneAttackOrReturn()
	m.updatePlaneWeaponFire()
	m.updateObjectTrails()
	m.updateShotBullets()
	m.updateMissionShips()
	m.updateMissionPlanes()
}
