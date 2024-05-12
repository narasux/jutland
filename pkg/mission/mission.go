package mission

import (
	"fmt"
	"log"
	"math/rand"

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
	// 指令集合 key 为 objUid + instrName
	// 注：同一对象，只能有一个同名指令（如：战舰不能有两个目标位置）
	instructions map[string]instr.Instruction

	// 任务日志 TODO 考虑直接用 chan ? 或者抽成 logger 包？
	// Logs []string
}

// NewManager ...
func NewManager(mission md.Mission) *MissionManager {
	return &MissionManager{
		state:        state.NewMissionState(mission),
		drawer:       drawer.NewDrawer(mission),
		instructions: map[string]instr.Instruction{},
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
	m.updateSelectedShips()
	m.updateShipTrails()
	m.updateMissionStatus()

	return m.state.MissionStatus, nil
}

// 更新指令集合
func (m *MissionManager) updateInstructions() {
	// 已经执行完的指令，就不再需要
	m.instructions = lo.PickBy(m.instructions, func(key string, instruction instr.Instruction) bool {
		return !instruction.IsExecuted()
	})

	// 战舰移动指令（鼠标右键点击确定目标位置）
	if pos := action.DetectMouseButtonClickOnMap(m.state, ebiten.MouseButtonRight); pos != nil {
		if len(m.state.SelectedShips) != 0 {
			for _, shipUid := range m.state.SelectedShips {
				m.instructions[fmt.Sprintf("%s-%s", shipUid, instr.NameShipMove)] = instr.NewShipMove(shipUid, *pos)
			}
		}
	}

	// 随机散开，用于战舰重叠的情况（按下 X 键）
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyX) {
		if len(m.state.SelectedShips) != 0 {
			for _, shipUid := range m.state.SelectedShips {
				// 如果战舰不是静止状态，则散开指令无效
				if m.state.Ships[shipUid].CurSpeed != 0 {
					continue
				}
				// 随机散开 [-2, 2] 的范围
				x, y := rand.Intn(5)-2, rand.Intn(5)-2
				// 通过
				m.instructions[fmt.Sprintf("%s-%s", shipUid, instr.NameShipMove)] = instr.NewShipMove(
					shipUid, obj.NewMapPos(
						m.state.Ships[shipUid].CurPos.MX+x,
						m.state.Ships[shipUid].CurPos.MY+y,
					),
				)
			}
		}
	}

	// 按下 w 键，如果任意选中战舰任意武器被禁用，则启用所有，否则禁用所有
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyW) {
		if len(m.state.SelectedShips) != 0 {
			anyWeaponDisabled := false
			for _, shipUid := range m.state.SelectedShips {
				ship := m.state.Ships[shipUid]
				if ship.Weapon.GunDisabled || ship.Weapon.TorpedoDisabled {
					anyWeaponDisabled = true
					break
				}
			}
			for _, shipUid := range m.state.SelectedShips {
				if anyWeaponDisabled {
					instrKey := fmt.Sprintf("%s-%s", shipUid, instr.NameEnableWeapon)
					m.instructions[instrKey] = instr.NewEnableWeapon(shipUid, obj.WeaponTypeAll)
				} else {
					instrKey := fmt.Sprintf("%s-%s", shipUid, instr.NameDisableWeapon)
					m.instructions[instrKey] = instr.NewDisableWeapon(shipUid, obj.WeaponTypeAll)
				}
			}
		}
	}

	// 按下 g 键，如果任意选中战舰任意火炮被禁用，则启用所有，否则禁用所有
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyG) {
		if len(m.state.SelectedShips) != 0 {
			anyGunDisabled := false
			for _, shipUid := range m.state.SelectedShips {
				ship := m.state.Ships[shipUid]
				if ship.Weapon.GunDisabled {
					anyGunDisabled = true
					break
				}
			}
			for _, shipUid := range m.state.SelectedShips {
				if anyGunDisabled {
					instrKey := fmt.Sprintf("%s-%s", shipUid, instr.NameEnableWeapon)
					m.instructions[instrKey] = instr.NewEnableWeapon(shipUid, obj.WeaponTypeGun)
				} else {
					instrKey := fmt.Sprintf("%s-%s", shipUid, instr.NameDisableWeapon)
					m.instructions[instrKey] = instr.NewDisableWeapon(shipUid, obj.WeaponTypeGun)
				}
			}
		}
	}

	// 按下 t 键，如果任意选中战舰任意鱼雷被禁用，则启用所有，否则禁用所有
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyT) {
		if len(m.state.SelectedShips) != 0 {
			anyTorpedoDisabled := false
			for _, shipUid := range m.state.SelectedShips {
				ship := m.state.Ships[shipUid]
				if ship.Weapon.TorpedoDisabled {
					anyTorpedoDisabled = true
					break
				}
			}
			for _, shipUid := range m.state.SelectedShips {
				if anyTorpedoDisabled {
					instrKey := fmt.Sprintf("%s-%s", shipUid, instr.NameEnableWeapon)
					m.instructions[instrKey] = instr.NewEnableWeapon(shipUid, obj.WeaponTypeTorpedo)
				} else {
					instrKey := fmt.Sprintf("%s-%s", shipUid, instr.NameDisableWeapon)
					m.instructions[instrKey] = instr.NewDisableWeapon(shipUid, obj.WeaponTypeTorpedo)
				}
			}
		}
	}

	// 按下 d 键，全局展示 / 不展示所有战舰状态
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyD) {
		m.state.GameOpts.ForceDisplayState = !m.state.GameOpts.ForceDisplayState
	}
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
