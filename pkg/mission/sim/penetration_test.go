package sim

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestRunPenetrationBuildsMatrix 验证 F0 穿深矩阵的参考点、图表和 finding。
// 场景：testdata/penetration-127mm.json5，120mm 穿深、20..200mm 装甲、0°/60°、10 步。
// 预期：Fidelity=F0、reference_ratio=1.2、10 个 chart 点、AP/HE 穿透计数 6/3、无 finding。
// 覆盖：LoadScenario -> RunPenetration，断言 metrics/charts/scorecard；不创建 manager。
func TestRunPenetrationBuildsMatrix(t *testing.T) {
	sc, err := LoadScenario(filepath.Join("testdata", "penetration-127mm.json5"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunPenetration(sc, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "F0", result.Fidelity)
	require.False(t, result.Mock)
	metrics := map[string]float64{}
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	require.InDelta(t, 1.2, metrics["reference_ratio"], 1e-9)
	require.Equal(t, 6.0, metrics["ap_penetrated"])
	require.Equal(t, 3.0, metrics["he_penetrated"])
	require.Len(t, result.Charts.Penetration, 10)
	require.Len(t, result.Charts.HitRateLabels, 10)
	require.Empty(t, result.Findings)
}
