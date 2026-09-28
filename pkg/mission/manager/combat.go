package manager

import (
	"math"
	"math/rand"

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
		chance = 1.0 / 5
	case diameter <= 155:
		chance = 1.0 / 9
	default:
		chance = 1.0 / 14
	}
	// 俯冲轰炸机、高空轰炸机收到放空炮火的伤害低一点
	if planeType == objUnit.PlaneTypeDiveBomber || planeType == objUnit.PlaneTypeLevelBomber {
		chance /= 3
	}
	return chance
}

// damageGroundPlanesNear 落点爆炸波及范围内的地面飞机结算伤害，
// excludeUid 用于排除已被直接命中的目标；返回是否造成伤害。
func (m *MissionManager) damageGroundPlanesNear(bt *objBullet.Bullet, excludeUid string) bool {
	m.rebuildCombatBuckets()
	hit := false
	m.combatBuckets.eachPlane(bt.CurPos.RX, bt.CurPos.RY, bombBlastRadius, func(plane *objUnit.Plane) bool {
		if plane.Uid == excludeUid || !plane.IsOnGround() {
			return false
		}
		if !m.state.UI.GameOpts.FriendlyFire && bt.BelongPlayer == plane.BelongPlayer {
			return false
		}
		if bt.CurPos.Distance(plane.CurPos) <= bombBlastRadius {
			plane.HurtBy(bt)
			hit = true
		}
		return false
	})
	return hit
}

// 更新战舰武器开火相关状态
// TODO 开火逻辑优化：主炮/鱼雷向射程内最大的，生命值比例最少目标开火，副炮向最近的目标开火
func (m *MissionManager) updateShipWeaponFire() {
	m.rebuildCombatBuckets()
	maxBulletDiameter := 0
	isTorpedoLaunched := false
	isRocketLaunched := false

	for _, ship := range m.state.Arena.Ships {
		if !ship.Weapon.AnyReloaded() {
			continue
		}
		inRangeEnemies := []objUnit.Hurtable{}

		target := m.state.Arena.Ships[ship.AttackTarget]
		// 若有指定攻击目标且在射程内，则优先攻击该目标；否则只看射程能盖到的格子
		if target != nil && ship.CurPos.Distance(target.CurPos) < ship.Weapon.MaxToShipRange {
			inRangeEnemies = append(inRangeEnemies, target)
		} else {
			m.combatBuckets.eachPlane(
				ship.CurPos.RX, ship.CurPos.RY, ship.Weapon.MaxToPlaneRange,
				func(enemy *objUnit.Plane) bool {
					if ship.BelongPlayer == enemy.BelongPlayer {
						return false
					}
					if ship.CurPos.Distance(enemy.CurPos) > ship.Weapon.MaxToPlaneRange {
						return false
					}
					inRangeEnemies = append(inRangeEnemies, enemy)
					return false
				},
			)
			m.combatBuckets.eachShip(
				ship.CurPos.RX, ship.CurPos.RY, ship.Weapon.MaxToShipRange,
				func(enemy *objUnit.BattleShip) bool {
					if ship.BelongPlayer == enemy.BelongPlayer || enemy.Uid == ship.AttackTarget {
						return false
					}
					if ship.CurPos.Distance(enemy.CurPos) > ship.Weapon.MaxToShipRange {
						return false
					}
					inRangeEnemies = append(inRangeEnemies, enemy)
					return false
				},
			)
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
			m.state.Arena.PutPlane(plane)
			// 给飞机下达攻击指令
			m.instructionSet.Add(instr.NewPlaneAttack(plane.Uid, object.TypeShip, target.ID()))
			continue
		}

		plane, targetType, targetUID, ok := m.takeOffFromBase(ship)
		if ok {
			m.state.Arena.PutPlane(plane)
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
			m.state.Arena.PutPlane(plane)
			m.instructionSet.Add(instr.NewPlaneAttack(plane.Uid, targetType, targetUID))
		}
	}
}

