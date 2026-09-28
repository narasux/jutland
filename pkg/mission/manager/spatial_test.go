package manager

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func TestCombatBucketsMatchDistanceFilter(t *testing.T) {
	near := &objUnit.BattleShip{Uid: "near", CurPos: objPos.NewR(1, 1)}
	mid := &objUnit.BattleShip{Uid: "mid", CurPos: objPos.NewR(6, 1)}
	far := &objUnit.BattleShip{Uid: "far", CurPos: objPos.NewR(40, 1)}
	idx := &combatIndex{}
	idx.rebuild(map[string]*objUnit.BattleShip{
		near.Uid: near,
		mid.Uid:  mid,
		far.Uid:  far,
	}, nil)

	const radius = 10.0
	got := map[string]bool{}
	idx.eachShip(1, 1, radius, func(ship *objUnit.BattleShip) bool {
		if math.Hypot(ship.CurPos.RX-1, ship.CurPos.RY-1) <= radius {
			got[ship.Uid] = true
		}
		return false
	})
	if !got["near"] || !got["mid"] || got["far"] {
		t.Fatalf("ships in range = %v", got)
	}
}

func TestUnloadedShipDoesNotFire(t *testing.T) {
	if config.G == nil {
		config.G = config.NewDefaultGameSettings()
	}
	shooter := &objUnit.BattleShip{
		Uid:          "shooter",
		BelongPlayer: faction.HumanAlpha,
		CurPos:       objPos.NewR(10, 10),
		Weapon: objUnit.ShipWeapon{
			HasMainGun: true,
			MainGuns: []*objUnit.Gun{{
				ReloadTime:    100,
				ReloadStartAt: 1 << 62,
				AntiShip:      true,
				Range:         20,
			}},
			MaxToShipRange: 20,
		},
	}
	target := &objUnit.BattleShip{
		Uid:          "target",
		BelongPlayer: faction.ComputerAlpha,
		CurPos:       objPos.NewR(12, 10),
		TotalHP:      100,
		CurHP:        100,
	}
	manager := &MissionManager{state: &state.MissionState{
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				shooter.Uid: shooter,
				target.Uid:  target,
			},
		},
	}}
	manager.updateShipWeaponFire()
	if len(manager.state.Arena.ForwardingBullets) != 0 {
		t.Fatalf("bullets = %d, want 0 while the only gun is reloading", len(manager.state.Arena.ForwardingBullets))
	}
	if target.CurHP != 100 {
		t.Fatalf("target HP = %.1f, want 100", target.CurHP)
	}
}
