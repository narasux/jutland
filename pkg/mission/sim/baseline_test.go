package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestApplyBaselineAnnotatesAndFlagsDrift 验证基线把 Baseline/Delta 写回指标并只对超容差项报 finding。
// 场景：结果有 shots=100、hit_rate=50；基线 shots=100、hit_rate=60，容差 shots=5、hit_rate=9。
// 预期：两行都有 Baseline/Delta；shots 不报 finding；hit_rate 产生 medium BASELINE-DRIFT。
// 覆盖：临时 baseline JSON -> LoadBaseline -> ApplyBaseline，断言 matched、字段、finding 数与严重度。
func TestApplyBaselineAnnotatesAndFlagsDrift(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "b.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
		"schemaVersion":1,"name":"test","scenario":"s","fidelity":"F1","commit":"abc",
		"metrics":{"shots":100,"hit_rate":60},"tolerance":{"shots":5,"hit_rate":9}
	}`), 0o644))
	b, err := LoadBaseline(path)
	require.NoError(t, err)
	result := &Result{
		Fidelity: "F1",
		Scenario: Scenario{Name: "s"},
		Metrics: []MetricRow{
			{Name: "shots", Value: 100},
			{Name: "hit_rate", Value: 50},
			{Name: "other", Value: 1},
		},
	}
	matched, err := ApplyBaseline(result, b)
	require.NoError(t, err)
	require.Equal(t, 2, matched)
	require.Equal(t, 100.0, result.Metrics[0].Baseline)
	require.InDelta(t, 0, result.Metrics[0].Delta, 1e-9)
	require.Equal(t, 60.0, result.Metrics[1].Baseline)
	require.InDelta(t, -10, result.Metrics[1].Delta, 1e-9)
	require.Len(t, result.Findings, 1)
	require.Equal(t, "BASELINE-DRIFT-hit_rate", result.Findings[0].ID)
	require.Equal(t, "medium", result.Findings[0].Severity)
	require.Equal(t, "baseline: test，matched=2，commit=abc。", result.DataQuality[0])
}

// TestWriteBaselineFromRunSnapshotsMetrics 验证从 result.json 生成可重新加载的 baseline。
// 场景：runs/r1/result.json 含 applied_damage=40、kill_rate=100；用 tolerance 覆盖两个指标。
// 预期：baseline 写入 baselines/f0-auto.json，LoadBaseline 后 metrics/tolerance/scenario 正确。
// 覆盖：临时 Reports 路径 -> WriteBaselineFromRun -> LoadBaseline；同时验证 ParseTolerances。
func TestWriteBaselineFromRunSnapshotsMetrics(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "runs", "r1")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	result := Result{
		Fidelity: "F0",
		Scenario: Scenario{Name: "f0-test"},
		Metrics: []MetricRow{
			{Name: "applied_damage", Value: 40, Unit: "HP"},
			{Name: "kill_rate", Value: 100, Unit: "%"},
		},
	}
	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "result.json"), payload, 0o644))
	paths := Paths{Reports: dir, Baselines: filepath.Join(dir, "baselines")}
	tolerances, err := ParseTolerances("applied_damage=1,kill_rate=1")
	require.NoError(t, err)
	path, err := WriteBaselineFromRun(paths, "r1", "f0-auto", tolerances)
	require.NoError(t, err)
	b, err := LoadBaseline(path)
	require.NoError(t, err)
	require.Equal(t, "f0-auto", b.Name)
	require.Equal(t, "f0-test", b.Scenario)
	require.Equal(t, 40.0, b.Metrics["applied_damage"])
	require.Equal(t, 1.0, b.Tolerance["kill_rate"])
}
