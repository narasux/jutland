package drawer

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestBattleReportTrendNiceAxis 验证纵轴上限与刻度取 1/2/5×10^n，并覆盖零值/非法值回退。
// 场景：曲线图纵轴要容纳双方累计值；直接取原最大值会让刻度出现 3、7、123 这类难读数字，
// 全零战斗和 NaN/Inf（异常统计）也不能把坐标轴画崩。
// 预期：0/负数/NaN/Inf→(1,1)；3→(3,1)；7→(8,2)；123→(150,50)；36500→(40000,10000)。
// 覆盖：表驱动 8 组输入，断言 axisMax 与 step 两个返回值。
func TestBattleReportTrendNiceAxis(t *testing.T) {
	cases := []struct {
		name     string
		maxVal   float64
		target   int
		wantMax  float64
		wantStep float64
	}{
		{name: "zero", maxVal: 0, target: 4, wantMax: 1, wantStep: 1},
		{name: "negative", maxVal: -3, target: 4, wantMax: 1, wantStep: 1},
		{name: "nan", maxVal: math.NaN(), target: 4, wantMax: 1, wantStep: 1},
		{name: "inf", maxVal: math.Inf(1), target: 4, wantMax: 1, wantStep: 1},
		{name: "single", maxVal: 1, target: 4, wantMax: 1, wantStep: 1},
		{name: "three", maxVal: 3, target: 4, wantMax: 3, wantStep: 1},
		{name: "seven", maxVal: 7, target: 4, wantMax: 8, wantStep: 2},
		{name: "hundred", maxVal: 123, target: 4, wantMax: 150, wantStep: 50},
		{name: "tonnage", maxVal: 36500, target: 4, wantMax: 40000, wantStep: 10000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			axisMax, step := battleReportTrendNiceAxis(tc.maxVal, tc.target)
			require.Equal(t, tc.wantMax, axisMax)
			require.Equal(t, tc.wantStep, step)
		})
	}
}

// TestBattleReportTrendTimeTicks 验证时间刻度等分且处理起止相同、反向区间与非法数量。
// 场景：横轴按 0%/33%/66%/100% 画时间文字；短战斗 start==end 不能除零，旧状态也可能给出反向拍数。
// 预期：(0,3600,4)→0/1200/2400/3600；(100,100,3)→[100]；(0,3600,1)→[0]；
// (3600,0,3) 交换后→0/1800/3600；count<=0→nil。
// 覆盖：五组输入直接比较完整刻度切片，边界组单独 require.Nil。
func TestBattleReportTrendTimeTicks(t *testing.T) {
	require.Equal(t, []int64{0, 1200, 2400, 3600}, battleReportTrendTimeTicks(0, 3600, 4))
	require.Equal(t, []int64{100}, battleReportTrendTimeTicks(100, 100, 3))
	require.Equal(t, []int64{0}, battleReportTrendTimeTicks(0, 3600, 1))
	require.Equal(t, []int64{0, 1800, 3600}, battleReportTrendTimeTicks(3600, 0, 3))
	require.Nil(t, battleReportTrendTimeTicks(0, 100, 0))
}

// TestBattleReportTrendTangentsAvoidOvershoot 验证单调三次插值的控制点不越出相邻采样点区间。
// 场景：累计曲线经常先平台后陡降（屏幕 Y 向下），普通 Catmull-Rom 会在平台段向下冲过 0
// 或把「损失」曲线画出不存在的波峰；控制点必须落在相邻两点 Y 之间。
// 预期：5 个点、4 个区间，每段两个控制点 Y 都在 [min(y0,y1), max(y0,y1)] 内。
// 覆盖：构造 x=0/10/20/30/40、y=100/100/50/50/20 的屏幕点列，逐段调用 Bezier 控制点断言。
func TestBattleReportTrendTangentsAvoidOvershoot(t *testing.T) {
	points := []battleReportTrendPoint{
		{X: 0, Y: 100},
		{X: 10, Y: 100},
		{X: 20, Y: 50},
		{X: 30, Y: 50},
		{X: 40, Y: 20},
	}
	tangents := battleReportTrendTangents(points)
	require.Len(t, tangents, len(points))
	for idx := 0; idx+1 < len(points); idx++ {
		_, c1y, _, c2y := battleReportTrendBezierControls(points, tangents, idx)
		low := math.Min(points[idx].Y, points[idx+1].Y)
		high := math.Max(points[idx].Y, points[idx+1].Y)
		require.GreaterOrEqual(t, c1y, low-1e-9)
		require.LessOrEqual(t, c1y, high+1e-9)
		require.GreaterOrEqual(t, c2y, low-1e-9)
		require.LessOrEqual(t, c2y, high+1e-9)
	}
}

