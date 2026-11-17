package drawer

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
	layoutUtils "github.com/narasux/jutland/pkg/utils/layout"
)

// battleReportTrendMaxDrawPoints 每张图每条曲线参与绘制的最大点数。
// 采样点可能积累到几百个，全量绘制会让软件光栅化变慢；累计曲线本身平滑，
// 抽稀到大约一屏宽度的点数对观感没有可见影响。
const battleReportTrendMaxDrawPoints = 200

// battleReportTrendMetric 趋势图的纵轴指标。
type battleReportTrendMetric int

const (
	battleReportTrendShipsBuilt battleReportTrendMetric = iota
	battleReportTrendShipsLost
	battleReportTrendPlanesBuilt
	battleReportTrendPlanesLost
	battleReportTrendSunkTonnage
	battleReportTrendDamageDealt
	// 当前在役兵力，曲线会随战损下降。
	battleReportTrendShipsRemaining
	battleReportTrendPlanesRemaining
	battleReportTrendSurvivingTonnage
)

// value 读取某个采样点上的指标值。
func (m battleReportTrendMetric) value(side state.BattleTimelineSide) float64 {
	switch m {
	case battleReportTrendShipsBuilt:
		return float64(side.ShipsBuilt)
	case battleReportTrendShipsLost:
		return float64(side.ShipsLost)
	case battleReportTrendPlanesBuilt:
		return float64(side.PlanesBuilt)
	case battleReportTrendPlanesLost:
		return float64(side.PlanesLost)
	case battleReportTrendSunkTonnage:
		return side.SunkTonnage
	case battleReportTrendDamageDealt:
		return side.DamageDealt
	case battleReportTrendShipsRemaining:
		return float64(side.ShipsRemaining)
	case battleReportTrendPlanesRemaining:
		return float64(side.PlanesRemaining)
	case battleReportTrendSurvivingTonnage:
		return side.SurvivingTonnage
	}
	return 0
}

// battleReportTrendSeries 一条曲线：阵营 + 指标 + 颜色 + 线型。
type battleReportTrendSeries struct {
	Label  string
	Color  color.RGBA
	Dashed bool
	Player faction.Player
	Metric battleReportTrendMetric
}

// battleReportTrendChart 一张趋势图。
type battleReportTrendChart struct {
	Title  string
	Unit   string
	Series []battleReportTrendSeries
}

// battleReportTrendPlot 曲线图内部坐标区（屏幕坐标）。
type battleReportTrendPlot struct {
	Left, Top, Right, Bottom float64
}

// battleReportTrendPoint 曲线上的一个屏幕坐标点。
type battleReportTrendPoint struct {
	X, Y float64
}

