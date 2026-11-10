package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/unitpanel"
)

func TestDiveToggleInstructionFollowsDiveState(t *testing.T) {
	surface := &objUnit.BattleShip{Uid: "surface", CurHP: 1, Type: objUnit.ShipTypeSubmarine}
	submerged := &objUnit.BattleShip{
		Uid: "submerged", CurHP: 1, Type: objUnit.ShipTypeSubmarine, DiveState: objUnit.DiveStateSubmerged,
	}
	diving := &objUnit.BattleShip{
		Uid: "diving", CurHP: 1, Type: objUnit.ShipTypeSubmarine, DiveState: objUnit.DiveStateDiving,
	}
	gunship := &objUnit.BattleShip{Uid: "gunship", CurHP: 1, Type: objUnit.ShipTypeDestroyer}
	ms := &state.MissionState{Arena: state.MissionArenaState{
		Ships: map[string]*objUnit.BattleShip{
			surface.Uid:   surface,
			submerged.Uid: submerged,
			diving.Uid:    diving,
			gunship.Uid:   gunship,
		},
	}}

	require.IsType(t, &instr.ShipDive{}, diveToggleInstruction(ms, surface.Uid))
	require.IsType(t, &instr.ShipSurface{}, diveToggleInstruction(ms, submerged.Uid))
	require.Nil(t, diveToggleInstruction(ms, diving.Uid))
	require.Nil(t, diveToggleInstruction(ms, gunship.Uid))
	require.Nil(t, diveToggleInstruction(ms, "missing"))
}

func TestToggleDiveActionQueuesAndExecutesInstruction(t *testing.T) {
	sub := &objUnit.BattleShip{Uid: "sub", CurHP: 100, Type: objUnit.ShipTypeSubmarine}
	missionState := &state.MissionState{Arena: state.MissionArenaState{
		Ships: map[string]*objUnit.BattleShip{sub.Uid: sub},
	}}
	missionManager := &MissionManager{state: missionState, instructionSet: NewInstructionSet()}

	missionManager.handleUnitPanelActions([]unitpanel.Action{{
		Kind: unitpanel.ActionToggleDive, ShipUids: []string{sub.Uid},
	}})
	require.NotNil(t, missionManager.instructionSet.Get(instr.GenInstrUid(instr.NameShipDive, sub.Uid)))

	missionManager.instructionSet.ExecAll(missionState)
	require.Equal(t, objUnit.DiveStateDiving, sub.DiveState)
}
