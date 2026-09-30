package computer

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

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
