package manager

import (
	"fmt"
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/object"
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
		CurPos: objPos.NewR(5, 5), FlightPhase: objUnit.PlaneFlightPhaseCruising,
	}
	planes := map[string]*objUnit.Plane{enemy.Uid: enemy}
	for idx := 0; idx < fighters; idx++ {
		uid := fmt.Sprintf("friendly-%02d", idx)
		planes[uid] = &objUnit.Plane{
			Uid: uid, Name: fighterName, Type: objUnit.PlaneTypeFighter,
			CurHP: 20, RemainRange: 100, BelongPlayer: faction.HumanAlpha,
			BelongShip: "carrier", CurPos: objPos.NewR(5, 5),
			FlightPhase: objUnit.PlaneFlightPhaseCruising,
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

// 同一架敌机最多被 maxAirAttackersPerTarget 架飞机追击，多出来的战斗机转去巡逻。
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

// 已经扑在同一架敌机上的超编飞机要拆开，否则旧存档里 20+ 架围殴一架侦察机的
// 局面会一直保持下去（已有攻击指令的飞机不会主动放弃目标）。
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

// 巡逻中的战斗机等到新目标后要脱离巡逻、转入攻击。
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
