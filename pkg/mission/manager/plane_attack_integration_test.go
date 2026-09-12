package manager

import (
	"testing"

	audioPlayer "github.com/narasux/jutland/pkg/audio/player"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/object"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

func TestBomberReleasesBeforeReturning(t *testing.T) {
	testCases := []struct {
		name      string
		planeName string
		enemyX    float64
	}{
		{name: "B-17G", planeName: "B-17G", enemyX: 100},
		{name: "B6N2", planeName: "B6N2", enemyX: 100},
		{name: "B6N2-full-range", planeName: "B6N2", enemyX: 220},
		{name: "D4Y3", planeName: "D4Y3", enemyX: 100},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testBomberReleasesBeforeReturning(t, testCase.planeName, testCase.enemyX)
		})
	}
}

func testBomberReleasesBeforeReturning(t *testing.T, planeName string, enemyX float64) {
	t.Helper()
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	airfield := objBuilding.NewAirfield(
		objPos.New(20, 150), 0, 6, 0.8, faction.HumanAlpha, 0,
		[]objUnit.PlaneGroup{{Name: planeName, MaxCount: 1}},
	)
	airfield.StockPlane(planeName)
	enemy := &objUnit.BattleShip{
		Uid:          "enemy",
		Type:         objUnit.ShipTypeCruiser,
		CurHP:        1000,
		TotalHP:      1000,
		CurPos:       objPos.NewR(enemyX, 150),
		BelongPlayer: faction.ComputerAlpha,
	}
	missionState := &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{
				MapCfg: &mapcfg.MapCfg{Width: 300, Height: 300},
			},
		},
		Arena: state.MissionArenaState{
			Airfields: map[string]*objBuilding.Airfield{airfield.Uid: airfield},
			Planes:    map[string]*objUnit.Plane{},
			Ships:     map[string]*objUnit.BattleShip{enemy.Uid: enemy},
		},
	}
	manager := &MissionManager{
		state:            missionState,
		instructionSet:   NewInstructionSet(),
		weaponFirePlayer: audioPlayer.NewWeaponFire(),
		targetingDirty:   true,
		targetingCursors: map[string]map[object.Type]int{},
	}
	refreshAlertTargetPlan(manager)
	manager.updateAirfieldAlertLaunch()
	if len(missionState.Arena.Planes) != 1 {
		t.Fatalf("bomber count = %d, want 1", len(missionState.Arena.Planes))
	}

	var bomber *objUnit.Plane
	for _, plane := range missionState.Arena.Planes {
		bomber = plane
	}
	if attack := manager.instructionSet.Items()[instr.GenInstrUid(instr.NamePlaneAttack, bomber.Uid)]; attack == nil {
		t.Fatal("bomber launched without an attack instruction")
	}
	for _, bomb := range bomber.Weapon.Bombs {
		if bomb.Released {
			t.Fatal("bomber inherited a released bomb before its first attack frame")
		}
	}

	released := false
	for frame := 0; frame < 5000; frame++ {
		manager.instructionSet.ExecAll(missionState)
		manager.updatePlaneAttackOrReturn()
		manager.updatePlaneWeaponFire()
		for _, bomb := range bomber.Weapon.Bombs {
			released = released || bomb.Released
		}
		for _, torpedo := range bomber.Weapon.Torpedoes {
			released = released || torpedo.Released
		}
		for _, rocket := range bomber.Weapon.Rockets {
			released = released || rocket.Exhausted()
		}
		if released {
			break
		}
		if bomber.FlightPhase == objUnit.PlaneFlightPhaseLandingStaging {
			t.Fatalf(
				"bomber returned before releasing: frame=%d pos=%s hp=%.1f remainRange=%.1f",
				frame,
				bomber.CurPos.String(),
				bomber.CurHP,
				bomber.RemainRange,
			)
		}
	}
	if !released {
		t.Fatalf("bomber did not release within test window: pos=%s target=%s", bomber.CurPos.String(), bomber.CurAttackTarget)
	}
}

func TestAirfieldDoesNotLaunchBomberBeyondRange(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	airfield := objBuilding.NewAirfield(
		objPos.New(20, 150), 0, 6, 0.8, faction.HumanAlpha, 0,
		[]objUnit.PlaneGroup{{Name: "B6N2", MaxCount: 1}},
	)
	airfield.StockPlane("B6N2")
	enemy := &objUnit.BattleShip{
		Uid:          "far-enemy",
		Type:         objUnit.ShipTypeCruiser,
		CurHP:        1000,
		TotalHP:      1000,
		CurPos:       objPos.New(260, 150),
		BelongPlayer: faction.ComputerAlpha,
	}
	missionState := &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{
				MapCfg: &mapcfg.MapCfg{Width: 300, Height: 300},
			},
		},
		Arena: state.MissionArenaState{
			Airfields: map[string]*objBuilding.Airfield{airfield.Uid: airfield},
			Planes:    map[string]*objUnit.Plane{},
			Ships:     map[string]*objUnit.BattleShip{enemy.Uid: enemy},
		},
	}
	manager := &MissionManager{
		state:            missionState,
		instructionSet:   NewInstructionSet(),
		weaponFirePlayer: audioPlayer.NewWeaponFire(),
		targetingDirty:   true,
		targetingCursors: map[string]map[object.Type]int{},
	}
	refreshAlertTargetPlan(manager)
	manager.updateAirfieldAlertLaunch()
	if len(missionState.Arena.Planes) != 0 {
		t.Fatalf("airfield launched %d bomber(s) against an out-of-range target", len(missionState.Arena.Planes))
	}
}

func TestPearlHarborAirfieldHasReachableStrikeTargets(t *testing.T) {
	manager := New("PearlHarbor1941")
	refreshAlertTargetPlan(manager)

	for _, airfield := range manager.state.Arena.Airfields {
		for _, group := range airfield.Aircraft.Groups {
			if group.TargetType != object.TypeShip || group.CurCount <= 0 {
				continue
			}
			queue := manager.targetingPlan.BaseQueues[airfield.Uid][object.TypeShip]
			if len(queue) == 0 {
				t.Fatalf("Pearl Harbor airfield has no reachable ship target for %s", group.Name)
			}
		}
	}
}

func TestPearlHarborEveryStrikeBaseHasReachableTargets(t *testing.T) {
	manager := New("PearlHarbor1941")
	refreshAlertTargetPlan(manager)

	for uid, ship := range manager.state.Arena.Ships {
		if !ship.Aircraft.HasPlane {
			continue
		}
		for _, group := range ship.Aircraft.Groups {
			if group.TargetType != object.TypeShip || group.CurCount <= 0 {
				continue
			}
			queue := manager.targetingPlan.BaseQueues[uid][object.TypeShip]
			if len(queue) == 0 {
				t.Fatalf("ship %s has no reachable ship target for %s", uid, group.Name)
			}
		}
	}
}
