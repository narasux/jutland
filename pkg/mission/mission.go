package mission

import (
	"log"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/common/constants"
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
	"github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/geometry"
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
	m.updateShipGroups()
	m.updateWeaponFire()
	m.updateShipTrails()
	m.updateShotBullets()
	m.updateMissionShips()
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
	}
	// 正在分组中，不可用
	if !m.state.IsGrouping {
		// 通过分组选中战舰
		groupID := action.GetGroupIDByPressedKey()
		if groupID != obj.GroupIDNone {
			shipInGroup := lo.Filter(lo.Values(m.state.Ships), func(ship *obj.BattleShip, _ int) bool {
				return ship.BelongPlayer == m.state.CurPlayer && ship.GroupID == groupID
			})
			m.state.SelectedShips = lo.Map(shipInGroup, func(ship *obj.BattleShip, _ int) string {
				return ship.Uid
			})
		}
	}

	// 检查选中的战舰，如果已经被摧毁，则要去掉
	m.state.SelectedShips = lo.Filter(m.state.SelectedShips, func(uid string, _ int) bool {
		ship, ok := m.state.Ships[uid]
		return ok && ship != nil && ship.CurHP > 0
	})
}

// 更新舰队编组状态（左 Ctrl + 0-9 编组）
func (m *MissionManager) updateShipGroups() {
	// 按下左边的 ctrl 键：进入 / 退出编组模式
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyControlLeft) {
		m.state.IsGrouping = !m.state.IsGrouping
	}
	// 设置编组后，如果松开 ctrl，则退出编组模式
	if action.DetectKeyboardKeyJustReleased(ebiten.KeyControlLeft) {
		m.state.IsGrouping = false
	}
	// 没有在编组模式，直接返回
	if !m.state.IsGrouping {
		return
	}
	groupID := action.GetGroupIDByPressedKey()
	// 没有设置合法的编组
	if groupID == obj.GroupIDNone {
		return
	}
	// 重新编组，只有当前选中的拥有这个编组
	for _, ship := range m.state.Ships {
		if ship.GroupID == groupID {
			ship.GroupID = obj.GroupIDNone
		}
	}
	for _, shipUid := range m.state.SelectedShips {
		m.state.Ships[shipUid].GroupID = groupID
	}
}

// 更新武器开火相关状态
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
				bullets := ship.Fire(enemy)
				if len(bullets) == 0 {
					continue
				}
				// 炮击声音（只有战舰在视野内才播放声音）
				// FIXME 需要修复声音重叠变大声的问题
				if audioPlayQuota > 0 && m.state.Camera.Contains(ship.CurPos) {
					audio.PlayAudioToEnd(audioRes.NewGunMK45())
					audioPlayQuota--
				}
				m.state.ForwardingBullets = slices.Concat(m.state.ForwardingBullets, bullets)
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
	for i := 0; i < len(m.state.ForwardingBullets); i++ {
		m.state.ForwardingBullets[i].Forward()
	}

	arrivedBullets, forwardingBullets := []*obj.Bullet{}, []*obj.Bullet{}
	for _, bullet := range m.state.ForwardingBullets {
		// 迷失的弹药，要及时消亡（如鱼雷没命中）
		if bullet.Life <= 0 {
			continue
		}
		if bullet.CurPos.MEqual(bullet.TargetPos) {
			arrivedBullets = append(arrivedBullets, bullet)
		} else {
			forwardingBullets = append(forwardingBullets, bullet)
		}
	}

	// 继续塔塔开的，保留
	m.state.ForwardingBullets = forwardingBullets
	// 已经到达目标地点的，存起来绘图用 FIXME 这里有个问题，自己赋值，会不会把正在绘制动画的搞没了
	m.state.ArrivedBullets = arrivedBullets
	// 尝试结算伤害
	for _, bt := range m.state.ArrivedBullets {
		for _, ship := range m.state.Ships {
			// 如果友军伤害没启用，则不对己方战舰造成伤害
			if !m.state.GameOpts.FriendlyFire && bt.BelongPlayer == ship.BelongPlayer {
				continue
			}

			// 判定命中，扣除战舰生命值，标记命中战舰
			if geometry.IsPointInRotatedRectangle(
				bt.CurPos.RX, bt.CurPos.RY, ship.CurPos.RX, ship.CurPos.RY,
				ship.Length/constants.MapBlockSize, ship.Width/constants.MapBlockSize,
				ship.CurRotation,
			) {
				ship.Hurt(bt.Damage)
				bt.HitShip = true
			} else {
				// TODO 其实还应该判断下，可能是 HitLand，后面再做吧
				bt.HitWater = true
			}
		}
	}
}

// 更新局内战舰
func (m *MissionManager) updateMissionShips() {
	audioPlayQuota := 2
	// 如果战舰 HP 为 0，则需要走消亡流程
	for uid, ship := range m.state.Ships {
		if ship.CurHP <= 0 {
			// 这里做了取巧，复用 CurHP 用于后续渲染爆炸效果
			ship.CurHP = texture.MaxExplodeState
			ship.CurSpeed = 0

			if audioPlayQuota > 0 && m.state.Camera.Contains(ship.CurPos) {
				audio.PlayAudioToEnd(audioRes.NewShipExplode())
				audioPlayQuota--
			}

			m.state.DestroyedShips = append(m.state.DestroyedShips, ship)
			delete(m.state.Ships, uid)
		}
	}

	// 消亡中的战舰会逐渐掉血到 0
	for _, ship := range m.state.DestroyedShips {
		ship.CurHP -= 0.5
	}

	// 移除已经完全消亡的战舰
	m.state.DestroyedShips = lo.Filter(m.state.DestroyedShips, func(ship *obj.BattleShip, _ int) bool {
		return ship.CurHP > 0
	})
}

// 计算下一帧任务状态
func (m *MissionManager) updateMissionStatus() {
	switch m.state.MissionStatus {
	case state.MissionRunning:
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			m.state.MissionStatus = state.MissionPaused
			return
		}
		// 检查所有战舰，判定胜利 / 失败
		anySelfShip, anyEnemyShip := false, false
		for _, ship := range m.state.Ships {
			if ship.BelongPlayer == m.state.CurPlayer {
				anySelfShip = true
			} else {
				anyEnemyShip = true
			}
		}
		// 自己的船都没了，失败
		if !anySelfShip && len(m.state.DestroyedShips) == 0 {
			m.state.MissionStatus = state.MissionFailed
			return
		}
		// 敌人都不存在，胜利
		if !anyEnemyShip && len(m.state.DestroyedShips) == 0 {
			m.state.MissionStatus = state.MissionSuccess
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