// buildBattleReportTrendCharts 按当前模式生成趋势曲线图。
// 累计模式展示建造（实线）/ 损失（虚线）/ 吨位损失 / 伤害；
// 当前兵力模式展示在役舰艇 / 飞机 / 吨位，曲线会随战损下降。
func buildBattleReportTrendCharts(
	own, enemy faction.Player, mode state.BattleReportTrendMode,
) []battleReportTrendChart {
	if mode != state.BattleReportTrendModeCumulative {
		return []battleReportTrendChart{
			{
				Title: i18n.Text(i18n.MsgBattleReportTrendShipsCurrentTitle),
				Unit:  i18n.Text(i18n.MsgBattleReportTrendUnitShips),
				Series: []battleReportTrendSeries{
					{
						Label: i18n.Text(i18n.MsgBattleReportPlayer), Color: battleReportOwnClr,
						Player: own, Metric: battleReportTrendShipsRemaining,
					},
					{
						Label: i18n.Text(i18n.MsgBattleReportEnemy), Color: battleReportLostClr,
						Player: enemy, Metric: battleReportTrendShipsRemaining,
					},
				},
			},
			{
				Title: i18n.Text(i18n.MsgBattleReportTrendPlanesCurrentTitle),
				Unit:  i18n.Text(i18n.MsgBattleReportTrendUnitPlanes),
				Series: []battleReportTrendSeries{
					{
						Label: i18n.Text(i18n.MsgBattleReportPlayer), Color: battleReportOwnClr,
						Player: own, Metric: battleReportTrendPlanesRemaining,
					},
					{
						Label: i18n.Text(i18n.MsgBattleReportEnemy), Color: battleReportLostClr,
						Player: enemy, Metric: battleReportTrendPlanesRemaining,
					},
				},
			},
			{
				Title: i18n.Text(i18n.MsgBattleReportTrendTonnageCurrentTitle),
				Unit:  i18n.Text(i18n.MsgBattleReportTrendUnitTonnage),
				Series: []battleReportTrendSeries{
					{
						Label: i18n.Text(i18n.MsgBattleReportPlayer), Color: battleReportOwnClr,
						Player: own, Metric: battleReportTrendSurvivingTonnage,
					},
					{
						Label: i18n.Text(i18n.MsgBattleReportEnemy), Color: battleReportLostClr,
						Player: enemy, Metric: battleReportTrendSurvivingTonnage,
					},
				},
			},
		}
	}
	return []battleReportTrendChart{
		{
			Title: i18n.Text(i18n.MsgBattleReportTrendShipsTitle),
			Unit:  i18n.Text(i18n.MsgBattleReportTrendUnitShips),
			Series: []battleReportTrendSeries{
				{
					Label: i18n.Text(i18n.MsgBattleReportTrendOwnBuilt), Color: battleReportOwnClr,
					Player: own, Metric: battleReportTrendShipsBuilt,
				},
				{
					Label: i18n.Text(i18n.MsgBattleReportTrendOwnLost), Color: battleReportOwnClr,
					Dashed: true, Player: own, Metric: battleReportTrendShipsLost,
				},
				{
					Label: i18n.Text(i18n.MsgBattleReportTrendEnemyBuilt), Color: battleReportLostClr,
					Player: enemy, Metric: battleReportTrendShipsBuilt,
				},
				{
					Label: i18n.Text(i18n.MsgBattleReportTrendEnemyLost), Color: battleReportLostClr,
					Dashed: true, Player: enemy, Metric: battleReportTrendShipsLost,
				},
			},
		},
		{
			Title: i18n.Text(i18n.MsgBattleReportTrendPlanesTitle),
			Unit:  i18n.Text(i18n.MsgBattleReportTrendUnitPlanes),
			Series: []battleReportTrendSeries{
				{
					Label: i18n.Text(i18n.MsgBattleReportTrendOwnBuilt), Color: battleReportOwnClr,
					Player: own, Metric: battleReportTrendPlanesBuilt,
				},
				{
					Label: i18n.Text(i18n.MsgBattleReportTrendOwnLost), Color: battleReportOwnClr,
					Dashed: true, Player: own, Metric: battleReportTrendPlanesLost,
				},
				{
					Label: i18n.Text(i18n.MsgBattleReportTrendEnemyBuilt), Color: battleReportLostClr,
					Player: enemy, Metric: battleReportTrendPlanesBuilt,
				},
				{
					Label: i18n.Text(i18n.MsgBattleReportTrendEnemyLost), Color: battleReportLostClr,
					Dashed: true, Player: enemy, Metric: battleReportTrendPlanesLost,
				},
			},
		},
		{
			Title: i18n.Text(i18n.MsgBattleReportTrendTonnageTitle),
			Unit:  i18n.Text(i18n.MsgBattleReportTrendUnitTonnage),
			Series: []battleReportTrendSeries{
				{
					Label: i18n.Text(i18n.MsgBattleReportPlayer), Color: battleReportOwnClr,
					Player: own, Metric: battleReportTrendSunkTonnage,
				},
				{
					Label: i18n.Text(i18n.MsgBattleReportEnemy), Color: battleReportLostClr,
					Player: enemy, Metric: battleReportTrendSunkTonnage,
				},
			},
		},
		{
			Title: i18n.Text(i18n.MsgBattleReportTrendDamageTitle),
			Series: []battleReportTrendSeries{
				{
					Label: i18n.Text(i18n.MsgBattleReportPlayer), Color: battleReportOwnClr,
					Player: own, Metric: battleReportTrendDamageDealt,
				},
				{
					Label: i18n.Text(i18n.MsgBattleReportEnemy), Color: battleReportLostClr,
					Player: enemy, Metric: battleReportTrendDamageDealt,
				},
			},
		},
	}
}

// battleReportTrendPlotRect 把图表卡片换算成内部坐标区。
// 左侧留纵轴刻度，顶部留标题和示例线，底部留时间刻度。
func battleReportTrendPlotRect(rect state.BattleReportRect, scale float64) battleReportTrendPlot {
	return battleReportTrendPlot{
		Left:   rect.X + 62*scale,
		Top:    rect.Y + 50*scale,
		Right:  rect.X + rect.W - 12*scale,
		Bottom: rect.Y + rect.H - 24*scale,
	}
}

// battleReportTrendHoverAt 返回鼠标所在图表的横向比例与图表下标。
// 四张图共用同一条时间轴，悬停任意一张都能联动全部曲线。
func battleReportTrendHoverAt(
	charts []state.BattleReportRect, scale float64,
) (ratio float64, index int, ok bool) {
	sx, sy := ebiten.CursorPosition()
	fx, fy := float64(sx), float64(sy)
	for idx, chart := range charts {
		if chart.Empty() {
			continue
		}
		plot := battleReportTrendPlotRect(chart, scale)
		if fx < plot.Left || fx > plot.Right || fy < plot.Top || fy > plot.Bottom {
			continue
		}
		width := plot.Right - plot.Left
		if width <= 0 {
			continue
		}
		return min(1, max(0, (fx-plot.Left)/width)), idx, true
	}
	return 0, 0, false
}

// battleReportTrendTimeTicks 把时间范围等分成 count 个刻度。
func battleReportTrendTimeTicks(startTick, endTick int64, count int) []int64 {
	if count <= 0 {
		return nil
	}
	if endTick < startTick {
		startTick, endTick = endTick, startTick
	}
	if endTick == startTick || count == 1 {
		return []int64{startTick}
	}
	ticks := make([]int64, 0, count)
	for idx := 0; idx < count; idx++ {
		ticks = append(ticks, startTick+(endTick-startTick)*int64(idx)/int64(count-1))
	}
	return ticks
}

