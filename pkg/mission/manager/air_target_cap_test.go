package manager

import (
	"fmt"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/targeting"
)

// registerCapTestFighter 注册一张临时战斗机模板，测试结束自动清掉。
func registerCapTestFighter(t *testing.T, name string) {
	t.Helper()
	old, had := objUnit.PlaneMap[name]
	objUnit.PlaneMap[name] = &objUnit.Plane{
		Name: name, Type: objUnit.PlaneTypeFighter,
		Weapon: objUnit.PlaneWeapon{Guns: []*objUnit.Gun{{}}},
	}
	t.Cleanup(func() {
		if had {
			objUnit.PlaneMap[name] = old
			return
		}
		delete(objUnit.PlaneMap, name)
	})
}

// newAirCapTestManager 造一个「一架可见敌机 + n 架己方战斗机」的最小调度环境。
// 基地队列只放这一架敌机，模拟迷雾下只有一架侦察机可见的局面。
func newAirCapTestManager(t *testing.T, fighters int) (*MissionManager, string) {
	t.Helper()
	const fighterName = "cap-test-fighter"
	registerCapTestFighter(t, fighterName)

	enemy := &objUnit.Plane{
		Uid: "enemy-plane", Name: fighterName, Type: objUnit.PlaneTypeFighter,
		CurHP: 20, RemainRange: 100, BelongPlayer: faction.ComputerAlpha,
		CurPos: objPos.NewR(5, 5), Flight: objUnit.FlightState{Phase: objUnit.PlaneFlightPhaseCruising},
	}
	planes := map[string]*objUnit.Plane{enemy.Uid: enemy}
	for idx := 0; idx < fighters; idx++ {
		uid := fmt.Sprintf("friendly-%02d", idx)
		planes[uid] = &objUnit.Plane{
			Uid: uid, Name: fighterName, Type: objUnit.PlaneTypeFighter,
			CurHP: 20, RemainRange: 100, BelongPlayer: faction.HumanAlpha,
			BelongShip: "carrier", CurPos: objPos.NewR(5, 5),
			Flight: objUnit.FlightState{Phase: objUnit.PlaneFlightPhaseCruising},
		}
	}
	manager := &MissionManager{
		state:          &state.MissionState{Arena: state.MissionArenaState{Planes: planes}},
		instructionSet: NewInstructionSet(),
		targetingPlan: targeting.Plan{
			BaseQueues: map[string]map[object.Type][]targeting.TargetRef{
				"carrier": {
					object.TypePlane: {{UID: enemy.Uid, TargetType: object.TypePlane}},
				},
			},
		},
		targetingCursors: map[string]map[object.Type]int{},
	}
	return manager, enemy.Uid
}

// countAirAttackers 统计己方飞机里拿到攻击指令的架数与转入巡逻的架数。
// 指令集是调度层的唯一事实来源：CurAttackTarget 要等 PlaneAttack 执行那一拍才同步。
func countAirAttackers(t *testing.T, m *MissionManager) (attackers, patrolling int) {
	t.Helper()
	for uid, plane := range m.state.Arena.Planes {
		if plane.BelongPlayer != faction.HumanAlpha {
			continue
		}
		if m.instructionSet.Exists(instr.GenInstrUid(instr.NamePlaneAttack, uid)) {
			attackers++
			continue
		}
		if m.instructionSet.Exists(instr.GenInstrUid(instr.NamePlanePatrol, uid)) {
			patrolling++
		}
	}
	return attackers, patrolling
}

// TestAirTargetCapSendsSurplusFightersOnPatrol 验证同一架敌机最多被 maxAirAttackersPerTarget=4 架追击，多出的战斗机转巡逻。
// 场景：基地队列只有 1 架可见敌机，7 架（4+3）己方战斗机同处 (5,5) 巡航，跑一帧 updatePlaneAttackOrReturn。
// 预期：4 架拿到 PlaneAttack 指令，3 架拿到 PlanePatrol 指令。
// 覆盖：nextTargetUIDForPlane 的 airTargetHasSlotFor 名额检查，以及无目标战斗机挂 PlanePatrol 的分支。
func TestAirTargetCapSendsSurplusFightersOnPatrol(t *testing.T) {
	const surplus = 3
	manager, _ := newAirCapTestManager(t, maxAirAttackersPerTarget+surplus)

	manager.updatePlaneAttackOrReturn()

	attackers, patrolling := countAirAttackers(t, manager)
	if attackers != maxAirAttackersPerTarget {
		t.Fatalf("attackers = %d, want %d", attackers, maxAirAttackersPerTarget)
	}
	if patrolling != surplus {
		t.Fatalf("patrolling = %d, want %d", patrolling, surplus)
	}
}

