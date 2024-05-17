package object

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"time"

	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/resources/images/mapblock"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

type ShipType string

const (
	// 航空母舰
	ShipTypeCarrier ShipType = "carrier"
	// 战列舰
	ShipTypeBattleship ShipType = "battleship"
	// 巡洋舰
	ShipTypeCruiser ShipType = "cruiser"
	// 驱逐舰
	ShipTypeDestroyer ShipType = "destroyer"
	// 护卫舰
	ShipTypeFrigate ShipType = "frigate"
	// 潜艇
	ShipTypeSubmarine ShipType = "submarine"
)

type WeaponType string

const (
	// 所有
	WeaponTypeAll WeaponType = "all"
	// 火炮
	WeaponTypeGun WeaponType = "gun"
	// 鱼雷
	WeaponTypeTorpedo WeaponType = "torpedo"
	// 导弹
	WeaponTypeMissile WeaponType = "missile"
)

const (
	// 顺时针
	RotateFlagClockwise = 1
	// 逆时针
	RotateFlagAnticlockwise = -1
)

// MapPos 位置
type MapPos struct {
	// 地图位置（用于通用计算，如小地图等）
	MX, MY int
	// 真实位置（用于计算屏幕位置，如不需要可不初始化）
	RX, RY float64
}

// NewMapPos ...
func NewMapPos(mx, my int) MapPos {
	return MapPos{MX: mx, MY: my, RX: float64(mx), RY: float64(my)}
}

// MEqual 判断位置是否相等（用地图位置判断，RX，RY 太准确一直没法到）
func (p *MapPos) MEqual(other MapPos) bool {
	return p.MX == other.MX && p.MY == other.MY
}

// Near 判断位置是否在指定范围内
func (p *MapPos) Near(other MapPos, distance int) bool {
	return geometry.CalcDistance(p.RX, p.RY, other.RX, other.RY) <= float64(distance)
}

// String ...
func (p *MapPos) String() string {
	return fmt.Sprintf("(%f/%d, %f/%d)", p.RX, p.MX, p.RY, p.MY)
}

// AssignRxy 重新赋值 RX，RY，并计算 MX，MY
func (p *MapPos) AssignRxy(rx, ry float64) {
	p.RX, p.RY = rx, ry
	p.MX, p.MY = int(math.Floor(rx)), int(math.Floor(ry))
}

// AssignMxy 重新赋值 MX，MY，同时计算 RX，RY
func (p *MapPos) AssignMxy(mx, my int) {
	p.MX, p.MY = mx, my
	p.RX, p.RY = float64(mx), float64(my)
}

// AddRx 修改 Rx，同时计算 Mx
func (p *MapPos) AddRx(rx float64) {
	p.RX += rx
	p.MX = int(math.Floor(p.RX))
}

// SubRx 修改 Rx，同时计算 Mx
func (p *MapPos) SubRx(rx float64) {
	p.RX -= rx
	p.MX = int(math.Floor(p.RX))
}

// AddRy 修改 Ry，同时计算 My
func (p *MapPos) AddRy(ry float64) {
	p.RY += ry
	p.MY = int(math.Floor(p.RY))
}

// SubRy 修改 Ry，同时计算 My
func (p *MapPos) SubRy(ry float64) {
	p.RY -= ry
	p.MY = int(math.Floor(p.RY))
}

// EnsureBorder 边界检查
func (p *MapPos) EnsureBorder(borderX, borderY float64) {
	p.RX = max(min(p.RX, borderX), 0)
	p.RY = max(min(p.RY, borderY), 0)
	p.MX = int(math.Floor(p.RX))
	p.MY = int(math.Floor(p.RY))
}

// 火炮 / 鱼雷弹药
type Bullet struct {
	// 固定参数
	// 弹药名称
	Name BulletName
	// 伤害数值
	Damage int

	// 动态参数
	// 唯一标识
	Uid string
	// 当前位置
	CurPos MapPos
	// 目标位置
	TargetPos MapPos
	// 旋转角度
	Rotation float64
	// 速度
	Speed float64

	// 所属战舰
	BelongShip string
	// 所属阵营（玩家）
	BelongPlayer faction.Player
}

// Forward 弹药前进
func (b *Bullet) Forward() {
	// 修改位置
	b.CurPos.AddRx(math.Sin(b.Rotation*math.Pi/180) * b.Speed)
	b.CurPos.SubRy(math.Cos(b.Rotation*math.Pi/180) * b.Speed)
}

type Gun struct {
	// 固定参数
	// 火炮名称
	Name GunName
	// 相对位置
	// 0.35 -> 从中心往舰首 35% 舰体长度
	// -0.3 -> 从中心往舰尾 30% 舰体长度
	PosPercent float64
	// 炮弹类型
	BulletName BulletName
	// 单次抛射炮弹数量
	BulletCount int
	// 装填时间（单位: s)
	ReloadTime int64
	// 射程
	Range float64
	// 炮弹散布
	BulletSpread int
	// 炮弹速度
	BulletSpeed float64

	// 动态参数
	// 当前火炮是否可用（如战损 / 禁用）
	Disable bool
	// 最后一次射击时间（时间戳)
	LastFireTime int64
}

