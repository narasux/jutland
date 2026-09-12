package manager

import (
	"testing"
	"time"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/targeting"
)

func TestNextTargetUIDSkipsDestroyedTargets(t *testing.T) {
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Ships: map[string]*objUnit.BattleShip{
					"alive": {Uid: "alive", CurHP: 10},
				},
			},
		},
		targetingPlan: targeting.Plan{
			BaseQueues: map[string]map[object.Type][]targeting.TargetRef{
				"base": {
					object.TypeShip: {
						{UID: "dead", TargetType: object.TypeShip},
						{UID: "alive", TargetType: object.TypeShip},
					},
				},
			},
		},
		targetingCursors: map[string]map[object.Type]int{},
	}

	uid, ok := manager.nextTargetUID("base", object.TypeShip)
	if !ok || uid != "alive" {
		t.Fatalf("next target = %q, %v; want alive", uid, ok)
	}
	if !manager.targetingDirty {
		t.Fatal("consuming a target should mark the plan dirty")
	}
}

func TestUpdateTargetPlanningConsumesBufferedPlan(t *testing.T) {
	oldTemplate, hadTemplate := objUnit.PlaneMap["test-bomber"]
	objUnit.PlaneMap["test-bomber"] = &objUnit.Plane{
		Name:   "test-bomber",
		Type:   objUnit.PlaneTypeDiveBomber,
		Weapon: objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{}}},
	}
	t.Cleanup(func() {
		if hadTemplate {
			objUnit.PlaneMap["test-bomber"] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, "test-bomber")
		}
	})

	af := objBuilding.NewAirfield(
		objPos.New(10, 10), 0, 6, 0.8, faction.HumanAlpha, 0,
		[]objUnit.PlaneGroup{{Name: "test-bomber", MaxCount: 1}},
	)
	af.Aircraft.Groups[0].TargetType = object.TypeShip
	af.Aircraft.Groups[0].CurCount = 1

	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Airfields: map[string]*objBuilding.Airfield{af.Uid: af},
				Ships: map[string]*objUnit.BattleShip{
					"enemy": {Uid: "enemy", CurHP: 100, CurPos: objPos.New(80, 80), BelongPlayer: faction.ComputerAlpha},
				},
			},
		},
		targetingResults: make(chan targeting.Plan, 1),
		targetingDirty:   true,
		targetingCursors: map[string]map[object.Type]int{},
	}
	manager.simTick = 1
	manager.updateTargetPlanning()
	if !manager.targetingBusy {
		t.Fatal("first targeting update should submit one background job")
	}

	select {
	case plan := <-manager.targetingResults:
		manager.targetingPlan = plan
		manager.targetingBusy = false
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for asynchronous target plan")
	}
	if len(manager.targetingPlan.BaseQueues[af.Uid][object.TypeShip]) == 0 {
		t.Fatal("completed target plan should contain a ship target for the airfield")
	}
}