// battleReportTrendNiceAxis 把最大观测值向上取整到 1/2/5×10^n 刻度。
// 返回纵轴上限与刻度间隔；所有值都为零时退化为 0..1。
func battleReportTrendNiceAxis(maxValue float64, targetTicks int) (axisMax, step float64) {
	if !(maxValue > 0) || math.IsNaN(maxValue) || math.IsInf(maxValue, 0) {
		return 1, 1
	}
	if targetTicks < 1 {
		targetTicks = 1
	}
	raw := maxValue / float64(targetTicks)
	base := math.Pow(10, math.Floor(math.Log10(raw)))
	step = base
	switch frac := raw / base; {
	case frac <= 1:
		step = base
	case frac <= 2:
		step = 2 * base
	case frac <= 5:
		step = 5 * base
	default:
		step = 10 * base
	}
	step = max(1, step)
	return math.Ceil(maxValue/step) * step, step
}

// battleReportTrendNearestSample 返回时间轴上离目标拍最近的采样点下标。
func battleReportTrendNearestSample(timeline []state.BattleTimelineSample, tick int64) int {
	if len(timeline) == 0 {
		return -1
	}
	best, bestDistance := 0, absInt64(timeline[0].Tick-tick)
	for idx := 1; idx < len(timeline); idx++ {
		if distance := absInt64(timeline[idx].Tick - tick); distance < bestDistance {
			best, bestDistance = idx, distance
		}
	}
	return best
}

// absInt64 返回 int64 的绝对值；采样时间轴不会接近 MinInt64。
func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

// battleReportTrendScreenPoints 把采样点映射成屏幕坐标。
// 曲线按 axisMax 归一化，时间按 [startTick, endTick] 线性展开。
func battleReportTrendScreenPoints(
	timeline []state.BattleTimelineSample,
	series battleReportTrendSeries,
	startTick, endTick int64,
	axisMax float64,
	plot battleReportTrendPlot,
) []battleReportTrendPoint {
	if len(timeline) == 0 {
		return nil
	}
	span := max(1, endTick-startTick)
	points := make([]battleReportTrendPoint, 0, len(timeline))
	for _, sample := range timeline {
		value := series.Metric.value(sample.TimelineSide(series.Player))
		ratioX := float64(sample.Tick-startTick) / float64(span)
		ratioX = min(1, max(0, ratioX))
		ratioY := 0.0
		if axisMax > 0 {
			ratioY = min(1, max(0, value/axisMax))
		}
		points = append(points, battleReportTrendPoint{
			X: plot.Left + ratioX*(plot.Right-plot.Left),
			Y: plot.Bottom - ratioY*(plot.Bottom-plot.Top),
		})
	}
	return points
}

// battleReportTrendTangents 用 Fritsch–Carlson 单调三次插值计算各点切线。
// 累计曲线只增不减，普通样条会在平台段产生下冲；这里保证贝塞尔控制点
// 落在相邻点纵向区间内，曲线不会画到 0 以下或出现视觉回摆。
func battleReportTrendTangents(points []battleReportTrendPoint) []float64 {
	n := len(points)
	tangents := make([]float64, n)
	if n < 2 {
		return tangents
	}
	slopes := make([]float64, n-1)
	for idx := 0; idx+1 < n; idx++ {
		dx := points[idx+1].X - points[idx].X
		if dx <= 0 {
			slopes[idx] = 0
			continue
		}
		slopes[idx] = (points[idx+1].Y - points[idx].Y) / dx
	}
	if n == 2 {
		tangents[0], tangents[1] = slopes[0], slopes[0]
		return tangents
	}
	tangents[0], tangents[n-1] = slopes[0], slopes[n-2]
	for idx := 1; idx+1 < n; idx++ {
		d0, d1 := slopes[idx-1], slopes[idx]
		if d0 == 0 || d1 == 0 || (d0 > 0) != (d1 > 0) {
			tangents[idx] = 0
			continue
		}
		h0 := points[idx].X - points[idx-1].X
		h1 := points[idx+1].X - points[idx].X
		if h0 <= 0 || h1 <= 0 {
			tangents[idx] = 0
			continue
		}
		w0, w1 := 2*h1+h0, h1+2*h0
		tangents[idx] = (w0 + w1) / (w0/d0 + w1/d1)
	}
	return tangents
}

// battleReportTrendBezierControls 返回区间 idx→idx+1 的三次贝塞尔控制点。
func battleReportTrendBezierControls(
	points []battleReportTrendPoint, tangents []float64, index int,
) (c1x, c1y, c2x, c2y float64) {
	p0, p1 := points[index], points[index+1]
	h := p1.X - p0.X
	return p0.X + h/3, p0.Y + tangents[index]*h/3,
		p1.X - h/3, p1.Y - tangents[index+1]*h/3
}

