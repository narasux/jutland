package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestWriteAnalysisBriefAggregatesHistory 验证跨 run 分析简报聚合 finding、红色指标、场景样本波动与帧门禁。
// 场景：r1 含 high finding + red scorecard + own_sunk=10/enemy_sunk=5；r2 同场景无 finding 且
// own_sunk=20/enemy_sunk=7，用来判断 r1 finding 已修复，同时让场景汇总出现 min/max 与样本不足提示。
// 预期：RecurringFindings/RedScorecards 各 1；F3-NO-SHOTS 的 Fixed=true；场景汇总 runs=2、
// own sunk 10~20、enemy sunk 5~7；Frame.samples=1、latestP99Ms=12.5；Prompt 非空，
// md 含 finding ID、「样本不足（runs < 3）」与「latest p99 12.500 ms」。
// 覆盖：临时 reports 目录写两条 history、r1/r2 result.json（含 metrics）与 frame/history.jsonl，
// 断言 JSON 汇总与 md 文案。
func TestWriteAnalysisBriefAggregatesHistory(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "runs", "r1")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	result := Result{
		SchemaVersion: 1,
		Scenario:      Scenario{Name: "f3-test"},
		Findings: []Finding{{
			ID: "F3-NO-SHOTS", Kind: "bug", Severity: "high", Title: "no shots", Status: "new",
		}},
		Scorecard: []ScorecardItem{{
			Dimension: "任务", Metric: "status", Target: "success", Status: "red",
		}},
		Metrics: []MetricRow{
			{Group: "损失", Name: "own_sunk", Value: 10, Unit: "ships"},
			{Group: "损失", Name: "enemy_sunk", Value: 5, Unit: "ships"},
		},
	}
	data, err := json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "result.json"), data, 0o644))
	r2Dir := filepath.Join(dir, "runs", "r2")
	require.NoError(t, os.MkdirAll(r2Dir, 0o755))
	r2 := Result{
		SchemaVersion: 1, Scenario: Scenario{Name: "f3-test"},
		Metrics: []MetricRow{
			{Group: "损失", Name: "own_sunk", Value: 20, Unit: "ships"},
			{Group: "损失", Name: "enemy_sunk", Value: 7, Unit: "ships"},
		},
	}
	r2Data, err := json.Marshal(r2)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(r2Dir, "result.json"), r2Data, 0o644))
	history := `{"runId":"r1","startedAt":"2026-11-19 10:00:00","fidelity":"F3","scorecard":{"green":1,"red":1},"findings":{"new":1},"runtimeMs":10,"report":"runs/r1/report.html"}
{"runId":"r2","startedAt":"2026-11-19 11:00:00","fidelity":"F3","scorecard":{"green":1},"findings":{},"runtimeMs":10,"report":"runs/r2/report.html"}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "history.jsonl"), []byte(history), 0o644))
	frameEntry := NewFrameHistoryEntry(
		FrameReport{Frames: 60, P99MS: 12.5, Budget: 16.7, Pass: true},
		"abc", false, time.Date(2026, 11, 20, 6, 0, 0, 0, time.UTC),
	)
	require.NoError(t, AppendFrameHistory(dir, frameEntry))
	out, err := WriteAnalysisBrief(Paths{Reports: dir}, 10)
	require.NoError(t, err)
	var brief AnalysisBrief
	payload, err := os.ReadFile(out.JSON)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(payload, &brief))
	require.Equal(t, 2, brief.Runs)
	require.Len(t, brief.RecurringFindings, 1)
	require.Equal(t, "F3-NO-SHOTS", brief.RecurringFindings[0].ID)
	require.True(t, brief.RecurringFindings[0].Fixed)
	require.Len(t, brief.RedScorecards, 1)
	require.Equal(t, "status", brief.RedScorecards[0].Metric)
	require.Len(t, brief.ScenarioSummaries, 1)
	require.Equal(t, "f3-test", brief.ScenarioSummaries[0].Scenario)
	summary := brief.ScenarioSummaries[0]
	require.Equal(t, 2, summary.Runs)
	require.InDelta(t, 15, summary.MeanOwnSunk, 1e-9)
	require.InDelta(t, 10, summary.OwnSunkMin, 1e-9)
	require.InDelta(t, 20, summary.OwnSunkMax, 1e-9)
	require.InDelta(t, 6, summary.MeanEnemySunk, 1e-9)
	require.InDelta(t, 5, summary.EnemySunkMin, 1e-9)
	require.InDelta(t, 7, summary.EnemySunkMax, 1e-9)
	require.NotEmpty(t, brief.Prompt)
	md, err := os.ReadFile(out.Markdown)
	require.NoError(t, err)
	require.Contains(t, string(md), "F3-NO-SHOTS")
	require.Contains(t, string(md), "样本不足（runs < 3）：f3-test")
	require.Contains(t, string(md), "latest p99 12.500 ms")
	require.Contains(t, string(md), "## 渲染帧门禁")
	require.Equal(t, 1, brief.Frame.Samples)
	require.InDelta(t, 12.5, brief.Frame.LatestP99MS, 1e-9)
	require.True(t, brief.Frame.LatestPass)
}