// TestBattleReportTrendDecimateKeepsEndpoints 验证绘制点抽稀保留首尾且不重排顺序。
// 场景：长时间战斗可能积累 720 个采样点，四条曲线逐帧全量描边会拖慢暂停界面；
// 抽稀必须让曲线仍从开局画到最新点，否则终点圆点和悬停位置会错位。
// 预期：1000 点抽到不超过 201 点，首点 X=0、末点 X=999，X 严格递增。
// 覆盖：构造 1000 个 X=idx 的点，调用 battleReportTrendDecimate(points, 200) 后检查长度、首尾与相邻顺序。
func TestBattleReportTrendDecimateKeepsEndpoints(t *testing.T) {
	points := make([]battleReportTrendPoint, 0, 1000)
	for idx := 0; idx < 1000; idx++ {
		points = append(points, battleReportTrendPoint{X: float64(idx), Y: float64(idx)})
	}
	decimated := battleReportTrendDecimate(points, 200)
	require.LessOrEqual(t, len(decimated), 201)
	require.Equal(t, 0.0, decimated[0].X)
	require.Equal(t, 999.0, decimated[len(decimated)-1].X)
	for idx := 1; idx < len(decimated); idx++ {
		require.Less(t, decimated[idx-1].X, decimated[idx].X)
	}
}

// TestBattleReportTrendScreenPointsMapsTimeAndValue 验证采样点按时间线性展开、按纵轴上限归一化。
// 场景：三张卡片的坐标区尺寸不同，但同一采样拍必须落在相同横向比例；数值 2/4 应画在纵向 50% 处。
// 预期：Tick 0/60/120 映射到 X=10/50/90，建造 0/2/4 映射到 Y=90/50/10（plot 10..90）。
// 覆盖：构造 HA 建造 0/2/4 的三点 timeline，调用 battleReportTrendScreenPoints 后逐点 InDelta 断言。
func TestBattleReportTrendScreenPointsMapsTimeAndValue(t *testing.T) {
	timeline := []state.BattleTimelineSample{
		{Tick: 0, Sides: map[faction.Player]state.BattleTimelineSide{faction.HumanAlpha: {ShipsBuilt: 0}}},
		{Tick: 60, Sides: map[faction.Player]state.BattleTimelineSide{faction.HumanAlpha: {ShipsBuilt: 2}}},
		{Tick: 120, Sides: map[faction.Player]state.BattleTimelineSide{faction.HumanAlpha: {ShipsBuilt: 4}}},
	}
	series := battleReportTrendSeries{
		Player: faction.HumanAlpha,
		Metric: battleReportTrendShipsBuilt,
	}
	plot := battleReportTrendPlot{Left: 10, Top: 10, Right: 90, Bottom: 90}
	points := battleReportTrendScreenPoints(timeline, series, 0, 120, 4, plot)
	require.Len(t, points, 3)
	require.InDelta(t, 10, points[0].X, 1e-9)
	require.InDelta(t, 90, points[0].Y, 1e-9)
	require.InDelta(t, 50, points[1].X, 1e-9)
	require.InDelta(t, 50, points[1].Y, 1e-9)
	require.InDelta(t, 90, points[2].X, 1e-9)
	require.InDelta(t, 10, points[2].Y, 1e-9)
}

// TestBuildBattleReportTrendChartsAssignsSidesAndMetrics 验证累计模式四张图的曲线数量、阵营归属与指标。
// 场景：舰艇/飞机图要同时呈现双方建造（实线）与损失（虚线），吨位/伤害图只按阵营分色；
// 阵营或指标接反会把「我方损失」画成敌方数据，结算结论会完全相反。
// 预期：4 张图，曲线数 4/4/2/2；首图双方顺序 HA/HA/CA/CA，指标 Built/Lost/Built/Lost；
// 吨位图指标 SunkTonnage；伤害图指标 DamageDealt；所有标题非空。
// 覆盖：以 HA 为我方、CA 为敌方、Cumulative 模式调用 buildBattleReportTrendCharts，
// 逐图断言长度、Player、Metric 与标题。
func TestBuildBattleReportTrendChartsAssignsSidesAndMetrics(t *testing.T) {
	charts := buildBattleReportTrendCharts(
		faction.HumanAlpha, faction.ComputerAlpha, state.BattleReportTrendModeCumulative,
	)
	require.Len(t, charts, 4)
	require.Len(t, charts[0].Series, 4)
	require.Len(t, charts[1].Series, 4)
	require.Len(t, charts[2].Series, 2)
	require.Len(t, charts[3].Series, 2)

	wantPlayers := []faction.Player{
		faction.HumanAlpha, faction.HumanAlpha, faction.ComputerAlpha, faction.ComputerAlpha,
	}
	wantMetrics := []battleReportTrendMetric{
		battleReportTrendShipsBuilt, battleReportTrendShipsLost,
		battleReportTrendShipsBuilt, battleReportTrendShipsLost,
	}
	for idx, series := range charts[0].Series {
		require.Equal(t, wantPlayers[idx], series.Player)
		require.Equal(t, wantMetrics[idx], series.Metric)
		require.Equal(t, idx == 1 || idx == 3, series.Dashed)
	}
	require.Equal(t, battleReportTrendSunkTonnage, charts[2].Series[0].Metric)
	require.Equal(t, battleReportTrendSunkTonnage, charts[2].Series[1].Metric)
	require.Equal(t, battleReportTrendDamageDealt, charts[3].Series[0].Metric)
	require.Equal(t, battleReportTrendDamageDealt, charts[3].Series[1].Metric)
	for _, chart := range charts {
		require.NotEmpty(t, chart.Title)
	}
}