// TestUpdatePlaneAttackOrReturnReleasesSurplusAttackers 验证已经围在同一架敌机上的超编飞机要拆开重新分配。
// 场景：5 架战斗机全部 CurAttackTarget=enemy 且已挂 PlaneAttack 指令（模拟旧存档 20+ 架围殴一架侦察机的围攻，此处用 5 架触发上限），跑一帧 updatePlaneAttackOrReturn。
// 预期：attackers 回落到 maxAirAttackersPerTarget=4，patrolling=1（多出那架转巡逻）。
// 覆盖：releaseSurplusAirAttackers 按 airAttackerCounts 回收名额，Remove 指令并清空 CurAttackTarget 后重新分配。
func TestUpdatePlaneAttackOrReturnReleasesSurplusAttackers(t *testing.T) {
	const fighters = 5
	manager, enemyUid := newAirCapTestManager(t, fighters)

	// 先让全部战斗机都锁定这架敌机，模拟围攻已经形成。
	for uid, plane := range manager.state.Arena.Planes {
		if plane.BelongPlayer != faction.HumanAlpha {
			continue
		}
		plane.CurAttackTarget = enemyUid
		manager.instructionSet.Add(instr.NewPlaneAttack(uid, object.TypePlane, enemyUid))
	}

	manager.updatePlaneAttackOrReturn()

	attackers, patrolling := countAirAttackers(t, manager)
	if attackers != maxAirAttackersPerTarget {
		t.Fatalf("attackers after rebalance = %d, want %d", attackers, maxAirAttackersPerTarget)
	}
	if patrolling != fighters-maxAirAttackersPerTarget {
		t.Fatalf("patrolling after rebalance = %d, want %d", patrolling, fighters-maxAirAttackersPerTarget)
	}
}

// TestPatrollingFighterLeavesPatrolForNewTarget 验证巡逻中的战斗机发现新目标后脱离巡逻转入攻击。
// 场景：1 架战斗机在 (5,5)，先挂 PlanePatrol；基地队列有 1 架敌机，跑一帧 updatePlaneAttackOrReturn。
// 预期：存在 PlaneAttack 指令，且 PlanePatrol 指令已被移除。
// 覆盖：updatePlaneAttackOrReturn 的巡逻分支——拿到目标后 Remove(patrolUid) 并 Add(PlaneAttack)。
func TestPatrollingFighterLeavesPatrolForNewTarget(t *testing.T) {
	manager, _ := newAirCapTestManager(t, 1)
	lone := manager.state.Arena.Planes["friendly-00"]
	manager.instructionSet.Add(instr.NewPlanePatrol(lone.Uid))

	manager.updatePlaneAttackOrReturn()

	if !manager.instructionSet.Exists(instr.GenInstrUid(instr.NamePlaneAttack, lone.Uid)) {
		t.Fatal("巡逻中的战斗机发现目标后应该转入攻击")
	}
	if manager.instructionSet.Exists(instr.GenInstrUid(instr.NamePlanePatrol, lone.Uid)) {
		t.Fatal("战斗机拿到目标后不该继续挂着巡逻指令")
	}
}

