package computer

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func TestComputerIssuesMoveOnlyEveryEightTicks(t *testing.T) {
	handler := NewHandler(faction.ComputerAlpha)
	misState := &state.MissionState{
		Arena: state.MissionArenaState{Ships: map[string]*objUnit.BattleShip{
			"own": {
				Uid: "own", BelongPlayer: faction.ComputerAlpha, TypeAbbr: "WaterDrop",
				CurPos: objPos.New(10, 10),
			},
			"foe": {
				Uid: "foe", BelongPlayer: faction.HumanAlpha,
				CurPos: objPos.New(30, 30),
			},
		}},
	}
	moveUID := instr.GenInstrUid(instr.NameShipMove, "own")

	misState.Core.SimTick = 1
	first := handler.Handle(nil, misState)
	if _, ok := first[moveUID]; !ok {
		t.Fatalf("tick 1 orders = %v, want move %s", keys(first), moveUID)
	}
	for tick := int64(2); tick <= 8; tick++ {
		misState.Core.SimTick = tick
		if got := handler.Handle(first, misState); len(got) != 0 {
			t.Fatalf("tick %d produced %v", tick, keys(got))
		}
	}
	misState.Core.SimTick = 9
	if _, ok := handler.Handle(nil, misState)[moveUID]; !ok {
		t.Fatal("tick 9 did not issue the next move")
	}
}

func keys(instructions map[string]instr.Instruction) []string {
	out := make([]string, 0, len(instructions))
	for uid := range instructions {
		out = append(out, uid)
	}
	return out
}
