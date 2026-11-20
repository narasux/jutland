// Package observer defines the read-only combat event stream used by headless
// simulation and analysis. Production code calls these methods only when an
// observer is attached; the default game path keeps the observer nil.
package observer

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
)

// Observer receives combat events. Implementations must be fast and must not
// mutate MissionManager or unit state from inside a callback.
type Observer interface {
	OnShot(Shot)
	OnHit(Hit)
	OnSink(Sink)
	OnLoss(Loss)
}

// Shot is one projectile leaving a weapon mount.
type Shot struct {
	Tick            int64          `json:"tick"`
	ShooterUID      string         `json:"shooterUid"`
	ShooterName     string         `json:"shooterName"`
	ShooterType     string         `json:"shooterType"`
	ShooterPlayer   faction.Player `json:"shooterPlayer"`
	ShooterIsPlane  bool           `json:"shooterIsPlane"`
	WeaponName      string         `json:"weaponName"`
	AmmoName        string         `json:"ammoName"`
	AmmoType        string         `json:"ammoType"`
	TargetUID       string         `json:"targetUid"`
	TargetType      object.Type    `json:"targetType"`
	FireDistance    float64        `json:"fireDistance"`
	FireRangeRatio  float64        `json:"fireRangeRatio"`
	Plunge          float64        `json:"plunge"`
	ProximityRadius float64        `json:"proximityRadius,omitempty"`
	BlastRadius     float64        `json:"blastRadius,omitempty"`
	MuzzleSpeed     float64        `json:"muzzleSpeed"`
}

// Hit is one damage application. For AoE weapons this is emitted per target,
// and TargetUID makes the attribution explicit (including friendly fire).
type Hit struct {
	Tick            int64          `json:"tick"`
	ShooterUID      string         `json:"shooterUid"`
	ShooterName     string         `json:"shooterName"`
	ShooterType     string         `json:"shooterType"`
	ShooterPlayer   faction.Player `json:"shooterPlayer"`
	ShooterIsPlane  bool           `json:"shooterIsPlane"`
	WeaponName      string         `json:"weaponName"`
	AmmoName        string         `json:"ammoName"`
	AmmoType        string         `json:"ammoType"`
	TargetUID       string         `json:"targetUid"`
	TargetName      string         `json:"targetName"`
	TargetType      string         `json:"targetType"`
	TargetPlayer    faction.Player `json:"targetPlayer"`
	TargetIsPlane   bool           `json:"targetIsPlane"`
	Kind            string         `json:"kind"` // direct, airburst, blast, depth_charge
	BaseDamage      float64        `json:"baseDamage"`
	AppliedDamage   float64        `json:"appliedDamage"`
	EffectiveDamage float64        `json:"effectiveDamage"`
	Overkill        float64        `json:"overkill"`
	HPBefore        float64        `json:"hpBefore"`
	HPAfter         float64        `json:"hpAfter"`
	Critical        int            `json:"critical"`
	FriendlyFire    bool           `json:"friendlyFire"`
	Killed          bool           `json:"killed"`
	// 以下字段仅在 config.G.EnablePenetration 打开且命中舰船时填充；未启用时保持零值。
	PenetrationMM    float64 `json:"penetrationMm,omitempty"`
	ArmorMM          float64 `json:"armorMm,omitempty"`
	EffectiveArmorMM float64 `json:"effectiveArmorMm,omitempty"`
	PenetrationRatio float64 `json:"penetrationRatio,omitempty"`
	Penetrated       bool    `json:"penetrated,omitempty"`
	Overmatch        bool    `json:"overmatch,omitempty"`
}

// Sink is a ship removed from the arena.
type Sink struct {
	Tick          int64          `json:"tick"`
	TargetUID     string         `json:"targetUid"`
	TargetName    string         `json:"targetName"`
	TargetType    string         `json:"targetType"`
	TargetPlayer  faction.Player `json:"targetPlayer"`
	Tonnage       float64        `json:"tonnage"`
	Carrier       bool           `json:"carrier"`
	DamageTaken   float64        `json:"damageTaken"`
	KillerUID     string         `json:"killerUid,omitempty"`
	KillerName    string         `json:"killerName"`
	KillerType    string         `json:"killerType"`
	KillerPlayer  faction.Player `json:"killerPlayer"`
	KillerIsPlane bool           `json:"killerIsPlane"`
	PlanesLost    int            `json:"planesLost"`
}

// Loss is a plane removed from the arena.
type Loss struct {
	Tick          int64          `json:"tick"`
	TargetUID     string         `json:"targetUid"`
	TargetName    string         `json:"targetName"`
	TargetType    string         `json:"targetType"`
	TargetPlayer  faction.Player `json:"targetPlayer"`
	BelongShip    string         `json:"belongShip,omitempty"`
	ShotDown      bool           `json:"shotDown"`
	DamageTaken   float64        `json:"damageTaken"`
	KillerUID     string         `json:"killerUid,omitempty"`
	KillerName    string         `json:"killerName"`
	KillerType    string         `json:"killerType"`
	KillerPlayer  faction.Player `json:"killerPlayer"`
	KillerIsPlane bool           `json:"killerIsPlane"`
}
