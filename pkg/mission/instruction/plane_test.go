package instruction

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

func TestPlaneReturnRecoversOnlyAfterLandingDeck(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	defer func() { config.G = oldSettings }()

	const planeName = "test-landing-plane"

	oldTemplate, hadTemplate := objUnit.PlaneMap[planeName]
	objUnit.PlaneMap[planeName] = &objUnit.Plane{
		Name:        planeName,
		TotalHP:     100,
		CurHP:       100,
		MaxSpeed:    0.5,
		RotateSpeed: 360,
		RemainRange: 100,
	}
	defer func() {
		if hadTemplate {
			objUnit.PlaneMap[planeName] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, planeName)
		}
	}()

	ship := &objUnit.BattleShip{
		Uid:          "carrier",
		Length:       128,
		CurPos:       objPos.NewR(50, 50),
		CurRotation:  0,
		BelongPlayer: faction.HumanAlpha,
		Aircraft: objUnit.ShipAircraft{
			Groups: []objUnit.PlaneGroup{{
				Name:     planeName,
				MaxCount: 1,
				CurCount: 0,
			}},
		},
	}
	plane := objUnit.NewPlane(planeName, objPos.NewR(50, 48), 0, ship.Uid, faction.HumanAlpha)
	ms := &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{
				MapCfg: &mapcfg.MapCfg{Width: 100, Height: 100},
			},
		},
		Arena: state.MissionArenaState{
			Ships:  map[string]*objUnit.BattleShip{ship.Uid: ship},
			Planes: map[string]*objUnit.Plane{plane.Uid: plane},
		},
	}
	returnInstr := NewPlaneReturn(plane.Uid)

	if err := returnInstr.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if ship.Aircraft.Groups[0].CurCount != 0 {
		t.Fatalf("plane recovered during approach start")
	}
	if plane.FlightPhase != objUnit.PlaneFlightPhaseLandingApproach {
		t.Fatalf("phase = %s, want %s", plane.FlightPhase, objUnit.PlaneFlightPhaseLandingApproach)
	}

	plane.CurPos = objPos.NewR(50, 51.9)
	plane.CurRotation = 180
	for i := 0; i < 20 && plane.FlightPhase != objUnit.PlaneFlightPhaseLandingDeck; i++ {
		if err := returnInstr.Exec(ms); err != nil {
			t.Fatal(err)
		}
	}
	if ship.Aircraft.Groups[0].CurCount != 0 {
		t.Fatalf("plane recovered before deck landing")
	}
	if plane.FlightPhase != objUnit.PlaneFlightPhaseLandingDeck {
		t.Fatalf("phase = %s, want %s", plane.FlightPhase, objUnit.PlaneFlightPhaseLandingDeck)
	}

	for i := 0; i < 20 && !returnInstr.Executed(); i++ {
		if err := returnInstr.Exec(ms); err != nil {
			t.Fatal(err)
		}
	}
	if !returnInstr.Executed() {
		t.Fatalf("return instruction did not finish")
	}
	if ship.Aircraft.Groups[0].CurCount != 1 {
		t.Fatalf("recovered plane count = %d, want 1", ship.Aircraft.Groups[0].CurCount)
	}
	if _, ok := ms.Arena.Planes[plane.Uid]; ok {
		t.Fatalf("plane still exists after recovery")
	}
}