// TestBuildBattleReportTrendChartsCurrentMode 验证当前兵力模式的三张图读取在役指标且不再使用面积填充。
// 场景：累计曲线只增不减，玩家需要能看出战损导致兵力下降的曲线；当前模式应展示在役舰艇 /
// 飞机 / 吨位，三条曲线都随损失下降，并且图表结构里没有面积填充开关。
// 预期：3 张图，每图 2 条曲线（HA/CA），指标依次 ShipsRemaining / PlanesRemaining /
// SurvivingTonnage；标题非空，Unit 分别非空。
// 覆盖：以 HA 为我方、CA 为敌方、Current 模式调用 buildBattleReportTrendCharts，
// 逐图断言曲线数、阵营顺序、指标与单位。
func TestBuildBattleReportTrendChartsCurrentMode(t *testing.T) {
	charts := buildBattleReportTrendCharts(
		faction.HumanAlpha, faction.ComputerAlpha, state.BattleReportTrendModeCurrent,
	)
	require.Len(t, charts, 3)
	wantMetrics := []battleReportTrendMetric{
		battleReportTrendShipsRemaining,
		battleReportTrendPlanesRemaining,
		battleReportTrendSurvivingTonnage,
	}
	for idx, chart := range charts {
		require.Len(t, chart.Series, 2)
		require.Equal(t, faction.HumanAlpha, chart.Series[0].Player)
		require.Equal(t, faction.ComputerAlpha, chart.Series[1].Player)
		require.Equal(t, wantMetrics[idx], chart.Series[0].Metric)
		require.Equal(t, wantMetrics[idx], chart.Series[1].Metric)
		require.NotEmpty(t, chart.Title)
		require.NotEmpty(t, chart.Unit)
	}
}

// TestBattleReportTrendNearestSample 验证悬停时按拍数选择最近采样点。
// 场景：鼠标横坐标换算出的目标拍不一定与采样点重合；取最近点才能让十字线吸附到真实数据。
// 预期：-10→0、29→0、31→60、119→120、999→120；空时间轴返回 -1。
// 覆盖：三点 timeline（0/60/120）逐拍断言下标，另用 nil 断言空输入。
func TestBattleReportTrendNearestSample(t *testing.T) {
	timeline := []state.BattleTimelineSample{{Tick: 0}, {Tick: 60}, {Tick: 120}}
	require.Equal(t, 0, battleReportTrendNearestSample(timeline, -10))
	require.Equal(t, 0, battleReportTrendNearestSample(timeline, 29))
	require.Equal(t, 1, battleReportTrendNearestSample(timeline, 31))
	require.Equal(t, 2, battleReportTrendNearestSample(timeline, 119))
	require.Equal(t, 2, battleReportTrendNearestSample(timeline, 999))
	require.Equal(t, -1, battleReportTrendNearestSample(nil, 0))
}

// TestBattleReportTrendMetricValueReadsAllFields 验证九种指标分别读取对应字段。
// 场景：趋势图的纵轴指标由 metric 枚举决定；字段映射写错会让「损失」曲线显示建造数，
// 或让「在役」曲线继续读累计建造数而永远单调增长。
// 预期：ShipsBuilt/ShipsLost/PlanesBuilt/PlanesLost/SunkTonnage/DamageDealt/
// ShipsRemaining/PlanesRemaining/SurvivingTonnage 各自返回对应值。
// 覆盖：构造一个字段互不相同的 BattleTimelineSide，对九个 metric 调 value 并逐项断言。
func TestBattleReportTrendMetricValueReadsAllFields(t *testing.T) {
	side := state.BattleTimelineSide{
		ShipsBuilt: 1, ShipsLost: 2, PlanesBuilt: 3, PlanesLost: 4,
		SunkTonnage: 5, DamageDealt: 6,
		ShipsRemaining: 7, PlanesRemaining: 8, SurvivingTonnage: 9,
	}
	require.Equal(t, 1.0, battleReportTrendShipsBuilt.value(side))
	require.Equal(t, 2.0, battleReportTrendShipsLost.value(side))
	require.Equal(t, 3.0, battleReportTrendPlanesBuilt.value(side))
	require.Equal(t, 4.0, battleReportTrendPlanesLost.value(side))
	require.Equal(t, 5.0, battleReportTrendSunkTonnage.value(side))
	require.Equal(t, 6.0, battleReportTrendDamageDealt.value(side))
	require.Equal(t, 7.0, battleReportTrendShipsRemaining.value(side))
	require.Equal(t, 8.0, battleReportTrendPlanesRemaining.value(side))
	require.Equal(t, 9.0, battleReportTrendSurvivingTonnage.value(side))
}

