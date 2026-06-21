package unit

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// UnitMovementState 单位机动状态
type UnitMovementState struct {
	CurPos      objPos.MapPos
	CurRotation float64
	CurSpeed    float64
}

// UnitGeometricSize 单位尺寸
type UnitGeometricSize struct {
	Length float64
	Width  float64
}

// Nation 单位所属国籍。
type Nation string

const (
	NationAll     Nation = "all"
	NationSpecial Nation = "special"
	NationCN      Nation = "cn"
	NationUS      Nation = "us"
	NationJP      Nation = "jp"
	NationDE      Nation = "de"
	NationUK      Nation = "uk"
	NationSU      Nation = "su"
)

// ToDisplay 国籍展示用名称。
func (n Nation) ToDisplay() string {
	switch n {
	case NationAll:
		return "全部"
	case NationCN:
		return "中国"
	case NationUS:
		return "美国"
	case NationJP:
		return "日本"
	case NationDE:
		return "德国"
	case NationUK:
		return "英国"
	case NationSU:
		return "苏联"
	default:
		return "特殊"
	}
}

// AvailableNations 返回图鉴筛选使用的稳定国籍顺序。
func AvailableNations() []Nation {
	return []Nation{NationAll, NationCN, NationUS, NationJP, NationDE, NationUK, NationSU, NationSpecial}
}

// CombatPowerInfo 单位的静态战力评估，仅用于图鉴与平衡分析。
type CombatPowerInfo struct {
	Total      int
	AntiShip   int
	AntiAir    int
	Survival   int
	Mobility   int
	Projection int
	Burst      int
	Hull       int
	Aviation   int
	Details    CombatPowerDetails
}

// CombatPowerDetails 战力各维度的原始值和武器贡献，用于图鉴说明。
type CombatPowerDetails struct {
	EffectiveHP             float64
	AntiShipDPS             float64
	AntiAirDPS              float64
	MaxProjectionRange      float64
	MaxProjectionDistanceKM float64
	BurstDamage             float64
	AntiShipContributions   []CombatPowerContribution
	AntiAirContributions    []CombatPowerContribution
	BurstContributions      []CombatPowerContribution
}

// CombatPowerContribution 单项武器或舰载机对某项能力的贡献。
type CombatPowerContribution struct {
	Name  string
	Value float64
}

const (
	// RotateFlagClockwise 顺时针
	RotateFlagClockwise = 1
	// RotateFlagAnticlockwise 逆时针
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
	ObjType() object.Type
	HurtBy(bullet *objBullet.Bullet)
}

// Attacker 攻击者
type Attacker interface {
	BattleUnit
	ObjType() object.Type
	Fire(enemy Hurtable) []*objBullet.Bullet
}

// AttackWeapon 攻击性武器
type AttackWeapon interface {
	Fire(shooter Attacker, enemy Hurtable) []*objBullet.Bullet
}