// battleReportTrendCurvePath 生成平滑曲线路径。
func battleReportTrendCurvePath(points []battleReportTrendPoint) *vector.Path {
	path := &vector.Path{}
	if len(points) == 0 {
		return path
	}
	path.MoveTo(float32(points[0].X), float32(points[0].Y))
	if len(points) < 2 {
		return path
	}
	tangents := battleReportTrendTangents(points)
	for idx := 0; idx+1 < len(points); idx++ {
		c1x, c1y, c2x, c2y := battleReportTrendBezierControls(points, tangents, idx)
		path.CubicTo(
			float32(c1x), float32(c1y), float32(c2x), float32(c2y),
			float32(points[idx+1].X), float32(points[idx+1].Y),
		)
	}
	return path
}

// battleReportTrendPolyline 把贝塞尔曲线离散成折线，供虚线描边使用。
func battleReportTrendPolyline(
	points []battleReportTrendPoint, tangents []float64, steps int,
) []battleReportTrendPoint {
	if len(points) < 2 || steps < 1 {
		return points
	}
	polyline := make([]battleReportTrendPoint, 0, (len(points)-1)*steps+1)
	polyline = append(polyline, points[0])
	for idx := 0; idx+1 < len(points); idx++ {
		c1x, c1y, c2x, c2y := battleReportTrendBezierControls(points, tangents, idx)
		p0, p1 := points[idx], points[idx+1]
		for step := 1; step <= steps; step++ {
			t := float64(step) / float64(steps)
			mt := 1 - t
			x := mt*mt*mt*p0.X + 3*mt*mt*t*c1x + 3*mt*t*t*c2x + t*t*t*p1.X
			y := mt*mt*mt*p0.Y + 3*mt*mt*t*c1y + 3*mt*t*t*c2y + t*t*t*p1.Y
			polyline = append(polyline, battleReportTrendPoint{X: x, Y: y})
		}
	}
	return polyline
}

// battleReportTrendDecimate 按固定步长抽稀绘制点，并保留最后一个点。
func battleReportTrendDecimate(
	points []battleReportTrendPoint, maxPoints int,
) []battleReportTrendPoint {
	if maxPoints < 2 || len(points) <= maxPoints {
		return points
	}
	stride := (len(points) + maxPoints - 1) / maxPoints
	decimated := make([]battleReportTrendPoint, 0, maxPoints+1)
	for idx := 0; idx < len(points); idx += stride {
		decimated = append(decimated, points[idx])
	}
	if last := points[len(points)-1]; decimated[len(decimated)-1].X != last.X {
		decimated = append(decimated, last)
	}
	return decimated
}

// battleReportTrendMaxValue 返回所有曲线在所有采样点上的最大值。
func battleReportTrendMaxValue(
	timeline []state.BattleTimelineSample, series []battleReportTrendSeries,
) float64 {
	maxValue := 0.0
	for _, item := range series {
		for _, sample := range timeline {
			maxValue = max(maxValue, item.Metric.value(sample.TimelineSide(item.Player)))
		}
	}
	return maxValue
}

// battleReportTrendValueText 按指标格式化数值；伤害保留小数值，其余取整。
func battleReportTrendValueText(metric battleReportTrendMetric, value float64) string {
	if metric == battleReportTrendDamageDealt {
		return battleReportDamageText(value)
	}
	return groupThousands(int64(math.Round(value)))
}

// battleReportTrendPathOptions 给矢量路径染色；零值 ColorScale 是白色，必须显式设置。
func battleReportTrendPathOptions(clr color.Color) *vector.DrawPathOptions {
	opts := &vector.DrawPathOptions{AntiAlias: true}
	opts.ColorScale.ScaleWithColor(clr)
	return opts
}

// drawBattleReportTrends 绘制趋势曲线页：顶部标题 + 2×2 曲线图 + 悬停提示。
func (d *Drawer) drawBattleReportTrends(
	screen *ebiten.Image,
	report state.BattleReportLayout,
	ms *state.MissionState,
	scale float64,
	bodyFont, monoFont *text.GoTextFaceSource,
) {
	timeline := ms.Stats.Timeline
	durationTick := ms.Core.SimTick
	if ms.Stats.Finalized {
		durationTick = ms.Stats.EndTick
	}

	startTick := int64(0)
	if len(timeline) > 0 {
		startTick = timeline[0].Tick
	}
	endTick := max(durationTick, startTick)
	if len(timeline) > 0 {
		endTick = max(endTick, timeline[len(timeline)-1].Tick)
	}
	if endTick <= startTick {
		endTick = startTick + 1
	}

	mode := ms.UI.BattleReportTrendMode
	if !mode.Valid() {
		mode = state.BattleReportTrendModeCurrent
	}
	header := report.TrendHeader
	modes := report.TrendModes
	title := i18n.Text(i18n.MsgBattleReportTrendTitle)
	titleRect := state.BattleReportRect{
		X: header.X, Y: header.Y,
		W: modes[0].X - 12*scale - header.X, H: header.H,
	}
	d.drawReportText(
		screen, title, titleRect,
		fitReportFontSize(title, titleRect.W, 22*scale, bodyFont),
		bodyFont, battleReportTitleClr, reportAlignLeft,
	)
	d.drawBattleReportTrendModes(screen, report, mode, scale, bodyFont)

	charts := buildBattleReportTrendCharts(ms.Player.CurPlayer, ms.Player.CurEnemy, mode)
	chartRects := report.TrendChartRects(mode)
	hoverTick := int64(0)
	hoverActive, hoveredChart := false, -1
	if ratio, index, ok := battleReportTrendHoverAt(chartRects, scale); ok && len(timeline) > 0 {
		hoverTick = startTick + int64(math.Round(ratio*float64(endTick-startTick)))
		hoverActive, hoveredChart = true, index
	}
	for idx, chart := range charts {
		if idx >= len(chartRects) {
			break
		}
		d.drawBattleReportTrendChart(
			screen, chartRects[idx], chart, timeline, startTick, endTick,
			hoverTick, hoverActive, scale, bodyFont, monoFont,
		)
	}
	// 浮层必须在所有图表绘制完之后再画：否则相邻图表会把浮层盖暗，
	// 截图里出现的「半透明浮层压在坐标轴上」就是绘制顺序导致的。
	if hoverActive && hoveredChart >= 0 && hoveredChart < len(charts) {
		if index := battleReportTrendNearestSample(timeline, hoverTick); index >= 0 {
			d.drawBattleReportTrendTooltip(
				screen, charts[hoveredChart], timeline[index],
				chartRects[hoveredChart], scale, bodyFont, monoFont,
			)
		}
	}

}

