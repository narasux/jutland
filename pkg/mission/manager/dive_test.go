package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/unitpanel"
)

// TestDiveToggleInstructionFollowsDiveState 验证下潜切换指令按当前深度状态选择下潜或上浮。
// 场景：同一潜艇分别处于水面、水下、下潜中、上浮中，另加一艘驱逐舰和不存在的 UID。
// 预期：水面给 *ShipDive，水下给 *ShipSurface；两种过渡态按最后指令反向；非潜艇与缺失 UID 为 nil。
// 覆盖：diveToggleInstruction(ms, uid) 对 5 个 UID 分别断言指令类型或 nil，覆盖稳定态、过渡态反向与非法输入。
func TestDiveToggleInstructionFollowsDiveState(t *testing.T) {
	surface := &objUnit.BattleShip{Uid: "surface", CurHP: 1, Type: objUnit.ShipTypeSubmarine}
	submerged := &objUnit.BattleShip{
		Uid: "submerged", CurHP: 1, Type: objUnit.ShipTypeSubmarine,
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSubmerged},
	}
	diving := &objUnit.BattleShip{
		Uid: "diving", CurHP: 1, Type: objUnit.ShipTypeSubmarine,
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateDiving},
	}
	surfacing := &objUnit.BattleShip{
		Uid: "surfacing", CurHP: 1, Type: objUnit.ShipTypeSubmarine,
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSurfacing},
	}
	gunship := &objUnit.BattleShip{Uid: "gunship", CurHP: 1, Type: objUnit.ShipTypeDestroyer}
	ms := &state.MissionState{Arena: state.MissionArenaState{
		Ships: map[string]*objUnit.BattleShip{
			surface.Uid:   surface,
			submerged.Uid: submerged,
			diving.Uid:    diving,
			surfacing.Uid: surfacing,
			gunship.Uid:   gunship,
		},
	}}

	require.IsType(t, &instr.ShipDive{}, diveToggleInstruction(ms, surface.Uid))
	require.IsType(t, &instr.ShipSurface{}, diveToggleInstruction(ms, submerged.Uid))
	// 过渡中按最后一次指令反向：下潜中按下即为上浮，上浮中按下即为下潜。
	require.IsType(t, &instr.ShipSurface{}, diveToggleInstruction(ms, diving.Uid))
	require.IsType(t, &instr.ShipDive{}, diveToggleInstruction(ms, surfacing.Uid))
	require.Nil(t, diveToggleInstruction(ms, gunship.Uid))
	require.Nil(t, diveToggleInstruction(ms, "missing"))
}

// TestToggleDiveActionQueuesAndExecutesInstruction 验证下潜按钮入队指令并在执行后进入下潜。
// 场景：水面潜艇 sub 的 UID 提交 unitpanel.ActionToggleDive，MissionManager 带空 instructionSet。
// 预期：处理后能取到 NameShipDive/sub 指令；ExecAll 后 DiveState 变为 DiveStateDiving。
// 覆盖：构造仅含 sub 的 MissionState，断言指令存在与执行后的深度状态，串联 UI 动作到指令执行的链路。
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
	require.Equal(t, objUnit.DiveStateDiving, sub.Submarine.DiveState)
}
