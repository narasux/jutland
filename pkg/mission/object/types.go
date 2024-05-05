package object

import (
	"slices"
	"time"
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

// MapPos 位置
type MapPos struct {
	// 地图位置（用于通用计算，如小地图等）
	MX, MY int
	// 真实位置（用于计算屏幕位置，如不需要可不初始化）
	RX, RY float64
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
	CurPosition MapPos
	// 目标位置
	TargetPosition MapPos
	// 旋转角度
	Rotate int
	// 速度
	Speed int

	// 动画（多图片）
	// TODO 补充入水 & 爆炸动画（指针数组）
}

type Gun struct {
	// 固定参数
	// 火炮名称
	Name GunName
	// 百分比位置（如：0.33 -> 距离舰首 1/3)
	PosPercent float64
	// 炮弹类型
	BulletName BulletName
	// 单次抛射炮弹数量
	BulletCount int
	// 装填时间（单位: s)
	ReloadTime int64
	// 射程
	Range int
	// 炮弹散布
	BulletSpread int
	// 炮弹速度
	BulletSpeed int

	// 动态参数
	// 当前火炮是否可用（如战损 / 禁用）
	Disable bool
	// 最后一次射击时间（时间戳)
	LastFireTime int64
}

// CanFire 是否可发射
func (g *Gun) CanFire() bool {
	if g.Disable {
		return false
	}
	return g.LastFireTime+g.ReloadTime <= time.Now().Unix()
}

// Fire 发射
func (g *Gun) Fire(curPos MapPos, targetPos MapPos) []*Bullet {
	if !g.CanFire() {
		return []*Bullet{}
	}
	g.LastFireTime = time.Now().Unix()
	// FIXME 初始化弹药（复数，考虑当前位置，散布，curPos 是战舰位置，还要计算相对位置，需要 uid）
	return []*Bullet{}
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
func (t *Torpedo) CanFire() bool {
	if t.Disable {
		return false
	}
	return t.LastFireTime+t.ReloadTime <= time.Now().Unix()
}

// Fire 发射
func (t *Torpedo) Fire(curPos MapPos, targetPos MapPos) []*Bullet {
	if !t.CanFire() {
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
}

// BattleShip 战舰
type BattleShip struct {
	// 固定参数
	// 名称
	Name ShipName
	// 类别
	Type ShipType

	// 初始生命值
	TotalHP int
	// 伤害减免（0.7 -> 仅受到击中的 70% 伤害)
	DamageReduction float64
	// 最大速度
	MaxSpeed float64
	// 转向速度（度）
	RotateSpeed int
	// 武器
	Weapon Weapon

	// 动态参数
	// 唯一标识
	Uid string
	// 当前生命值
	CurHP int
	// 当前位置
	Pos MapPos
	// 旋转角度
	Rotate int
	// 当前速度
	CurSpeed float64
}

// DisableWeapon 禁用武器
func (s *BattleShip) DisableWeapon(t WeaponType) {
	if t == WeaponTypeAll || t == WeaponTypeGun {
		for i := 0; i < len(s.Weapon.Guns); i++ {
			s.Weapon.Guns[i].Disable = true
		}
	}
	if t == WeaponTypeAll || t == WeaponTypeTorpedo {
		for i := 0; i < len(s.Weapon.Torpedoes); i++ {
			s.Weapon.Torpedoes[i].Disable = true
		}
	}
}

// EnableWeapon 启用武器
func (s *BattleShip) EnableWeapon(t WeaponType) {
	if t == WeaponTypeAll || t == WeaponTypeGun {
		for i := 0; i < len(s.Weapon.Guns); i++ {
			s.Weapon.Guns[i].Disable = false
		}
	}
	if t == WeaponTypeAll || t == WeaponTypeTorpedo {
		for i := 0; i < len(s.Weapon.Torpedoes); i++ {
			s.Weapon.Torpedoes[i].Disable = false
		}
	}
}

// Fire 向指定目标发射武器
func (s *BattleShip) Fire(curPos MapPos, targetPos MapPos) []*Bullet {
	bullets := []*Bullet{}
	for i := 0; i < len(s.Weapon.Guns); i++ {
		bullets = slices.Concat(bullets, s.Weapon.Guns[i].Fire(curPos, targetPos))
	}
	for i := 0; i < len(s.Weapon.Torpedoes); i++ {
		bullets = slices.Concat(bullets, s.Weapon.Torpedoes[i].Fire(curPos, targetPos))
	}
	return bullets
}

func (s *BattleShip) MoveTo(targetPos MapPos) (arrive bool) {
	// FIXME 修改自身的位置，角度，速度
	return true
}