// drawBattleReportTrendModes 绘制「当前兵力 / 建造与损失」切换按钮。
func (d *Drawer) drawBattleReportTrendModes(
	screen *ebiten.Image,
	report state.BattleReportLayout,
	active state.BattleReportTrendMode,
	scale float64,
	bodyFont *text.GoTextFaceSource,
) {
	labels := [2]string{
		i18n.Text(i18n.MsgBattleReportTrendModeCurrent),
		i18n.Text(i18n.MsgBattleReportTrendModeCumulative),
	}
	sx, sy := ebiten.CursorPosition()
	for idx, rect := range report.TrendModes {
		if rect.Empty() {
			continue
		}
		fill, border := battleReportTabFill, battleReportTabBorder
		textClr := battleReportLabelClr
		if state.BattleReportTrendMode(idx) == active {
			fill, border, textClr = battleReportTabActive, battleReportButtonBrd, colorx.White
		} else if rect.Contains(sx, sy) {
			fill, border = battleReportHoverFill, battleReportHoverBrd
		}
		vector.FillRect(
			screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
			fill, false,
		)
		vector.StrokeRect(
			screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
			float32(max(1, scale)), border, false,
		)
		label := labels[idx]
		d.drawReportText(
			screen, label, rect,
			fitReportFontSize(label, rect.W-12*scale, 13*scale, bodyFont),
			bodyFont, textClr, reportAlignCenter,
		)
	}
}

// drawBattleReportTrendChart 绘制单张曲线图。
func (d *Drawer) drawBattleReportTrendChart(
	screen *ebiten.Image,
	rect state.BattleReportRect,
	chart battleReportTrendChart,
	timeline []state.BattleTimelineSample,
	startTick, endTick, hoverTick int64,
	hoverActive bool,
	scale float64,
	bodyFont, monoFont *text.GoTextFaceSource,
) {
	if rect.Empty() {
		return
	}
	vector.FillRect(
		screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
		battleReportCardFill, false,
	)
	vector.StrokeRect(
		screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
		float32(max(1, scale)), battleReportCardBorder, false,
	)

	title := chart.Title
	if chart.Unit != "" {
		title += "  ·  " + chart.Unit
	}
	titleRect := state.BattleReportRect{
		X: rect.X + 14*scale, Y: rect.Y + 6*scale, W: rect.W - 28*scale, H: 18 * scale,
	}
	d.drawReportText(
		screen, title, titleRect,
		fitReportFontSize(title, titleRect.W, 15*scale, bodyFont),
		bodyFont, battleReportTitleClr, reportAlignLeft,
	)
	d.drawBattleReportTrendLegend(screen, rect, chart.Series, scale, bodyFont)

	plot := battleReportTrendPlotRect(rect, scale)
	if plot.Right <= plot.Left || plot.Bottom <= plot.Top {
		return
	}
	vector.FillRect(
		screen, float32(plot.Left), float32(plot.Top),
		float32(plot.Right-plot.Left), float32(plot.Bottom-plot.Top),
		color.RGBA{R: 5, G: 11, B: 15, A: 150}, false,
	)

	if len(timeline) < 2 {
		empty := i18n.Text(i18n.MsgBattleReportTrendEmpty)
		emptyRect := state.BattleReportRect{
			X: plot.Left, Y: plot.Top, W: plot.Right - plot.Left, H: plot.Bottom - plot.Top,
		}
		d.drawReportText(
			screen, empty, emptyRect,
			fitReportFontSize(empty, emptyRect.W, 15*scale, bodyFont),
			bodyFont, battleReportSubtitleClr, reportAlignCenter,
		)
		return
	}

	maxValue := battleReportTrendMaxValue(timeline, chart.Series)
	axisMax, axisStep := battleReportTrendNiceAxis(maxValue, 4)
	d.drawBattleReportTrendAxes(screen, plot, startTick, endTick, axisMax, axisStep, scale, monoFont)

	for _, series := range chart.Series {
		points := battleReportTrendDecimate(
			battleReportTrendScreenPoints(timeline, series, startTick, endTick, axisMax, plot),
			battleReportTrendMaxDrawPoints,
		)
		if len(points) == 0 {
			continue
		}
		if series.Dashed {
			tangents := battleReportTrendTangents(points)
			polyline := battleReportTrendPolyline(points, tangents, 2)
			d.drawBattleReportTrendDashed(
				screen, polyline, 4*scale, 3.5*scale, max(1, 1.3*scale), series.Color,
			)
		} else {
			path := battleReportTrendCurvePath(points)
			vector.StrokePath(
				screen, path,
				&vector.StrokeOptions{
					Width:    float32(max(1, 1.6*scale)),
					LineCap:  vector.LineCapRound,
					LineJoin: vector.LineJoinRound,
				},
				battleReportTrendPathOptions(series.Color),
			)
		}
		// 终点圆点标出最新值，方便和曲线末端的图例 / 轴刻度对照。
		last := points[len(points)-1]
		vector.FillCircle(
			screen, float32(last.X), float32(last.Y),
			float32(max(1.6, 2.2*scale)), series.Color, true,
		)
	}

	if hoverActive {
		d.drawBattleReportTrendHover(
			screen, chart, timeline, startTick, endTick, hoverTick,
			plot, axisMax, scale,
		)
	}
}

