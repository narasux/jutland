package manager

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"

	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/common/constants"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objExplosion "github.com/narasux/jutland/pkg/mission/object/explosion"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// bombBlastRadius 炸弹/曲射炮弹落点的爆炸波及半径（地图格）：
// 半径内的地面飞机（停放与滑行中）都会受到伤害，而不是要求落点
// 精确落在机身矩形内。
const bombBlastRadius = 1.0

// 舰对空弹药的单发命中率（每结算帧独立判定，数值为平衡用常数）：
//   - 小口径速射防空炮（≤40mm）：弹幕密集，命中率最高；
//   - 中口径高平两用炮（≤155mm）：76~155mm 弹靠少量直击与近炸破片；
//   - 大口径舰炮（>155mm）：弹丸散布远大于机体，直击概率极低。
//
// 俯冲轰炸机贴舰俯冲时姿态与高度变化剧烈，防空弹幕难以构成提前量，
// 命中率统一再降为三分之一。
func aaHitChance(diameter int, planeType objUnit.PlaneType) float64 {
	var chance float64
	switch {
	case diameter <= 40:
		chance = 1.0 / 6
	case diameter <= 155:
		chance = 1.0 / 12
	default:
		chance = 1.0 / 18
	}
	if planeType == objUnit.PlaneTypeDiveBomber {
		chance /= 3
	}
	return chance
}

// damageGroundPlanesNear 落点爆炸波及范围内的地面飞机结算伤害，
// excludeUid 用于排除已被直接命中的目标；返回是否造成伤害。
func (m *MissionManager) damageGroundPlanesNear(bt *objBullet.Bullet, excludeUid string) bool {
	hit := false
	for uid, plane := range m.state.Arena.Planes {
		if uid == excludeUid || !plane.IsOnGround() {
			continue
		}
		// 如果友军伤害没启用，则不对己方地面飞机造成伤害
		if !m.state.UI.GameOpts.FriendlyFire && bt.BelongPlayer == plane.BelongPlayer {
			continue
		}
		if bt.CurPos.Distance(plane.CurPos) <= bombBlastRadius {
			plane.HurtBy(bt)
			hit = true
		}
	}
	return hit
}

