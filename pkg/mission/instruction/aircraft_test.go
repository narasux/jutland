package instruction

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func TestAircraftToggleInstructions(t *testing.T) {
	ship := &objUnit.BattleShip{Uid: "carrier"}
	missionState := &state.MissionState{
		Arena: state.MissionArenaState{Ships: map[string]*objUnit.BattleShip{ship.Uid: ship}},
	}

	disable := NewDisableAircraft(ship.Uid)
	if err := disable.Exec(missionState); err != nil {
		t.Fatal(err)
	}
	if !ship.Aircraft.Disable || !disable.Executed() {
		t.Fatal("disable instruction did not disable takeoff")
	}

	enable := NewEnableAircraft(ship.Uid)
	if err := enable.Exec(missionState); err != nil {
		t.Fatal(err)
	}
	if ship.Aircraft.Disable || !enable.Executed() {
		t.Fatal("enable instruction did not enable takeoff")
	}
}
