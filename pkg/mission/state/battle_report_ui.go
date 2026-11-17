package state

import (
	"fmt"
	"math"

	"github.com/narasux/jutland/pkg/common/constants"
)

// 战报面板的设计尺寸（1280×760 基准），实际布局按屏幕尺寸整体缩放。
// 结算页现在独占一屏，比首版弹窗宽得多，可以容纳双方分栏与完整明细。
const (
	battleReportDesignW      = 1280.0
	battleReportDesignH      = 760.0
	battleReportMaxScale     = 3
	battleReportScreenMargin = 32.0
	battleReportPadX         = 40.0
	battleReportBadgeW       = 120.0
	battleReportBadgeH       = 34.0
	battleReportTabW         = 92.0
	battleReportTabH         = 34.0
	battleReportTabGap       = 8.0
	battleReportButtonW      = 220.0
	battleReportButtonH      = 46.0
	battleReportButtonGap    = 20.0
	battleReportCardGap      = 16.0
)

// BattleReportPage 战报面板当前页。
type BattleReportPage int

const (
	// BattleReportPageOverview 总览
	BattleReportPageOverview BattleReportPage = iota
	// BattleReportPageShips 舰艇明细
	BattleReportPageShips
	// BattleReportPagePlanes 飞机明细
	BattleReportPagePlanes
	// BattleReportPageTrends 趋势曲线
	BattleReportPageTrends
	// BattleReportPageEvents 战况记录
	BattleReportPageEvents
	battleReportPageCount
)

// Valid 是否为有效页签。
func (p BattleReportPage) Valid() bool {
	return p >= 0 && int(p) < int(battleReportPageCount)
}

// 明细列表的固定几何（设计尺寸，绘制与滚动上限共用同一套计算）。
const (
	battleReportDetailListTop = 76.0
	// BattleReportDetailRowHeight 明细列表行高（设计尺寸）；26 可在同一屏内容纳 13 个舰级。
	BattleReportDetailRowHeight = 26.0
	battleReportDetailTotalH    = 32.0
	battleReportDetailFooterGap = 6.0
	battleReportDetailPad       = 18.0
	battleReportDetailMinRows   = 13
)

// BattleReportDetailVisibleRows 返回明细列表面板当前能完整显示的行数。
// panelH 为屏幕高度，scale 为战报整体缩放。
func BattleReportDetailVisibleRows(panelH, scale float64) int {
	if panelH <= 0 || scale <= 0 {
		return 0
	}
	listTop := battleReportDetailListTop * scale
	listBottom := panelH - battleReportDetailPad*scale -
		battleReportDetailTotalH*scale - battleReportDetailFooterGap*scale
	listH := listBottom - listTop
	if listH <= 0 {
		return 0
	}
	visible := max(1, int(listH/(BattleReportDetailRowHeight*scale)))
	// 当前布局在常规窗口下正好能完整放下 12 行；浮点误差或边界取整不能让它少算一行。
	required := (battleReportDetailListTop +
		float64(battleReportDetailMinRows)*BattleReportDetailRowHeight +
		battleReportDetailPad + battleReportDetailTotalH + battleReportDetailFooterGap) * scale
	if panelH >= required {
		visible = max(visible, battleReportDetailMinRows)
	}
	return visible
}

// BattleReportEventFilter 详情页战况筛选。
type BattleReportEventFilter int

const (
	// BattleReportEventFilterAll 显示全部战况。
	BattleReportEventFilterAll BattleReportEventFilter = iota
	// BattleReportEventFilterKills 只显示击沉和击落。
	BattleReportEventFilterKills
)

// BattleReportTrendMode 趋势页的展示模式。
type BattleReportTrendMode int

const (
	// BattleReportTrendModeCurrent 当前兵力：在役舰艇 / 飞机 / 吨位，可升可降。
	BattleReportTrendModeCurrent BattleReportTrendMode = iota
	// BattleReportTrendModeCumulative 建造与损失累计曲线。
	BattleReportTrendModeCumulative
	battleReportTrendModeCount
)

