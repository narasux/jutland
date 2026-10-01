package instruction

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func TestPlaneScoutLoitersThenReturns(t *testing.T) {
	point := objPos.New(3, 3)
	plane := &objUnit.Plane{
		Uid: "scout", Type: objUnit.PlaneTypeScout, CurHP: 10, RemainRange: 80,
		CurPos: point, FlightPhase: objUnit.PlaneFlightPhaseCruising, SightRange: 36,
		BelongPlayer: faction.HumanAlpha,
	}
	ms := &state.MissionState{
		Core: state.MissionCoreState{SimTick: 10},
		Arena: state.MissionArenaState{
			Planes: map[string]*objUnit.Plane{plane.Uid: plane},
		},
	}
	order := NewPlaneScout(plane.Uid, point, true)
	if err := order.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if order.loiterUntil <= ms.Core.SimTick {
		t.Fatal("arrival should start a 30 second loiter")
	}
	ms.Core.SimTick = order.loiterUntil
	if err := order.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if !plane.ForceReturn || !order.Executed() {
		t.Fatal("loiter should end in a return")
	}
}

func TestTrackFleetHoldsOutsideAntiAir(t *testing.T) {
	vision := state.NewFactionVision(24, 24)
	vision.Stamp(10, 10, 12)
	ship := &objUnit.BattleShip{
		Uid: "enemy", CurHP: 10, BelongPlayer: faction.ComputerAlpha,
		CurPos: objPos.New(10, 10),
		Weapon: objUnit.ShipWeapon{MaxToPlaneRange: 6},
	}
	ms := &state.MissionState{
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
		},
	}
	plane := &objUnit.Plane{
		BelongPlayer: faction.HumanAlpha,
		CurPos:       objPos.NewR(10, 16.5),
		SightRange:   36,
	}
	_, hold := trackFleet(ms, plane, vision)
	if !hold {
		t.Fatal("scout inside the anti-air margin should hold")
	}
	plane.CurPos = objPos.NewR(10, 22)
	move, hold := trackFleet(ms, plane, vision)
	if !move || hold {
		t.Fatal("scout outside anti-air should close on the fleet")
	}
}
