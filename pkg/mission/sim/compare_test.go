package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWriteComparisonDiffsRuns 验证两个 run 的指标、scorecard、finding 差异。
// 场景：A shots=100/red/D1，B shots=120/green/D2；B 新出现 F-NEW，A 的 F-GONE 消失。
// 预期：shots delta=20、score change 一条、FindingsAdded/FindingsGone 各一条；md/json 生成。
// 覆盖：临时 runs/r1、r2 写 result.json，调用 WriteComparison 断言 JSON 与 Markdown。
func TestWriteComparisonDiffsRuns(t *testing.T) {
	dir := t.TempDir()
	writeResult := func(run string, result Result) {
		runDir := filepath.Join(dir, "runs", run)
		require.NoError(t, os.MkdirAll(runDir, 0o755))
		payload, err := json.Marshal(result)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(runDir, "result.json"), payload, 0o644))
	}
	writeResult("r1", Result{
		Scenario:  Scenario{Name: "s"},
		Metrics:   []MetricRow{{Group: "命中", Name: "shots", Value: 100, Unit: "shots"}},
		Findings:  []Finding{{ID: "F-GONE", Title: "gone", Severity: "low", Status: "new"}},
		Scorecard: []ScorecardItem{{Dimension: "任务", Metric: "status", Status: "red"}},
	})
	writeResult("r2", Result{
		Scenario:  Scenario{Name: "s"},
		Metrics:   []MetricRow{{Group: "命中", Name: "shots", Value: 120, Unit: "shots"}},
		Findings:  []Finding{{ID: "F-NEW", Title: "new", Severity: "high", Status: "new"}},
		Scorecard: []ScorecardItem{{Dimension: "任务", Metric: "status", Status: "green"}},
	})
	out, err := WriteComparison(Paths{Reports: dir}, "r1", "r2")
	require.NoError(t, err)
	var report CompareReport
	payload, err := os.ReadFile(out.JSON)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(payload, &report))
	require.Len(t, report.Metrics, 1)
	require.Equal(t, 20.0, report.Metrics[0].Delta)
	require.Len(t, report.FindingsAdded, 1)
	require.Equal(t, "F-NEW", report.FindingsAdded[0].ID)
	require.Len(t, report.FindingsGone, 1)
	require.Equal(t, "F-GONE", report.FindingsGone[0].ID)
	require.Len(t, report.ScoreChanges, 1)
	md, err := os.ReadFile(out.Markdown)
	require.NoError(t, err)
	require.Contains(t, string(md), "F-NEW")
}