// CanFire 是否可发射
func (g *Gun) CanFire(curPos, targetPos MapPos) bool {
	// 未启用，不可发射
	if g.Disable {
		return false
	}
	// 在重新装填，不可发射
	if g.LastFireTime+g.ReloadTime > time.Now().Unix() {
		return false
	}
	// 不在射程内，不可发射
	distance := geometry.CalcDistance(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	if distance > g.Range {
		return false
	}
	return true
}

// Fire 发射
func (g *Gun) Fire(
	shipUid string,
	player faction.Player,
	shipLength, shipRotation float64,
	curPos, targetPos MapPos,
) []*Bullet {
	shotBullets := []*Bullet{}
	if !g.CanFire(curPos, targetPos) {
		return shotBullets
	}
	g.LastFireTime = time.Now().Unix()

	// 炮塔距离战舰中心的距离
	distance := g.PosPercent * shipLength / mapblock.BlockSize
	curPos.AddRx(math.Sin(shipRotation*math.Pi/180) * distance)
	curPos.SubRy(math.Cos(shipRotation*math.Pi/180) * distance)

	// FIXME 其实还要考虑提前量（依赖敌舰速度，角度），除此之外，散布应该随着距离减小而减小
	for i := 0; i < g.BulletCount; i++ {
		pos := deepcopy.Copy(targetPos).(MapPos)
		// 炮弹散布的半径
		radius := float64(g.BulletSpread) / mapblock.BlockSize
		// rand.Intn(3) - 1 算方向，rand.Float64() 算距离
		pos.AddRx(float64(rand.Intn(3)-1) * rand.Float64() * radius)
		pos.AddRy(float64(rand.Intn(3)-1) * rand.Float64() * radius)

		shotBullets = append(shotBullets, NewBullets(g.BulletName, curPos, pos, g.BulletSpeed, shipUid, player))
	}

	return shotBullets
}

type Torpedo struct {
	// 固定参数
	// 百分比位置（如：0.33 -> 距离舰首 1/3)
	PosPercent float64
	// 鱼雷类型
	Bullet Bullet
	// 单次发射鱼雷数量
	BulletCount int
	// 装填时间（单位: s)
	ReloadTime int64
	// 射程
	Range float64
	// 鱼雷速度
	BulletSpeed float64

	// 动态参数
	// 当前鱼雷是否可用（如战损 / 禁用）
	Disable bool
	// 最后一次发射时间（时间戳)
	LastFireTime int64
}

// CanFire 是否可发射
func (t *Torpedo) CanFire(curPos, targetPos MapPos) bool {
	// 未启用，不可发射
	if t.Disable {
		return false
	}
	// 在重新装填，不可发射
	if t.LastFireTime+t.ReloadTime > time.Now().Unix() {
		return false
	}
	// 不在射程内，不可发射
	if geometry.CalcDistance(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY) > t.Range {
		return false
	}
	return true
}

// Fire 发射
func (t *Torpedo) Fire(
	shipUid string,
	player faction.Player,
	shipLength, shipRotation float64,
	curPos, targetPos MapPos,
) []*Bullet {
	if !t.CanFire(curPos, targetPos) {
		return []*Bullet{}
	}
	t.LastFireTime = time.Now().Unix()
	// FIXME 初始化弹药（复数，考虑当前位置，curPos 是战舰位置，还要计算相对位置，需要 uid）
	return []*Bullet{}
}

// Weapon 武器系统
type Weapon struct {
	// 火炮
	Guns []*Gun
	// 鱼雷
	Torpedoes []*Torpedo
	// 最大射程（各类武器射程最大值）
	MaxRange float64
	// 武器禁用情况
	GunDisabled     bool
	TorpedoDisabled bool
}

// BattleShip 战舰
type BattleShip struct {
	// 固定参数
	// 名称
	Name ShipName
	// 类别
	Type ShipType

	// 初始生命值
	TotalHP float64
	// 伤害减免（0.7 -> 仅受到击中的 70% 伤害)
	DamageReduction float64
	// 最大速度
	MaxSpeed float64
	// 转向速度（度）
	RotateSpeed float64
	// 战舰长度
	Length float64
	// 战舰宽度
	Width float64
	// 武器
	Weapon Weapon

	// 动态参数
	// 唯一标识
	Uid string
	// 当前生命值
	CurHP float64
	// 当前位置
	CurPos MapPos
	// 旋转角度
	CurRotation float64
	// 当前速度
	CurSpeed float64

	// 所属阵营（玩家）
	BelongPlayer faction.Player
}

// DisableWeapon 禁用武器
func (s *BattleShip) DisableWeapon(t WeaponType) {
	if t == WeaponTypeAll || t == WeaponTypeGun {
		for i := 0; i < len(s.Weapon.Guns); i++ {
			s.Weapon.Guns[i].Disable = true
		}
		s.Weapon.GunDisabled = true
	}
	if t == WeaponTypeAll || t == WeaponTypeTorpedo {
		for i := 0; i < len(s.Weapon.Torpedoes); i++ {
			s.Weapon.Torpedoes[i].Disable = true
		}
		s.Weapon.TorpedoDisabled = true
	}
}

// EnableWeapon 启用武器
func (s *BattleShip) EnableWeapon(t WeaponType) {
	if t == WeaponTypeAll || t == WeaponTypeGun {
		for i := 0; i < len(s.Weapon.Guns); i++ {
			s.Weapon.Guns[i].Disable = false
		}
		s.Weapon.GunDisabled = false
	}
	if t == WeaponTypeAll || t == WeaponTypeTorpedo {
		for i := 0; i < len(s.Weapon.Torpedoes); i++ {
			s.Weapon.Torpedoes[i].Disable = false
		}
		s.Weapon.TorpedoDisabled = false
	}
}

// InMaxRange 是否在最大射程内
func (s *BattleShip) InMaxRange(targetPos MapPos) bool {
	return geometry.CalcDistance(s.CurPos.RX, s.CurPos.RY, targetPos.RX, targetPos.RY) <= s.Weapon.MaxRange
}

// Fire 向指定目标发射武器
func (s *BattleShip) Fire(targetPos MapPos) []*Bullet {
	shotBullets := []*Bullet{}
	// 如果生命值为 0，那还 Fire 个锤子，直接返回
	if s.CurHP <= 0 {
		return shotBullets
	}
	for i := 0; i < len(s.Weapon.Guns); i++ {
		shotBullets = slices.Concat(shotBullets, s.Weapon.Guns[i].Fire(
			s.Uid, s.BelongPlayer, s.Length, s.CurRotation, s.CurPos, targetPos,
		))
	}
	for i := 0; i < len(s.Weapon.Torpedoes); i++ {
		shotBullets = slices.Concat(shotBullets, s.Weapon.Torpedoes[i].Fire(
			s.Uid, s.BelongPlayer, s.Length, s.CurRotation, s.CurPos, targetPos,
		))
	}
	return shotBullets
}

// MoveTo 移动到指定位置
// Q: 为什么不是移动的同时进行转向，而是原地转向后再移动？
// A：边移动边转向对实时计算的要求较高，且不利于后续的路线规划设计，
// 尾流渲染也不好做，先从简单的上手，后面有条件再搞，毕竟摊煎饼谁不喜欢呢？
//
// TODO 路线规划 -> 绕过陆地
func (s *BattleShip) MoveTo(targetPos MapPos, borderX, borderY int) (arrive bool) {
	// 如果生命值为 0，肯定是走不动，直接返回
	if s.CurHP <= 0 {
		return true
	}
	// 差不多到目标位置即可，不要强求准确，否则需要微调，视觉效果不佳
	if s.CurPos.Near(targetPos, 1) {
		s.CurSpeed = 0
		return true
	}
	// 未到达目标位置，逐渐加速
	if s.CurSpeed < s.MaxSpeed {
		s.CurSpeed = min(s.MaxSpeed, s.CurSpeed+s.MaxSpeed/5)
	}
	// 到目标位置附近，逐渐减速
	if s.CurPos.Near(targetPos, 2) {
		s.CurSpeed = max(s.MaxSpeed/5, s.CurSpeed-s.MaxSpeed/5)
	}
	targetRotation := geometry.CalcAngleBetweenPoints(s.CurPos.RX, s.CurPos.RY, targetPos.RX, targetPos.RY)
	// 逐渐转向
	if s.CurRotation != targetRotation {
		// 默认顺时针旋转
		rotateFlag := RotateFlagClockwise
		// 如果逆时针夹角小于顺时针夹角，则需要逆时针旋转
		if math.Mod(targetRotation-s.CurRotation+360, 360) > 180 {
			rotateFlag = RotateFlagAnticlockwise
		}
		s.CurRotation += float64(rotateFlag) * min(math.Abs(targetRotation-s.CurRotation), s.RotateSpeed)
		s.CurRotation = math.Mod(s.CurRotation+360, 360)
		// 原地旋转到差不多角度，才开始移动
		if math.Abs(s.CurRotation-targetRotation) > 1 {
			s.CurSpeed = 0
		}
	}
	// 修改位置
	s.CurPos.AddRx(math.Sin(s.CurRotation*math.Pi/180) * s.CurSpeed)
	s.CurPos.SubRy(math.Cos(s.CurRotation*math.Pi/180) * s.CurSpeed)

	// 防止出边界
	s.CurPos.EnsureBorder(float64(borderX-1), float64(borderY-1))

	return false
}

// ShipTrail 战舰尾流
type ShipTrail struct {
	Pos      MapPos
	Rotation float64
	Size     float64
	Life     int
}

// NewShipTrail ...
func NewShipTrail(pos MapPos, rotation, size float64, life int) *ShipTrail {
	return &ShipTrail{Pos: pos, Rotation: rotation, Size: size, Life: life}
}
