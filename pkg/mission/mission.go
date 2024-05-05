package mission

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/action"
	"github.com/narasux/jutland/pkg/mission/drawer"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	md "github.com/narasux/jutland/pkg/mission/metadata"
	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
)

// MissionManager 任务管理器
type MissionManager struct {
	state  *state.MissionState
	drawer *drawer.Drawer
	// 指令集合
	instructions []instr.Instruction

	// 任务日志 TODO 考虑直接用 chan ? 或者抽成 logger 包？
	// Logs []string
}

// NewManager ...
func NewManager(mission md.Mission) *MissionManager {
	return &MissionManager{
		state: state.NewMissionState(
			mission, md.Get(mission).InitCameraPos,
		),
		drawer:       drawer.NewDrawer(),
		instructions: []instr.Instruction{},
	}
}

// Draw 绘制任务图像
func (m *MissionManager) Draw(screen *ebiten.Image) {
	m.drawer.Draw(screen, m.state)
}

func (m *MissionManager) Update() (state.MissionStatus, error) {
	m.dropExecutedInstructions()
	m.executeInstructions()
	m.updateCameraPosition()
	m.updateMissionStatus()

	// FIXME 移除这段测试代码
	if len(m.instructions) == 0 {
		var ship *obj.BattleShip
		for _, s := range m.state.Ships {
			ship = s
			break
		}
		tarPos := obj.MapPos{MX: 48, MY: 54, RX: 48.5, RY: 54.5}
		if ship.CurPos.MX == 48 {
			tarPos = obj.MapPos{MX: 36, MY: 40, RX: 36.5, RY: 40.5}
		}
		m.instructions = append(m.instructions, instr.NewShipMove(ship.Uid, tarPos))
	}
	return m.state.MissionStatus, nil
}

// 已经执行完的指令不需要重复执行
func (m *MissionManager) dropExecutedInstructions() {
	m.instructions = lo.Filter(m.instructions, func(i instr.Instruction, _ int) bool {
		return !i.IsExecuted()
	})
}

// 逐条执行指令（移动/炮击/雷击/建造）
func (m *MissionManager) executeInstructions() {
	for _, i := range m.instructions {
		if err := i.Exec(m.state); err != nil {
			// TODO 某个指令执行失败，不影响流程，但是应该有错误信息输出到游戏界面？
			log.Printf("Instruction %s exec error: %s", i.String(), err)
			continue
		}
	}
}

// 计算下一帧相机位置
func (m *MissionManager) updateCameraPosition() {
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
	s.Camera.Pos.MX = lo.Max([]int{s.Camera.Pos.MX, 0})
	s.Camera.Pos.MY = lo.Max([]int{s.Camera.Pos.MY, 0})
	s.Camera.Pos.MX = lo.Min([]int{s.Camera.Pos.MX, s.MissionMD.MapCfg.Width - s.Camera.Width - 1})
	s.Camera.Pos.MY = lo.Min([]int{s.Camera.Pos.MY, s.MissionMD.MapCfg.Height - s.Camera.Height - 1})
}

// TODO 计算下一帧任务状态
func (m *MissionManager) updateMissionStatus() {
	switch m.state.MissionStatus {
	case state.MissionRunning:
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			m.state.MissionStatus = state.MissionPaused
			return
		}
	case state.MissionPaused:
		if ebiten.IsKeyPressed(ebiten.KeyQ) {
			m.state.MissionStatus = state.MissionFailed
			return
		} else if ebiten.IsKeyPressed(ebiten.KeyEnter) {
			m.state.MissionStatus = state.MissionRunning
			return
		}
	default:
		m.state.MissionStatus = state.MissionRunning
	}
}
