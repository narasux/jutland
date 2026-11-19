package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWriteAnalysisBriefAggregatesHistory 验证跨 run 分析简报聚合 finding 与红色指标。
// 场景：r1 含 high finding + red scorecard；r2 同场景但无 finding，用来判断 r1 finding 已修复。
// 预期：RecurringFindings/RedScorecards 各 1；F3-NO-SHOTS 的 Fixed=true；Prompt 非空，md 含 finding ID。
// 覆盖：临时 reports 目录写两条 history 与 r1/r2 result.json，验证按场景最新 run 判定 fixed。
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
	r2Dir := filepath.Join(dir, "runs", "r2")
	require.NoError(t, os.MkdirAll(r2Dir, 0o755))
	r2 := Result{SchemaVersion: 1, Scenario: Scenario{Name: "f3-test"}}
	r2Data, err := json.Marshal(r2)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(r2Dir, "result.json"), r2Data, 0o644))
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