// Valid 是否为有效的趋势模式。
func (m BattleReportTrendMode) Valid() bool {
	return m >= 0 && int(m) < int(battleReportTrendModeCount)
}

// 战报滚动条拖动目标。
const (
	BattleReportScrollDragNone = iota
	BattleReportScrollDragEvents
	BattleReportScrollDragDetailLeft
	BattleReportScrollDragDetailRight
)

// BattleReportScrollbar 战报列表滚动条的几何信息。
type BattleReportScrollbar struct {
	TrackX, TrackY, TrackW, TrackH float64
	ThumbY, ThumbH                 float64
	MaxScroll                      int
}

// BattleReportEventScrollbar 计算详情页滚动条几何；记录不足一屏时返回 false。
func BattleReportEventScrollbar(
	panel BattleReportRect, total, scroll int, scale float64,
) (BattleReportScrollbar, bool) {
	if scale <= 0 || total <= BattleReportEventRows {
		return BattleReportScrollbar{}, false
	}
	pad := 18 * scale
	contentRight := panel.X + panel.W - pad - 14*scale
	trackX := contentRight + 4*scale
	trackW := 6 * scale
	trackY := panel.Y + 54*scale
	trackH := float64(BattleReportEventRows) * 30 * scale
	maxScroll := total - BattleReportEventRows
	scroll = min(max(0, scroll), maxScroll)
	thumbH := max(18*scale, trackH*float64(BattleReportEventRows)/float64(total))
	thumbY := trackY + (trackH-thumbH)*float64(scroll)/float64(maxScroll)
	return BattleReportScrollbar{
		TrackX: trackX, TrackY: trackY, TrackW: trackW, TrackH: trackH,
		ThumbY: thumbY, ThumbH: thumbH, MaxScroll: maxScroll,
	}, true
}

// BattleReportDetailScrollbar 计算舰艇 / 飞机明细滚动条几何；列表不足一屏时返回 false。
func BattleReportDetailScrollbar(
	panel BattleReportRect, total, scroll int, scale float64,
) (BattleReportScrollbar, bool) {
	if scale <= 0 {
		return BattleReportScrollbar{}, false
	}
	visible := BattleReportDetailVisibleRows(panel.H, scale)
	if visible <= 0 || total <= visible {
		return BattleReportScrollbar{}, false
	}
	pad := 18 * scale
	contentW := panel.W - 2*pad - 14*scale
	trackX := panel.X + pad + contentW + 4*scale
	trackW := 5 * scale
	trackY := panel.Y + 76*scale
	trackH := panel.H - pad - 32*scale - 6*scale - 76*scale
	if trackH <= 0 {
		return BattleReportScrollbar{}, false
	}
	maxScroll := total - visible
	scroll = min(max(0, scroll), maxScroll)
	thumbH := max(18*scale, trackH*float64(visible)/float64(total))
	thumbY := trackY + (trackH-thumbH)*float64(scroll)/float64(maxScroll)
	return BattleReportScrollbar{
		TrackX: trackX, TrackY: trackY, TrackW: trackW, TrackH: trackH,
		ThumbY: thumbY, ThumbH: thumbH, MaxScroll: maxScroll,
	}, true
}

// BattleReportRect 战报面板中的矩形区域（屏幕坐标）。
type BattleReportRect struct {
	X, Y, W, H float64
}

// Contains 判断屏幕坐标是否落在矩形内。
func (r BattleReportRect) Contains(x, y int) bool {
	fx, fy := float64(x), float64(y)
	return fx >= r.X && fx <= r.X+r.W && fy >= r.Y && fy <= r.Y+r.H
}

// Empty 矩形尚未分配（例如实时统计页没有按钮区）。
func (r BattleReportRect) Empty() bool {
	return r.W <= 0 || r.H <= 0
}