// battleReportTrendLegendItem 一个图例项在绘制行内的横向位置。
type battleReportTrendLegendItem struct {
	LineX, LineW   float64
	LabelX, LabelW float64
}

// battleReportTrendLegendItems 把图例项按自然宽度连续排列，并整体右对齐。
// 可用宽度不足时先压缩字号，再重新测量；返回最终字号与每项位置，
// 最后一项的右边界正好落在 right 上。
func battleReportTrendLegendItems(
	labels []string,
	right, fontSize, lineW, labelGap, itemGap, available float64,
	measure func(string, float64) float64,
) ([]battleReportTrendLegendItem, float64) {
	if len(labels) == 0 {
		return nil, fontSize
	}
	fixed := float64(len(labels))*(lineW+labelGap) + float64(len(labels)-1)*itemGap
	labelSum := 0.0
	for _, label := range labels {
		labelSum += measure(label, fontSize)
	}
	if labelSum > 0 && fixed+labelSum > available {
		availableForLabels := available - fixed
		if availableForLabels <= 0 {
			fontSize = 8
		} else {
			fontSize = max(8, fontSize*availableForLabels/labelSum)
		}
	}
	widths := make([]float64, len(labels))
	total := fixed
	for idx, label := range labels {
		widths[idx] = measure(label, fontSize)
		total += widths[idx]
	}
	items := make([]battleReportTrendLegendItem, 0, len(labels))
	x := right - total
	for idx := range labels {
		items = append(items, battleReportTrendLegendItem{
			LineX: x, LineW: lineW,
			LabelX: x + lineW + labelGap, LabelW: widths[idx],
		})
		x += lineW + labelGap + widths[idx] + itemGap
	}
	return items, fontSize
}

// drawBattleReportTrendLegend 在图表标题下方绘制右对齐的一行图例。
func (d *Drawer) drawBattleReportTrendLegend(
	screen *ebiten.Image,
	rect state.BattleReportRect,
	series []battleReportTrendSeries,
	scale float64,
	bodyFont *text.GoTextFaceSource,
) {
	if len(series) == 0 {
		return
	}
	pad := 14 * scale
	y := rect.Y + 26*scale
	height := 15 * scale
	lineW := 16 * scale
	labelGap := 5 * scale
	itemGap := 14 * scale
	labels := make([]string, len(series))
	for idx := range series {
		labels[idx] = series[idx].Label
	}
	items, fontSize := battleReportTrendLegendItems(
		labels, rect.X+rect.W-pad, 11.5*scale, lineW, labelGap, itemGap,
		rect.W-2*pad,
		func(label string, size float64) float64 {
			return layoutUtils.CalcTextWidth(label, size, font.ForText(label, bodyFont))
		},
	)
	for idx, item := range items {
		cy := y + height/2
		if series[idx].Dashed {
			vector.StrokeLine(
				screen, float32(item.LineX), float32(cy),
				float32(item.LineX+item.LineW*0.4), float32(cy),
				float32(max(1, 1.4*scale)), series[idx].Color, true,
			)
			vector.StrokeLine(
				screen, float32(item.LineX+item.LineW*0.62), float32(cy),
				float32(item.LineX+item.LineW), float32(cy),
				float32(max(1, 1.4*scale)), series[idx].Color, true,
			)
		} else {
			vector.StrokeLine(
				screen, float32(item.LineX), float32(cy),
				float32(item.LineX+item.LineW), float32(cy),
				float32(max(1, 1.4*scale)), series[idx].Color, true,
			)
		}
		d.drawReportText(
			screen, series[idx].Label,
			state.BattleReportRect{
				X: item.LabelX, Y: y, W: item.LabelW, H: height,
			},
			fontSize, bodyFont, battleReportLabelClr, reportAlignLeft,
		)
	}
}

