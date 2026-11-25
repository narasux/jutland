package manager

import (
	"math"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objDamage "github.com/narasux/jutland/pkg/mission/object/damage"
	objExplosion "github.com/narasux/jutland/pkg/mission/object/explosion"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// ramContactCellSize 撞击接触检测的网格边长（地图格）。
// 舰体通常只有 1~2 格长，用炮弹的 8 格桶会把大量不相邻的船放进同一候选集。
const ramContactCellSize = 2.0

// maxContactSeparation 单舰单拍的最大分离位移（地图格）。
// 深穿透时不要一拍瞬移出去；常规接触（0.2~0.5 格）一到两拍即可推开。
const maxContactSeparation = 0.5

// contactSpeedKeep 舰艏正撞后保留的航速比例：正撞几乎停船，擦碰只掉少量航速。
const contactSpeedKeep = 0.2

// updateShipContacts 检测敌方舰船之间的接触：结算撞击伤害，并做碰撞分离与掉速。
// 碰撞响应与 enableRamming 无关：即使关闭撞击伤害，舰船也不会再互相穿过。
// 友军不参与：不检测、不阻挡、不分离，保持现有的穿行行为。
func (m *MissionManager) updateShipContacts() {
	ships := m.state.Arena.OrderedShips()
	if len(ships) == 0 {
		return
	}

	buckets := make(map[uint64][]*objUnit.BattleShip, len(ships))
	maxHalf, maxMove := 0.0, 0.0
	for _, ship := range ships {
		if !contactShipAlive(ship) {
			continue
		}
		half := math.Hypot(ship.Length, ship.Width) / 2 / constants.MapBlockSize
		maxHalf = max(maxHalf, half)
		maxMove = max(maxMove, ship.CurSpeed)
		key := ramBucketKey(ramBucket(ship.CurPos.RX), ramBucket(ship.CurPos.RY))
		buckets[key] = append(buckets[key], ship)
	}
	for _, a := range ships {
		if !contactShipAlive(a) {
			continue
		}
		halfA := math.Hypot(a.Length, a.Width) / 2 / constants.MapBlockSize
		rings := int(math.Ceil((halfA + maxHalf + maxMove) / ramContactCellSize))
		bx, by := ramBucket(a.CurPos.RX), ramBucket(a.CurPos.RY)
		for iy := by - rings; iy <= by+rings; iy++ {
			for ix := bx - rings; ix <= bx+rings; ix++ {
				for _, b := range buckets[ramBucketKey(ix, iy)] {
					if b.Uid <= a.Uid || b.BelongPlayer == a.BelongPlayer {
						continue
					}
					if !contactShipAlive(b) {
						continue
					}
					ct, ok := contactBetweenShips(a, b)
					if !ok {
						continue
					}
					// 先按撞击瞬间的航速结算伤害，再掉速 / 分离；
					// 否则正撞会先把自己减到最低速，低于 minSpeed 而漏掉伤害。
					var beganA, beganB bool
					if a.Ram.Enabled() {
						beganA = a.RamContactBegan(b.Uid)
					}
					if b.Ram.Enabled() {
						beganB = b.RamContactBegan(a.Uid)
					}
					m.tryRamShip(a, b, ct.PointX, ct.PointY, ct.NormalX, ct.NormalY, beganA)
					m.tryRamShip(b, a, ct.PointX, ct.PointY, -ct.NormalX, -ct.NormalY, beganB)
					m.resolveShipContact(a, b, ct, beganA, beganB)
				}
			}
		}
	}

	m.updateShipPlaneContacts(ships)
}

// updateShipPlaneContacts 处理 canHitPlanes 特殊单位（水滴）的撞机。
func (m *MissionManager) updateShipPlaneContacts(ships []*objUnit.BattleShip) {
	for _, ship := range ships {
		if !contactShipAlive(ship) || !ship.Ram.CanHitPlanes || !ramSettingActive(ship) {
			continue
		}
		for _, plane := range m.state.Arena.OrderedPlanes() {
			if plane.CurHP <= 0 || plane.BelongPlayer == ship.BelongPlayer {
				continue
			}
			ct, ok := contactBetweenShipPlane(ship, plane)
			if !ok {
				continue
			}
			m.tryRamPlane(ship, plane, ct.PointX, ct.PointY, ct.NormalX, ct.NormalY)
		}
	}
}

// ramSweepMinSpeed 超过该内部速度（地图格/拍）的舰船需要做路径扫掠，
// 否则水滴这类 1000 节单位会在一拍内直接穿过目标。
const ramSweepMinSpeed = 0.05

// contactBetweenShips 返回从 B 指向 A 的接触信息。
// 快速移动的一方按本拍位移做子步采样，避免单点 OBB 检测漏掉高速穿透。
func contactBetweenShips(a, b *objUnit.BattleShip) (geometry.RectContact, bool) {
	if a.CurSpeed >= b.CurSpeed && a.CurSpeed > ramSweepMinSpeed {
		return sweptShipContact(a, b)
	}
	if b.CurSpeed > ramSweepMinSpeed {
		ct, ok := sweptShipContact(b, a)
		if ok {
			ct.NormalX, ct.NormalY = -ct.NormalX, -ct.NormalY
		}
		return ct, ok
	}
	ct := shipRectContactAt(a, a.CurPos.RX, a.CurPos.RY, b)
	return ct, ct.OK
}

func contactBetweenShipPlane(ship *objUnit.BattleShip, plane *objUnit.Plane) (geometry.RectContact, bool) {
	if ship.CurSpeed <= ramSweepMinSpeed {
		ct := shipPlaneRectContactAt(ship, ship.CurPos.RX, ship.CurPos.RY, plane)
		return ct, ct.OK
	}
	dist := ship.CurSpeed
	step := max(0.03, min(ship.Width, plane.Width)/constants.MapBlockSize)
	n := min(int(math.Ceil(dist/step)), 64)
	if n <= 1 {
		ct := shipPlaneRectContactAt(ship, ship.CurPos.RX, ship.CurPos.RY, plane)
		return ct, ct.OK
	}
	s, c := math.Sin(ship.CurRotation*math.Pi/180), math.Cos(ship.CurRotation*math.Pi/180)
	startX := ship.CurPos.RX - s*dist
	startY := ship.CurPos.RY + c*dist
	for k := 1; k <= n; k++ {
		t := float64(k) / float64(n)
		x := startX + (ship.CurPos.RX-startX)*t
		y := startY + (ship.CurPos.RY-startY)*t
		ct := shipPlaneRectContactAt(ship, x, y, plane)
		if ct.OK {
			return ct, true
		}
	}
	return geometry.RectContact{}, false
}

// sweptShipContact 让 mover 沿本拍位移（当前位置减去航向×速度）采样，
// 返回从 other 指向 mover 的接触信息。
func sweptShipContact(mover, other *objUnit.BattleShip) (geometry.RectContact, bool) {
	dist := mover.CurSpeed
	step := max(0.03, min(mover.Width, other.Width)/constants.MapBlockSize)
	n := min(int(math.Ceil(dist/step)), 64)
	if n <= 1 {
		ct := shipRectContactAt(mover, mover.CurPos.RX, mover.CurPos.RY, other)
		return ct, ct.OK
	}
	s, c := math.Sin(mover.CurRotation*math.Pi/180), math.Cos(mover.CurRotation*math.Pi/180)
	startX := mover.CurPos.RX - s*dist
	startY := mover.CurPos.RY + c*dist
	for k := 1; k <= n; k++ {
		t := float64(k) / float64(n)
		x := startX + (mover.CurPos.RX-startX)*t
		y := startY + (mover.CurPos.RY-startY)*t
		ct := shipRectContactAt(mover, x, y, other)
		if ct.OK {
			return ct, true
		}
	}
	return geometry.RectContact{}, false
}

func shipRectContactAt(
	mover *objUnit.BattleShip, x, y float64, other *objUnit.BattleShip,
) geometry.RectContact {
	return geometry.RotatedRectContact(
		x, y,
		mover.Length/constants.MapBlockSize, mover.Width/constants.MapBlockSize, mover.CurRotation,
		other.CurPos.RX, other.CurPos.RY,
		other.Length/constants.MapBlockSize, other.Width/constants.MapBlockSize, other.CurRotation,
	)
}

func shipPlaneRectContactAt(
	ship *objUnit.BattleShip, x, y float64, plane *objUnit.Plane,
) geometry.RectContact {
	return geometry.RotatedRectContact(
		x, y,
		ship.Length/constants.MapBlockSize, ship.Width/constants.MapBlockSize, ship.CurRotation,
		plane.CurPos.RX, plane.CurPos.RY,
		plane.Length/constants.MapBlockSize, plane.Width/constants.MapBlockSize, plane.CurRotation,
	)
}

func contactShipAlive(ship *objUnit.BattleShip) bool {
	return ship != nil && ship.CurHP > 0 && !ship.Submerged()
}

// ramSettingActive 普通战舰受游戏设置开关控制；配置 alwaysOn 的特殊单位不受影响。
func ramSettingActive(ship *objUnit.BattleShip) bool {
	if ship == nil || !ship.Ram.Enabled() {
		return false
	}
	if ship.Ram.AlwaysOn {
		return true
	}
	return config.G != nil && config.G.EnableRamming
}

// ramMass 碰撞分离使用的质量代理：优先吨位，其次总血量，缺失时按 1。
func ramMass(ship *objUnit.BattleShip) float64 {
	if ship.Tonnage > 0 {
		return ship.Tonnage
	}
	if ship.TotalHP > 0 {
		return ship.TotalHP
	}
	return 1
}

// resolveShipContact 处理一次接触的物理响应：掉速 + 按吨位分离。
// 掉速只在接触开始那一拍施加，持续重叠时不会把航速逐拍乘到 0。
func (m *MissionManager) resolveShipContact(
	a, b *objUnit.BattleShip, ct geometry.RectContact, beganA, beganB bool,
) {
	if beganA {
		applyContactSpeedLoss(a, ct.NormalX, ct.NormalY)
	}
	if beganB {
		applyContactSpeedLoss(b, -ct.NormalX, -ct.NormalY)
	}
	if ct.Depth <= 0 {
		return
	}
	massA, massB := ramMass(a), ramMass(b)
	total := massA + massB
	if total <= 0 {
		total = 1
	}
	// 法线从 B 指向 A：A 沿法线推出，B 沿反方向推出；大舰的位移更小。
	m.shiftShipOutOfContact(a, ct.NormalX, ct.NormalY, min(ct.Depth*massB/total, maxContactSeparation))
	m.shiftShipOutOfContact(b, -ct.NormalX, -ct.NormalY, min(ct.Depth*massA/total, maxContactSeparation))
}

// applyContactSpeedLoss 吃掉舰船朝对方运动的那部分航速。
// 正撞（运动方向与接触法线几乎相反）保留 contactSpeedKeep，擦碰只损失少量航速。
func applyContactSpeedLoss(ship *objUnit.BattleShip, nx, ny float64) {
	if ship.CurSpeed <= 0 {
		return
	}
	sin, cos := math.Sin(ship.CurRotation*math.Pi/180), math.Cos(ship.CurRotation*math.Pi/180)
	into := sin*(-nx) + (-cos)*(-ny)
	if into <= 0 {
		return
	}
	into = min(into, 1)
	ship.CurSpeed *= 1 - (1-contactSpeedKeep)*into
}

// shiftShipOutOfContact 把舰船沿法线推开 dist 格；会把舰体推上岸或推出地图时放弃这次位移。
func (m *MissionManager) shiftShipOutOfContact(ship *objUnit.BattleShip, nx, ny, dist float64) {
	if dist <= 0 {
		return
	}
	next := ship.CurPos
	next.AddRx(nx * dist)
	next.AddRy(ny * dist)
	if mapCfg := m.state.Core.MissionMD.MapCfg; mapCfg != nil {
		next.EnsureBorder(float64(mapCfg.Width-2), float64(mapCfg.Height-2))
		if !ship.CanOnLand() {
			before := objUnit.HullLandOverlapSamples(mapCfg, ship.CurPos, ship.CurRotation, ship.Length)
			after := objUnit.HullLandOverlapSamples(mapCfg, next, ship.CurRotation, ship.Length)
			if after > before {
				return
			}
		}
	}
	ship.CurPos = next
}

// tryRamShip 对单个方向做舰艏、角度、接近速度与冷却判定，通过后结算撞击。
// began 表示本拍是该次接触的第一拍：持续重叠不再重复掉血。
func (m *MissionManager) tryRamShip(
	attacker, target *objUnit.BattleShip, px, py, nx, ny float64, began bool,
) {
	if !ramSettingActive(attacker) || target.CurHP <= 0 || target.Submerged() {
		return
	}
	originX, originY := ramContactOrigin(attacker)
	if !contactInBow(attacker, originX, originY, px, py) {
		return
	}
	bearing := ramBearing(attacker, originX, originY, target.CurPos.RX, target.CurPos.RY)
	if bearing > attacker.Ram.BowArc {
		return
	}
	closing := ramClosingSpeed(attacker, target, nx, ny)
	if !began || closing < attacker.Ram.MinSpeed || !attacker.RamReady(target.Uid) {
		return
	}
	res := objDamage.ResolveRam(
		attacker.Ram.Damage, closing, attacker.MaxSpeedKnots(), bearing,
		attacker.Ram.SelfDamageRatio, attacker.Tonnage, target.Tonnage,
		target.HorizontalDamageReduction,
	)
	if res.TargetDamage <= 0 {
		return
	}
	attacker.MarkRamHit(target.Uid)
	ix, iy := ramImpactPoint(attacker, px, py)
	bt := m.newRamBullet(attacker, target.Uid, object.TypeShip, ix, iy, res.BaseDamage, "ram")
	m.hurtShip(target, bt)
	bt.HitObjType = object.TypeShip
	m.recordRamHit(bt, target.TotalHP)
	m.applyRamSelfDamage(attacker, res.SelfDamage)
}

// tryRamPlane 处理特殊单位的撞机；飞机没有水平减伤，伤害直接取减伤前的 base。
func (m *MissionManager) tryRamPlane(
	attacker *objUnit.BattleShip, plane *objUnit.Plane, px, py, nx, ny float64,
) {
	if !ramSettingActive(attacker) || plane.CurHP <= 0 {
		return
	}
	if !attacker.RamContactBegan(plane.Uid) {
		return
	}
	originX, originY := ramContactOrigin(attacker)
	if !contactInBow(attacker, originX, originY, px, py) {
		return
	}
	bearing := ramBearing(attacker, originX, originY, plane.CurPos.RX, plane.CurPos.RY)
	if bearing > attacker.Ram.BowArc {
		return
	}
	closing := ramPlaneClosingSpeed(attacker, plane, nx, ny)
	if closing < attacker.Ram.MinSpeed || !attacker.RamReady(plane.Uid) {
		return
	}
	res := objDamage.ResolveRam(
		attacker.Ram.Damage, closing, attacker.MaxSpeedKnots(), bearing,
		attacker.Ram.SelfDamageRatio, attacker.Tonnage, 0, 0,
	)
	if res.TargetDamage <= 0 {
		return
	}
	attacker.MarkRamHit(plane.Uid)
	ix, iy := ramImpactPoint(attacker, px, py)
	bt := m.newRamBullet(attacker, plane.Uid, object.TypePlane, ix, iy, res.BaseDamage, "ram")
	m.hurtPlane(plane, bt)
	bt.HitObjType = object.TypePlane
	m.recordRamHit(bt, plane.TotalHP)
	m.applyRamSelfDamage(attacker, res.SelfDamage)
}

func (m *MissionManager) newRamBullet(
	attacker *objUnit.BattleShip, targetUID string, targetType object.Type,
	px, py, baseDamage float64, hitKind string,
) *objBullet.Bullet {
	pos := objPos.NewR(px, py)
	return &objBullet.Bullet{
		Name: "Ram", Type: objBullet.TypeRam,
		CurPos: pos, TargetPos: pos, Rotation: attacker.CurRotation,
		Damage: baseDamage, Plunge: 0, CriticalRate: 0, PenetrationMM: 0,
		Shooter: attacker.Uid, ShooterName: attacker.Name, ShooterType: string(attacker.Type),
		ShooterObjType: object.TypeShip, BelongPlayer: attacker.BelongPlayer,
		TargetObjType: targetType, TargetUID: targetUID,
		WeaponName: "Ram", HitKind: hitKind,
	}
}

// recordRamHit 记录伤害统计并生成撞击表现。
func (m *MissionManager) recordRamHit(bt *objBullet.Bullet, targetMaxHP float64) {
	m.state.Stats.RecordDamage(bt.BelongPlayer, bt.RealDamage)
	m.spawnRamDamageNumber(bt, targetMaxHP)
	m.spawnRamEffect(bt)
	if !m.headless && m.state.View.Camera.Contains(bt.CurPos) {
		m.weaponFirePlayer.PlayRamImpact()
	}
}

func (m *MissionManager) spawnRamDamageNumber(bt *objBullet.Bullet, targetMaxHP float64) {
	if m.headless || !m.state.UI.GameOpts.DisplayDamageNumber || bt.RealDamage <= 0 {
		return
	}
	ratio := 1.0
	if targetMaxHP > 0 {
		ratio = bt.RealDamage / targetMaxHP
	}
	fontSize, clr := 16.0, colorx.White
	switch {
	case ratio >= 0.15:
		fontSize, clr = 24, colorx.Red
	case ratio >= 0.05:
		fontSize, clr = 20, colorx.Yellow
	}
	mark := objMark.NewText(bt.CurPos, objMark.DamageFlagText(bt.RealDamage), fontSize, clr, 20)
	m.state.UI.GameMarks[mark.ID] = mark
}

func (m *MissionManager) spawnRamEffect(bt *objBullet.Bullet) {
	if m.headless || !m.state.UI.GameOpts.DisplayHitEffects ||
		len(m.state.Arena.Explosions) >= maxLiveEffects {
		return
	}
	kind, size, life, ok := hitEffect(bt)
	if !ok {
		return
	}
	m.effectSeq++
	m.state.Arena.Explosions = append(m.state.Arena.Explosions, objExplosion.NewHitEffect(
		kind, bt.CurPos, bt.Rotation, size, life, m.effectSeq, bt.BelongPlayer,
	))
}

// applyRamSelfDamage 结算撞击方的反作用伤害，并写入一条「自伤」战况。
// 自伤不是对敌伤害，不增加 DamageDealt，但必须进入 DamageTaken 与详情列表。
func (m *MissionManager) applyRamSelfDamage(attacker *objUnit.BattleShip, selfDamage float64) {
	if selfDamage <= 0 {
		return
	}
	attacker.CurHP = max(0, attacker.CurHP-selfDamage)
	attacker.Damage.DamageTaken += selfDamage
	attacker.Damage.LastHit = objUnit.HitInfo{WeaponType: "ram", Damage: selfDamage}
	m.state.Stats.RecordBattleLog(state.BattleLogEntry{
		Tick:         objUnit.SimTick(),
		ActorPlayer:  attacker.BelongPlayer,
		ActorName:    attacker.Name,
		ActorType:    string(attacker.Type),
		TargetPlayer: attacker.BelongPlayer,
		TargetName:   attacker.Name,
		TargetType:   string(attacker.Type),
		Action:       state.BattleLogShipSelfHit,
		Hit:          attacker.Damage.LastHit,
	})
}

// ramContactOrigin 返回判定撞击时的参考舰体中心。
// 高速舰的接触点在对齐路径上可能已经落到当前舰体后方，用本拍起点做参考，
// 否则 1000 节的水滴会被判定成「舰艉撞人」而漏掉伤害。
func ramContactOrigin(ship *objUnit.BattleShip) (float64, float64) {
	if ship.CurSpeed <= ramSweepMinSpeed {
		return ship.CurPos.RX, ship.CurPos.RY
	}
	s, c := math.Sin(ship.CurRotation*math.Pi/180), math.Cos(ship.CurRotation*math.Pi/180)
	return ship.CurPos.RX - s*ship.CurSpeed, ship.CurPos.RY + c*ship.CurSpeed
}

// contactInBow 接触点是否落在参考中心前方 25% 舰长内。
func contactInBow(ship *objUnit.BattleShip, originX, originY, px, py float64) bool {
	s, c := math.Sin(ship.CurRotation*math.Pi/180), math.Cos(ship.CurRotation*math.Pi/180)
	forward := (px-originX)*s + (py-originY)*-c
	return forward > ship.Length/constants.MapBlockSize*0.25
}

// ramImpactPoint 把撞击表现锚点夹到撞击舰的舰艏。
// SAT 接触点在深穿透或高速扫掠后会落到舰体中段甚至目标中心，
// 按原坐标画闪光 / 伤害数字会看起来「没撞上」；这里保留接触点的横向偏移，
// 纵向固定在舰艏内 90% 半长处。
func ramImpactPoint(ship *objUnit.BattleShip, px, py float64) (float64, float64) {
	s, c := math.Sin(ship.CurRotation*math.Pi/180), math.Cos(ship.CurRotation*math.Pi/180)
	dx, dy := px-ship.CurPos.RX, py-ship.CurPos.RY
	halfLen := ship.Length / constants.MapBlockSize / 2
	halfWid := ship.Width / constants.MapBlockSize / 2
	lateral := min(max(dx*c+dy*s, -halfWid), halfWid)
	return ship.CurPos.RX + s*halfLen*0.9 + c*lateral,
		ship.CurPos.RY - c*halfLen*0.9 + s*lateral
}

// ramBearing 返回撞击舰航向与「参考中心 -> 目标」方位的夹角（度）。
func ramBearing(ship *objUnit.BattleShip, originX, originY, tx, ty float64) float64 {
	dx, dy := tx-originX, ty-originY
	dist := math.Hypot(dx, dy)
	if dist < 1e-9 {
		return 0
	}
	s, c := math.Sin(ship.CurRotation*math.Pi/180), math.Cos(ship.CurRotation*math.Pi/180)
	cosBearing := (dx*s + dy*-c) / dist
	cosBearing = min(max(cosBearing, -1), 1)
	return math.Acos(cosBearing) * 180 / math.Pi
}

// ramClosingSpeed 沿接触法线（从目标指向撞击舰）计算接近速度，单位节。
func ramClosingSpeed(attacker, target *objUnit.BattleShip, nx, ny float64) float64 {
	avx, avy := shipVelocity(attacker)
	tvx, tvy := shipVelocity(target)
	return (avx-tvx)*-nx + (avy-tvy)*-ny
}

// ramPlaneClosingSpeed 撞机只计入舰体自身的接近速度：飞机航速用 km/h 口径，
// 与舰船的节混算会把速度系数顶到上限，直接按舰速投影到接触法线更稳定。
func ramPlaneClosingSpeed(attacker *objUnit.BattleShip, _ *objUnit.Plane, nx, ny float64) float64 {
	avx, avy := shipVelocity(attacker)
	return avx*-nx + avy*-ny
}

func shipVelocity(ship *objUnit.BattleShip) (float64, float64) {
	s, c := math.Sin(ship.CurRotation*math.Pi/180), math.Cos(ship.CurRotation*math.Pi/180)
	speed := ship.CurSpeedKnots()
	return s * speed, -c * speed
}

func ramBucket(v float64) int {
	return int(math.Floor(v / ramContactCellSize))
}

func ramBucketKey(x, y int) uint64 {
	return uint64(uint32(int32(x)))<<32 | uint64(uint32(int32(y)))
}
