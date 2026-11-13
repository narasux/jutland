package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/state"
)

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
