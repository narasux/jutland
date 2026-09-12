package targeting

import (
	"fmt"
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
)

func TestBuildPlanCoversFourSpatialFleets(t *testing.T) {
	const (
		leftBaseUID  = "left-airfield"
		rightBaseUID = "right-airfield"
	)
	snapshot := Snapshot{
		Tick: 30,
		Bases: []Base{
			{
				UID: leftBaseUID, Player: faction.HumanAlpha, Pos: Point{X: 103, Y: 156},
				Groups: []Group{{TargetType: object.TypeShip, Available: 12, Range: 1000}},
			},
			{
				UID: rightBaseUID, Player: faction.HumanAlpha, Pos: Point{X: 161, Y: 166},
				Groups: []Group{{TargetType: object.TypeShip, Available: 12, Range: 1000}},
			},
		},
		EnemyShips: []EnemyShip{
			{UID: "nw-carrier", Player: faction.ComputerAlpha, Pos: Point{X: 40, Y: 40}, Value: 1},
			{UID: "nw-battleship", Player: faction.ComputerAlpha, Pos: Point{X: 46, Y: 40}, Value: 0.85},
			{UID: "sw-carrier", Player: faction.ComputerAlpha, Pos: Point{X: 40, Y: 215}, Value: 1},
			{UID: "sw-battleship", Player: faction.ComputerAlpha, Pos: Point{X: 46, Y: 215}, Value: 0.85},
			{UID: "ne-carrier", Player: faction.ComputerAlpha, Pos: Point{X: 215, Y: 40}, Value: 1},
			{UID: "ne-battleship", Player: faction.ComputerAlpha, Pos: Point{X: 210, Y: 40}, Value: 0.85},
			{UID: "se-carrier", Player: faction.ComputerAlpha, Pos: Point{X: 215, Y: 215}, Value: 1},
			{UID: "se-battleship", Player: faction.ComputerAlpha, Pos: Point{X: 210, Y: 215}, Value: 0.85},
		},
	}

	plan := BuildPlan(snapshot)
	if plan.ExpiresTick != snapshot.Tick+PlanTTLTicks {
		t.Fatalf("expiry tick = %d, want %d", plan.ExpiresTick, snapshot.Tick+PlanTTLTicks)
	}

	fleets := make(map[string]struct{})
	for _, baseUID := range []string{leftBaseUID, rightBaseUID} {
		queue := plan.BaseQueues[baseUID][object.TypeShip]
		for idx := 0; idx < 2 && idx < len(queue); idx++ {
			fleets[queue[idx].FleetID] = struct{}{}
		}
	}
	if len(fleets) != 4 {
		t.Fatalf("initial base queues cover %d fleets, want all 4: %+v", len(fleets), plan.BaseQueues)
	}
}

func TestBuildPlanDoesNotMutateSnapshotOrder(t *testing.T) {
	snapshot := Snapshot{
		Tick: 1,
		Bases: []Base{{
			UID: "base", Player: faction.HumanAlpha, Pos: Point{X: 0, Y: 0},
			Groups: []Group{{TargetType: object.TypePlane, Available: 1, Range: 1000}},
		}},
		EnemyPlanes: []EnemyPlane{
			{UID: "z-plane", Player: faction.ComputerAlpha, Pos: Point{X: 2, Y: 0}, Threat: 0.5, Airborne: true},
			{UID: "a-plane", Player: faction.ComputerAlpha, Pos: Point{X: 1, Y: 0}, Threat: 0.8, Airborne: true},
		},
	}

	plan := BuildPlan(snapshot)
	if plan.BaseQueues["base"][object.TypePlane][0].UID != "a-plane" {
		t.Fatalf("air queue = %+v, want highest-scoring a-plane first", plan.BaseQueues["base"][object.TypePlane])
	}
	if snapshot.EnemyPlanes[0].UID != "z-plane" {
		t.Fatal("BuildPlan mutated caller-owned snapshot ordering")
	}
}

func TestBuildPlanSkipsOutOfRangeTargets(t *testing.T) {
	snapshot := Snapshot{
		Tick: 1,
		Bases: []Base{{
			UID: "short-range", Player: faction.HumanAlpha, Pos: Point{X: 0, Y: 0},
			Groups: []Group{{TargetType: object.TypeShip, Available: 1, Range: 10}},
		}},
		EnemyShips: []EnemyShip{{
			UID: "far-ship", Player: faction.ComputerAlpha, Pos: Point{X: 50, Y: 0}, Value: 1,
		}},
	}

	plan := BuildPlan(snapshot)
	if queue := plan.BaseQueues["short-range"][object.TypeShip]; len(queue) != 0 {
		t.Fatalf("short-range base queue = %+v, want no out-of-range target", queue)
	}
}

