package computer

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestOrdersWaitForTheirOwnFrame 验证电脑召唤只在 60 拍的编组轮上计算。
// 场景：场上只有 1 艘敌战列舰和 1 个集结点，决策按 8/30/60 三档错开轮转：
// 8 拍只做沉船标记，30 拍没有到期的编组或航路任务，都不该重新召唤。
// 预期：SimTick=0 与 60 各下 1 条召唤，8、30 拍返回空；失败说明召唤被挂到了
// 错误的轮转档，增援会在不该出现的拍子反复排队。
// 覆盖：makeTemplate("bb", 战列舰, 对舰 20, TimeCost 10) 配集结点 rp(0,0)，
// 依次断言 summonNames 数量为 1、0、0、1。
func TestOrdersWaitForTheirOwnFrame(t *testing.T) {
	useTemplates(t, makeTemplate("bb", objUnit.ShipTypeBattleShip, 20, 0, 0, 0, 10))
	enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeBattleShip, 2, 0)
	enemy.CombatPower.AntiShip = 10
	point := makePoint("rp", 0, 0, 2, "bb")
	handler := NewHandler(faction.ComputerAlpha)
	misState := newMission(0, []*objUnit.BattleShip{enemy}, []*objBuilding.ReinforcePoint{point})

	if len(summonNames(handler.Handle(nil, misState))) != 1 {
		t.Fatal("tick 0 did not summon")
	}
	misState.Core.SimTick = 8
	if got := handler.Handle(nil, misState); len(got) != 0 {
		t.Fatalf("tick 8 issued %v", summonNames(got))
	}
	misState.Core.SimTick = 30
	if got := handler.Handle(nil, misState); len(got) != 0 {
		t.Fatalf("tick 30 issued %v", summonNames(got))
	}
	misState.Core.SimTick = 60
	if len(summonNames(handler.Handle(nil, misState))) != 1 {
		t.Fatal("tick 60 did not summon")
	}
}

// TestPathMaintenanceIsStaggered 验证同组航路按成员序号错开、目标未动就不重下。
// 场景：两艘己方护卫舰 a(10,10)、b(12,10) 编入进攻组，敌驱逐舰在 (50,10)；
// 航路每 30 拍只轮到序号匹配的一艘，避免整队同时挤进只有 2 个工人的寻路线程。
// 预期：0 拍只给 a 下 ShipMove，29 拍只给 b 下 ShipMovePath；30 拍敌舰未动、
// 与上次目的地相距不到 8 格时不刷新；敌舰东移 9 格到 (59,10) 后 90 拍才重下 a 的航路。
// 覆盖：TypeAbbr="WaterDrop" 让 a 走直线指令、b 走异步寻路，逐拍断言指令键集合。
func TestPathMaintenanceIsStaggered(t *testing.T) {
	first := makeShip("a", faction.ComputerAlpha, objUnit.ShipTypeFrigate, 10, 10)
	first.TypeAbbr = "WaterDrop"
	first.CombatPower.AntiShip = 10
	second := makeShip("b", faction.ComputerAlpha, objUnit.ShipTypeFrigate, 12, 10)
	second.CombatPower.AntiShip = 10
	enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 50, 10)
	enemy.CombatPower.AntiShip = 10
	ships := []*objUnit.BattleShip{first, second, enemy}
	handler := NewHandler(faction.ComputerAlpha)
	misState := newMission(0, ships, nil)

	opened := handler.Handle(nil, misState)
	moveA := instr.GenInstrUid(instr.NameShipMove, "a")
	moveB := instr.GenInstrUid(instr.NameShipMove, "b")
	if _, ok := opened[moveA].(*instr.ShipMove); !ok {
		t.Fatalf("tick 0 move A = %T", opened[moveA])
	}
	if _, ok := opened[moveB]; ok {
		t.Fatal("tick 0 also moved B")
	}

	misState.Core.SimTick = 29
	next := handler.Handle(opened, misState)
	if _, ok := next[moveB].(*instr.ShipMovePath); !ok {
		t.Fatalf("tick 29 move B = %T", next[moveB])
	}
	if _, ok := next[moveA]; ok {
		t.Fatal("tick 29 also moved A")
	}

	misState.Core.SimTick = 30
	if got := handler.Handle(opened, misState); len(got) != 0 {
		t.Fatalf("tick 30 refreshed an unmoved target: %v", keys(got))
	}

	enemy.CurPos = objPos.New(enemy.CurPos.MX+9, enemy.CurPos.MY)
	misState.Core.SimTick = 90
	refreshed := handler.Handle(opened, misState)
	if _, ok := refreshed[moveA]; !ok {
		t.Fatal("tick 90 did not refresh after the target moved")
	}
}

// TestSunkTargetIsMarkedWithoutPathing 验证目标沉没时先标重选、当拍不乱下航路。
// 场景：进攻组已锁定 foe，随后 foe 从 Ships 表消失；8 拍是沉船检查拍但不重建编组，
// 30 拍才是下一轮航路与重选拍。
// 预期：8 拍返回空且 attack.needsReselect=true；补入敌舰 next 后 30 拍
// needsReselect 清零、targetUID="next"。失败意味着整组会继续开向已沉目标。
// 覆盖：own(a) 对舰 10、foe 对舰 10、next 对舰 1；直接断言 attack 的两个字段。
func TestSunkTargetIsMarkedWithoutPathing(t *testing.T) {
	own := makeShip("a", faction.ComputerAlpha, objUnit.ShipTypeFrigate, 0, 0)
	own.CombatPower.AntiShip = 10
	enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 40, 0)
	enemy.CombatPower.AntiShip = 10
	other := makeShip("next", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 80, 0)
	other.CombatPower.AntiShip = 1
	handler := NewHandler(faction.ComputerAlpha)
	ships := map[string]*objUnit.BattleShip{own.Uid: own, enemy.Uid: enemy}
	misState := newMission(0, nil, nil)
	misState.Arena.Ships = ships

	handler.Handle(nil, misState)
	delete(ships, enemy.Uid)
	misState.Core.SimTick = 8
	if got := handler.Handle(nil, misState); len(got) != 0 {
		t.Fatalf("tick 8 issued %v", keys(got))
	}
	if handler.attack == nil || !handler.attack.needsReselect {
		t.Fatalf("attack = %+v, want needs reselect", handler.attack)
	}

	ships[other.Uid] = other
	misState.Core.SimTick = 30
	handler.Handle(nil, misState)
	if handler.attack.needsReselect || handler.attack.targetUID != "next" {
		t.Fatalf("reselect = %+v", handler.attack)
	}
}

func keys(instructions map[string]instr.Instruction) []string {
	out := make([]string, 0, len(instructions))
	for uid := range instructions {
		out = append(out, uid)
	}
	return out
}
