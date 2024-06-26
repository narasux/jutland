package manager

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/controller/computer"
	"github.com/narasux/jutland/pkg/mission/controller/human"
	"github.com/narasux/jutland/pkg/mission/drawer"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/state"
)

// MissionManager 任务管理器
type MissionManager struct {
	state              *state.MissionState
	drawer             *drawer.Drawer
	instructions       map[string]instr.Instruction
	playerAlphaHandler controller.InputHandler
	playerBetaHandler  controller.InputHandler
}

// NewManager ...
func New(mission string) *MissionManager {
	return &MissionManager{
		state:  state.NewMissionState(mission),
		drawer: drawer.NewDrawer(mission),
		// 指令集合 key 为 objUid + instrName
		// 注：同一对象，只能有一个同名指令（如：战舰不能有两个目标位置）
		instructions: map[string]instr.Instruction{},
		// 目前用户一只能是人类，用户二是电脑 TODO 支持多人远程联机
		playerAlphaHandler: human.NewHandler(faction.HumanAlpha),
		playerBetaHandler:  computer.NewHandler(faction.ComputerAlpha),
	}
}

// Draw 绘制任务图像
func (m *MissionManager) Draw(screen *ebiten.Image) {
	m.drawer.Draw(screen, m.state)
}

func (m *MissionManager) Update() (state.MissionStatus, error) {
	// 如果是暂停，不要继续刷新
	if m.state.MissionStatus == state.MissionRunning {
		m.updateGameOptions()
		m.updateInstructions()
		m.executeInstructions()
		m.updateCameraPosition()
		m.updateGameMarks()
		m.updateSelectedShips()
		m.updateShipGroups()
		m.updateWeaponFire()
		m.updateObjectTrails()
		m.updateShotBullets()
		m.updateMissionShips()
	}
	m.updateMissionStatus()

	return m.state.MissionStatus, nil
}
