package sim

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunDiagnosticsFlagsInvariants 验证诊断规则能抓非有限/负值/命中矛盾/友伤/预算。
// 场景：shots=10、hits=11、hit_rate=10、friendly=1、p99=20、kill_rate=0，另加 NaN 指标。
// 预期：HITS_GT_SHOTS、HIT_RATE_MISMATCH、FRIENDLY_FIRE、P99_OVER_BUDGET、ZERO_KILL_RATE、NONFINITE 均出现。
// 覆盖：RunDiagnostics 一次多规则；断言 ID 集合，不依赖顺序。
func TestRunDiagnosticsFlagsInvariants(t *testing.T) {
	result := &Result{
		Scenario: Scenario{Mode: ModeDuel},
		Metrics: []MetricRow{
			{Name: "shots", Value: 10},
			{Name: "hits", Value: 11},
			{Name: "hit_rate", Value: 10},
			{Name: "friendly_hits", Value: 1},
			{Name: "tick_p99_ms", Value: 20},
			{Name: "kill_rate", Value: 0},
			{Name: "bad", Value: math.NaN()},
		},
	}
	diags := RunDiagnostics(result)
	ids := map[string]bool{}
	for _, diag := range diags {
		ids[diag.ID] = true
	}
	for _, id := range []string{
		"DIAG-HITS-GT-SHOTS", "DIAG-HIT-RATE-MISMATCH", "DIAG-FRIENDLY-FIRE",
		"DIAG-P99-OVER-BUDGET", "DIAG-ZERO-KILL-RATE", "DIAG-NONFINITE-bad",
	} {
		require.True(t, ids[id], "missing %s in %v", id, ids)
	}
}

// TestRunDiagnosticsCleanResult 验证正常结果不产生误报。
// 场景：10 发 5 中、hit_rate=50、friendly=0、p99=1、kill_rate=100。
// 预期：Diagnostics 为空。
// 覆盖：RunDiagnostics 的干净分支。
func TestRunDiagnosticsCleanResult(t *testing.T) {
	result := &Result{
		Scenario: Scenario{Mode: ModeDuel},
		Metrics: []MetricRow{
			{Name: "shots", Value: 10},
			{Name: "hits", Value: 5},
			{Name: "hit_rate", Value: 50},
			{Name: "friendly_hits", Value: 0},
			{Name: "tick_p99_ms", Value: 1},
			{Name: "kill_rate", Value: 100},
		},
	}
	require.Empty(t, RunDiagnostics(result))
}

// TestBuildSummaryIncludesDiagnostics 验证 Markdown 摘要包含诊断段。
// 场景：Result 带一条 DIAG-ZERO-SHOTS，buildSummary 生成人类摘要。
// 预期：summary 含 "## 诊断" 和诊断 ID；失败说明诊断只进 JSON 不进摘要。
// 覆盖：buildSummary 的 Diagnostics 分支。
func TestBuildSummaryIncludesDiagnostics(t *testing.T) {
	result := &Result{
		Scenario: Scenario{Name: "s"},
		Diagnostics: []Diagnostic{{
			ID: "DIAG-ZERO-SHOTS", Severity: "high", Message: "no shots",
		}},
	}
	summary := buildSummary(result, Manifest{})
	require.Contains(t, summary, "## 诊断")
	require.Contains(t, summary, "DIAG-ZERO-SHOTS")
}