// TestSurplusAirAttackerReleaseKeepsTakingOffFightersMoving 验证名额回收不会拆掉滑跑中飞机的攻击指令，否则它们永远停在跑道/舰尾。
// 场景：机场与航母各放飞 maxAirAttackersPerTarget+2=6 架仍处 PlaneFlightPhaseTakingOff 的战斗机，全部认领同一架敌机；先调 releaseSurplusAirAttackers，再执行 900 帧 instructionSet.ExecAll + updatePlaneAttackOrReturn。
// 预期：回收后每架滑跑机仍保留 PlaneAttack 指令；900 帧后没有飞机仍处于 TakingOff（全部离地转巡航）。
// 覆盖：releaseSurplusAirAttackers 跳过 !IsCruising 的飞机，以及攻击指令的 UpdateTakeoff 分支持续推滑跑（机场停跑道、航母卡舰尾的回归）。
func TestSurplusAirAttackerReleaseKeepsTakingOffFightersMoving(t *testing.T) {
	const fighterName = "cap-test-takeoff-fighter"
	registerCapTestFighter(t, fighterName)
	// 模板默认没有速度/生命，这里补上真实战斗机的数值，才能观察滑跑是否推进。
	objUnit.PlaneMap[fighterName].TotalHP = 20
	objUnit.PlaneMap[fighterName].CurHP = 20
	objUnit.PlaneMap[fighterName].MaxSpeed = 0.05
	objUnit.PlaneMap[fighterName].Acceleration = 0.01
	objUnit.PlaneMap[fighterName].RotateSpeed = 6
	objUnit.PlaneMap[fighterName].RemainRange = 100

	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	testCases := []struct {
		name  string
		build func() (objUnit.AircraftBase, *state.MissionState)
	}{
		{
			name: "airfield",
			build: func() (objUnit.AircraftBase, *state.MissionState) {
				// 机场起飞点为双点、冷却为 0，便于同一拍连续放飞。
				airfield := objBuilding.NewAirfield(
					objPos.NewR(50, 50), 0, 6, 0.8, faction.HumanAlpha, 0,
					[]objUnit.PlaneGroup{{Name: fighterName, MaxCount: 32}},
				)
				for range 32 {
					airfield.StockPlane(fighterName)
				}
				return airfield, &state.MissionState{Arena: state.MissionArenaState{
					Ships:     map[string]*objUnit.BattleShip{},
					Airfields: map[string]*objBuilding.Airfield{airfield.Uid: airfield},
				}}
			},
		},
		{
			name: "carrier",
			build: func() (objUnit.AircraftBase, *state.MissionState) {
				deck := &objUnit.CarrierDeck{
					Name:          "capTestDeck",
					TakeoffPoints: []objUnit.TakeoffPoint{{Forward: 0.4, Lateral: 0, RunLength: 0.6}},
					Landing: objUnit.LandingConfig{
						Mode: objUnit.LandingModeDeck, Forward: 0.5, ApproachLength: 3,
					},
				}
				deck.Validate()
				carrier := &objUnit.BattleShip{
					Uid: "cap-test-carrier", CurHP: 1000, CurPos: objPos.NewR(50, 50),
					Length: 250, Width: 30, BelongPlayer: faction.HumanAlpha,
					Aircraft: objUnit.ShipAircraft{
						Groups: []objUnit.PlaneGroup{
							{
								Name: fighterName, MaxCount: 32, CurCount: 32,
								TargetType: object.TypePlane,
							},
						},
						HasPlane: true,
					},
				}
				carrier.Aircraft.ResolveDeck(deck)
				return carrier, &state.MissionState{Arena: state.MissionArenaState{
					Ships:     map[string]*objUnit.BattleShip{carrier.Uid: carrier},
					Airfields: map[string]*objBuilding.Airfield{},
				}}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			base, missionState := testCase.build()
			enemy := &objUnit.Plane{
				Uid: "cap-enemy", Name: fighterName, Type: objUnit.PlaneTypeFighter,
				CurHP: 20, RemainRange: 100, BelongPlayer: faction.ComputerAlpha,
				CurPos: objPos.NewR(60, 60), Flight: objUnit.FlightState{Phase: objUnit.PlaneFlightPhaseCruising},
			}
			missionState.Arena.Planes = map[string]*objUnit.Plane{enemy.Uid: enemy}
			manager := &MissionManager{
				state:          missionState,
				instructionSet: NewInstructionSet(),
			}

			takingOff := []*objUnit.Plane{}
			for idx := 0; idx < maxAirAttackersPerTarget+2; idx++ {
				plane := base.BaseAircraft().TakeOff(base, object.TypePlane)
				if plane == nil {
					t.Fatalf("第 %d 架未能起飞（起飞点/库存不足）", idx)
				}
				if plane.Flight.Phase != objUnit.PlaneFlightPhaseTakingOff {
					t.Fatalf("起飞的飞机应处于滑跑阶段, got %s", plane.Flight.Phase)
				}
				manager.state.Arena.PutPlane(plane)
				// 起飞时认领同一架敌机，触发名额上限
				plane.CurAttackTarget = enemy.Uid
				manager.instructionSet.Add(instr.NewPlaneAttack(plane.Uid, object.TypePlane, enemy.Uid))
				takingOff = append(takingOff, plane)
			}

			manager.releaseSurplusAirAttackers(manager.airAttackerCounts())

			for _, plane := range takingOff {
				if !manager.instructionSet.Exists(instr.GenInstrUid(instr.NamePlaneAttack, plane.Uid)) {
					t.Fatalf("滑跑中的飞机 %s 被拆掉了指令，将永远停在原地不动", plane.Uid)
				}
			}

			// 持续调度若干帧：所有飞机都必须离地（进入巡航），而不是留在跑道上。
			for range 900 {
				manager.instructionSet.ExecAll(manager.state)
				manager.updatePlaneAttackOrReturn()
			}
			for _, plane := range takingOff {
				if plane.Flight.Phase == objUnit.PlaneFlightPhaseTakingOff {
					t.Fatalf("飞机 %s 900 帧后仍在滑跑, speed=%.5f", plane.Uid, plane.CurSpeed)
				}
			}
		})
	}
}
