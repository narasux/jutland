package instruction

import (
	"testing"

	"github.com/stretchr/testify/require"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestNewDiveToggleChoosesDirectionFromSubmarineState 验证深浅切换指令按当前稳定态/过渡态选择方向。
// 场景：玩家反复点“下潜/上浮”按钮；水面/上浮中应下潜，水下/下潜中应上浮，过渡中允许反向。
// 预期：Surface/Surfacing → *ShipDive；Submerged/Diving → *ShipSurface；Uid 与目标状态对应。
// 覆盖：表驱动四种 DiveState，构造潜艇并调用 NewDiveToggle，断言指令类型与 Uid。
func TestNewDiveToggleChoosesDirectionFromSubmarineState(t *testing.T) {
	cases := []struct {
		name      string
		state     objUnit.DiveState
		wantDive  bool
		wantInstr string
	}{
		{name: "surface dives", state: objUnit.DiveStateSurface, wantDive: true},
		{name: "diving surfaces", state: objUnit.DiveStateDiving, wantDive: false},
		{name: "submerged surfaces", state: objUnit.DiveStateSubmerged, wantDive: false},
		{name: "surfacing dives", state: objUnit.DiveStateSurfacing, wantDive: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sub := &objUnit.BattleShip{
				Uid: "sub", Type: objUnit.ShipTypeSubmarine, CurHP: 100, DiveState: tc.state,
			}
			instr := NewDiveToggle(sub)
			require.NotNil(t, instr)
			if tc.wantDive {
				require.IsType(t, &ShipDive{}, instr)
				require.Equal(t, GenInstrUid(NameShipDive, "sub"), instr.Uid())
			} else {
				require.IsType(t, &ShipSurface{}, instr)
				require.Equal(t, GenInstrUid(NameShipSurface, "sub"), instr.Uid())
			}
		})
	}
}

// TestNewDiveToggleRejectsUnavailableShips 验证非潜艇、空指针和已沉没潜艇不会生成深浅指令。
// 场景：AI/单位面板可能对驱逐舰或已击沉的潜艇调用统一切换函数；必须返回 nil 而不是下发无效指令。
// 预期：nil、非潜艇、CurHP<=0 的潜艇都返回 nil。
// 覆盖：分别传入 nil、ShipTypeDestroyer、CurHP=0 的潜艇，断言 NewDiveToggle 为 nil。
func TestNewDiveToggleRejectsUnavailableShips(t *testing.T) {
	require.Nil(t, NewDiveToggle(nil))
	require.Nil(t, NewDiveToggle(&objUnit.BattleShip{
		Uid: "dd", Type: objUnit.ShipTypeDestroyer, CurHP: 100,
	}))
	require.Nil(t, NewDiveToggle(&objUnit.BattleShip{
		Uid: "dead", Type: objUnit.ShipTypeSubmarine, CurHP: 0,
	}))
}

// TestShipDiveInstructionExecSetsSubmarineState 验证下潜指令执行后潜艇进入 Diving 并标记已执行。
// 场景：玩家点击下潜/快捷键下发 ShipDive；任务主循环执行指令后，潜艇应开始过渡动画。
// 预期：Exec 返回 nil、Executed=true、DiveState=Diving；String/Uid 使用舰船 UID。
// 覆盖：构造水面潜艇与 MissionState.Arena.Ships，调用 NewShipDive("sub").Exec 后逐项断言。
func TestShipDiveInstructionExecSetsSubmarineState(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", Type: objUnit.ShipTypeSubmarine, CurHP: 100,
	}
	ms := &state.MissionState{Arena: state.MissionArenaState{
		Ships: map[string]*objUnit.BattleShip{sub.Uid: sub},
	}}

	instr := NewShipDive(sub.Uid)
	require.False(t, instr.Executed())
	require.NoError(t, instr.Exec(ms))
	require.True(t, instr.Executed())
	require.Equal(t, objUnit.DiveStateDiving, sub.DiveState)
	require.Equal(t, "Dive ship sub", instr.String())
	require.Equal(t, GenInstrUid(NameShipDive, sub.Uid), instr.Uid())
}

// TestShipSurfaceInstructionExecSetsSubmarineState 验证上浮指令执行后潜艇进入 Surfacing。
// 场景：水下潜艇需要上浮换气/规避；ShipSurface 执行后应开始上浮过渡，而不是停留在 Submerged。
// 预期：Exec 返回 nil、Executed=true、DiveState=Surfacing。
// 覆盖：构造 DiveStateSubmerged 的潜艇，调用 NewShipSurface("sub").Exec 后断言状态与 UID。
func TestShipSurfaceInstructionExecSetsSubmarineState(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", Type: objUnit.ShipTypeSubmarine, CurHP: 100,
		DiveState: objUnit.DiveStateSubmerged,
	}
	ms := &state.MissionState{Arena: state.MissionArenaState{
		Ships: map[string]*objUnit.BattleShip{sub.Uid: sub},
	}}

	instr := NewShipSurface(sub.Uid)
	require.False(t, instr.Executed())
	require.NoError(t, instr.Exec(ms))
	require.True(t, instr.Executed())
	require.Equal(t, objUnit.DiveStateSurfacing, sub.DiveState)
	require.Equal(t, "Surface ship sub", instr.String())
	require.Equal(t, GenInstrUid(NameShipSurface, sub.Uid), instr.Uid())
}

// TestDiveInstructionMissingShipStillExecutes 验证目标舰船已不存在时指令仍标记完成，不阻塞队列。
// 场景：潜艇在指令执行前被击沉，Arena.Ships 已移除；指令若返回 error/未完成，指令队列会卡住。
// 预期：Exec 返回 nil、Executed=true，不 panic。
// 覆盖：空 MissionState 上执行 NewShipDive("missing")，断言无错误且已完成。
func TestDiveInstructionMissingShipStillExecutes(t *testing.T) {
	ms := &state.MissionState{}
	instr := NewShipDive("missing")
	require.NoError(t, instr.Exec(ms))
	require.True(t, instr.Executed())
	require.Equal(t, GenInstrUid(NameShipDive, "missing"), instr.Uid())
}
