package mission

import (
	"log"
	"math"
	"slices"
	"strconv"

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
	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
	"github.com/narasux/jutland/pkg/resources/images/bullet"
	"github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
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
func NewManager(mission string) *MissionManager {
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
		m.updateInstructions()
		m.executeInstructions()
		m.updateCameraPosition()
		m.updateGameOptions()
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
			log.Printf("Instruction %s exec error: %s\n", i.String(), err)
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

	// 按下 n 键，全局展示 / 不展示所有伤害数字
	if action.DetectKeyboardKeyJustPressed(ebiten.KeyN) {
		m.state.GameOpts.DisplayDamageNumber = !m.state.GameOpts.DisplayDamageNumber
	}
}

// 更新游戏标识
func (m *MissionManager) updateGameMarks() {
	for markID, mark := range m.state.GameMarks {
		mark.Life--

		// 检查游戏标识，如果生命值为 0，则删除
		if mark.Life <= 0 {
			delete(m.state.GameMarks, markID)
		}
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

			// 如果当前选中的分组不是当前按键的分组，则更新记录
			if m.state.SelectedGroupID != groupID {
				m.state.SelectedGroupID = groupID
			} else {
				// 如果当前选中的分组再次被选中，移动相机中心位置到当前分组的第一艘战舰处
				if len(m.state.SelectedShips) > 0 {
					m.state.Camera.Pos = m.state.Ships[m.state.SelectedShips[0]].CurPos.Copy()
					m.state.Camera.Pos.SubMx(m.state.Camera.Width / 2)
					m.state.Camera.Pos.SubMy(m.state.Camera.Height / 2)
				}
			}
		}
	}

	// 检查选中的战舰，如果已经被摧毁，则要去掉
	m.state.SelectedShips = lo.Filter(m.state.SelectedShips, func(uid string, _ int) bool {
		ship, ok := m.state.Ships[uid]
		return ok && ship != nil && ship.CurHP > 0
	})
	// 没有战舰被选中，应该重置 SelectedGroupID
	if m.state.SelectedGroupID != obj.GroupIDNone && len(m.state.SelectedShips) == 0 {
		m.state.SelectedGroupID = obj.GroupIDNone
	}
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

// 更新武器开火相关状态，目前是攻击最近的目标
// TODO 开火逻辑优化：主炮/鱼雷向射程内最大的，生命值比例最少目标开火，副炮向最近的目标开火
func (m *MissionManager) updateWeaponFire() {
	maxBulletDiameter := 0
	isTorpedoLaunched := false

	for shipUid, ship := range m.state.Ships {
		var nearestEnemy *obj.BattleShip
		var nearestEnemyDistance float64 = 0

		for enemyUid, enemy := range m.state.Ships {
			// 不能炮击自己，也不能主动炮击己方的战舰
			if shipUid == enemyUid || ship.BelongPlayer == enemy.BelongPlayer {
				continue
			}
			// 如果不在最大射程内，跳过
			distance := geometry.CalcDistance(ship.CurPos.RX, ship.CurPos.RY, enemy.CurPos.RX, enemy.CurPos.RY)
			if distance > ship.Weapon.MaxRange {
				continue
			}
			// 找到最近的敌人
			if nearestEnemy == nil || distance < nearestEnemyDistance {
				nearestEnemy = enemy
				nearestEnemyDistance = distance
			}
		}
		if nearestEnemy != nil {
			bullets := ship.Fire(nearestEnemy)
			if len(bullets) == 0 {
				continue
			}
			// 镜头内的才统计
			if m.state.Camera.Contains(ship.CurPos) {
				for _, bt := range bullets {
					if bt.Type == obj.BulletTypeTorpedo {
						isTorpedoLaunched = true
					} else {
						// 只有炮弹才计算口径，鱼雷发射都是一个声音
						maxBulletDiameter = max(maxBulletDiameter, bt.Diameter)
					}
				}
			}
			m.state.ForwardingBullets = slices.Concat(m.state.ForwardingBullets, bullets)
			break
		}
	}

	// 口径即是真理，只有最大的才能在本轮说话
	if maxBulletDiameter > 0 {
		audio.PlayAudioToEnd(audioRes.NewGunFire(maxBulletDiameter))
	} else if isTorpedoLaunched {
		audio.PlayAudioToEnd(audioRes.NewTorpedoLaunch())
	}
}

// 更新尾流状态（战舰，鱼雷，炮弹）
func (m *MissionManager) updateObjectTrails() {
	for i := 0; i < len(m.state.Trails); i++ {
		m.state.Trails[i].Update()
	}
	// 生命周期结束的，不再需要
	m.state.Trails = lo.Filter(m.state.Trails, func(t *obj.Trail, _ int) bool {
		return t.IsAlive()
	})
	for _, ship := range m.state.Ships {
		if ship.CurSpeed > 0 {
			offset := ship.Length / constants.MapBlockSize
			sinVal := math.Sin(ship.CurRotation * math.Pi / 180)
			cosVal := math.Cos(ship.CurRotation * math.Pi / 180)

			frontPos, backPos := ship.CurPos.Copy(), ship.CurPos.Copy()
			frontPos.AddRx(sinVal * offset * 0.25)
			frontPos.SubRy(cosVal * offset * 0.25)
			backPos.SubRx(sinVal * offset * 0.2)
			backPos.AddRy(cosVal * offset * 0.2)

			m.state.Trails = append(
				m.state.Trails,
				obj.NewTrail(
					frontPos,
					texture.TrailShapeCircle,
					ship.Width*0.6,
					1.2,
					ship.Length/6+150*ship.CurSpeed,
					1,
					0,
					0,
				),
				obj.NewTrail(
					backPos,
					texture.TrailShapeCircle,
					ship.Width,
					0.4,
					ship.Length/7+155*ship.CurSpeed,
					1.5,
					0,
					0,
				),
			)
		}
	}
	for _, bt := range m.state.ForwardingBullets {
		if bt.HitObjectType == obj.HitObjectTypeNone && bt.ForwardAge > 10 {
			diffusionRate, multipleSizeAsLife, lifeReductionRate := 0.1, 7.0, 2.0
			if bt.Type == obj.BulletTypeTorpedo {
				diffusionRate, multipleSizeAsLife, lifeReductionRate = 0.5, 8.0, 3.0
			}
			size := float64(bullet.GetImgWidth(bt.Name, bt.Type, bt.Diameter))
			m.state.Trails = append(
				m.state.Trails,
				obj.NewTrail(
					bt.CurPos,
					texture.TrailShapeRect,
					size,
					diffusionRate,
					size*multipleSizeAsLife,
					lifeReductionRate,
					0,
					bt.Rotation,
				),
			)
		}
	}
}

// 更新弹药状态
func (m *MissionManager) updateShotBullets() {
	for i := 0; i < len(m.state.ForwardingBullets); i++ {
		m.state.ForwardingBullets[i].Forward()
	}

	// 结算伤害
	resolveDamage := func(bt *obj.Bullet) bool {
		for _, ship := range m.state.Ships {
			// 总不能不小心打死自己吧，真是不应该 :D
			if bt.BelongShip == ship.Uid {
				continue
			}
			// 如果友军伤害没启用，则不对己方战舰造成伤害
			if !m.state.GameOpts.FriendlyFire && bt.BelongPlayer == ship.BelongPlayer {
				continue
			}

			// 如果是直射弹药，要检查多个等分点，避免速度过快，只在终点算伤害（虚空过穿）
			// 以最小宽度 10 计算，速度在 0.3125 内时，4 个等分点内不会丢失伤害，超过则需要倍增
			speedInterval := 0.3125
			checkPoint := lo.Ternary(
				bt.ShotType == obj.BulletShotTypeDirect, 4*int(math.Ceil(bt.Speed/speedInterval)), 1,
			)
			for cp := 0; cp < checkPoint; cp++ {
				pos := bt.CurPos.Copy()
				pos.SubRx(math.Sin(bt.Rotation*math.Pi/180) * bt.Speed / float64(checkPoint) * float64(cp))
				pos.AddRy(math.Cos(bt.Rotation*math.Pi/180) * bt.Speed / float64(checkPoint) * float64(cp))

				// 判定命中，扣除战舰生命值，标记命中战舰
				if geometry.IsPointInRotatedRectangle(
					pos.RX, pos.RY, ship.CurPos.RX, ship.CurPos.RY,
					ship.Length/constants.MapBlockSize, ship.Width/constants.MapBlockSize,
					ship.CurRotation,
				) {
					ship.Hurt(bt)
					bt.HitObjectType = obj.HitObjectTypeShip
					return true
				}
			}
		}
		return false
	}

	arrivedBullets, forwardingBullets := []*obj.Bullet{}, []*obj.Bullet{}
	for _, bt := range m.state.ForwardingBullets {
		// 迷失的弹药，要及时消亡（如鱼雷没命中）
		if bt.Life <= 0 {
			// TODO 其实还应该判断下，可能是 HitLand，后面再做吧
			bt.HitObjectType = obj.HitObjectTypeWater
			continue
		}
		if bt.ShotType == obj.BulletShotTypeArcing {
			// 曲射炮弹只要到达目的地，就不会再走了（只有到目的地才有伤害）
			// FIXME 这里可能会有问题，MEqual 太粗略了，会结算异常
			if bt.CurPos.MEqual(bt.TargetPos) {
				if resolveDamage(bt) {
					bt.HitObjectType = obj.HitObjectTypeShip
				} else {
					// TODO 其实还应该判断下，可能是 HitLand，后面再做吧
					bt.HitObjectType = obj.HitObjectTypeWater
				}
				arrivedBullets = append(arrivedBullets, bt)
			} else {
				forwardingBullets = append(forwardingBullets, bt)
			}
		} else if bt.ShotType == obj.BulletShotTypeDirect {
			// 鱼雷碰撞到陆地，应该不再前进
			if bt.Type == obj.BulletTypeTorpedo && m.state.MissionMD.MapCfg.Map.IsLand(bt.CurPos.MX, bt.CurPos.MY) {
				bt.HitObjectType = obj.HitObjectTypeLand
				arrivedBullets = append(arrivedBullets, bt)
			} else if resolveDamage(bt) {
				// 鱼雷 / 直射炮弹没有目的地的说法，碰到就爆炸
				arrivedBullets = append(arrivedBullets, bt)
			} else {
				forwardingBullets = append(forwardingBullets, bt)
			}
		}
	}

	// 继续塔塔开的，保留
	m.state.ForwardingBullets = forwardingBullets
	// 已经到达目标地点的，转换成爆炸 & 伤害数值
	// TODO 支持命中爆炸
	for _, bt := range arrivedBullets {
		if bt.HitObjectType != obj.HitObjectTypeShip {
			continue
		}

		if m.state.GameOpts.DisplayDamageNumber {
			fontSize, clr := 0.0, colorx.White
			switch bt.CriticalType {
			case obj.CriticalTypeNone:
				fontSize, clr = float64(16), colorx.White
			case obj.CriticalTypeThreeTimes:
				fontSize, clr = float64(20), colorx.Yellow
			case obj.CriticalTypeTenTimes:
				fontSize, clr = float64(24), colorx.Red
			}
			mark := obj.NewTextMark(bt.CurPos, strconv.Itoa(int(bt.RealDamage)), fontSize, clr, 20)
			m.state.GameMarks[mark.ID] = mark
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

// GetState 获取当前任务状态
func (m *MissionManager) GetState() *state.MissionState {
	return m.state
}
