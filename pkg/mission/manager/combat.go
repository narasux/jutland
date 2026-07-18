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

// 更新战舰武器开火相关状态
// TODO 开火逻辑优化：主炮/鱼雷向射程内最大的，生命值比例最少目标开火，副炮向最近的目标开火
func (m *MissionManager) updateShipWeaponFire() {
	maxBulletDiameter := 0
	isTorpedoLaunched := false
	isRocketLaunched := false

	for _, ship := range m.state.Arena.Ships {
		inRangeEnemies := []objUnit.Hurtable{}

		if target := m.state.Arena.Ships[ship.AttackTarget]; target != nil {
			// 如果有目标敌人，则检查是否在射程内，如果有直接选中即可
			if ship.CurPos.Distance(target.CurPos) < ship.Weapon.MaxToShipRange {
				inRangeEnemies = append(inRangeEnemies, target)
			}
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

// 飞机出动 & 攻击
func (m *MissionManager) updatePlaneAttackOrReturn() {
	for _, ship := range m.state.Arena.Ships {
		// 战舰上没有飞机的，跳过
		if !ship.Aircraft.HasPlane {
			continue
		}

		inRangeEnemies := []objUnit.Hurtable{}
		// TODO 目前飞机目标都没考虑是否在攻击范围内，未来还是需要考虑的
		if target := m.state.Arena.Ships[ship.AttackTarget]; target != nil {
			// 如果有目标敌人，则直接选中即可
			inRangeEnemies = append(inRangeEnemies, target)
		} else {
			// 敌机
			for _, enemy := range m.state.Arena.Planes {
				// 不能攻击己方的战机
				if ship.BelongPlayer == enemy.BelongPlayer {
					continue
				}
				inRangeEnemies = append(inRangeEnemies, enemy)
			}
			// 敌舰
			for _, enemy := range m.state.Arena.Ships {
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
			m.state.Arena.Planes[plane.Uid] = plane
			// 给飞机下达攻击指令
			m.instructionSet.Add(instr.NewPlaneAttack(plane.Uid, enemy.ObjType(), enemy.ID()))
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
		// 如果战机已经有攻击目标，则跳过
		if m.instructionSet.Exists(instrUid) {
			continue
		}

		// 有剩余燃料 & 没有攻击目标，按攻击类型选一个新的
		inRangeEnemies := []objUnit.Hurtable{}

		if plane.AttackObjType() == object.TypePlane {
			// 敌机
			for _, enemy := range m.state.Arena.Planes {
				// 不能攻击己方的战机
				if plane.BelongPlayer == enemy.BelongPlayer {
					continue
				}
				inRangeEnemies = append(inRangeEnemies, enemy)
			}
		} else if plane.AttackObjType() == object.TypeShip {
			// 敌舰
			for _, enemy := range m.state.Arena.Ships {
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
	bombReleased, rocketLaunched, torpedoLaunched := false, false, false

	for _, plane := range m.state.Arena.Planes {
		if !plane.IsCruising() {
			continue
		}
		inRangeEnemies := []objUnit.Hurtable{}
		if plane.AttackObjType() == object.TypePlane {
			// 敌机
			for _, enemy := range m.state.Arena.Planes {
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
		} else if plane.AttackObjType() == object.TypeShip {
			// 敌舰
			for _, enemy := range m.state.Arena.Ships {
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
		}
		if total := len(inRangeEnemies); total != 0 {
			// 射程内的敌人都会被攻击
			enemy := inRangeEnemies[rand.Intn(total)]
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
				// 如果是舰对空，需要设置 “擦肩而过” 率，现在命中率太高（昭和防空，十防九空）
				if bt.ShooterObjType == object.TypeShip {
					if plane.Type == objUnit.PlaneTypeDiveBomber {
						// 俯冲轰炸机要飞得很近，插肩而过率得高一些
						if rand.Intn(24) != 0 {
							continue
						}
					} else {
						// 鱼雷机/战斗机按 1/8 的概率被击中
						if rand.Intn(8) != 0 {
							continue
						}
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

	// resolveRocketDamage 处理火箭近炸破片范围伤害，并创建局部爆炸效果。
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
		if bt.Type == objBullet.TypeRocket && bt.TargetObjType == object.TypePlane {
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
