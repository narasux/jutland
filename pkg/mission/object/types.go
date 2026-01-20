package object

import "github.com/narasux/jutland/pkg/mission/faction"

type ObjectType int

const (
	// ObjectTypeNone 无
	ObjectTypeNone ObjectType = iota
	// ObjectTypeShip 战舰
	ObjectTypeShip
	// ObjectTypePlane 战机
	ObjectTypePlane
	// ObjectTypeWater 水面
	ObjectTypeWater
	// ObjectTypeLand 陆地
	ObjectTypeLand
)

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
	// ID 单位ID
	ID() string
	// Detail 详细信息，调试时候使用
	Detail() string
	// Player 所属玩家
	Player() faction.Player
	// MovementState 机动状态（速度，方向，位置等信息）
	MovementState() UnitMovementState
	// GeometricSize 几何尺寸（长、宽等信息）
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

	ObjType() ObjectType
	Fire(enemy Hurtable) []*Bullet
}

// AttackWeapon 攻击性武器
type AttackWeapon interface {
	Fire(shooter Attacker, enemy Hurtable) []*Bullet
}
