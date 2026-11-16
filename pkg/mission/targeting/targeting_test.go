package targeting

import (
	"fmt"
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
)

// TestBuildPlanCoversFourSpatialFleets 验证舰队覆盖：两个机场的对舰队列首批要覆盖四个方向的全部敌舰队。
// 场景：4 个舰队分布在 (40,40)/(40,215)/(215,40)/(215,215)，各由相距 5-6 格的航母与战列舰组成（半径 20）。
// 预期：ExpiresTick=30+180=210；左右基地对舰队列前 2 项 FleetID 去重后为 4，失败即有舰队无人攻击。
// 覆盖：2 个 Available 12、Range 1000 的 Base 与 8 艘 EnemyShip，BuildPlan 后统计队列 FleetID 集合。
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

// TestBuildPlanDoesNotMutateSnapshotOrder 验证 BuildPlan 只读 Snapshot，不在调用方切片上原地排序。
// 场景：快照在后台与主线程间传递，敌机 z-plane（Threat 0.5、距 2）与 a-plane（Threat 0.8、距 1）乱序输入。
// 预期：队列首项为评分更高的 a-plane，而 snapshot.EnemyPlanes[0] 仍为 z-plane，失败即后台规划篡改主线程数据。
// 覆盖：单基地 Available 1、Range 1000 的 TypePlane 队列，BuildPlan 后断言 queue[0].UID 与原切片首元素。
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

// TestBuildPlanDoesNotRepeatAirTargets 验证同一架敌机在同一基地的对空队列里只出现一次。
// 场景：基地有 12 架可用 CAP，敌空域只有 2 架飞机；若去重失败，整个 CAP 会重复扑向同一目标。
// 预期：队列长度等于敌机数 2，且每个 UID 只出现 1 次。
// 覆盖：构造 1 个基地/2 架敌机，BuildPlan 后遍历队列统计 UID 次数。
func TestBuildPlanDoesNotRepeatAirTargets(t *testing.T) {
	snapshot := Snapshot{
		Tick: 1,
		Bases: []Base{{
			UID: "base", Player: faction.HumanAlpha, Pos: Point{X: 0, Y: 0},
			Groups: []Group{{TargetType: object.TypePlane, Available: 12, Range: 3000}},
		}},
		EnemyPlanes: []EnemyPlane{
			{UID: "scout", Player: faction.ComputerAlpha, Pos: Point{X: 5, Y: 0}, Threat: 0.5, Airborne: true},
			{UID: "bomber", Player: faction.ComputerAlpha, Pos: Point{X: 8, Y: 0}, Threat: 0.5, Airborne: true},
		},
	}

	plan := BuildPlan(snapshot)
	queue := plan.BaseQueues["base"][object.TypePlane]
	if len(queue) != 2 {
		t.Fatalf("air queue = %+v, want each enemy plane exactly once", queue)
	}
	seen := map[string]int{}
	for _, ref := range queue {
		seen[ref.UID]++
	}
	for uid, count := range seen {
		if count != 1 {
			t.Fatalf("air queue repeats %s %d times: %+v", uid, count, queue)
		}
	}
}

// TestBuildPlanSkipsOutOfRangeTargets 验证基地按 Group.Range 过滤可达目标，射程外的敌舰不进对舰队列。
// 场景：short-range 基地射程 10，far-ship 在 (50,0) 距离 50；不过滤会让飞机起飞后永远追不上目标。
// 预期：BaseQueues["short-range"][TypeShip] 长度为 0，失败意味着编队被派往不可达目标。
// 覆盖：单基地（Pos 0,0、Available 1、Range 10）与单敌舰（Pos 50,0、Value 1），断言队列为空。
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

// TestBuildPlanUsesLongestAvailableShipRange 验证同基地多机组按最大 Range 判可达，短腿机组不拖累远程。
// 场景：mixed-range 基地同时有 Range 20 与 Range 100 的对舰机组，far-ship 在 (50,0)，只有远程组够得到。
// 预期：队列非空且首项为 far-ship，失败意味着长航程机型被同基地短程机型过滤，永远拿不到远处目标。
// 覆盖：两个 Available 1 的机组（20/100）与 Pos(50,0) 敌舰，BuildPlan 后断言 queue[0].UID。
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

// TestBuildPlanUsesNearestReachableFleetRepresentative 验证代表舰优先取舰队内航程可达者，而不是单纯按价值。
// 场景：基地射程 25，远航母在 (30,0)、Value 1 超程，近驱逐在 (10,0)、Value 0.35，两者相距 20 格同队。
// 预期：队列首项为 near-destroyer，失败意味着队列指向飞机够不着的目标，出勤后只能空转。
// 覆盖：单基地 Available 1、Range 25 的 TypeShip 机组与两艘敌舰，断言队列非空且 queue[0].UID。
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

// TestBuildPlanFillsEveryReachableShipBase 验证覆盖分配后，每个有对舰能力且目标可达的基地都要独立填满队列。
// 场景：primary(0,0) 与 secondary(8,0) 各 12 架、射程 100，同一艘 enemy 在 (12,0)，覆盖阶段只给一个基地归属。
// 预期：两个基地的 TypeShip 队列都非空，失败意味着没分到舰队的航母/机场只能反复起飞战斗机。
// 覆盖：两个 Base 与单艘 EnemyShip 的 Snapshot，BuildPlan 后遍历 snapshot.Bases 断言各队列长度大于 0。
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
