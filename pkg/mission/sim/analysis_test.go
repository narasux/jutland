package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWriteAnalysisBriefAggregatesHistory 验证跨 run 分析简报聚合 finding 与红色指标。
// 场景：history 中一条 run，result.json 含一个 high finding 与一个 red scorecard；另一条 run 缺失 result 也要跳过。
// 预期：生成 analysis-brief.md/json；JSON 中 RecurringFindings/RedScorecards 各 1，Prompt 非空，md 含 finding ID。
// 覆盖：临时 reports 目录写 history.jsonl + runs/r1/result.json，调用 WriteAnalysisBrief 断言产物与聚合。
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
	}
	data, err := json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "result.json"), data, 0o644))
	history := `{"runId":"r1","startedAt":"2026-11-19 10:00:00","fidelity":"F3","scorecard":{"green":1,"red":1},"findings":{"new":1},"runtimeMs":10,"report":"runs/r1/report.html"}
{"runId":"r2","startedAt":"2026-11-19 11:00:00","fidelity":"F3","scorecard":{"green":1},"findings":{},"runtimeMs":10,"report":"runs/r2/report.html"}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "history.jsonl"), []byte(history), 0o644))
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
	require.NotEmpty(t, brief.Prompt)
	md, err := os.ReadFile(out.Markdown)
	require.NoError(t, err)
	require.Contains(t, string(md), "F3-NO-SHOTS")
}
