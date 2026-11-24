package macro

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/sim"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestRunMacroSameSeedDeterministic 验证 F3 同 seed 两次运行在同步寻路下指标一致。
// 场景：PearlHarbor1941，12000 tick，seeded RNG/固定时钟/稳定 ID；此前异步寻路会让单舰航路在 tick 76 漂移。
// 预期：两次 Metrics/Scorecard/Findings 完全一致；失败说明仍有未接管的异步路径或 map 顺序。
// 覆盖：同一 scenario 调 RunMacro 两次，比较聚合结果；路径搜索由 headless 切换为同步。
func TestRunMacroSameSeedDeterministic(t *testing.T) {
	sc, err := sim.LoadScenario(filepath.Join("..", "testdata", "macro-pearl.json5"))
	if err != nil {
		t.Fatal(err)
	}
	sc.Macro.MaxTicks = 12000
	at := time.Date(2026, 11, 19, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))
	a, err := RunMacro(sc, at)
	if err != nil {
		t.Fatal(err)
	}
	b, err := RunMacro(sc, at)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, deterministicMetrics(a.Metrics), deterministicMetrics(b.Metrics))
	require.Equal(t, deterministicScorecard(a.Scorecard), deterministicScorecard(b.Scorecard))
	require.Equal(t, a.Findings, b.Findings)
}

// TestRunMacroSeedsAggregatesSequentialRuns 验证 in-process 多 seed 顺序聚合。
// 场景：PearlHarbor1941，seed 1/2，6000 tick；每个 trial 独立 manager。
// 预期：Runs=2、metrics 中 runs=2、shots>0、DataQuality 含 multi-seed；数值为均值。
// 覆盖：LoadScenario -> RunMacroSeeds，按 metric 名断言；不要求每个 trial 结束。
func TestRunMacroSeedsAggregatesSequentialRuns(t *testing.T) {
	sc, err := sim.LoadScenario(filepath.Join("..", "testdata", "macro-pearl.json5"))
	if err != nil {
		t.Fatal(err)
	}
	sc.Macro.MaxTicks = 6000
	result, err := RunMacroSeeds(sc, 2, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, 2, result.Runs)
	metrics := map[string]float64{}
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	require.Equal(t, 2.0, metrics["runs"])
	require.Greater(t, metrics["shots"], 0.0)
	require.NotEmpty(t, result.Charts.Survival)
	_, ok := metrics["mission_success"]
	require.True(t, ok)
	require.Contains(t, result.DataQuality[len(result.DataQuality)-1], "multi-seed")
	require.NotNil(t, result.BattleReport)
	require.Contains(t, result.BattleReport.Note, "最后一局")
}

// TestRunMacroPearlHarbor 验证 F3 能通过生产任务状态和双方 AI 跑出真实射击/损失。
// 场景：testdata/macro-pearl.json5，PearlHarbor1941，playerSide=P1，6000 tick；AI 双方 computer controller。
// 预期：Fidelity=F3、ticks<=6000、shots>0、hits>0、own/enemy sunk 有数据，且双方被空袭击沉的舰数
// 在 [0, 总沉没数] 内（飞机击杀归因不能超过总损失）；不要求任务结束（可能 timeout）。
// 覆盖：LoadScenario -> 覆盖 MaxTicks -> RunMacro，按 metric 名断言射击、损失与 own/enemy_sunk_by_aircraft 上界；不启动 UI/窗口。
func TestRunMacroPearlHarbor(t *testing.T) {
	sc, err := sim.LoadScenario(filepath.Join("..", "testdata", "macro-pearl.json5"))
	if err != nil {
		t.Fatal(err)
	}
	sc.Macro.MaxTicks = 6000
	result, err := RunMacro(sc, time.Now())
	if err != nil {
		t.Fatalf("RunMacro() error = %v", err)
	}
	require.Equal(t, "F3", result.Fidelity)
	require.False(t, result.Mock)
	metrics := map[string]float64{}
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	require.LessOrEqual(t, metrics["ticks"], 6000.0)
	require.Greater(t, metrics["shots"], 0.0)
	require.Greater(t, metrics["hits"], 0.0)
	require.Greater(t, metrics["own_sunk"]+metrics["enemy_sunk"], 0.0)
	// 空袭归因：两个指标必须存在，且不能超过对应的总沉没数。
	require.GreaterOrEqual(t, metrics["own_sunk_by_aircraft"], 0.0)
	require.GreaterOrEqual(t, metrics["enemy_sunk_by_aircraft"], 0.0)
	require.LessOrEqual(t, metrics["own_sunk_by_aircraft"], metrics["own_sunk"])
	require.LessOrEqual(t, metrics["enemy_sunk_by_aircraft"], metrics["enemy_sunk"])
	// 空中火力按阵营拆分：四个指标必须存在且非负（本任务双方都有飞机）。
	for _, name := range []string{"own_plane_shots", "own_plane_hits", "enemy_plane_shots", "enemy_plane_hits"} {
		require.GreaterOrEqual(t, metrics[name], 0.0, name)
	}
	for _, finding := range result.Findings {
		require.NotEqual(t, "F3-NO-SHOTS", finding.ID)
	}
	// 战报导出：总览卡、趋势采样和完整战况必须随 F3 结果一起返回。
	require.NotNil(t, result.BattleReport)
	require.Len(t, result.BattleReport.Cards, 4)
	require.NotEmpty(t, result.BattleReport.Trend)
	require.NotEmpty(t, result.BattleLog)
}

func deterministicMetrics(rows []sim.MetricRow) []sim.MetricRow {
	out := make([]sim.MetricRow, 0, len(rows))
	for _, row := range rows {
		if strings.HasPrefix(row.Name, "tick_") {
			continue
		}
		out = append(out, row)
	}
	return out
}

func deterministicScorecard(rows []sim.ScorecardItem) []sim.ScorecardItem {
	out := make([]sim.ScorecardItem, 0, len(rows))
	for _, row := range rows {
		if row.Dimension == "性能" {
			continue
		}
		out = append(out, row)
	}
	return out
}

// TestMacroObjectiveMet 验证可选任务目标（坚持到第 N 拍）的判定规则。
// 场景：engine 只认「全歼/被全灭」，而防守型任务需要「坚持到 N 拍」这类目标；
// 判定必须区分提前胜利、中途全灭、以及仍在运行但未到目标拍数。
// 预期：success 恒为 true；failed 恒为 false；running 时 ticks>=surviveTicks 才为 true；
// surviveTicks<=0 视为未配置并返回 false。
// 覆盖：表驱动 6 组 (status, ticks, surviveTicks)，调用 macroObjectiveMet 断言布尔值。
func TestMacroObjectiveMet(t *testing.T) {
	cases := []struct {
		name         string
		status       state.MissionStatus
		ticks        int64
		surviveTicks int64
		want         bool
	}{
		{name: "success early", status: state.MissionSuccess, ticks: 1000, surviveTicks: 12000, want: true},
		{name: "failed early", status: state.MissionFailed, ticks: 1000, surviveTicks: 12000, want: false},
		{name: "running just below", status: state.MissionRunning, ticks: 11999, surviveTicks: 12000, want: false},
		{name: "running at limit", status: state.MissionRunning, ticks: 12000, surviveTicks: 12000, want: true},
		{name: "running past limit", status: state.MissionRunning, ticks: 20000, surviveTicks: 12000, want: true},
		{name: "no objective", status: state.MissionRunning, ticks: 12000, surviveTicks: 0, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, macroObjectiveMet(tc.status, tc.ticks, tc.surviveTicks))
		})
	}
}
