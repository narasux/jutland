package manager

import (
	"log"
	"math"
	"slices"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/common/constants"
	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
	audioRes "github.com/narasux/jutland/pkg/resources/audio"
	"github.com/narasux/jutland/pkg/resources/images/bullet"
	"github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

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
			if bt.CurPos.Near(bt.TargetPos, 0.05) {
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
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if m.state.GameOpts.MapDisplayMode == state.MapDisplayModeFull {
				// 如果是在全屏查看地图模式，退出查看
				m.state.GameOpts.MapDisplayMode = state.MapDisplayModeNone
			} else {
				// 否则暂停游戏
				m.state.MissionStatus = state.MissionPaused
			}
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