// TestBattleReportTrendTooltipStaysInsidePlot 验证浮层被夹紧在绘图区内，不会压住时间刻度或相邻图表。
// 场景：鼠标停在图表右下角时，朝右下展开的浮层会越过绘图区，盖住横轴时间刻度甚至下一张图；
// 旧实现只按窗口边界夹紧，截图里出现过浮层与坐标轴、下一张图重叠。
// 预期：右下角悬停时浮层翻到鼠标左上并夹紧，x/y 都不小于绘图区 +6 的安全边距，
// 且右下边界不超出绘图区；左上角悬停时同样完整落在绘图区内。
// 覆盖：plot 0..500×0..140、浮层 200×100，分别用 (480,130) 与 (50,50) 两个光标位置断言四条边界。
func TestBattleReportTrendTooltipStaysInsidePlot(t *testing.T) {
	plot := battleReportTrendPlot{Left: 0, Top: 0, Right: 500, Bottom: 140}
	const boxW, boxH = 200.0, 100.0

	x, y := battleReportTrendTooltipPos(480, 130, boxW, boxH, plot, 1)
	require.GreaterOrEqual(t, x, plot.Left+6)
	require.GreaterOrEqual(t, y, plot.Top+6)
	require.LessOrEqual(t, x+boxW, plot.Right-6)
	require.LessOrEqual(t, y+boxH, plot.Bottom-6)

	x, y = battleReportTrendTooltipPos(50, 50, boxW, boxH, plot, 1)
	require.GreaterOrEqual(t, x, plot.Left+6)
	require.GreaterOrEqual(t, y, plot.Top+6)
	require.LessOrEqual(t, x+boxW, plot.Right-6)
	require.LessOrEqual(t, y+boxH, plot.Bottom-6)
}

// TestBattleReportTrendLegendItemsRightAligned 验证图例按自然宽度连续排列并整体贴右，空间不足时压缩字号。
// 场景：两系列图例如果等分整行，会一个贴左、一个居中，和标题左对齐的曲线图不协调；
// 四系列图例也需要整体靠右，并且俄语等长标签不能压出卡片。
// 预期：两标签时最后一项右边界=500，所有项按顺序不重叠；四段长标签在 300 宽度内自动缩小字号，
// 仍保持最后一项贴右且总宽度不超过可用宽度。
// 覆盖：用字符数×字号×0.6 的假测量函数，分别对 2 项短标签和 4 项长标签调用
// battleReportTrendLegendItems，断言字号、右边界与相邻项间距。
func TestBattleReportTrendLegendItemsRightAligned(t *testing.T) {
	measure := func(label string, size float64) float64 {
		return float64(len([]rune(label))) * size * 0.6
	}
	shortItems, shortFont := battleReportTrendLegendItems(
		[]string{"我方", "敌方"}, 500, 12, 16, 5, 14, 300, measure,
	)
	require.Equal(t, 12.0, shortFont)
	require.Len(t, shortItems, 2)
	require.InDelta(t, 500, shortItems[1].LabelX+shortItems[1].LabelW, 1e-6)
	for idx := 0; idx+1 < len(shortItems); idx++ {
		require.Less(t, shortItems[idx].LabelX+shortItems[idx].LabelW, shortItems[idx+1].LineX)
	}

	longItems, longFont := battleReportTrendLegendItems(
		[]string{"Постройка", "Потери", "Постройка", "Потери"},
		500, 12, 16, 5, 14, 300, measure,
	)
	require.Less(t, longFont, 12.0)
	require.GreaterOrEqual(t, longFont, 8.0)
	require.InDelta(t, 500, longItems[3].LabelX+longItems[3].LabelW, 1e-6)
	require.GreaterOrEqual(t, longItems[0].LineX, 200-1e-6)
	for idx := 0; idx+1 < len(longItems); idx++ {
		require.Less(t, longItems[idx].LabelX+longItems[idx].LabelW, longItems[idx+1].LineX)
	}
}
