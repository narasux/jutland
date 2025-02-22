package object

import "github.com/narasux/jutland/pkg/mission/faction"

// UnitMovementState 单位机动状态
type UnitMovementState struct {
	CurPos      MapPos
	CurRotation float64
	CurSpeed    float64
}

// UnitGeometricSize 单位尺寸
type UnitGeometricSize struct {
	Length float64
	Width  float64
}

// BattleUnit 战斗单位
type BattleUnit interface {
	ID() string
	Player() faction.Player
	MovementState() UnitMovementState
	GeometricSize() UnitGeometricSize
}

// Hurtable 可被伤害的对象
type Hurtable interface {
	BattleUnit

	ObjType() ObjectType
	HurtBy(bullet *Bullet)
}

// Attacker 攻击者
type Attacker interface {
	BattleUnit

	Fire(enemy Hurtable) []*Bullet
}

// AttackWeapon 攻击性武器
type AttackWeapon interface {
	Fire(shooter Attacker, enemy Hurtable) []*Bullet
}