func TestTakeOffFromBaseRotatesTargetTypes(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	const fighterName = "rotate-fighter"
	const bomberName = "rotate-bomber"
	oldFighter, hadFighter := objUnit.PlaneMap[fighterName]
	oldBomber, hadBomber := objUnit.PlaneMap[bomberName]
	objUnit.PlaneMap[fighterName] = &objUnit.Plane{
		Name: fighterName, Type: objUnit.PlaneTypeFighter,
		Weapon: objUnit.PlaneWeapon{Guns: []*objUnit.Gun{{}}},
	}
	objUnit.PlaneMap[bomberName] = &objUnit.Plane{
		Name: bomberName, Type: objUnit.PlaneTypeDiveBomber,
		Weapon: objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{}}},
	}
	t.Cleanup(func() {
		if hadFighter {
			objUnit.PlaneMap[fighterName] = oldFighter
		} else {
			delete(objUnit.PlaneMap, fighterName)
		}
		if hadBomber {
			objUnit.PlaneMap[bomberName] = oldBomber
		} else {
			delete(objUnit.PlaneMap, bomberName)
		}
	})

	airfield := objBuilding.NewAirfield(
		objPos.New(10, 10), 0, 6, 0.8, faction.HumanAlpha, 0,
		[]objUnit.PlaneGroup{
			{Name: fighterName, MaxCount: 1},
			{Name: bomberName, MaxCount: 1},
		},
	)
	airfield.StockPlane(fighterName)
	airfield.StockPlane(bomberName)
	enemyPlane := &objUnit.Plane{Uid: "enemy-plane", CurHP: 10, FlightPhase: objUnit.PlaneFlightPhaseCruising}
	enemyShip := &objUnit.BattleShip{Uid: "enemy-ship", CurHP: 10}
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Planes: map[string]*objUnit.Plane{enemyPlane.Uid: enemyPlane},
				Ships:  map[string]*objUnit.BattleShip{enemyShip.Uid: enemyShip},
			},
		},
		targetingPlan: targeting.Plan{
			BaseQueues: map[string]map[object.Type][]targeting.TargetRef{
				airfield.Uid: {
					object.TypePlane: {{UID: enemyPlane.Uid, TargetType: object.TypePlane}},
					object.TypeShip:  {{UID: enemyShip.Uid, TargetType: object.TypeShip}},
				},
			},
		},
		targetingCursors:   map[string]map[object.Type]int{},
		takeoffTypeCursors: map[string]int{},
	}

	_, firstType, _, ok := manager.takeOffFromBase(airfield)
	if !ok || firstType != object.TypePlane {
		t.Fatalf(
			"first sortie type = %v, %v; want plane; groups=%+v plan=%+v",
			firstType,
			ok,
			airfield.Aircraft.Groups,
			manager.targetingPlan.BaseQueues[airfield.Uid],
		)
	}
	_, secondType, _, ok := manager.takeOffFromBase(airfield)
	if !ok || secondType != object.TypeShip {
		t.Fatalf("second sortie type = %v, %v; want ship", secondType, ok)
	}
}

func TestTakeOffFromBaseChoosesPlaneThatCanReachTarget(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	const shortName = "rotate-short"
	const longName = "rotate-long"
	oldShort, hadShort := objUnit.PlaneMap[shortName]
	oldLong, hadLong := objUnit.PlaneMap[longName]
	objUnit.PlaneMap[shortName] = &objUnit.Plane{
		Name: shortName, Type: objUnit.PlaneTypeDiveBomber, Range: 20,
		Weapon: objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{}}},
	}
	objUnit.PlaneMap[longName] = &objUnit.Plane{
		Name: longName, Type: objUnit.PlaneTypeDiveBomber, Range: 100,
		Weapon: objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{}}},
	}
	t.Cleanup(func() {
		if hadShort {
			objUnit.PlaneMap[shortName] = oldShort
		} else {
			delete(objUnit.PlaneMap, shortName)
		}
		if hadLong {
			objUnit.PlaneMap[longName] = oldLong
		} else {
			delete(objUnit.PlaneMap, longName)
		}
	})

	airfield := objBuilding.NewAirfield(
		objPos.New(0, 0), 0, 6, 0.8, faction.HumanAlpha, 0,
		[]objUnit.PlaneGroup{
			{Name: shortName, MaxCount: 1},
			{Name: longName, MaxCount: 1},
		},
	)
	airfield.StockPlane(shortName)
	airfield.StockPlane(longName)
	enemyShip := &objUnit.BattleShip{
		Uid: "far-enemy", CurHP: 10, CurPos: objPos.New(80, 0),
		BelongPlayer: faction.ComputerAlpha,
	}
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Airfields: map[string]*objBuilding.Airfield{airfield.Uid: airfield},
				Ships:     map[string]*objUnit.BattleShip{enemyShip.Uid: enemyShip},
			},
		},
		targetingPlan: targeting.Plan{
			BaseQueues: map[string]map[object.Type][]targeting.TargetRef{
				airfield.Uid: {
					object.TypeShip: {{UID: enemyShip.Uid, TargetType: object.TypeShip}},
				},
			},
		},
		targetingCursors:   map[string]map[object.Type]int{},
		takeoffTypeCursors: map[string]int{},
	}

	plane, targetType, targetUID, ok := manager.takeOffFromBase(airfield)
	if !ok || targetType != object.TypeShip || targetUID != enemyShip.Uid {
		t.Fatalf("takeoff = %v, %v, %q; want ship target %s", plane, targetType, targetUID, enemyShip.Uid)
	}
	if plane == nil || plane.Name != longName {
		t.Fatalf("plane = %v, want long-range group %s", plane, longName)
	}
}
