package unit

import (
	"github.com/narasux/jutland/pkg/i18n"
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
	// NationAll 表示图鉴筛选中的全部国籍。
	NationAll Nation = "all"
	// NationSpecial 表示不属于常规国家分类的特殊单位。
	NationSpecial Nation = "special"
	// NationCN 表示中国。
	NationCN Nation = "cn"
	// NationUS 表示美国。
	NationUS Nation = "us"
	// NationJP 表示日本。
	NationJP Nation = "jp"
	// NationDE 表示德国。
	NationDE Nation = "de"
	// NationUK 表示英国。
	NationUK Nation = "uk"
	// NationSU 表示苏联。
	NationSU Nation = "su"
)

// ToDisplay 国籍展示用名称。
func (n Nation) ToDisplay() string {
	switch n {
	case NationAll:
		return i18n.Text(i18n.MsgNationAll)
	case NationCN:
		return i18n.Text(i18n.MsgNationChina)
	case NationUS:
		return i18n.Text(i18n.MsgNationUnitedStates)
	case NationJP:
		return i18n.Text(i18n.MsgNationJapan)
	case NationDE:
		return i18n.Text(i18n.MsgNationGermany)
	case NationUK:
		return i18n.Text(i18n.MsgNationUnitedKingdom)
	case NationSU:
		return i18n.Text(i18n.MsgNationSovietUnion)
	default:
		return i18n.Text(i18n.MsgNationSpecial)
	}
}

// AvailableNations 返回图鉴筛选使用的稳定国籍顺序。
func AvailableNations() []Nation {
	return []Nation{NationAll, NationCN, NationUS, NationJP, NationDE, NationUK, NationSU, NationSpecial}
}

// CombatPowerInfo 单位的静态战力评估，仅用于图鉴与平衡分析。
type CombatPowerInfo struct {
	// FormationSize 表示本条战力覆盖的单位数量：舰船为 1，飞机为 10 架标准编队。
	FormationSize int
	Total         int
	AntiShip      int
	AntiAir       int
	Survival      int
	Mobility      int
	Projection    int
	Burst         int
	Hull          int
	Aviation      int
	Details       CombatPowerDetails
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
