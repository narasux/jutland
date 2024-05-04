package mission

import (
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Position 位置
type Position struct {
	X int
	Y int
}

// 火炮 / 鱼雷弹药
type Bullet struct {
	Img *ebiten.Image

	// 固定参数
	// 弹药名称
	Name string
	// 伤害数值
	Damage int

	// 动态参数
	// 当前位置
	CurrentPosition Position
	// 目标位置
	TargetPosition Position
	// 旋转角度
	Rotate float64
	// 速度
	Speed float64

	// 动画（多图片）
	// TODO 补充入水 & 爆炸动画（指针数组）
}

type Gun struct {
	// 固定参数
	// 火炮名称
	Name string
	// 百分比位置（如：0.33 -> 距离舰首 1/3)
	PosPercent float64
	// 炮弹类型
	Bullet Bullet
	// 单次抛射炮弹数量
	BulletCount int
	// 装填时间（单位: s)
	ReloadTime int64
	// 射程
	Range float64
	// 炮弹散布
	BulletSpread float64
	// 炮弹速度
	BulletSpeed float64

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
func (g *Gun) Fire(curPos Position, targetPos Position) []*Bullet {
	if !g.CanFire() {
		return []*Bullet{}
	}
	g.LastFireTime = time.Now().Unix()
	// FIXME 初始化弹药（复数，考虑当前位置，散布，curPos 是战舰位置，还要计算相对位置）
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
func (t *Torpedo) Fire(curPos Position, targetPos Position) []*Bullet {
	if !t.CanFire() {
		return []*Bullet{}
	}
	t.LastFireTime = time.Now().Unix()
	// FIXME 初始化弹药（复数，考虑当前位置，curPos 是战舰位置，还要计算相对位置）
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
	img *ebiten.Image

	// 固定参数
	// 名称
	Name string
	// 类别
	Type ShipType

	// 初始生命值
	TotalHP int
	// 伤害减免（0.7 -> 仅受到击中的 70% 伤害)
	DamageReduction float64
	// 武器
	Weapon Weapon
	// 最大速度
	MaxSpeed float64

	// 动态参数
	// 当前生命值
	CurHP int
	// 当前位置
	Pos Position
	// 旋转角度
	Rotate float64
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

// 向指定目标发射炮弹
func (s *BattleShip) Fire(curPos Position, targetPos Position) []*Bullet {
	bullets := []*Bullet{}
	for i := 0; i < len(s.Weapon.Guns); i++ {
		bullets = slices.Concat(bullets, s.Weapon.Guns[i].Fire(curPos, targetPos))
	}
	for i := 0; i < len(s.Weapon.Torpedoes); i++ {
		bullets = slices.Concat(bullets, s.Weapon.Torpedoes[i].Fire(curPos, targetPos))
	}
	return bullets
}
