package unit

import "testing"

func TestAircraftStatusClassifiesFlightPhasesAndLosses(t *testing.T) {
	aircraft := ShipAircraft{Groups: []PlaneGroup{
		{Name: "fighter", MaxCount: 10, CurCount: 4},
		{Name: "bomber", MaxCount: 6, CurCount: 2},
	}}
	planes := map[string]*Plane{
		"f1": {Name: "fighter", BelongShip: "carrier", FlightPhase: PlaneFlightPhaseTakingOff},
		"f2": {Name: "fighter", BelongShip: "carrier", FlightPhase: PlaneFlightPhaseCruising},
		"f3": {Name: "fighter", BelongShip: "carrier", FlightPhase: PlaneFlightPhaseLandingStaging},
		"f4": {Name: "fighter", BelongShip: "carrier", FlightPhase: PlaneFlightPhaseLandingDeck},
		"b1": {Name: "bomber", BelongShip: "carrier", FlightPhase: ""},
		"x1": {Name: "fighter", BelongShip: "other", FlightPhase: PlaneFlightPhaseCruising},
	}

	status := aircraft.Status("carrier", planes)
	fighter := status.Groups[0]
	if fighter.Standby != 4 || fighter.InCombat != 2 || fighter.Returning != 2 || fighter.Lost != 2 {
		t.Fatalf("fighter status = %+v", fighter)
	}
	bomber := status.Groups[1]
	if bomber.Standby != 2 || bomber.InCombat != 1 || bomber.Returning != 0 || bomber.Lost != 3 {
		t.Fatalf("bomber status = %+v", bomber)
	}
	if status.Total.Alive() != 11 || status.Total.Initial() != 16 {
		t.Fatalf("total status = %+v", status.Total)
	}
}

func TestDisabledAircraftCannotTakeOff(t *testing.T) {
	aircraft := ShipAircraft{Disable: true, Groups: []PlaneGroup{{Name: "fighter", MaxCount: 1, CurCount: 1}}}
	if plane := aircraft.TakeOff(&BattleShip{}, 0); plane != nil {
		t.Fatal("disabled aircraft group took off")
	}
}
