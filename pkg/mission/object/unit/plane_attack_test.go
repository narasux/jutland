package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

func TestAttackLeadSpeedUsesWeaponBallistics(t *testing.T) {
	plane := &Plane{
		Type:     PlaneTypeFighter,
		CurSpeed: 0.1,
		Weapon: PlaneWeapon{
			Guns: []*Gun{{
				BulletSpeed:  0.1875,
				AntiAircraft: true,
			}},
		},
	}
	if got := plane.AttackLeadSpeed(object.TypePlane); math.Abs(got-0.1875) > 1e-9 {
		t.Fatalf("attack lead speed = %v, want weapon speed 0.1875", got)
	}
	if got := plane.AttackLeadSpeed(object.TypeShip); math.Abs(got-plane.CurSpeed) > 1e-9 {
		t.Fatalf("attack lead speed without matching weapon = %v, want plane speed %v", got, plane.CurSpeed)
	}
}

func TestAttackLeadSpeedPrefersShipReleaseWeapon(t *testing.T) {
	plane := &Plane{
		Type: PlaneTypeTorpedoBomber,
		Weapon: PlaneWeapon{
			Guns: []*Gun{{
				BulletSpeed: 0.2,
				AntiShip:    true,
			}},
			Torpedoes: []*Releaser{{
				BulletSpeed: 0.1,
			}},
		},
	}
	if got := plane.AttackLeadSpeed(object.TypeShip); math.Abs(got-0.1) > 1e-9 {
		t.Fatalf("ship attack lead speed = %v, want release weapon speed 0.1", got)
	}
}

func TestFighterPursuitAlignsNoseWithWeaponLeadPoint(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	enemyPos := objPos.NewR(10, 9.7)
	leadPos := objPos.NewR(10, 9)
	plane := &Plane{
		Uid:          "fighter",
		CurHP:        100,
		CurPos:       objPos.NewR(10, 10),
		CurRotation:  0,
		CurSpeed:     0.12,
		MaxSpeed:     0.12,
		RotateSpeed:  30,
		RemainRange:  100,
		BelongPlayer: faction.HumanAlpha,
	}

	(&FighterPursuitStrategy{}).MoveTo(plane, nil, leadPos, enemyPos, 0.06)

	bearing := plane.CurPos.Angle(enemyPos)
	relativeBearing := math.Mod(bearing-plane.CurRotation+540, 360) - 180
	if math.Abs(relativeBearing) > 20 {
		t.Fatalf(
			"fighter drifted outside its forward firing arc: bearing=%.2f rotation=%.2f",
			bearing,
			plane.CurRotation,
		)
	}
}
