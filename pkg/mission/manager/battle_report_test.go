package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/state"
)

// TestFinalizeBattleStatsMapsMissionStatusToEndReason 验证任务状态到战斗结束原因的映射与终局收尾。
// 场景：MissionSuccess、普通失败、确认放弃的失败，以及 Running/Paused 共 5 种状态。
// 预期：胜/败/放弃映射 Victory/Defeat/Aborted，写 EndTick=10 并切总览；运行/暂停不收尾且留在舰艇页。
// 覆盖：表驱动设置状态/确认放弃/SimTick=10，断言 Finalized、EndReason、EndTick 与页面。
func TestFinalizeBattleStatsMapsMissionStatusToEndReason(t *testing.T) {
	tests := []struct {
		name          string
		status        state.MissionStatus
		confirmQuit   bool
		wantReason    state.BattleEndReason
		wantFinalized bool
	}{
		{
			name:          "victory",
			status:        state.MissionSuccess,
			wantReason:    state.BattleEndVictory,
			wantFinalized: true,
		},
		{
			name:          "defeat",
			status:        state.MissionFailed,
			wantReason:    state.BattleEndDefeat,
			wantFinalized: true,
		},
		{
			name:          "aborted",
			status:        state.MissionFailed,
			confirmQuit:   true,
			wantReason:    state.BattleEndAborted,
			wantFinalized: true,
		},
		{
			name:   "running is not finalized",
			status: state.MissionRunning,
		},
		{
			name:   "paused is not finalized",
			status: state.MissionPaused,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &MissionManager{state: &state.MissionState{
				Core: state.MissionCoreState{
					MissionStatus:      tt.status,
					ConfirmQuitMission: tt.confirmQuit,
					SimTick:            10,
				},
				UI: state.MissionUIState{BattleReportPage: state.BattleReportPageShips},
			}}
			manager.state.Stats = state.NewBattleStats()

			manager.finalizeBattleStats()

			require.Equal(t, tt.wantFinalized, manager.state.Stats.Finalized)
			if tt.wantFinalized {
				require.Equal(t, tt.wantReason, manager.state.Stats.EndReason)
				require.Equal(t, int64(10), manager.state.Stats.EndTick)
				require.Equal(t, state.BattleReportPageOverview, manager.state.UI.BattleReportPage)
			} else {
				require.Equal(t, state.BattleReportPageShips, manager.state.UI.BattleReportPage)
			}
		})
	}
}

// TestBattleReportPageAccessors 验证战报页签按总览→舰艇→飞机→战况循环切换且拒绝非法页。
// 场景：从舰艇页连续 Shift(1) 三次，再 Shift(-1) 回退，最后 SetBattleReportPage(9)。
// 预期：依次到飞机、战况、总览，再回战况；非法页 9 被忽略，页面保持战况。
// 覆盖：零值 MissionState 上调用 Set/Shift/BattleReportPage，断言四页顺序、回绕与 Valid 过滤。
func TestBattleReportPageAccessors(t *testing.T) {
	manager := &MissionManager{state: &state.MissionState{}}

	manager.SetBattleReportPage(state.BattleReportPageShips)
	require.Equal(t, state.BattleReportPageShips, manager.BattleReportPage())

	manager.ShiftBattleReportPage(1)
	require.Equal(t, state.BattleReportPagePlanes, manager.BattleReportPage())

	manager.ShiftBattleReportPage(1)
	require.Equal(t, state.BattleReportPageEvents, manager.BattleReportPage())

	manager.ShiftBattleReportPage(1)
	require.Equal(t, state.BattleReportPageOverview, manager.BattleReportPage())

	manager.ShiftBattleReportPage(-1)
	require.Equal(t, state.BattleReportPageEvents, manager.BattleReportPage())

	manager.SetBattleReportPage(state.BattleReportPage(9))
	require.Equal(t, state.BattleReportPageEvents, manager.BattleReportPage())
}
