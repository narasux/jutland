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

// TestNextTargetUIDSkipsDestroyedTargets 验证基地目标游标会跳过已从战场消失/已沉没的队列项。
// 场景：目标计划里基地 "base" 的 TypeShip 队列依次是 "dead" 与 "alive"，
// 但 Arena.Ships 只登记了 CurHP=10 的 "alive"，模拟计划生成后目标被击沉的情况。
// 预期：nextTargetUID 返回 "alive" 且 ok=true，消费后 targetingDirty 变为 true 以便重算计划；
// 失败意味着派机会锁定一个不存在的目标，飞机原地盘旋或直接飞向空海域。
// 覆盖：手写 targeting.Plan 队列与 targetingCursors，调用一次 nextTargetUID 后检查返回值与脏标记。
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

// TestUpdateTargetPlanningConsumesBufferedPlan 验证 updateTargetPlanning 只提交一个后台计划并消费其结果。
// 场景：临时在 PlaneMap 注册俯冲轰炸机 "test-bomber"（带 1 个炸弹释放器），机场 (10,10) 库存 1 架、
// 编组 TargetType=TypeShip、CurCount=1，敌舰在 (80,80)；simTick=1、targetingDirty=true，
// targetingResults 是容量 1 的缓冲通道。
// 预期：首次调用后 targetingBusy=true（同一时刻只跑一个后台任务）；2 秒内能从通道读到计划，
// 且计划里该机场的 TypeShip 队列非空。失败意味着提交逻辑重复并发、主线程阻塞，或计划不含敌舰。
// 覆盖：注册/清理模板，手工接收 targetingResults 回填 targetingPlan，最后检查 BaseQueues 长度。
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
					"enemy": {
						Uid:          "enemy",
						CurHP:        100,
						CurPos:       objPos.New(80, 80),
						BelongPlayer: faction.ComputerAlpha,
					},
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

// TestTakeOffFromBaseRotatesTargetTypes 验证机场起飞会在不同打击目标类型之间轮转。
// 场景：同一个机场混编 "rotate-fighter"（战斗机，目标类型 Plane）与 "rotate-bomber"
// （俯冲轰炸机，目标类型 Ship），两者各有 1 架库存；目标计划同时给出 TypePlane 与 TypeShip 队列。
// 预期：第一次 takeOffFromBase 返回 object.TypePlane，第二次返回 object.TypeShip；
// 失败意味着 takeoffTypeCursors 不推进，所有架次都扑向同一类目标，空袭或制空会失衡。
// 覆盖：临时注册两种模板并在 Cleanup 中还原，连续两次调用 takeOffFromBase(airfield,nil) 比较返回类型。
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

	_, firstType, _, ok := manager.takeOffFromBase(airfield, nil)
	if !ok || firstType != object.TypePlane {
		t.Fatalf(
			"first sortie type = %v, %v; want plane; groups=%+v plan=%+v",
			firstType,
			ok,
			airfield.Aircraft.Groups,
			manager.targetingPlan.BaseQueues[airfield.Uid],
		)
	}
	_, secondType, _, ok := manager.takeOffFromBase(airfield, nil)
	if !ok || secondType != object.TypeShip {
		t.Fatalf("second sortie type = %v, %v; want ship", secondType, ok)
	}
}

// TestTakeOffFromBaseChoosesPlaneThatCanReachTarget 验证派机时按“往返航程”挑选能返航的机型。
// 场景：机场在 (0,0)，编组里 "rotate-short" 航程 20、"rotate-long" 航程 200，均为俯冲轰炸机；
// 敌舰在 (80,0)，单程 80、往返 160，目标计划只给出 TypeShip 队列。
// 预期：takeOffFromBase 返回长航程编组的飞机、类型为 TypeShip、目标 UID 为 "far-enemy"；
// 失败意味着短航程机型被派出去后无法返航，或者反过来有航程足够的机型却不敢起飞。
// 覆盖：临时注册两档 Range 模板，断言 plane.Name=="rotate-long" 与返回的 targetType/targetUID/ok。
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
	// 航程 200 覆盖到 (80,0) 的往返距离 160；短腿机型（20）往返不够，仍被跳过。
	objUnit.PlaneMap[longName] = &objUnit.Plane{
		Name: longName, Type: objUnit.PlaneTypeDiveBomber, Range: 200,
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

	plane, targetType, targetUID, ok := manager.takeOffFromBase(airfield, nil)
	if !ok || targetType != object.TypeShip || targetUID != enemyShip.Uid {
		t.Fatalf("takeoff = %v, %v, %q; want ship target %s", plane, targetType, targetUID, enemyShip.Uid)
	}
	if plane == nil || plane.Name != longName {
		t.Fatalf("plane = %v, want long-range group %s", plane, longName)
	}
}