func TestBuildPlanUsesLongestAvailableShipRange(t *testing.T) {
	snapshot := Snapshot{
		Tick: 1,
		Bases: []Base{{
			UID: "mixed-range", Player: faction.HumanAlpha, Pos: Point{X: 0, Y: 0},
			Groups: []Group{
				{TargetType: object.TypeShip, Available: 1, Range: 20},
				{TargetType: object.TypeShip, Available: 1, Range: 100},
			},
		}},
		EnemyShips: []EnemyShip{{
			UID: "far-ship", Player: faction.ComputerAlpha, Pos: Point{X: 50, Y: 0}, Value: 1,
		}},
	}

	plan := BuildPlan(snapshot)
	queue := plan.BaseQueues["mixed-range"][object.TypeShip]
	if len(queue) == 0 || queue[0].UID != "far-ship" {
		t.Fatalf("queue = %+v, want target reachable by the long-range group", queue)
	}
}

func TestBuildPlanUsesNearestReachableFleetRepresentative(t *testing.T) {
	snapshot := Snapshot{
		Tick: 1,
		Bases: []Base{{
			UID: "base", Player: faction.HumanAlpha, Pos: Point{X: 0, Y: 0},
			Groups: []Group{{TargetType: object.TypeShip, Available: 1, Range: 25}},
		}},
		EnemyShips: []EnemyShip{
			{UID: "far-carrier", Player: faction.ComputerAlpha, Pos: Point{X: 30, Y: 0}, Value: 1},
			{UID: "near-destroyer", Player: faction.ComputerAlpha, Pos: Point{X: 10, Y: 0}, Value: 0.35},
		},
	}

	plan := BuildPlan(snapshot)
	queue := plan.BaseQueues["base"][object.TypeShip]
	if len(queue) == 0 || queue[0].UID != "near-destroyer" {
		t.Fatalf("queue = %+v, want nearest reachable fleet representative first", queue)
	}
}

func TestBuildPlanFillsEveryReachableShipBase(t *testing.T) {
	snapshot := Snapshot{
		Tick: 1,
		Bases: []Base{
			{
				UID: "primary", Player: faction.HumanAlpha, Pos: Point{X: 0, Y: 0},
				Groups: []Group{{TargetType: object.TypeShip, Available: 12, Range: 100}},
			},
			{
				UID: "secondary", Player: faction.HumanAlpha, Pos: Point{X: 8, Y: 0},
				Groups: []Group{{TargetType: object.TypeShip, Available: 12, Range: 100}},
			},
		},
		EnemyShips: []EnemyShip{{
			UID: "enemy", Player: faction.ComputerAlpha, Pos: Point{X: 12, Y: 0}, Value: 1,
		}},
	}

	plan := BuildPlan(snapshot)
	for _, base := range snapshot.Bases {
		if queue := plan.BaseQueues[base.UID][object.TypeShip]; len(queue) == 0 {
			t.Fatalf("base %s has no ship target queue: %+v", base.UID, plan.BaseQueues)
		}
	}
}

func BenchmarkBuildPlan(b *testing.B) {
	snapshot := Snapshot{
		Tick: 30,
		Bases: []Base{
			{UID: "left", Player: faction.HumanAlpha, Pos: Point{X: 103, Y: 156}, Groups: []Group{
				{TargetType: object.TypePlane, Available: 12, Range: 3000},
				{TargetType: object.TypeShip, Available: 12, Range: 3000},
			}},
			{UID: "right", Player: faction.HumanAlpha, Pos: Point{X: 161, Y: 166}, Groups: []Group{
				{TargetType: object.TypePlane, Available: 12, Range: 3000},
				{TargetType: object.TypeShip, Available: 12, Range: 3000},
			}},
		},
	}
	for idx := 0; idx < 60; idx++ {
		snapshot.EnemyShips = append(snapshot.EnemyShips, EnemyShip{
			UID:    fmt.Sprintf("ship-%02d", idx),
			Player: faction.ComputerAlpha,
			Pos:    Point{X: float64(20 + (idx%6)*4), Y: float64(20 + (idx/6)*4)},
			Value:  0.5,
		})
	}
	for idx := 0; idx < 200; idx++ {
		snapshot.EnemyPlanes = append(snapshot.EnemyPlanes, EnemyPlane{
			UID:      fmt.Sprintf("plane-%03d", idx),
			Player:   faction.ComputerAlpha,
			Pos:      Point{X: float64(80 + idx%20), Y: float64(80 + idx/20)},
			Threat:   0.5,
			Airborne: true,
		})
	}

	b.ResetTimer()
	for range b.N {
		_ = BuildPlan(snapshot)
	}
}
