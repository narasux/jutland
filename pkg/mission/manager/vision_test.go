package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func TestUpdateVisionLightsInsideRadiusOnly(t *testing.T) {
	vision := state.NewFactionVision(20, 20)
	ms := &state.MissionState{
		Core: state.MissionCoreState{FogOfWar: true},
		Player: state.MissionPlayerState{
			CurPlayer: faction.HumanAlpha,
			Visions: map[faction.Player]*state.FactionVision{
				faction.HumanAlpha: vision,
			},
		},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				"alive": {
					Uid: "alive", CurHP: 10, SightRange: 2,
					BelongPlayer: faction.HumanAlpha, CurPos: objPos.NewR(5.2, 5.2),
				},
				"dead": {
					Uid: "dead", CurHP: 0, SightRange: 8,
					BelongPlayer: faction.HumanAlpha, CurPos: objPos.New(15, 15),
				},
			},
			Airfields: map[string]*objBuilding.Airfield{
				"field": {
					Uid: "field", Disabled: true, BelongPlayer: faction.HumanAlpha,
					Pos: objPos.New(1, 1),
				},
			},
		},
	}

	(&MissionManager{state: ms}).updateVision()

	if !vision.VisibleAt(5, 5) {
		t.Fatal("cell under the ship should be visible")
	}
	if vision.VisibleAt(15, 15) {
		t.Fatal("dead ship should not light the map")
	}
	if !vision.VisibleAt(1, 1) {
		t.Fatal("disabled airfield should still light its cell")
	}
	if !vision.ExploredAt(5, 5) {
		t.Fatal("visible cells should become explored")
	}
}

func TestUpdateVisionSkipsWhenFogDisabled(t *testing.T) {
	ms := &state.MissionState{
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				"s": {
					Uid: "s", CurHP: 10, SightRange: 4,
					BelongPlayer: faction.HumanAlpha, CurPos: objPos.New(2, 2),
				},
			},
		},
	}
	(&MissionManager{state: ms}).updateVision()
	if ms.Player.Visions != nil {
		t.Fatal("disabled fog should not create vision maps")
	}
}