// BattleReportLayout 保存战报面板各区域的共享布局，
// 绘制、命中测试与测试断言都读同一份计算结果。
type BattleReportLayout struct {
	Panel       BattleReportRect
	Title       BattleReportRect
	Subtitle    BattleReportRect
	Badge       BattleReportRect
	Tabs        [5]BattleReportRect
	Cards       [4]BattleReportRect
	Sides       [2]BattleReportRect
	DetailSides [2]BattleReportRect
	Content     BattleReportRect
	// TrendHeader 趋势页顶部说明区
	TrendHeader BattleReportRect
	// TrendCharts 趋势页四张曲线图的 2×2 布局（当前兵力模式只用前三格，第三张横跨整行）
	TrendCharts [4]BattleReportRect
	// TrendModes 趋势页「当前兵力 / 建造与损失」切换按钮
	TrendModes   [2]BattleReportRect
	EventFilters [2]BattleReportRect
	Note         BattleReportRect
	Footer       BattleReportRect
	Buttons      BattleReportRect
	BackButton   BattleReportRect
	RetryButton  BattleReportRect
	Hint         BattleReportRect
	Scale        float64
}

// PageAt 返回鼠标位置命中的战报页签。
func (l BattleReportLayout) PageAt(x, y int) (BattleReportPage, bool) {
	for idx, tab := range l.Tabs {
		if tab.Contains(x, y) {
			return BattleReportPage(idx), true
		}
	}
	return BattleReportPageOverview, false
}

// TrendChartRects 返回当前趋势模式实际绘制的图表区域。
// 累计模式用 2×2 四张图；当前兵力只有舰艇 / 飞机 / 吨位三张，
// 第三张横跨两列，给吨位曲线更长的横轴。
func (l BattleReportLayout) TrendChartRects(mode BattleReportTrendMode) []BattleReportRect {
	if mode == BattleReportTrendModeCumulative {
		return l.TrendCharts[:]
	}
	wide := BattleReportRect{
		X: l.TrendCharts[0].X,
		Y: l.TrendCharts[2].Y,
		W: l.TrendCharts[1].X + l.TrendCharts[1].W - l.TrendCharts[0].X,
		H: l.TrendCharts[2].H,
	}
	return []BattleReportRect{l.TrendCharts[0], l.TrendCharts[1], wide}
}

