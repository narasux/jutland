package state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func battleReportRectsOverlap(a, b BattleReportRect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

func requireInsideRect(t *testing.T, inner, outer BattleReportRect) {
	t.Helper()
	const eps = 1e-6
	require.GreaterOrEqual(t, inner.X, outer.X-eps)
	require.GreaterOrEqual(t, inner.Y, outer.Y-eps)
	require.LessOrEqual(t, inner.X+inner.W, outer.X+outer.W+eps)
	require.LessOrEqual(t, inner.Y+inner.H, outer.Y+outer.H+eps)
}

func requireInsideScreen(t *testing.T, rect BattleReportRect, screenW, screenH int) {
	t.Helper()
	const eps = 1e-6
	require.GreaterOrEqual(t, rect.X, -eps)
	require.GreaterOrEqual(t, rect.Y, -eps)
	require.LessOrEqual(t, rect.X+rect.W, float64(screenW)+eps)
	require.LessOrEqual(t, rect.Y+rect.H, float64(screenH)+eps)
}

func TestBattleReportLayoutFitsCommonScreens(t *testing.T) {
	sizes := []struct{ w, h int }{
		{1920, 1080},
		{1280, 720},
		{1024, 768},
		{800, 600},
	}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("%dx%d", size.w, size.h), func(t *testing.T) {
			report := CalcBattleReportLayout(size.w, size.h, true)
			require.Greater(t, report.Scale, 0.0)
			requireInsideScreen(t, report.Panel, size.w, size.h)

			for idx, card := range report.Cards {
				requireInsideRect(t, card, report.Content)
				if idx > 0 {
					require.False(t, battleReportRectsOverlap(report.Cards[idx-1], card))
				}
			}
			for idx, tab := range report.Tabs {
				requireInsideRect(t, tab, report.Panel)
				require.False(t, battleReportRectsOverlap(tab, report.Title))
				require.False(t, battleReportRectsOverlap(tab, report.Badge))
				if idx > 0 {
					require.False(t, battleReportRectsOverlap(report.Tabs[idx-1], tab))
				}
			}
			for _, rect := range []BattleReportRect{
				report.Title, report.Subtitle, report.Badge, report.Content,
				report.Note, report.Footer, report.BackButton, report.RetryButton,
				report.Sides[0], report.Sides[1],
				report.DetailSides[0], report.DetailSides[1],
			} {
				requireInsideRect(t, rect, report.Panel)
			}
			require.False(t, battleReportRectsOverlap(report.Title, report.Badge))
			require.False(t, battleReportRectsOverlap(report.BackButton, report.RetryButton))
			require.InDelta(t, (float64(size.w)-report.Panel.W)/2, report.Panel.X, 1e-6)
			require.InDelta(t, (float64(size.h)-report.Panel.H)/2, report.Panel.Y, 1e-6)

			for idx, tab := range report.Tabs {
				page, ok := report.PageAt(int(tab.X+tab.W/2), int(tab.Y+tab.H/2))
				require.True(t, ok)
				require.Equal(t, BattleReportPage(idx), page)
			}

			// 纵向顺序：卡片 → 双方汇总 → 页脚 → 按钮。
			cardsBottom := report.Cards[0].Y + report.Cards[0].H
			require.LessOrEqual(t, cardsBottom, report.Sides[0].Y+1e-6)
			require.LessOrEqual(t, report.Sides[0].Y+report.Sides[0].H, report.Note.Y+1e-6)
			require.LessOrEqual(t, report.Note.Y+report.Note.H, report.Footer.Y+1e-6)
			require.LessOrEqual(t, report.Footer.Y+report.Footer.H, report.BackButton.Y+1e-6)
			requireInsideRect(t, report.Sides[0], report.Content)
			requireInsideRect(t, report.Sides[1], report.Content)
			requireInsideRect(t, report.DetailSides[0], report.Content)
			requireInsideRect(t, report.DetailSides[1], report.Content)
			require.False(t, battleReportRectsOverlap(report.Sides[0], report.Sides[1]))
			require.False(t, battleReportRectsOverlap(report.DetailSides[0], report.DetailSides[1]))
		})
	}
}

func TestBattleReportLayoutButtonsOptional(t *testing.T) {
	withButtons := CalcBattleReportLayout(1920, 1080, true)
	require.False(t, withButtons.BackButton.Empty())
	require.False(t, withButtons.RetryButton.Empty())
	require.False(t, withButtons.Buttons.Empty())
	require.True(t, withButtons.Hint.Empty())

	noButtons := CalcBattleReportLayout(1920, 1080, false)
	require.True(t, noButtons.BackButton.Empty())
	require.True(t, noButtons.RetryButton.Empty())
	require.True(t, noButtons.Buttons.Empty())
	require.False(t, noButtons.Hint.Empty())
	requireInsideRect(t, noButtons.Hint, noButtons.Panel)
	require.Equal(t, withButtons.Panel, noButtons.Panel)
}

func TestBattleReportDurationText(t *testing.T) {
	require.Equal(t, "00:00", BattleReportDurationText(0))
	require.Equal(t, "00:59", BattleReportDurationText(59*60))
	require.Equal(t, "01:01", BattleReportDurationText(61*60))
	require.Equal(t, "1:02:03", BattleReportDurationText(int64(3723*60)))
	require.Equal(t, "00:00", BattleReportDurationText(-100))
}

// 结算页按钮居中并排：左侧是低频的重新挑战，右侧是主要操作返回主界面。
func TestBattleReportButtonsCenterRetryLeftAndBackRight(t *testing.T) {
	report := CalcBattleReportLayout(1920, 1080, true)
	require.Equal(t, report.RetryButton.Y, report.BackButton.Y)
	require.Less(t, report.RetryButton.X, report.BackButton.X)
	require.InDelta(
		t,
		battleReportButtonGap*report.Scale,
		report.BackButton.X-(report.RetryButton.X+report.RetryButton.W),
		1e-6,
	)
	pairCenter := (report.RetryButton.X + report.BackButton.X + report.BackButton.W) / 2
	require.InDelta(t, report.Panel.X+report.Panel.W/2, pairCenter, 1e-6)
}
