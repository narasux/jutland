package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
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

// TestBattleReportPageAccessors 验证战报页签只通过 SetBattleReportPage 切换且拒绝非法页。
// 场景：零值 MissionState 上依次设置舰艇、飞机、战况、总览，最后 SetBattleReportPage(9)。
// 预期：每次设置后 BattleReportPage 返回对应页；非法页 9 被忽略，页面保持总览。
// 覆盖：调用 SetBattleReportPage/BattleReportPage，断言四页赋值顺序与 Valid 过滤。
func TestBattleReportPageAccessors(t *testing.T) {
	manager := &MissionManager{state: &state.MissionState{}}

	manager.SetBattleReportPage(state.BattleReportPageShips)
	require.Equal(t, state.BattleReportPageShips, manager.BattleReportPage())

	manager.SetBattleReportPage(state.BattleReportPagePlanes)
	require.Equal(t, state.BattleReportPagePlanes, manager.BattleReportPage())

	manager.SetBattleReportPage(state.BattleReportPageEvents)
	require.Equal(t, state.BattleReportPageEvents, manager.BattleReportPage())

	manager.SetBattleReportPage(state.BattleReportPageOverview)
	require.Equal(t, state.BattleReportPageOverview, manager.BattleReportPage())

	manager.SetBattleReportPage(state.BattleReportPage(9))
	require.Equal(t, state.BattleReportPageOverview, manager.BattleReportPage())
}

// TestHandleBattleReportTrendModeClick 验证趋势页模式按钮只在趋势页生效，并写入对应模式。
// 场景：趋势页提供「当前兵力 / 建造与损失」两个按钮；其它页签也有相同坐标区域，
// 点击不能被误消费，否则会把页签点击吞掉。
// 预期：趋势页点击累计模式按钮返回 true 且模式变为 Cumulative；点击空白返回 false；
// 切到总览页后点击同一位置返回 false，模式保持 Cumulative。
// 覆盖：1920×1080 布局下用 TrendModes[1] 中心与面板空白坐标调用，
// 断言返回值与 UI.BattleReportTrendMode。
func TestHandleBattleReportTrendModeClick(t *testing.T) {
	manager := &MissionManager{state: &state.MissionState{
		View: state.MissionViewState{
			Layout: layout.ScreenLayout{Width: 1920, Height: 1080},
		},
		UI: state.MissionUIState{BattleReportPage: state.BattleReportPageTrends},
	}}
	report := manager.BattleReportLayout(true)
	cumulative := report.TrendModes[int(state.BattleReportTrendModeCumulative)]

	require.True(t, manager.HandleBattleReportTrendModeClick(
		int(cumulative.X+cumulative.W/2), int(cumulative.Y+cumulative.H/2),
	))
	require.Equal(t, state.BattleReportTrendModeCumulative, manager.state.UI.BattleReportTrendMode)

	require.False(t, manager.HandleBattleReportTrendModeClick(
		int(report.Panel.X+report.Panel.W/2), int(report.Footer.Y),
	))

	manager.state.UI.BattleReportPage = state.BattleReportPageOverview
	require.False(t, manager.HandleBattleReportTrendModeClick(
		int(cumulative.X+cumulative.W/2), int(cumulative.Y+cumulative.H/2),
	))
	require.Equal(t, state.BattleReportTrendModeCumulative, manager.state.UI.BattleReportTrendMode)
}
