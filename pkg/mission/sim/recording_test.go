package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSummarizeAndCompareRecording 验证游戏 JSONL 录制聚合与 sim run 对比。
// 场景：录制 1 shot/1 hit/1 sink，命中伤害 30、友伤 0；sim 指标 shots=2、hits=1、enemy_sunk=1。
// 预期：Summary shots=1、hits=1、Sinks=1、AppliedDamage=30；compare 输出 md/json 且 shots delta=+1。
// 覆盖：SummarizeRecording + WriteRecordingComparison 临时目录。
func TestSummarizeAndCompareRecording(t *testing.T) {
	dir := t.TempDir()
	recPath := filepath.Join(dir, "record.jsonl")
	lines := []string{
		`{"type":"shot","event":{"tick":10,"shooterUid":"HA/DD-1"}}`,
		`{"type":"hit","event":{"tick":11,"appliedDamage":30,"effectiveDamage":30}}`,
		`{"type":"sink","event":{"tick":12,"targetUid":"CA/DD-1"}}`,
	}
	var payload []byte
	for _, line := range lines {
		payload = append(payload, []byte(line+"\n")...)
	}
	require.NoError(t, os.WriteFile(recPath, payload, 0o644))
	summary, err := SummarizeRecording(recPath)
	require.NoError(t, err)
	require.Equal(t, 1, summary.Shots)
	require.Equal(t, 1, summary.Hits)
	require.Equal(t, 1, summary.Sinks)
	require.Equal(t, 30.0, summary.AppliedDamage)
	require.Equal(t, int64(10), summary.FirstTick)

	reports := filepath.Join(dir, "reports")
	runDir := filepath.Join(reports, "runs", "r1")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	result := Result{
		Scenario: Scenario{Name: "s"},
		Metrics: []MetricRow{
			{Name: "shots", Value: 2},
			{Name: "hits", Value: 1},
			{Name: "applied_damage", Value: 30},
			{Name: "enemy_sunk", Value: 1},
		},
	}
	resultPayload, err := json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "result.json"), resultPayload, 0o644))
	out, err := WriteRecordingComparison(Paths{Reports: reports}, recPath, "r1")
	require.NoError(t, err)
	var report CompareReport
	data, err := os.ReadFile(out.JSON)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &report))
	require.NotEmpty(t, report.Metrics)
	found := false
	for _, metric := range report.Metrics {
		if metric.Name == "shots" {
			require.Equal(t, 1.0, metric.Delta)
			found = true
		}
	}
	require.True(t, found)
}
