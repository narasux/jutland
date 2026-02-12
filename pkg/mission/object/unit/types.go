package unit

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objCommon "github.com/narasux/jutland/pkg/mission/object/common"
)

// UnitMovementState 单位机动状态
type UnitMovementState struct {
	CurPos      objCommon.MapPos
	CurRotation float64
	CurSpeed    float64
}

// UnitGeometricSize 单位尺寸
type UnitGeometricSize struct {
	Length float64
	Width  float64
}

const (
	// 顺时针
	RotateFlagClockwise = 1
	// 逆时针
	RotateFlagAnticlockwise = -1
)

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

// 下面是向前声明的接口，用于避免循环引用
// 实际类型在各自的包中定义

// Hurtable 可被伤害的对象
type Hurtable interface {
	BattleUnit
	ObjType() objCommon.ObjectType
	HurtBy(bullet *objBullet.Bullet)
}

// Attacker 攻击者
type Attacker interface {
	BattleUnit
	ObjType() objCommon.ObjectType
	Fire(enemy Hurtable) []*objBullet.Bullet
}

// AttackWeapon 攻击性武器
type AttackWeapon interface {
	Fire(shooter Attacker, enemy Hurtable) []*objBullet.Bullet
}