// CalcBattleReportLayout 根据屏幕尺寸计算战报面板布局。
// withButtons 为 true 时是结算页（含返回 / 重试按钮），false 时是暂停实时统计。
func CalcBattleReportLayout(screenW, screenH int, withButtons bool) BattleReportLayout {
	availW := float64(screenW) - 2*battleReportScreenMargin
	availH := float64(screenH) - 2*battleReportScreenMargin
	scale := min(battleReportMaxScale, availW/battleReportDesignW, availH/battleReportDesignH)
	if !(scale > 0) || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}

	panelW, panelH := battleReportDesignW*scale, battleReportDesignH*scale
	panelX := (float64(screenW) - panelW) / 2
	panelY := (float64(screenH) - panelH) / 2

	// rect 把设计坐标系（面板左上角为原点）换算成屏幕坐标。
	rect := func(x, y, w, h float64) BattleReportRect {
		return BattleReportRect{
			X: panelX + x*scale,
			Y: panelY + y*scale,
			W: w * scale,
			H: h * scale,
		}
	}

	innerW := battleReportDesignW - 2*battleReportPadX

	tabsW := float64(battleReportPageCount)*battleReportTabW +
		float64(battleReportPageCount-1)*battleReportTabGap
	tabsX := battleReportPadX + innerW - battleReportBadgeW - 16 - tabsW
	tabs := [5]BattleReportRect{}
	for idx := range tabs {
		tabs[idx] = rect(
			tabsX+float64(idx)*(battleReportTabW+battleReportTabGap),
			28, battleReportTabW, battleReportTabH,
		)
	}

	cardW := (innerW - 3*battleReportCardGap) / 4
	cards := [4]BattleReportRect{}
	for idx := range cards {
		cards[idx] = rect(
			battleReportPadX+float64(idx)*(cardW+battleReportCardGap),
			136, cardW, 104,
		)
	}

	sideGap := 20.0
	sideW := (innerW - sideGap) / 2
	sides := [2]BattleReportRect{
		rect(battleReportPadX, 252, sideW, 364),
		rect(battleReportPadX+sideW+sideGap, 252, sideW, 364),
	}
	detailSides := [2]BattleReportRect{
		rect(battleReportPadX, 136, sideW, 480),
		rect(battleReportPadX+sideW+sideGap, 136, sideW, 480),
	}

	// 趋势页：顶部说明条 + 2×2 曲线图。图高固定，保证四张图的坐标轴刻度对齐。
	trendGap := 16.0
	trendTop, trendH := 172.0, 214.0
	trendW := (innerW - trendGap) / 2
	trendCharts := [4]BattleReportRect{
		rect(battleReportPadX, trendTop, trendW, trendH),
		rect(battleReportPadX+trendW+trendGap, trendTop, trendW, trendH),
		rect(battleReportPadX, trendTop+trendH+trendGap, trendW, trendH),
		rect(battleReportPadX+trendW+trendGap, trendTop+trendH+trendGap, trendW, trendH),
	}
	// 模式切换按钮放在说明条右侧；按钮宽度固定，保证两种语言下都能完整显示。
	modeW, modeH, modeGap := 112.0, 24.0, 6.0
	modeX := battleReportPadX + innerW - modeW*2 - modeGap
	trendModes := [2]BattleReportRect{
		rect(modeX, 138, modeW, modeH),
		rect(modeX+modeW+modeGap, 138, modeW, modeH),
	}

	filterW, filterH, filterGap := 58.0, 22.0, 6.0
	filtersW := 2*filterW + filterGap
	filtersX := battleReportDesignW - 18 - 220 - 12 - filtersW
	filterRects := [2]BattleReportRect{
		rect(filtersX, 142, filterW, filterH),
		rect(filtersX+filterW+filterGap, 142, filterW, filterH),
	}

	layout := BattleReportLayout{
		Panel:        BattleReportRect{X: panelX, Y: panelY, W: panelW, H: panelH},
		Title:        rect(battleReportPadX, 28, tabsX-20-battleReportPadX, 44),
		Badge:        rect(battleReportPadX+innerW-battleReportBadgeW, 28, battleReportBadgeW, battleReportBadgeH),
		Tabs:         tabs,
		Subtitle:     rect(battleReportPadX, 80, innerW, 26),
		Cards:        cards,
		Sides:        sides,
		DetailSides:  detailSides,
		Content:      rect(battleReportPadX, 128, innerW, 496),
		TrendHeader:  rect(battleReportPadX, 136, innerW, 28),
		TrendCharts:  trendCharts,
		TrendModes:   trendModes,
		EventFilters: filterRects,
		Note:         rect(battleReportPadX, 630, innerW, 24),
		Footer:       rect(battleReportPadX, 660, innerW, 30),
		Scale:        scale,
	}
	if withButtons {
		// 两个按钮居中并排：左侧是低频的重新挑战，右侧是主要操作返回主界面。
		totalW := battleReportButtonW*2 + battleReportButtonGap
		buttonX := battleReportPadX + (innerW-totalW)/2
		layout.Buttons = rect(battleReportPadX, 696, innerW, battleReportButtonH)
		layout.RetryButton = rect(
			buttonX,
			696, battleReportButtonW, battleReportButtonH,
		)
		layout.BackButton = rect(
			buttonX+battleReportButtonW+battleReportButtonGap,
			696, battleReportButtonW, battleReportButtonH,
		)
	} else {
		layout.Hint = rect(battleReportPadX, 696, innerW, 28)
	}
	return layout
}

// BattleReportDurationText 把模拟拍换算成作战时长文本。
// 暂停不推进 SimTick，速度倍率下是游戏内等效时长，不是墙钟时间。
func BattleReportDurationText(simTick int64) string {
	seconds := max(0, simTick) / constants.MaxTPS
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}
