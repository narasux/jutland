package game

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/state"
)

func TestMissionResultClickActionMapsButtons(t *testing.T) {
	report := state.CalcBattleReportLayout(1920, 1080, true)
	center := func(rect state.BattleReportRect) (int, int) {
		return int(rect.X + rect.W/2), int(rect.Y + rect.H/2)
	}

	backX, backY := center(report.BackButton)
	require.Equal(t, missionResultActionBack, missionResultClickAction(report, backX, backY))

	retryX, retryY := center(report.RetryButton)
	require.Equal(t, missionResultActionRetry, missionResultClickAction(report, retryX, retryY))

	require.Equal(
		t,
		missionResultActionNone,
		missionResultClickAction(report, int(report.Panel.X)+4, int(report.Panel.Y)+4),
	)
}

func TestEnterMissionSplashStartsReportFlow(t *testing.T) {
	g := &Game{resultSplashTicks: 999}
	g.enterMissionSplash(GameModeMissionSuccess)
	require.Equal(t, GameMode(GameModeMissionSuccess), g.mode)
	require.Equal(t, GameMode(GameModeMissionSuccess), g.resultMode)
	require.Zero(t, g.resultSplashTicks)

	g.resultSplashTicks = 7
	g.enterMissionSplash(GameModeMissionFailed)
	require.Equal(t, GameMode(GameModeMissionFailed), g.mode)
	require.Equal(t, GameMode(GameModeMissionFailed), g.resultMode)
	require.Zero(t, g.resultSplashTicks)
}
