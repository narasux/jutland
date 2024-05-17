package mission

import (
	"log"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/mission/action"
	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/controller/computer"
	"github.com/narasux/jutland/pkg/mission/controller/human"
	"github.com/narasux/jutland/pkg/mission/drawer"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	md "github.com/narasux/jutland/pkg/mission/metadata"
	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
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
func NewManager(mission md.Mission) *MissionManager {
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
	m.updateInstructions()
	m.executeInstructions()
	m.updateCameraPosition()
	m.updateGameOptions()
	m.updateSelectedShips()
	m.updateWeaponFire()
	m.updateShipTrails()
	m.updateShotBullets()
	m.updateMissionStatus()

	return m.state.MissionStatus, nil
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
	s.Camera.Pos.AssignMxy(
		lo.Max([]int{s.Camera.Pos.MX, 0}),
		lo.Max([]int{s.Camera.Pos.MY, 0}),
	)
	s.Camera.Pos.AssignMxy(
		lo.Min([]int{s.Camera.Pos.MX, s.MissionMD.MapCfg.Width - s.Camera.Width - 1}),
		lo.Min([]int{s.Camera.Pos.MY, s.MissionMD.MapCfg.Height - s.Camera.Height - 1}),
	)
}

// 更新游戏选项
func (m *MissionManager) updateGameOptions() {
	// 按下 d 键，全局展示 / 不展示所有战舰状态
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyD) {
		m.state.GameOpts.ForceDisplayState = !m.state.GameOpts.ForceDisplayState
	}
}

// 更新选择的战舰列表
func (m *MissionManager) updateSelectedShips() {
	// 选择一个区域中的所有战舰
	if area := action.DetectCursorSelectArea(m.state); area != nil {
		m.state.SelectedShips = []string{}
		for _, ship := range m.state.Ships {
			// 被鼠标划区区域选中的我方战舰
			if ship.BelongPlayer == m.state.CurPlayer && area.Contain(ship.CurPos) {
				m.state.SelectedShips = append(m.state.SelectedShips, ship.Uid)
			}
		}
		return
	}
}

func (m *MissionManager) updateWeaponFire() {
	// 限制单次循环内的开火声音次数，避免过于嘈杂
	audioPlayQuota := 3

	for shipUid, ship := range m.state.Ships {
		for enemyUid, enemy := range m.state.Ships {
			// 不能炮击自己，也不能主动炮击己方的战舰
			if shipUid == enemyUid || ship.BelongPlayer == enemy.BelongPlayer {
				continue
			}
			// 如果在最大射程内，立刻开火（只对该目标开火）
			if ship.InMaxRange(enemy.CurPos) {
				bullets := ship.Fire(enemy.CurPos)
				if len(bullets) == 0 {
					continue
				}
				// 炮击声音（只有战舰在视野内才播放声音）
				// FIXME 需要修复声音重叠变大声的问题
				if audioPlayQuota > 0 && m.state.Camera.Contains(ship.CurPos) {
					audio.PlayAudioToEnd(audioRes.NewGunMK45())
					audioPlayQuota--
				}
				m.state.ShotBullets = slices.Concat(m.state.ShotBullets, bullets)
				break
			}
		}
	}
}

// 更新战舰尾流状态
func (m *MissionManager) updateShipTrails() {
	for i := 0; i < len(m.state.ShipTrails); i++ {
		// 尾流尺寸越来越大，但是留存时间越来越短
		m.state.ShipTrails[i].Size += 0.2
		m.state.ShipTrails[i].Life -= 1
	}
	// 生命周期结束的，不再需要
	m.state.ShipTrails = lo.Filter(m.state.ShipTrails, func(t *obj.ShipTrail, _ int) bool {
		return t.Life > 0
	})
	for _, ship := range m.state.Ships {
		if ship.CurSpeed > 0 {
			// TODO 考虑下这里的 size 要不要是战舰图片的 size ？好像不是很有必要？
			// TODO 尾流的 Life 应该和速度相关，是否和战舰类型相关？
			m.state.ShipTrails = append(m.state.ShipTrails, obj.NewShipTrail(ship.CurPos, ship.CurRotation, 20, 60))
		}
	}
}

// 更新弹药状态
func (m *MissionManager) updateShotBullets() {
	for i := 0; i < len(m.state.ShotBullets); i++ {
		m.state.ShotBullets[i].Forward()
	}
	m.state.ShotBullets = lo.Filter(m.state.ShotBullets, func(b *obj.Bullet, _ int) bool {
		// 如果已经到达目标地点，则不再需要保留
		if b.CurPos.MEqual(b.TargetPos) {
			// TODO 伤害结算
			return false
		}
		return true
	})
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
