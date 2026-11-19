package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWriteEvaluationAggregatesSuite 验证 suite summary 能聚合成跨 fidelity 评估页。
// 场景：r1 为 F1，scorecard 1 green + 1 red，1 finding，1 diagnostic，shots=10；suite summary 引用 r1。
// 预期：生成 -evaluation.html/md/json；HTML/MD 含 r1、green=1、red=1，JSON Runs=1。
// 覆盖：WriteEvaluation 的加载、状态计数、指标映射、HTML/Markdown/JSON 输出。
func TestWriteEvaluationAggregatesSuite(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "runs", "r1")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	result := Result{
		Scenario: Scenario{Name: "s", Mode: ModeDuel}, Fidelity: "F1",
		Scorecard: []ScorecardItem{
			{Dimension: "d", Metric: "m", Status: "green"},
			{Dimension: "d", Metric: "n", Status: "red"},
		},
		Findings:    []Finding{{ID: "f"}},
		Diagnostics: []Diagnostic{{ID: "d"}},
		Metrics:     []MetricRow{{Name: "shots", Value: 10}},
	}
	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "result.json"), payload, 0o644))
	suite := SuiteSummary{Name: "full", Runs: []SuiteRunResult{{RunID: "r1"}}}
	suitePayload, err := json.Marshal(suite)
	require.NoError(t, err)
	summaryPath := filepath.Join(dir, "suites", "full.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(summaryPath), 0o755))
	require.NoError(t, os.WriteFile(summaryPath, suitePayload, 0o644))
	out, err := WriteEvaluation(Paths{Reports: dir}, summaryPath)
	require.NoError(t, err)
	html, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Contains(t, string(html), "r1")
	require.Contains(t, string(html), "1")
	md, err := os.ReadFile(filepath.Join(dir, "suites", "full-evaluation.md"))
	require.NoError(t, err)
	require.Contains(t, string(md), "green: 1")
	var report EvaluationReport
	raw, err := os.ReadFile(filepath.Join(dir, "suites", "full-evaluation.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &report))
	require.Equal(t, 1, report.Runs)
}
