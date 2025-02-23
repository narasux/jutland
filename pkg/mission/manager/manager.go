package manager

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/controller/computer"
	"github.com/narasux/jutland/pkg/mission/controller/human"
	"github.com/narasux/jutland/pkg/mission/drawer"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/hacker"
	"github.com/narasux/jutland/pkg/mission/state"
)

// MissionManager 任务管理器
type MissionManager struct {
	state              *state.MissionState
	drawer             *drawer.Drawer
	terminal           *hacker.Terminal
	instructionSet     *InstructionSet
	playerAlphaHandler controller.InputHandler
	playerBetaHandler  controller.InputHandler
}

// NewManager ...
func New(mission string) *MissionManager {
	return &MissionManager{
		state:          state.NewMissionState(mission),
		drawer:         drawer.NewDrawer(mission),
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
}

func (m *MissionManager) Update() (state.MissionStatus, error) {
	// 如果是暂停，不要继续刷新
	switch m.state.MissionStatus {
	case state.MissionRunning:
		m.updateGameOptions()
		m.updateInstructions()
		m.executeInstructions()
		m.updateCameraPosition()
		m.updateGameMarks()
		m.updateBuildings()
		m.updateSelectedShips()
		m.updateShipGroups()
		m.updateShipWeaponFire()
		m.updatePlaneWeaponFire()
		m.updateObjectTrails()
		m.updateShotBullets()
		m.updateMissionShips()
	case state.MissionInMap:
		m.updateInstructions()
		m.executeInstructions()
		m.updateCameraPosition()
		m.updateGameMarks()
		m.updateBuildings()
		m.updateShipWeaponFire()
		m.updatePlaneWeaponFire()
		m.updateObjectTrails()
		m.updateShotBullets()
		m.updateMissionShips()
	case state.MissionInBuilding:
		m.updateInstructions()
		m.executeInstructions()
		m.updateReinforcePoints()
		m.updateGameMarks()
		m.updateBuildings()
		m.updateShipWeaponFire()
		m.updatePlaneWeaponFire()
		m.updateObjectTrails()
		m.updateShotBullets()
		m.updateMissionShips()
	case state.MissionInTerminal:
		m.updateTerminal()
	}
	m.updateMissionStatus()

	return m.state.MissionStatus, nil
}
