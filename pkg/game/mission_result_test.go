package game

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/state"
)

// TestMissionResultClickActionMapsButtons 验证结算页点击坐标只映射到返回按钮。
// 场景：1920×1080 战报布局下点返回按钮中心，以及面板左上角内 4px 的空白处。
// 预期：依次得到 missionResultActionBack、None；失败意味着按钮错位或空白区误触。
// 覆盖：CalcBattleReportLayout(1920,1080,true) 后取矩形中心，断言两种点击结果。
func TestMissionResultClickActionMapsButtons(t *testing.T) {
	report := state.CalcBattleReportLayout(1920, 1080, true)
	center := func(rect state.BattleReportRect) (int, int) {
		return int(rect.X + rect.W/2), int(rect.Y + rect.H/2)
	}

	backX, backY := center(report.BackButton)
	require.Equal(t, missionResultActionBack, missionResultClickAction(report, backX, backY))

	require.Equal(
		t,
		missionResultActionNone,
		missionResultClickAction(report, int(report.Panel.X)+4, int(report.Panel.Y)+4),
	)
}

// TestEnterMissionSplashStartsReportFlow 验证进入胜负全屏图时同步模式并清零帧计数。
// 场景：上一局已累计 999 帧，或中途 7 帧时再次进入失败页，都必须重新开始最短展示计时。
// 预期：mode/resultMode 分别等于成功或失败，resultSplashTicks 均为 0；失败意味着沿用旧计时或战报背景。
// 覆盖：先以 999 帧、再置 7 帧调用 enterMissionSplash，分别断言三个字段。
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
