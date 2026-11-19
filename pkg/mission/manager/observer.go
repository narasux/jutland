package manager

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/observer"
)

// SetObserver attaches a read-only combat event receiver.
// Passing nil disables the observer path; the game's default stays nil.
func (m *MissionManager) SetObserver(o observer.Observer) {
	m.combatObserver = o
}

// observeShots emits one event per projectile immediately after a weapon fires.
// A nil observer costs one branch and no allocation.
func (m *MissionManager) observeShots(bullets []*objBullet.Bullet) {
	if m.combatObserver == nil {
		return
	}
	tick := objUnit.SimTick()
	for _, bt := range bullets {
		if bt == nil {
			continue
		}
		m.combatObserver.OnShot(shotFromBullet(bt, tick))
	}
}

func shotFromBullet(bt *objBullet.Bullet, tick int64) observer.Shot {
	return observer.Shot{
		Tick:            tick,
		ShooterUID:      bt.Shooter,
		ShooterName:     bt.ShooterName,
		ShooterType:     bt.ShooterType,
		ShooterPlayer:   bt.BelongPlayer,
		ShooterIsPlane:  bt.ShooterObjType == object.TypePlane,
		WeaponName:      bt.WeaponName,
		AmmoName:        bt.Name,
		AmmoType:        string(bt.Type),
		TargetUID:       bt.TargetUID,
		TargetType:      bt.TargetObjType,
		FireDistance:    bt.FireDistance,
		FireRangeRatio:  bt.FireRangeRatio,
		Plunge:          bt.Plunge,
		ProximityRadius: bt.ProximityRadius,
		BlastRadius:     bt.BlastRadius,
		MuzzleSpeed:     bt.Speed,
	}
}

// observeHit emits one damage event against a concrete target.
// It is called before the early return for a killing hit, so every hit is
// attributed even when no hit log entry is written.
func (m *MissionManager) observeHit(
	bt *objBullet.Bullet,
	targetUID, targetName, targetType string,
	targetPlayer faction.Player,
	targetIsPlane bool,
	hpBefore, hpAfter, applied, effective, overkill float64,
) {
	if m.combatObserver == nil {
		return
	}
	kind := bt.HitKind
	if kind == "" {
		kind = "direct"
	}
	m.combatObserver.OnHit(observer.Hit{
		Tick:            objUnit.SimTick(),
		ShooterUID:      bt.Shooter,
		ShooterName:     bt.ShooterName,
		ShooterType:     bt.ShooterType,
		ShooterPlayer:   bt.BelongPlayer,
		ShooterIsPlane:  bt.ShooterObjType == object.TypePlane,
		WeaponName:      bt.WeaponName,
		AmmoName:        bt.Name,
		AmmoType:        string(bt.Type),
		TargetUID:       targetUID,
		TargetName:      targetName,
		TargetType:      targetType,
		TargetPlayer:    targetPlayer,
		TargetIsPlane:   targetIsPlane,
		Kind:            kind,
		BaseDamage:      bt.Damage,
		AppliedDamage:   applied,
		EffectiveDamage: effective,
		Overkill:        overkill,
		HPBefore:        hpBefore,
		HPAfter:         hpAfter,
		Critical:        int(bt.CriticalType),
		FriendlyFire:    bt.BelongPlayer == targetPlayer,
		Killed:          hpAfter <= 0,
	})
}

// observeSink emits after RecordShipLost, before CurHP is reused for the
// wreck animation. planesLost is counted before RecordShipLost clears hangars.
func (m *MissionManager) observeSink(ship *objUnit.BattleShip, planesLost int) {
	if m.combatObserver == nil || ship == nil {
		return
	}
	killerName, killerType := "", ""
	killerPlayer := faction.Player("")
	killerIsPlane := false
	if ship.SunkByHit {
		killerPlayer = ship.LastAttackerPlayer
		killerName = ship.LastAttackerName
		killerType = ship.LastAttackerType
		killerIsPlane = ship.LastAttackerIsPlane
	}
	m.combatObserver.OnSink(observer.Sink{
		Tick:          objUnit.SimTick(),
		TargetUID:     ship.Uid,
		TargetName:    ship.Name,
		TargetType:    string(ship.Type),
		TargetPlayer:  ship.BelongPlayer,
		Tonnage:       ship.Tonnage,
		Carrier:       ship.Type == objUnit.ShipTypeAircraftCarrier,
		DamageTaken:   ship.DamageTaken,
		KillerName:    killerName,
		KillerType:    killerType,
		KillerPlayer:  killerPlayer,
		KillerIsPlane: killerIsPlane,
		PlanesLost:    planesLost,
	})
}

// observeLoss emits after RecordPlaneLost, before CurHP is reused for the
// falling animation.
func (m *MissionManager) observeLoss(plane *objUnit.Plane) {
	if m.combatObserver == nil || plane == nil {
		return
	}
	killerName, killerType := "", ""
	killerPlayer := faction.Player("")
	killerIsPlane := false
	if plane.ShotDown {
		killerPlayer = plane.LastAttackerPlayer
		killerName = plane.LastAttackerName
		killerType = plane.LastAttackerType
		killerIsPlane = plane.LastAttackerIsPlane
	}
	m.combatObserver.OnLoss(observer.Loss{
		Tick:          objUnit.SimTick(),
		TargetUID:     plane.Uid,
		TargetName:    plane.Name,
		TargetType:    string(plane.Type),
		TargetPlayer:  plane.BelongPlayer,
		BelongShip:    plane.BelongShip,
		ShotDown:      plane.ShotDown,
		DamageTaken:   plane.DamageTaken,
		KillerName:    killerName,
		KillerType:    killerType,
		KillerPlayer:  killerPlayer,
		KillerIsPlane: killerIsPlane,
	})
}