// 更新战舰武器开火相关状态
// TODO 开火逻辑优化：主炮/鱼雷向射程内最大的，生命值比例最少目标开火，副炮向最近的目标开火
func (m *MissionManager) updateShipWeaponFire() {
	maxBulletDiameter := 0
	isTorpedoLaunched := false
	isRocketLaunched := false

	for _, ship := range m.state.Arena.Ships {
		inRangeEnemies := []objUnit.Hurtable{}

		target := m.state.Arena.Ships[ship.AttackTarget]
		// 若有指定攻击目标且在射程内，则优先攻击该目标；否则扫描沿途其他敌人
		if target != nil && ship.CurPos.Distance(target.CurPos) < ship.Weapon.MaxToShipRange {
			inRangeEnemies = append(inRangeEnemies, target)
		} else {
			// 敌机
			for _, enemy := range m.state.Arena.Planes {
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
			for enemyUid, enemy := range m.state.Arena.Ships {
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
			if m.state.View.Camera.Contains(ship.CurPos) {
				for _, bt := range bullets {
					if bt.Type == objBullet.TypeTorpedo {
						isTorpedoLaunched = true
					} else if bt.Type == objBullet.TypeRocket {
						isRocketLaunched = true
					} else {
						// 只有炮弹才计算口径，鱼雷发射都是一个声音
						maxBulletDiameter = max(maxBulletDiameter, bt.Diameter)
					}
				}
			}
			m.state.Arena.ForwardingBullets = append(m.state.Arena.ForwardingBullets, bullets...)
		}
	}

	m.weaponFirePlayer.PlayShipFire(maxBulletDiameter, isTorpedoLaunched, isRocketLaunched)
}

// updatePlaneAttackOrReturn 负责飞机出动、目标重分配和返航决策。
// 出动机型由基地队列决定；已经锁定有效目标的飞机保持目标粘性。
func (m *MissionManager) updatePlaneAttackOrReturn() {
	for _, ship := range m.state.Arena.Ships {
		// 战舰上没有飞机的，跳过
		if !ship.Aircraft.HasPlane {
			continue
		}

		if target := m.state.Arena.Ships[ship.AttackTarget]; target != nil {
			plane := ship.Aircraft.TakeOff(ship, object.TypeShip)
			// 没有合适的飞机，那就跳过
			if plane == nil {
				continue
			}
			// 加入到对局飞机数据集中
			m.state.Arena.Planes[plane.Uid] = plane
			// 给飞机下达攻击指令
			m.instructionSet.Add(instr.NewPlaneAttack(plane.Uid, object.TypeShip, target.ID()))
			continue
		}

		plane, targetType, targetUID, ok := m.takeOffFromBase(ship)
		if ok {
			m.state.Arena.Planes[plane.Uid] = plane
			m.instructionSet.Add(instr.NewPlaneAttack(plane.Uid, targetType, targetUID))
		}
	}

	for _, plane := range m.state.Arena.Planes {
		if !plane.IsCruising() {
			continue
		}
		// 剩余燃料为 0，需要返航
		if plane.MustReturn() {
			// 添加返航指令
			m.instructionSet.Add(instr.NewPlaneReturn(plane.Uid))
			continue
		}
		instrUid := instr.GenInstrUid(instr.NamePlaneAttack, plane.Uid)
		targetType := plane.AttackObjType()
		if plane.CurAttackTarget != "" && !m.targetReachableByPlane(plane, plane.CurAttackTarget, targetType) {
			m.instructionSet.Remove(instrUid)
			plane.CurAttackTarget = ""
			m.markTargetingDirty()
		}
		// 如果战机已经有攻击目标，则跳过
		if m.instructionSet.Exists(instrUid) {
			continue
		}

		if targetUID, ok := m.nextTargetUIDForPlane(plane); ok {
			m.instructionSet.Add(instr.NewPlaneAttack(plane.Uid, targetType, targetUID))
		} else {
			// 没有可攻击对象，返航
			m.instructionSet.Add(instr.NewPlaneReturn(plane.Uid))
		}
	}
}

// updateAirfieldAlertLaunch 机场警戒起飞：不设警戒范围，从异步目标计划中
// 消费对空 / 对舰基地队列。无计划时保持待命，不在主循环扫描全场敌人。
func (m *MissionManager) updateAirfieldAlertLaunch() {
	for _, af := range m.state.Arena.Airfields {
		if !af.CanAlertLaunch() {
			continue
		}
		for range 2 {
			plane, targetType, targetUID, ok := m.takeOffFromBase(af)
			if !ok {
				break
			}
			m.state.Arena.Planes[plane.Uid] = plane
			m.instructionSet.Add(instr.NewPlaneAttack(plane.Uid, targetType, targetUID))
		}
	}
}

// 更新战机武器开火相关状态
func (m *MissionManager) updatePlaneWeaponFire() {
	bombReleased, rocketLaunched, torpedoLaunched := false, false, false

	for _, plane := range m.state.Arena.Planes {
		if !plane.IsCruising() {
			continue
		}
		// 对舰机型飞行途中的自卫防空火力（与主目标攻击互不影响）
		m.updatePlaneDefensiveFire(plane)

		if enemy := m.planeFireTarget(plane); enemy != nil {
			// 投放前检查飞机到预计命中点的航迹，只有陆地真正挡在
			// 鱼雷与目标之间时才放弃本次投放。
			if plane.Type == objUnit.PlaneTypeTorpedoBomber &&
				plane.TorpedoPathCrossesLand(enemy, &m.state.Core.MissionMD.MapCfg.Map) {
				m.retargetTorpedoBomber(plane, enemy.ID())
				continue
			}
			bullets := plane.Fire(enemy)
			if len(bullets) == 0 {
				continue
			}
			// 镜头内的才统计
			if m.state.View.Camera.Contains(plane.CurPos) {
				for _, bt := range bullets {
					// 统计一个就好，不要吵吵
					if bombReleased || rocketLaunched || torpedoLaunched {
						break
					}
					if bt.Type == objBullet.TypeBomb {
						bombReleased = true
					} else if bt.Type == objBullet.TypeRocket {
						rocketLaunched = true
					} else if bt.Type == objBullet.TypeTorpedo {
						torpedoLaunched = true
					}
				}
			}
			m.state.Arena.ForwardingBullets = append(m.state.Arena.ForwardingBullets, bullets...)
		}
	}

	m.weaponFirePlayer.PlayPlaneFire(bombReleased, rocketLaunched, torpedoLaunched)
}

// planeFireTarget 优先返回飞机当前追击目标；目标不在射程内或缺失时再选择其他候选。
// 这样战斗机不会因为随机切目标而把机头对准射界外的敌人。
func (m *MissionManager) planeFireTarget(plane *objUnit.Plane) objUnit.Hurtable {
	if target := m.preferredPlaneFireTarget(plane); target != nil {
		return target
	}
	candidates := m.planeFireCandidates(plane)
	if len(candidates) == 0 {
		return nil
	}
	return candidates[rand.Intn(len(candidates))]
}

// preferredPlaneFireTarget 优先返回当前追击目标，前提是目标有效且位于武器最大射程内。
func (m *MissionManager) preferredPlaneFireTarget(plane *objUnit.Plane) objUnit.Hurtable {
	if plane.CurAttackTarget == "" {
		return nil
	}
	if target := m.state.Arena.Planes[plane.CurAttackTarget]; target != nil && canPlaneFireAtTarget(plane, target) {
		return target
	}
	if target := m.state.Arena.Ships[plane.CurAttackTarget]; target != nil && canPlaneFireAtTarget(plane, target) {
		return target
	}
	return nil
}

// planeFireCandidates 收集飞机当前最大射程内、且类型符合攻击模式的敌我目标。
func (m *MissionManager) planeFireCandidates(plane *objUnit.Plane) []objUnit.Hurtable {
	candidates := []objUnit.Hurtable{}
	switch plane.AttackObjType() {
	case object.TypePlane:
		for _, enemy := range m.state.Arena.Planes {
			if canPlaneFireAtTarget(plane, enemy) {
				candidates = append(candidates, enemy)
			}
		}
	case object.TypeShip:
		for _, enemy := range m.state.Arena.Ships {
			if canPlaneFireAtTarget(plane, enemy) {
				candidates = append(candidates, enemy)
			}
		}
		for _, enemy := range m.state.Arena.Planes {
			if canPlaneFireAtTarget(plane, enemy) {
				candidates = append(candidates, enemy)
			}
		}
	}
	return candidates
}

// canPlaneFireAtTarget 判断目标是否敌方、类型是否匹配，以及是否在对应武器最大射程内。
// 射界与装填仍由具体武器在 Fire 阶段检查。
func canPlaneFireAtTarget(plane *objUnit.Plane, target objUnit.Hurtable) bool {
	if target == nil || target.Player() == plane.Player() {
		return false
	}
	switch plane.AttackObjType() {
	case object.TypePlane:
		return target.ObjType() == object.TypePlane &&
			plane.CurPos.Distance(target.MovementState().CurPos) <= plane.Weapon.MaxToPlaneRange
	case object.TypeShip:
		switch enemy := target.(type) {
		case *objUnit.BattleShip:
			return plane.CurPos.Distance(enemy.CurPos) <= plane.Weapon.MaxToShipRange
		case *objUnit.Plane:
			return plane.Type != objUnit.PlaneTypeTorpedoBomber && enemy.IsOnGround() &&
				plane.CurPos.Distance(enemy.CurPos) <= plane.Weapon.MaxToShipRange
		}
	}
	return false
}

// updatePlaneDefensiveFire 对舰机型（轰炸机/鱼雷机）飞行途中的自卫防空火力：
// 以往对空开火目标只来自 AttackObjType（对舰机型只会打敌舰与地面目标），
// 导致轰炸机在空中被战斗机咬住时炮塔全程哑火、纯挨打。这里扫描射程内的
// 空中敌机并还击，具体能否开火由武器自行过滤——Gun.Fire 只响应
// antiAircraft 武器且要求目标在射界/射程内，炸弹/鱼雷释放器不会对空中
// 目标投放，因此自卫火力不会误投弹药，也不打断既定的对舰攻击节奏。
// 每帧至多集火一个自卫目标：一次开火后相关炮塔即进入装填，其余炮塔
// 等下一帧再对其他目标开火，与 updateShipWeaponFire 等一帧一目标的
// 择敌模式保持一致。
func (m *MissionManager) updatePlaneDefensiveFire(plane *objUnit.Plane) {
	// 战斗机的对空交战由主目标逻辑负责；无对空武力的机型 MaxToPlaneRange 为 0
	if plane.AttackObjType() != object.TypeShip || plane.Weapon.MaxToPlaneRange <= 0 {
		return
	}
	for _, enemy := range m.state.Arena.Planes {
		// 只还击空中敌机：起飞/降落中的地面目标由对舰投放逻辑处理
		if enemy.BelongPlayer == plane.BelongPlayer || !enemy.IsCruising() {
			continue
		}
		// 不在对空火力射程内，跳过
		if plane.CurPos.Distance(enemy.CurPos) > plane.Weapon.MaxToPlaneRange {
			continue
		}
		// 该目标在所有炮塔射界内都无法开火时继续扫描，否则集火后结束本轮
		bullets := plane.Fire(enemy)
		if len(bullets) == 0 {
			continue
		}
		m.state.Arena.ForwardingBullets = append(m.state.Arena.ForwardingBullets, bullets...)
		break
	}
}

// retargetTorpedoBomber 让鱼雷机放弃当前不安全的投放对象，改为追踪其他敌舰。
// 若没有其他敌舰，则保留原指令，等飞离陆地后再尝试投放。
func (m *MissionManager) retargetTorpedoBomber(plane *objUnit.Plane, skippedTargetUid string) {
	targets := []objUnit.Hurtable{}
	for _, enemy := range m.state.Arena.Ships {
		if plane.BelongPlayer == enemy.BelongPlayer || enemy.Uid == skippedTargetUid {
			continue
		}
		targets = append(targets, enemy)
	}
	if len(targets) == 0 {
		return
	}

	enemy := targets[rand.Intn(len(targets))]
	plane.CurAttackTarget = enemy.ID()
	m.instructionSet.Add(instr.NewPlaneAttack(plane.Uid, enemy.ObjType(), enemy.ID()))
}

// 更新弹药状态
func (m *MissionManager) updateShotBullets() {
	for i := 0; i < len(m.state.Arena.ForwardingBullets); i++ {
		m.state.Arena.ForwardingBullets[i].Forward()
	}

	// 结算伤害
	resolveDamage := func(bt *objBullet.Bullet) bool {
		prevPos := bt.CurPos.Copy()
		prevPos.SubRx(math.Sin(bt.Rotation*math.Pi/180) * bt.Speed)
		prevPos.AddRy(math.Cos(bt.Rotation*math.Pi/180) * bt.Speed)

		switch bt.TargetObjType {
		case object.TypeShip:
			for _, ship := range m.state.Arena.Ships {
				// 总不能不小心打死自己吧，真是不应该 :D
				if bt.Shooter == ship.Uid {
					continue
				}
				// 如果友军伤害没启用，则不对己方战舰造成伤害
				if !m.state.UI.GameOpts.FriendlyFire && bt.BelongPlayer == ship.BelongPlayer {
					continue
				}

				if bt.ShotType == objBullet.ShotTypeDirect {
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
				} else if bt.ShotType == objBullet.ShotTypeArcing {
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
			// 曲射炮弹 / 炸弹落点爆炸波及地面飞机（停放与滑行）：按爆炸半径
			// 结算，不要求落点精确落在机身矩形内（机场本体无敌，但地面
			// 飞机会被炸毁）
			if bt.ShotType == objBullet.ShotTypeArcing && bt.HitObjType == object.TypeNone {
				if m.damageGroundPlanesNear(bt, "") {
					bt.HitObjType = object.TypePlane
				}
			}
		case object.TypePlane:
			for _, plane := range m.state.Arena.Planes {
				// 总不能不小心打死自己吧，真是不应该 :D
				if bt.Shooter == plane.Uid {
					continue
				}
				// 如果友军伤害没启用，则不对己方战舰造成伤害
				if !m.state.UI.GameOpts.FriendlyFire && bt.BelongPlayer == plane.BelongPlayer {
					continue
				}
				// 如果是舰对空，需要设置 “擦肩而过” 率：命中率按弹药口径分级，
				// 小口径速射防空炮弹幕密集，大口径舰炮弹散布远大于机体，直击罕见
				if bt.ShooterObjType == object.TypeShip {
					if rand.Float64() >= aaHitChance(bt.Diameter, plane.Type) {
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
					// 炸弹直接命中一架地面飞机时，爆炸同时波及附近其他地面飞机
					if bt.ShotType == objBullet.ShotTypeArcing {
						m.damageGroundPlanesNear(bt, plane.Uid)
					}
					break
				}
			}
		default:
			return false
		}
		return bt.HitObjType != object.TypeNone
	}

	rocketShouldExplode := func(bt *objBullet.Bullet) bool {
		if bt.Life <= 0 || bt.CurPos.Near(bt.TargetPos, bt.ProximityRadius) {
			return true
		}
		for _, plane := range m.state.Arena.Planes {
			if bt.Shooter == plane.Uid {
				continue
			}
			if !m.state.UI.GameOpts.FriendlyFire && bt.BelongPlayer == plane.BelongPlayer {
				continue
			}
			if bt.CurPos.Distance(plane.CurPos) <= bt.ProximityRadius {
				return true
			}
		}
		return false
	}

	// resolveRocketDamage 处理火箭及对空编程炮弹的近炸破片范围伤害。
	resolveRocketDamage := func(bt *objBullet.Bullet) {
		for _, plane := range m.state.Arena.Planes {
			if bt.Shooter == plane.Uid {
				continue
			}
			if !m.state.UI.GameOpts.FriendlyFire && bt.BelongPlayer == plane.BelongPlayer {
				continue
			}
			if bt.CurPos.Distance(plane.CurPos) > bt.BlastRadius {
				continue
			}
			plane.HurtBy(bt)
			bt.HitObjType = object.TypePlane
		}
		if bt.HitObjType == object.TypeNone {
			bt.HitObjType = object.TypeWater
		}
		m.state.Arena.Explosions = append(
			m.state.Arena.Explosions,
			objExplosion.NewRocket(bt.CurPos.Copy(), bt.Rotation),
		)
		if m.state.View.Camera.Contains(bt.CurPos) {
			m.weaponFirePlayer.PlayRocketExplode()
		}
	}

	arrivedBullets, forwardingBullets := []*objBullet.Bullet{}, []*objBullet.Bullet{}
	for _, bt := range m.state.Arena.ForwardingBullets {
		if bt.HasAirburst() {
			if rocketShouldExplode(bt) {
				resolveRocketDamage(bt)
				arrivedBullets = append(arrivedBullets, bt)
			} else {
				forwardingBullets = append(forwardingBullets, bt)
			}
			continue
		}
		// 迷失的弹药，要及时消亡（如鱼雷没命中）
		if bt.Life <= 0 {
			// TODO 其实还应该判断下，可能是 HitLand，后面再做吧
			bt.HitObjType = object.TypeWater
			continue
		}
		if bt.ShotType == objBullet.ShotTypeArcing {
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
		} else if bt.ShotType == objBullet.ShotTypeDirect {
			// 鱼雷碰撞到陆地，应该不再前进
			if bt.Type == objBullet.TypeTorpedo && m.state.Core.MissionMD.MapCfg.Map.IsLand(bt.CurPos.MX, bt.CurPos.MY) {
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
	m.state.Arena.ForwardingBullets = forwardingBullets
	// 已经到达目标地点的，转换成爆炸 & 伤害数值
	// TODO 支持命中爆炸
	for _, bt := range arrivedBullets {
		// 击中的是战舰 / 飞机，才会有伤害数值
		if bt.HitObjType != object.TypeShip && bt.HitObjType != object.TypePlane {
			continue
		}

		if m.state.UI.GameOpts.DisplayDamageNumber {
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
			if m.state.UI.DebugFlags.DamageColorByTeam {
				if bt.BelongPlayer == m.state.Player.CurPlayer {
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
			m.state.UI.GameMarks[mark.ID] = mark
		}
	}
}
