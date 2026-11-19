package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCheckPerfGate 验证 p99 门禁读取 result.json 并按预算判定。
// 场景：runs/r1/result.json 的 tick_p99_ms=10；同一 run 用 16.7ms 应通过，用 5ms 应失败。
// 预期：Pass 字段随预算变化，reports/perf/latest.json 写入；缺 tick_p99_ms 的 run 返回错误。
// 覆盖：临时 reports 目录 -> CheckPerf 两次；再写一个 F1 result 验证缺指标错误。
func TestCheckPerfGate(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "runs", "r1")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	result := Result{Fidelity: "F3", Metrics: []MetricRow{{Name: "tick_p99_ms", Value: 10, Unit: "ms"}}}
	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "result.json"), payload, 0o644))
	paths := Paths{Reports: dir}
	pass, err := CheckPerf(paths, "r1", 16.7)
	require.NoError(t, err)
	require.True(t, pass.Pass)
	require.Equal(t, 10.0, pass.P99MS)
	require.FileExists(t, filepath.Join(dir, "perf", "latest.json"))
	fail, err := CheckPerf(paths, "r1", 5)
	require.NoError(t, err)
	require.False(t, fail.Pass)

	badDir := filepath.Join(dir, "runs", "r2")
	require.NoError(t, os.MkdirAll(badDir, 0o755))
	bad := Result{Fidelity: "F1", Metrics: []MetricRow{{Name: "shots", Value: 1}}}
	raw, err := json.Marshal(bad)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(badDir, "result.json"), raw, 0o644))
	_, err = CheckPerf(paths, "r2", 16.7)
	require.Error(t, err)
}
