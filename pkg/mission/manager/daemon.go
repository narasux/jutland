package manager

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/common/constants"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/object/trail"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// 逐条执行指令（移动/炮击/雷击/建造）
func (m *MissionManager) executeInstructions() {
	m.instructionSet.ExecAll(m.state)
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

// 更新建筑物
func (m *MissionManager) updateBuildings() {
	// 增援点当然算是建筑物！
	for _, rp := range m.state.ReinforcePoints {
		if ship := rp.Update(
			m.state.ShipUidGenerators[rp.BelongPlayer],
			// FIXME 目前电脑玩家先不限制金钱
			lo.Ternary(rp.BelongPlayer == m.state.CurPlayer, m.state.CurFunds, 50000),
		); ship != nil {
			m.state.Ships[ship.Uid] = ship
			if rp.BelongPlayer == m.state.CurPlayer {
				m.state.CurFunds -= ship.FundsCost
			}
			// 战舰移动到集结点 & 随机散开 [-3, 3] 的范围（通过 ShipMove 指令实现）
			x, y := rand.Intn(7)-3, rand.Intn(7)-3
			targetPos := objPos.New(rp.RallyPos.MX+x, rp.RallyPos.MY+y)
			m.instructionSet.Add(instr.NewShipMove(ship.Uid, targetPos))
		}
	}

	// 油井当然算是建筑物！
	fontSize := float64(24)
	for _, op := range m.state.OilPlatforms {
		text := fmt.Sprintf("+%d $", op.Yield)
		for _, ship := range m.state.Ships {
			if ship.Type != objUnit.ShipTypeCargo || ship.BelongPlayer != m.state.CurPlayer {
				continue
			}

			// TODO 目前是只要货轮在油井附近就给钱，后续考虑开采 & 存储 & 货轮容量的设计
			if ship.CurPos.Near(op.Pos, float64(op.Radius)) {
				op.AddShip(ship)
			} else {
				op.RemoveShip(ship.Uid)
			}
		}

		for uid, ship := range op.LoadingOilShips {
			// 如果货轮不在了，需要及时移除掉
			cargo, ok := m.state.Ships[uid]
			if !ok {
				op.RemoveShip(uid)
			} else if cargo.BelongPlayer == m.state.CurPlayer && ship.Update() {
				m.state.CurFunds += int64(ship.FundYield)
				mark := objMark.NewText(cargo.CurPos, text, fontSize, colorx.Gold, 50)
				m.state.GameMarks[mark.ID] = mark
			}
		}
	}
}

// 更新战舰武器开火相关状态
// TODO 开火逻辑优化：主炮/鱼雷向射程内最大的，生命值比例最少目标开火，副炮向最近的目标开火
func (m *MissionManager) updateShipWeaponFire() {
	maxBulletDiameter := 0
	isTorpedoLaunched := false

	for _, ship := range m.state.Ships {
		inRangeEnemies := []objUnit.Hurtable{}

		if target := m.state.Ships[ship.AttackTarget]; target != nil {
			// 如果有目标敌人，则检查是否在射程内，如果有直接选中即可
			if ship.CurPos.Distance(target.CurPos) < ship.Weapon.MaxToShipRange {
				inRangeEnemies = append(inRangeEnemies, target)
			}
		} else {
			// 敌机
			for _, enemy := range m.state.Planes {
				// 不能攻击己方的战机
				if ship.BelongPlayer == enemy.BelongPlayer {
					continue
				}
				// 如果不在 对空 最大射程内，跳过
				if ship.CurPos.Distance(enemy.CurPos) > ship.Weapon.MaxToPlaneRange {
					continue
				}
				inRangeEnemies = append(inRangeEnemies, enemy)
			}

			// 敌舰
			for enemyUid, enemy := range m.state.Ships {
				// 不能主动炮击己方的战舰（包括自己），目标敌人的也可以跳过（前面已处理）
				if ship.BelongPlayer == enemy.BelongPlayer ||
					enemyUid == ship.AttackTarget {
					continue
				}
				// 如果不在 对舰 最大射程内，跳过
				if ship.CurPos.Distance(enemy.CurPos) > ship.Weapon.MaxToShipRange {
					continue
				}
				inRangeEnemies = append(inRangeEnemies, enemy)
			}
		}

		if total := len(inRangeEnemies); total != 0 {
			// 射程内的敌人都会被攻击
			enemy := inRangeEnemies[rand.Intn(total)]
			bullets := ship.Fire(enemy)
			if len(bullets) == 0 {
				continue
			}
			// 镜头内的才统计
			if m.state.Camera.Contains(ship.CurPos) {
				for _, bt := range bullets {
					if bt.Type == objBullet.BulletTypeTorpedo {
						isTorpedoLaunched = true
					} else {
						// 只有炮弹才计算口径，鱼雷发射都是一个声音
						maxBulletDiameter = max(maxBulletDiameter, bt.Diameter)
					}
				}
			}
			m.state.ForwardingBullets = slices.Concat(m.state.ForwardingBullets, bullets)
		}
	}

	// 口径即是真理，只有最大的才能在本轮说话
	if maxBulletDiameter > 0 {
		audio.PlayAudioToEnd(audioRes.NewGunFire(maxBulletDiameter))
	} else if isTorpedoLaunched {
		audio.PlayAudioToEnd(audioRes.NewTorpedoLaunch())
	}
}

// 飞机出动 & 攻击
func (m *MissionManager) updatePlaneAttackOrReturn() {
	for _, ship := range m.state.Ships {
		// 战舰上没有飞机的，跳过
		if !ship.Aircraft.HasPlane {
			continue
		}

		inRangeEnemies := []objUnit.Hurtable{}
		// TODO 目前飞机目标都没考虑是否在攻击范围内，未来还是需要考虑的
		if target := m.state.Ships[ship.AttackTarget]; target != nil {
			// 如果有目标敌人，则直接选中即可
			inRangeEnemies = append(inRangeEnemies, target)
		} else {
			// 敌机
			for _, enemy := range m.state.Planes {
				// 不能攻击己方的战机
				if ship.BelongPlayer == enemy.BelongPlayer {
					continue
				}
				inRangeEnemies = append(inRangeEnemies, enemy)
			}
			// 敌舰
			for _, enemy := range m.state.Ships {
				// 不能主动炮击己方的战舰（包括自己），目标敌人的也可以跳过（前面已处理）
				if ship.BelongPlayer == enemy.BelongPlayer {
					continue
				}
				inRangeEnemies = append(inRangeEnemies, enemy)
			}
		}

		if total := len(inRangeEnemies); total != 0 {
			// 射程内的敌人都会被攻击
			enemy := inRangeEnemies[rand.Intn(total)]
			plane := ship.Aircraft.TakeOff(ship, enemy.ObjType())
			// 没有合适的飞机，那就跳过
			if plane == nil {
				continue
			}
			// 加入到对局飞机数据集中
			m.state.Planes[plane.Uid] = plane
			// 给飞机下达攻击指令
			m.instructionSet.Add(instr.NewPlaneAttack(plane.Uid, enemy.ObjType(), enemy.ID()))
		}
	}

	for _, plane := range m.state.Planes {
		// 剩余燃料为 0，需要返航
		if plane.MustReturn() {
			// 添加返航指令
			m.instructionSet.Add(instr.NewPlaneReturn(plane.Uid))
			continue
		}
		instrUid := instr.GenInstrUid(instr.NamePlaneAttack, plane.Uid)
		// 如果战机已经有攻击目标，则跳过
		if m.instructionSet.Exists(instrUid) {
			continue
		}

		// 有剩余燃料 & 没有攻击目标，按攻击类型选一个新的
		inRangeEnemies := []objUnit.Hurtable{}

		if plane.AttackObjType() == object.TypePlane {
			// 敌机
			for _, enemy := range m.state.Planes {
				// 不能攻击己方的战机
				if plane.BelongPlayer == enemy.BelongPlayer {
					continue
				}
				inRangeEnemies = append(inRangeEnemies, enemy)
			}
		} else if plane.AttackObjType() == object.TypeShip {
			// 敌舰
			for _, enemy := range m.state.Ships {
				// 不能主动炮击己方的战舰（包括自己），目标敌人的也可以跳过（前面已处理）
				if plane.BelongPlayer == enemy.BelongPlayer {
					continue
				}
				inRangeEnemies = append(inRangeEnemies, enemy)
			}
		}
		if total := len(inRangeEnemies); total != 0 {
			// 射程内的敌人都会被攻击
			enemy := inRangeEnemies[rand.Intn(total)]
			// 给飞机下达攻击指令
			m.instructionSet.Add(instr.NewPlaneAttack(plane.Uid, enemy.ObjType(), enemy.ID()))
		} else {
			// 没有可攻击对象，返航
			m.instructionSet.Add(instr.NewPlaneReturn(plane.Uid))
		}
	}
}

// 更新战机武器开火相关状态
func (m *MissionManager) updatePlaneWeaponFire() {
	bombReleased, torpedoLaunched := false, false

	for _, plane := range m.state.Planes {
		inRangeEnemies := []objUnit.Hurtable{}
		// 敌机
		for _, enemy := range m.state.Planes {
			// 不能攻击己方的战机（包括自己）
			if plane.BelongPlayer == enemy.BelongPlayer {
				continue
			}
			// 如果不在 对空 最大射程内，跳过
			if plane.CurPos.Distance(enemy.CurPos) > plane.Weapon.MaxToPlaneRange {
				continue
			}
			inRangeEnemies = append(inRangeEnemies, enemy)
		}
		// 敌舰
		for _, enemy := range m.state.Ships {
			// 不能攻击自己，也不能攻击己方的战舰
			if plane.BelongPlayer == enemy.BelongPlayer {
				continue
			}
			// 如果不在 对舰 最大射程内，跳过
			if plane.CurPos.Distance(enemy.CurPos) > plane.Weapon.MaxToShipRange {
				continue
			}
			inRangeEnemies = append(inRangeEnemies, enemy)
		}
		if total := len(inRangeEnemies); total != 0 {
			// 射程内的敌人都会被攻击
			enemy := inRangeEnemies[rand.Intn(total)]
			bullets := plane.Fire(enemy)
			if len(bullets) == 0 {
				continue
			}
			// 镜头内的才统计
			if m.state.Camera.Contains(plane.CurPos) {
				for _, bt := range bullets {
					// 统计一个就好，不要吵吵
					if bombReleased || torpedoLaunched {
						break
					}
					if bt.Type == objBullet.BulletTypeBomb {
						bombReleased = true
					} else if bt.Type == objBullet.BulletTypeTorpedo {
						torpedoLaunched = true
					}
				}
			}
			m.state.ForwardingBullets = slices.Concat(m.state.ForwardingBullets, bullets)
		}
	}

	if bombReleased {
		// 有炸弹就发声音
		audio.PlayAudioToEnd(audioRes.NewBombSpawn())
	} else if torpedoLaunched {
		// 有鱼雷就发声音
		audio.PlayAudioToEnd(audioRes.NewTorpedoLaunch())
	}
}

// 更新尾流状态（战舰，鱼雷，炮弹）
func (m *MissionManager) updateObjectTrails() {
	for i := 0; i < len(m.state.Trails); i++ {
		m.state.Trails[i].Update()
	}
	// 生命周期结束的，不再需要
	m.state.Trails = lo.Filter(m.state.Trails, func(t *trail.Trail, _ int) bool {
		return t.IsAlive()
	})
	for _, ship := range m.state.Ships {
		if trails := ship.GenTrails(); trails != nil {
			m.state.Trails = append(m.state.Trails, trails...)
		}
	}
	for _, bt := range m.state.ForwardingBullets {
		// 炸弹目前没有尾流
		if bt.Type == objBullet.BulletTypeBomb {
			continue
		}
		if trails := bt.GenTrails(); trails != nil {
			m.state.Trails = append(m.state.Trails, trails...)
		}
	}
}

// 更新弹药状态
func (m *MissionManager) updateShotBullets() {
	for i := 0; i < len(m.state.ForwardingBullets); i++ {
		m.state.ForwardingBullets[i].Forward()
	}

	// 结算伤害
	resolveDamage := func(bt *objBullet.Bullet) bool {
		prevPos := bt.CurPos.Copy()
		prevPos.SubRx(math.Sin(bt.Rotation*math.Pi/180) * bt.Speed)
		prevPos.AddRy(math.Cos(bt.Rotation*math.Pi/180) * bt.Speed)

		switch bt.TargetObjType {
		case object.TypeShip:
			for _, ship := range m.state.Ships {
				// 总不能不小心打死自己吧，真是不应该 :D
				if bt.Shooter == ship.Uid {
					continue
				}
				// 如果友军伤害没启用，则不对己方战舰造成伤害
				if !m.state.GameOpts.FriendlyFire && bt.BelongPlayer == ship.BelongPlayer {
					continue
				}

				if bt.ShotType == objBullet.BulletShotTypeDirect {
					// 直射则检查线段是否与矩形相交
					if geometry.IsSegmentIntersectRotatedRectangle(
						prevPos.RX, prevPos.RY,
						bt.CurPos.RX, bt.CurPos.RY,
						ship.CurPos.RX, ship.CurPos.RY,
						// 转换成实际地图上的尺寸
						ship.Length/constants.MapBlockSize,
						ship.Width/constants.MapBlockSize,
						ship.CurRotation,
					) {
						ship.HurtBy(bt)
						bt.HitObjType = object.TypeShip
						break
					}
				} else if bt.ShotType == objBullet.BulletShotTypeArcing {
					// 弧线炮弹，只要命中一个目标，就不再继续搜索
					if geometry.IsPointInRotatedRectangle(
						prevPos.RX, prevPos.RY,
						ship.CurPos.RX, ship.CurPos.RY,
						// 转换成实际地图上的尺寸
						ship.Length/constants.MapBlockSize,
						ship.Width/constants.MapBlockSize,
						ship.CurRotation,
					) {
						ship.HurtBy(bt)
						bt.HitObjType = object.TypeShip
						break
					}
				}
			}
		case object.TypePlane:
			for _, plane := range m.state.Planes {
				// 总不能不小心打死自己吧，真是不应该 :D
				if bt.Shooter == plane.Uid {
					continue
				}
				// 如果友军伤害没启用，则不对己方战舰造成伤害
				if !m.state.GameOpts.FriendlyFire && bt.BelongPlayer == plane.BelongPlayer {
					continue
				}
				// 如果是舰对空，设置 7/8 的 “擦肩而过” 率，现在命中率太高了
				if bt.ShooterObjType == object.TypeShip {
					if rand.Intn(8) == 0 {
						continue
					}
				}

				// 对空射击都认为是直射，检查线段是否与矩形相交
				if geometry.IsSegmentIntersectRotatedRectangle(
					prevPos.RX, prevPos.RY,
					bt.CurPos.RX, bt.CurPos.RY,
					plane.CurPos.RX, plane.CurPos.RY,
					// 转换成实际地图上的尺寸
					plane.Length/constants.MapBlockSize,
					plane.Width/constants.MapBlockSize,
					plane.CurRotation,
				) {
					plane.HurtBy(bt)
					bt.HitObjType = object.TypePlane
					break
				}
			}
		default:
			return false
		}
		return bt.HitObjType != object.TypeNone
	}

	arrivedBullets, forwardingBullets := []*objBullet.Bullet{}, []*objBullet.Bullet{}
	for _, bt := range m.state.ForwardingBullets {
		// 迷失的弹药，要及时消亡（如鱼雷没命中）
		if bt.Life <= 0 {
			// TODO 其实还应该判断下，可能是 HitLand，后面再做吧
			bt.HitObjType = object.TypeWater
			continue
		}
		if bt.ShotType == objBullet.BulletShotTypeArcing {
			// 曲射炮弹只要到达目的地，就不会再走了（只有到目的地才有伤害）
			if bt.CurPos.Near(bt.TargetPos, 0.05) {
				if !resolveDamage(bt) {
					// TODO 其实还应该判断下，可能是 HitLand，后面再做吧
					bt.HitObjType = object.TypeWater
				}
				arrivedBullets = append(arrivedBullets, bt)
			} else {
				forwardingBullets = append(forwardingBullets, bt)
			}
		} else if bt.ShotType == objBullet.BulletShotTypeDirect {
			// 鱼雷碰撞到陆地，应该不再前进
			if bt.Type == objBullet.BulletTypeTorpedo && m.state.MissionMD.MapCfg.Map.IsLand(bt.CurPos.MX, bt.CurPos.MY) {
				bt.HitObjType = object.TypeLand
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
		// 击中的是战舰 / 飞机，才会有伤害数值
		if bt.HitObjType != object.TypeShip && bt.HitObjType != object.TypePlane {
			continue
		}

		if m.state.GameOpts.DisplayDamageNumber {
			fontSize, clr := 0.0, colorx.White
			switch bt.CriticalType {
			case objBullet.CriticalTypeNone:
				fontSize, clr = float64(16), colorx.White
			case objBullet.CriticalTypeThreeTimes:
				fontSize, clr = float64(20), colorx.Yellow
			case objBullet.CriticalTypeTenTimes:
				fontSize, clr = float64(24), colorx.Red
			}
			// DEBUG: 调试用逻辑，区分敌我伤害
			if m.state.DebugFlags.DamageColorByTeam {
				if bt.BelongPlayer == m.state.CurPlayer {
					clr = colorx.Cyan
				} else {
					clr = colorx.DarkRed
				}
			}
			// 如果是大于 1 的，则取整，否则保留两位小数
			flagText := lo.Ternary(
				bt.RealDamage > 1,
				strconv.Itoa(int(bt.RealDamage)),
				fmt.Sprintf("%.2f", bt.RealDamage),
			)
			mark := objMark.NewText(bt.CurPos, flagText, fontSize, clr, 20)
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
			ship.CurHP = textureImg.MaxShipExplodeState

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
		// 支持逐渐减速的效果，而不是直接就变成 0
		ship.CurSpeed = max(0, ship.CurSpeed-ship.MaxSpeed/30)
	}

	// 移除已经完全消亡的战舰
	m.state.DestroyedShips = lo.Filter(
		m.state.DestroyedShips, func(ship *objUnit.BattleShip, _ int) bool { return ship.CurHP > 0 },
	)
}

// 更新局内战机
func (m *MissionManager) updateMissionPlanes() {
	// 如果战机 HP 为 0，则需要走消亡流程
	for uid, plane := range m.state.Planes {
		if plane.CurHP <= 0 {
			// 这里做了取巧，复用 CurHP 用于后续渲染爆炸效果
			plane.CurHP = textureImg.MaxPlaneExplodeState

			// TODO 评估是否要参考战舰爆炸一样播放声音？
			m.state.DestroyedPlanes = append(m.state.DestroyedPlanes, plane)
			delete(m.state.Planes, uid)
		}
	}

	// 消亡中的战机会逐渐掉血到 0
	for _, plane := range m.state.DestroyedPlanes {
		plane.CurHP -= 0.5
		// FIXME 如何模拟出被击落的效果？
	}

	// 移除已经完全消亡的战舰
	m.state.DestroyedPlanes = lo.Filter(
		m.state.DestroyedPlanes, func(plane *objUnit.Plane, _ int) bool { return plane.CurHP > 0 },
	)
}

// 计算下一帧任务状态
func (m *MissionManager) updateMissionStatus() {
	calcNextStatusByShips := func(curStatus state.MissionStatus) state.MissionStatus {
		// 还有战舰在沉没，游戏继续
		if len(m.state.DestroyedShips) != 0 {
			return curStatus
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
			return state.MissionFailed
		}
		// 敌人都不存在，胜利
		if !anyEnemyShip && len(m.state.DestroyedShips) == 0 {
			return state.MissionSuccess
		}
		return curStatus
	}

	switch m.state.MissionStatus {
	case state.MissionRunning:
		// 暂停游戏
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			m.state.MissionStatus = state.MissionPaused
		}
		m.state.MissionStatus = calcNextStatusByShips(m.state.MissionStatus)
	case state.MissionPaused:
		if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
			m.state.MissionStatus = state.MissionFailed
		} else if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			m.state.MissionStatus = state.MissionRunning
		}
	case state.MissionInMap:
		// 退出全屏地图模式
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			m.state.MissionStatus = state.MissionRunning
		}
		m.state.MissionStatus = calcNextStatusByShips(m.state.MissionStatus)
	case state.MissionInTerminal:
		// 退出终端模式
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			m.state.MissionStatus = state.MissionRunning
		}
		return
	case state.MissionInBuilding:
		// 退出建筑物交互模式
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			m.state.MissionStatus = state.MissionRunning
		}
	default:
		m.state.MissionStatus = state.MissionRunning
	}

	// 按下 m 键，切换地图展示模式
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		m.state.MissionStatus = lo.Ternary(
			m.state.MissionStatus != state.MissionInMap,
			state.MissionInMap,
			state.MissionRunning,
		)
	}

	// 按下 b 键，开启查看增援点模式
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		m.state.MissionStatus = lo.Ternary(
			m.state.MissionStatus != state.MissionInBuilding,
			state.MissionInBuilding,
			state.MissionRunning,
		)
	}

	// 按下 LeftCtrl，LeftShift 的同时按下 ` 键开启终端
	if ebiten.IsKeyPressed(ebiten.KeyControlLeft) &&
		ebiten.IsKeyPressed(ebiten.KeyShiftLeft) &&
		inpututil.IsKeyJustPressed(ebiten.KeyBackquote) {
		m.state.MissionStatus = state.MissionInTerminal
		audio.PlayAudioToEnd(audioRes.NewCheating())
		// 进入终端会按下 ctrl，此时会导致进入编组模式，需要强制退出下
		m.state.IsGrouping = false
	}
}
