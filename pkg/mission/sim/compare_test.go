package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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

// TestWriteComparisonToPreviousUsesHistory 验证单 run compare 会找同 scenario 的上一条历史。
// 场景：r1 10:00、r2 11:00，均为 scenario=s；history 两条；只传 r2。
// 预期：生成 compare/r1__r2.md，A/B runId 正确；找不到前序时返回错误。
// 覆盖：LoadHistory + 同 scenario 回溯 + WriteComparison。
func TestWriteComparisonToPreviousUsesHistory(t *testing.T) {
	dir := t.TempDir()
	writeRun := func(run, started string, value float64) {
		runDir := filepath.Join(dir, "runs", run)
		require.NoError(t, os.MkdirAll(runDir, 0o755))
		result := Result{
			Scenario:  Scenario{Name: "s"},
			StartedAt: mustTime(t, started),
			Metrics:   []MetricRow{{Name: "shots", Value: value}},
		}
		payload, err := json.Marshal(result)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(runDir, "result.json"), payload, 0o644))
	}
	writeRun("r1", "2026-11-19T10:00:00+08:00", 10)
	writeRun("r2", "2026-11-19T11:00:00+08:00", 20)
	history := `{"runId":"r1","startedAt":"2026-11-19 10:00:00"}
{"runId":"r2","startedAt":"2026-11-19 11:00:00"}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "history.jsonl"), []byte(history), 0o644))
	out, err := WriteComparisonToPrevious(Paths{Reports: dir}, "r2")
	require.NoError(t, err)
	require.Contains(t, out.Markdown, "r1__r2")
	_, err = WriteComparisonToPrevious(Paths{Reports: dir}, "r1")
	require.Error(t, err)
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed
}
