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

// TestBattleReportLayoutFitsCommonScreens 验证四种常见分辨率下战报面板完整落在屏幕内且各区域不重叠。
// 场景：结算页按 1280×760 设计尺寸缩放，需在 1920×1080、1280×720、1024×768、800×600 都不越界。
// 预期：Scale>0、Panel 居中；卡片/页签/趋势图互不重叠且 PageAt 命中自身；纵向 Cards→Sides→Note→Footer→按钮。
// 覆盖：遍历四组尺寸，用 requireInsideRect/insideScreen/overlap 与 InDelta 断言所有子矩形与顺序。
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
				report.Note, report.Footer, report.BackButton,
				report.Sides[0], report.Sides[1],
				report.DetailSides[0], report.DetailSides[1],
			} {
				requireInsideRect(t, rect, report.Panel)
			}
			require.False(t, battleReportRectsOverlap(report.Title, report.Badge))
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

			// 趋势页：顶部说明条、模式按钮与 2×2 图表都必须在内容区内且互不重叠。
			requireInsideRect(t, report.TrendHeader, report.Content)
			for idx, mode := range report.TrendModes {
				requireInsideRect(t, mode, report.TrendHeader)
				if idx > 0 {
					require.False(t, battleReportRectsOverlap(report.TrendModes[idx-1], mode))
				}
			}
			for idx, chart := range report.TrendCharts {
				requireInsideRect(t, chart, report.Content)
				require.False(t, battleReportRectsOverlap(chart, report.TrendHeader))
				for prev := 0; prev < idx; prev++ {
					require.False(t, battleReportRectsOverlap(report.TrendCharts[prev], chart))
				}
			}
		})
	}
}

// TestBattleReportLayoutButtonsOptional 验证 withButtons 分支：结算页给返回按钮，暂停统计页给提示且面板不跳变。
// 场景：同一套 CalcBattleReportLayout 既画战斗结算，也画暂停实时统计，后者不能出现“返回主界面”按钮。
// 预期：true 时 Back/Buttons 非空、Hint 为空；false 时三者 Empty、Hint 在 Panel 内，两者 Panel 相等。
// 覆盖：1920×1080 下分别传 true/false，断言 Empty() 布尔值与 Panel 相等。
func TestBattleReportLayoutButtonsOptional(t *testing.T) {
	withButtons := CalcBattleReportLayout(1920, 1080, true)
	require.False(t, withButtons.BackButton.Empty())
	require.False(t, withButtons.Buttons.Empty())
	require.True(t, withButtons.Hint.Empty())

	noButtons := CalcBattleReportLayout(1920, 1080, false)
	require.True(t, noButtons.BackButton.Empty())
	require.True(t, noButtons.Buttons.Empty())
	require.False(t, noButtons.Hint.Empty())
	requireInsideRect(t, noButtons.Hint, noButtons.Panel)
	require.Equal(t, withButtons.Panel, noButtons.Panel)
}

// TestBattleReportDurationText 验证 BattleReportDurationText 按 60 TPS 把 SimTick 换算成作战时长文本。
// 场景：页脚显示游戏内等效时长，暂停不推进 SimTick；需覆盖整分、跨分、跨小时与负 tick 输入。
// 预期：0→00:00、59*60→00:59、61*60→01:01、3723*60→1:02:03、-100→00:00，跨小时才用 H:mm:ss。
// 覆盖：五组 tick 直接断言字符串，避免 61 秒被写成 01:61 或负值出现 '-'。
func TestBattleReportDurationText(t *testing.T) {
	require.Equal(t, "00:00", BattleReportDurationText(0))
	require.Equal(t, "00:59", BattleReportDurationText(59*60))
	require.Equal(t, "01:01", BattleReportDurationText(61*60))
	require.Equal(t, "1:02:03", BattleReportDurationText(int64(3723*60)))
	require.Equal(t, "00:00", BattleReportDurationText(-100))
}

// TestBattleReportBackButtonAnchoredBottomRight 验证结算页唯一的返回按钮贴齐面板右下角且尺寸固定。
// 场景：结算页只剩“返回主界面”这一个次要操作，需让出底部居中位置并靠右下角，避免抢走战报主体视线。
// 预期：按钮右边界 = 面板右边界 - battleReportPadX*Scale，下边界 = 面板下边界 - battleReportButtonBottomGap*Scale，
// 宽高分别等于 battleReportButtonW/H * Scale；失败意味着按钮漂移或尺寸回退。
// 覆盖：用 1920x1080 布局计算 report，逐项断言右边界、下边界与宽高。
func TestBattleReportBackButtonAnchoredBottomRight(t *testing.T) {
	report := CalcBattleReportLayout(1920, 1080, true)
	require.InDelta(
		t,
		report.Panel.X+report.Panel.W-battleReportPadX*report.Scale,
		report.BackButton.X+report.BackButton.W,
		1e-6,
	)
	require.InDelta(
		t,
		report.Panel.Y+report.Panel.H-battleReportButtonBottomGap*report.Scale,
		report.BackButton.Y+report.BackButton.H,
		1e-6,
	)
	require.InDelta(t, battleReportButtonW*report.Scale, report.BackButton.W, 1e-6)
	require.InDelta(t, battleReportButtonH*report.Scale, report.BackButton.H, 1e-6)
}

// TestBattleReportTrendChartRectsSwitchLayout 验证趋势页两种模式使用不同的图表排布。
// 场景：累计模式需要四张图（2×2）；当前兵力只有舰艇 / 飞机 / 吨位三张，第三张应横跨整行，
// 否则右下会留下一块空白，吨位曲线也失去更长的横轴。
// 预期：累计返回 4 个矩形且等于 TrendCharts；当前返回 3 个，第三块 X/W 覆盖前一行左右两张，
// Y/H 与第二行一致，三个矩形互不重叠。
// 覆盖：1920×1080 布局下调用 TrendChartRects 两次，用 Equal/InDelta/overlap 断言数量与几何。
func TestBattleReportTrendChartRectsSwitchLayout(t *testing.T) {
	report := CalcBattleReportLayout(1920, 1080, true)
	cumulative := report.TrendChartRects(BattleReportTrendModeCumulative)
	require.Equal(t, report.TrendCharts[:], cumulative)

	current := report.TrendChartRects(BattleReportTrendModeCurrent)
	require.Len(t, current, 3)
	require.Equal(t, report.TrendCharts[0], current[0])
	require.Equal(t, report.TrendCharts[1], current[1])
	require.Equal(t, report.TrendCharts[2].Y, current[2].Y)
	require.Equal(t, report.TrendCharts[2].H, current[2].H)
	require.InDelta(t, report.TrendCharts[0].X, current[2].X, 1e-9)
	require.InDelta(
		t,
		report.TrendCharts[1].X+report.TrendCharts[1].W-report.TrendCharts[0].X,
		current[2].W,
		1e-9,
	)
	for idx := range current {
		for prev := 0; prev < idx; prev++ {
			require.False(t, battleReportRectsOverlap(current[prev], current[idx]))
		}
	}
}