// drawBattleReportTrendAxes 绘制网格、纵轴刻度与时间刻度。
func (d *Drawer) drawBattleReportTrendAxes(
	screen *ebiten.Image,
	plot battleReportTrendPlot,
	startTick, endTick int64,
	axisMax, axisStep float64,
	scale float64,
	monoFont *text.GoTextFaceSource,
) {
	steps := max(1, int(math.Round(axisMax/axisStep)))
	for idx := 0; idx <= steps; idx++ {
		value := float64(idx) * axisStep
		y := plot.Bottom - value/axisMax*(plot.Bottom-plot.Top)
		gridClr := color.RGBA{R: 214, G: 201, B: 178, A: 46}
		if idx == 0 {
			gridClr.A = 96
		}
		vector.StrokeLine(
			screen, float32(plot.Left), float32(y), float32(plot.Right), float32(y),
			float32(max(1, scale)), gridClr, true,
		)
		label := groupThousands(int64(math.Round(value)))
		labelRect := state.BattleReportRect{
			X: plot.Left - 56*scale, Y: y - 8*scale, W: 50 * scale, H: 16 * scale,
		}
		d.drawReportText(
			screen, label, labelRect,
			fitReportFontSize(label, labelRect.W, 11*scale, monoFont),
			monoFont, battleReportSubtitleClr, reportAlignRight,
		)
	}

	ticks := battleReportTrendTimeTicks(startTick, endTick, 4)
	for idx, tick := range ticks {
		ratio := float64(tick-startTick) / float64(max(1, endTick-startTick))
		x := plot.Left + ratio*(plot.Right-plot.Left)
		vector.StrokeLine(
			screen, float32(x), float32(plot.Top), float32(x), float32(plot.Bottom),
			float32(max(1, scale)), color.RGBA{R: 214, G: 201, B: 178, A: 38}, true,
		)
		label := state.BattleReportDurationText(tick)
		labelW := 62 * scale
		labelX := x - labelW/2
		align := reportAlignCenter
		switch idx {
		case 0:
			labelX, align = x, reportAlignLeft
		case len(ticks) - 1:
			labelX, align = x-labelW, reportAlignRight
		}
		labelRect := state.BattleReportRect{
			X: labelX, Y: plot.Bottom + 4*scale, W: labelW, H: 14 * scale,
		}
		d.drawReportText(
			screen, label, labelRect,
			fitReportFontSize(label, labelRect.W, 10.5*scale, monoFont),
			monoFont, battleReportSubtitleClr, align,
		)
	}
}

// drawBattleReportTrendHover 绘制联动十字线与采样点圆点。
// 数值浮层由调用方在所有图表绘制完成后统一绘制，避免被相邻图表覆盖。
func (d *Drawer) drawBattleReportTrendHover(
	screen *ebiten.Image,
	chart battleReportTrendChart,
	timeline []state.BattleTimelineSample,
	startTick, endTick, hoverTick int64,
	plot battleReportTrendPlot,
	axisMax float64,
	scale float64,
) {
	index := battleReportTrendNearestSample(timeline, hoverTick)
	if index < 0 {
		return
	}
	sample := timeline[index]
	ratio := float64(sample.Tick-startTick) / float64(max(1, endTick-startTick))
	ratio = min(1, max(0, ratio))
	x := plot.Left + ratio*(plot.Right-plot.Left)
	vector.StrokeLine(
		screen, float32(x), float32(plot.Top), float32(x), float32(plot.Bottom),
		float32(max(1, scale)), color.RGBA{R: 232, G: 224, B: 198, A: 135}, true,
	)

	for _, series := range chart.Series {
		value := series.Metric.value(sample.TimelineSide(series.Player))
		y := plot.Bottom - min(1, max(0, value/axisMax))*(plot.Bottom-plot.Top)
		vector.FillCircle(
			screen, float32(x), float32(y),
			float32(max(2, 2.6*scale)), series.Color, true,
		)
	}
}

// battleReportTrendTooltipPos 计算浮层左上角，保证它完整落在图表的绘图区内。
// 鼠标右侧 / 下方放不下时翻到另一侧，再夹紧到绘图区；浮层比绘图区还大时居中，
// 宁可盖住曲线，也不压住横轴刻度、图例或相邻图表。
func battleReportTrendTooltipPos(
	cursorX, cursorY, boxW, boxH float64,
	plot battleReportTrendPlot,
	scale float64,
) (x, y float64) {
	gap := 6 * scale
	minX, maxX := plot.Left+gap, plot.Right-boxW-gap
	minY, maxY := plot.Top+gap, plot.Bottom-boxH-gap
	x = cursorX + 14*scale
	y = cursorY + 14*scale
	if maxX >= minX {
		if x > maxX {
			x = cursorX - boxW - 14*scale
		}
		x = max(minX, min(x, maxX))
	} else {
		x = plot.Left + (plot.Right-plot.Left-boxW)/2
	}
	if maxY >= minY {
		if y > maxY {
			y = cursorY - boxH - 14*scale
		}
		y = max(minY, min(y, maxY))
	} else {
		y = plot.Top + (plot.Bottom-plot.Top-boxH)/2
	}
	return x, y
}