// 更新战机武器开火相关状态
func (m *MissionManager) updatePlaneWeaponFire() {
	m.rebuildCombatBuckets()
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
	if !plane.Weapon.AnyReloaded() {
		return nil
	}
	candidates := []objUnit.Hurtable{}
	switch plane.AttackObjType() {
	case object.TypePlane:
		m.combatBuckets.eachPlane(
			plane.CurPos.RX, plane.CurPos.RY, plane.Weapon.MaxToPlaneRange,
			func(enemy *objUnit.Plane) bool {
				if canPlaneFireAtTarget(plane, enemy) {
					candidates = append(candidates, enemy)
				}
				return false
			},
		)
	case object.TypeShip:
		m.combatBuckets.eachShip(
			plane.CurPos.RX, plane.CurPos.RY, plane.Weapon.MaxToShipRange,
			func(enemy *objUnit.BattleShip) bool {
				if canPlaneFireAtTarget(plane, enemy) {
					candidates = append(candidates, enemy)
				}
				return false
			},
		)
		m.combatBuckets.eachPlane(
			plane.CurPos.RX, plane.CurPos.RY, plane.Weapon.MaxToShipRange,
			func(enemy *objUnit.Plane) bool {
				if canPlaneFireAtTarget(plane, enemy) {
					candidates = append(candidates, enemy)
				}
				return false
			},
		)
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
	if !plane.Weapon.AntiAircraftReady() {
		return
	}
	m.combatBuckets.eachPlane(
		plane.CurPos.RX, plane.CurPos.RY, plane.Weapon.MaxToPlaneRange,
		func(enemy *objUnit.Plane) bool {
			// 只还击空中敌机：起飞/降落中的地面目标由对舰投放逻辑处理
			if enemy.BelongPlayer == plane.BelongPlayer || !enemy.IsCruising() {
				return false
			}
			if plane.CurPos.Distance(enemy.CurPos) > plane.Weapon.MaxToPlaneRange {
				return false
			}
			bullets := plane.Fire(enemy)
			if len(bullets) == 0 {
				return false
			}
			m.state.Arena.ForwardingBullets = append(m.state.Arena.ForwardingBullets, bullets...)
			return true
		},
	)
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

// targetHitRadius 返回目标矩形中心到角点的距离，单位是地图格。
// 半长不够：弹道擦过舷侧角点时，到舰心的距离可以大于舰长的一半。
func targetHitRadius(length, width float64) float64 {
	return 0.5 * math.Hypot(length, width) / constants.MapBlockSize
}

// beyondHitReach 用距离平方判断这一拍是否不可能相交。
func beyondHitReach(bx, by, tx, ty, reach float64) bool {
	dx := bx - tx
	dy := by - ty
	return dx*dx+dy*dy > reach*reach
}

// 更新弹药状态
func (m *MissionManager) updateShotBullets() {
	for i := 0; i < len(m.state.Arena.ForwardingBullets); i++ {
		m.state.Arena.ForwardingBullets[i].Forward()
	}
	m.rebuildCombatBuckets()

	// 结算伤害
	resolveDamage := func(bt *objBullet.Bullet) bool {
		// 这一拍的飞行方向只算一次，粗筛和精确相交共用。
		sinR := math.Sin(bt.Rotation * math.Pi / 180)
		cosR := math.Cos(bt.Rotation * math.Pi / 180)
		prevRX := bt.CurPos.RX - sinR*bt.Speed
		prevRY := bt.CurPos.RY + cosR*bt.Speed
		// 查询半径要盖住舰体对角线，不能只按弹速，否则桶边缘的船会被漏掉。
		shipQuery := bt.Speed + maxHullHitRadius
		if bt.ShotType == objBullet.ShotTypeArcing {
			shipQuery = maxHullHitRadius
		}

		switch bt.TargetObjType {
		case object.TypeShip:
			m.combatBuckets.eachShip(bt.CurPos.RX, bt.CurPos.RY, shipQuery, func(ship *objUnit.BattleShip) bool {
				if bt.Shooter == ship.Uid {
					return false
				}
				if !m.state.UI.GameOpts.FriendlyFire && bt.BelongPlayer == ship.BelongPlayer {
					return false
				}

				if bt.ShotType == objBullet.ShotTypeDirect {
					reach := targetHitRadius(ship.Length, ship.Width) + bt.Speed
					if beyondHitReach(bt.CurPos.RX, bt.CurPos.RY, ship.CurPos.RX, ship.CurPos.RY, reach) {
						return false
					}
					if geometry.IsSegmentIntersectRotatedRectangle(
						prevRX, prevRY,
						bt.CurPos.RX, bt.CurPos.RY,
						ship.CurPos.RX, ship.CurPos.RY,
						ship.Length/constants.MapBlockSize,
						ship.Width/constants.MapBlockSize,
						ship.CurRotation,
					) {
						ship.HurtBy(bt)
						bt.HitObjType = object.TypeShip
						return true
					}
				} else if bt.ShotType == objBullet.ShotTypeArcing {
					reach := targetHitRadius(ship.Length, ship.Width)
					if beyondHitReach(bt.CurPos.RX, bt.CurPos.RY, ship.CurPos.RX, ship.CurPos.RY, reach) {
						return false
					}
					if geometry.IsPointInRotatedRectangle(
						bt.CurPos.RX, bt.CurPos.RY,
						ship.CurPos.RX, ship.CurPos.RY,
						ship.Length/constants.MapBlockSize,
						ship.Width/constants.MapBlockSize,
						ship.CurRotation,
					) {
						ship.HurtBy(bt)
						bt.HitObjType = object.TypeShip
						return true
					}
				}
				return false
			})
			if bt.ShotType == objBullet.ShotTypeArcing && bt.HitObjType == object.TypeNone {
				if m.damageGroundPlanesNear(bt, "") {
					bt.HitObjType = object.TypePlane
				}
			}
		case object.TypePlane:
			m.combatBuckets.eachPlane(
				bt.CurPos.RX, bt.CurPos.RY, bt.Speed+maxHullHitRadius,
				func(plane *objUnit.Plane) bool {
					if bt.Shooter == plane.Uid {
						return false
					}
					if !m.state.UI.GameOpts.FriendlyFire && bt.BelongPlayer == plane.BelongPlayer {
						return false
					}
					if bt.ShooterObjType == object.TypeShip {
						if rand.Float64() >= aaHitChance(bt.Diameter, plane.Type) {
							return false
						}
					}
					reach := targetHitRadius(plane.Length, plane.Width) + bt.Speed
					if beyondHitReach(bt.CurPos.RX, bt.CurPos.RY, plane.CurPos.RX, plane.CurPos.RY, reach) {
						return false
					}
					if geometry.IsSegmentIntersectRotatedRectangle(
						prevRX, prevRY,
						bt.CurPos.RX, bt.CurPos.RY,
						plane.CurPos.RX, plane.CurPos.RY,
						plane.Length/constants.MapBlockSize,
						plane.Width/constants.MapBlockSize,
						plane.CurRotation,
					) {
						plane.HurtBy(bt)
						bt.HitObjType = object.TypePlane
						if bt.ShotType == objBullet.ShotTypeArcing {
							m.damageGroundPlanesNear(bt, plane.Uid)
						}
						return true
					}
					return false
				},
			)
		default:
			return false
		}
		return bt.HitObjType != object.TypeNone
	}

	rocketShouldExplode := func(bt *objBullet.Bullet) bool {
		if bt.Life <= 0 || bt.CurPos.Near(bt.TargetPos, bt.ProximityRadius) {
			return true
		}
		found := false
		m.combatBuckets.eachPlane(bt.CurPos.RX, bt.CurPos.RY, bt.ProximityRadius, func(plane *objUnit.Plane) bool {
			if bt.Shooter == plane.Uid {
				return false
			}
			if !m.state.UI.GameOpts.FriendlyFire && bt.BelongPlayer == plane.BelongPlayer {
				return false
			}
			if bt.CurPos.Distance(plane.CurPos) <= bt.ProximityRadius {
				found = true
				return true
			}
			return false
		})
		return found
	}

	// resolveRocketDamage 处理火箭及对空编程炮弹的近炸破片范围伤害。
	resolveRocketDamage := func(bt *objBullet.Bullet) {
		m.combatBuckets.eachPlane(bt.CurPos.RX, bt.CurPos.RY, bt.BlastRadius, func(plane *objUnit.Plane) bool {
			if bt.Shooter == plane.Uid {
				return false
			}
			if !m.state.UI.GameOpts.FriendlyFire && bt.BelongPlayer == plane.BelongPlayer {
				return false
			}
			if bt.CurPos.Distance(plane.CurPos) > bt.BlastRadius {
				return false
			}
			plane.HurtBy(bt)
			bt.HitObjType = object.TypePlane
			return false
		})
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
			flagText := objMark.DamageFlagText(bt.RealDamage)
			mark := objMark.NewText(bt.CurPos, flagText, fontSize, clr, 20)
			m.state.UI.GameMarks[mark.ID] = mark
		}
	}
}