// drawBattleReportTrendTooltip 在鼠标旁绘制采样点数值浮层。
// 浮层被限制在对应图表的绘图区内，避免压住横轴时间刻度、图例或相邻图表。
func (d *Drawer) drawBattleReportTrendTooltip(
	screen *ebiten.Image,
	chart battleReportTrendChart,
	sample state.BattleTimelineSample,
	rect state.BattleReportRect,
	scale float64,
	bodyFont, monoFont *text.GoTextFaceSource,
) {
	timeText := i18n.Format(i18n.MsgBattleReportTrendTooltipTime, map[string]any{
		"Time": state.BattleReportDurationText(sample.Tick),
	})
	headerSize := 13 * scale
	rowSize := 12.5 * scale
	padding := 9 * scale
	rowH := 17 * scale
	dotSize := 6 * scale
	dotGap := 6 * scale
	valueGap := 14 * scale

	labelW := layoutUtils.CalcTextWidth(timeText, headerSize, font.ForText(timeText, monoFont))
	valueW := 0.0
	values := make([]string, len(chart.Series))
	for idx, series := range chart.Series {
		value := series.Metric.value(sample.TimelineSide(series.Player))
		values[idx] = battleReportTrendValueText(series.Metric, value)
		labelW = max(labelW, layoutUtils.CalcTextWidth(
			series.Label, rowSize, font.ForText(series.Label, bodyFont),
		))
		valueW = max(valueW, layoutUtils.CalcTextWidth(
			values[idx], rowSize, font.ForText(values[idx], monoFont),
		))
	}
	boxW := padding*2 + dotSize + dotGap + labelW + valueGap + valueW
	boxH := padding*2 + headerSize + 3*scale + float64(len(chart.Series))*rowH

	sx, sy := ebiten.CursorPosition()
	plot := battleReportTrendPlotRect(rect, scale)
	x, y := battleReportTrendTooltipPos(float64(sx), float64(sy), boxW, boxH, plot, scale)

	vector.FillRect(
		screen, float32(x), float32(y), float32(boxW), float32(boxH),
		color.RGBA{R: 10, G: 20, B: 26, A: 252}, false,
	)
	vector.StrokeRect(
		screen, float32(x), float32(y), float32(boxW), float32(boxH),
		float32(max(1, scale)), battleReportPanelBorder, false,
	)
	d.drawReportText(
		screen, timeText,
		state.BattleReportRect{
			X: x + padding, Y: y + padding, W: boxW - 2*padding, H: headerSize,
		},
		headerSize, monoFont, battleReportNoteClr, reportAlignLeft,
	)

	rowsTop := y + padding + headerSize + 3*scale
	for idx, series := range chart.Series {
		rowY := rowsTop + float64(idx)*rowH
		vector.FillCircle(
			screen,
			float32(x+padding+dotSize/2), float32(rowY+rowH/2),
			float32(dotSize/2), series.Color, true,
		)
		labelRect := state.BattleReportRect{
			X: x + padding + dotSize + dotGap, Y: rowY,
			W: labelW, H: rowH,
		}
		d.drawReportText(
			screen, series.Label, labelRect, rowSize,
			bodyFont, battleReportLabelClr, reportAlignLeft,
		)
		valueRect := state.BattleReportRect{
			X: x + boxW - padding - valueW, Y: rowY, W: valueW, H: rowH,
		}
		d.drawReportText(
			screen, values[idx], valueRect, rowSize,
			monoFont, battleReportValueClr, reportAlignRight,
		)
	}
}

// drawBattleReportTrendDashed 沿折线绘制虚线，保持曲线的圆角与间距一致。
func (d *Drawer) drawBattleReportTrendDashed(
	screen *ebiten.Image,
	points []battleReportTrendPoint,
	dash, gap, width float64,
	clr color.Color,
) {
	if len(points) < 2 || dash <= 0 || gap <= 0 {
		return
	}
	drawing, remaining := true, dash
	for idx := 0; idx+1 < len(points); idx++ {
		p0, p1 := points[idx], points[idx+1]
		segmentLength := math.Hypot(p1.X-p0.X, p1.Y-p0.Y)
		if segmentLength <= 0 {
			continue
		}
		position := 0.0
		for position < segmentLength {
			step := min(remaining, segmentLength-position)
			next := position + step
			if drawing {
				startX := p0.X + (p1.X-p0.X)*(position/segmentLength)
				startY := p0.Y + (p1.Y-p0.Y)*(position/segmentLength)
				endX := p0.X + (p1.X-p0.X)*(next/segmentLength)
				endY := p0.Y + (p1.Y-p0.Y)*(next/segmentLength)
				vector.StrokeLine(
					screen, float32(startX), float32(startY),
					float32(endX), float32(endY), float32(width), clr, true,
				)
			}
			position = next
			remaining -= step
			if remaining <= 1e-6 {
				drawing = !drawing
				if drawing {
					remaining = dash
				} else {
					remaining = gap
				}
			}
		}
	}
}
