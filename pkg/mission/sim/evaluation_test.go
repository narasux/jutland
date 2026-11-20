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
	_, out, err := WriteEvaluation(Paths{Reports: dir}, summaryPath)
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

// TestEvaluationGateError 验证评测门禁只拦「缺 run」和「smoke 红」，balance 红放行。
// 场景：CI 里跑完整 suite 后需要明确的退出码；缺 run 表示报告链断了，
// smoke 红（F2 合同作战/冒烟）表示引擎或场景真的坏了，都必须失败；
// balance 红是「可能需要重新平衡」的信号，不应阻断提交流程。
// 预期：干净报告返回 nil；Missing 非空返回含 missing 的错误；SmokeRed>0 返回含 smoke red 的错误；
// BugFindings>0 返回含 bug finding 的错误；只有 BalanceRed/BalanceFindings 时返回 nil。
// 覆盖：表驱动 5 个 EvaluationReport，调用 EvaluationGateError 断言错误存在性与关键字。
func TestEvaluationGateError(t *testing.T) {
	cases := []struct {
		name    string
		report  EvaluationReport
		wantErr string
	}{
		{name: "clean", report: EvaluationReport{Status: map[string]int{"green": 6}}},
		{name: "missing", report: EvaluationReport{Missing: []string{"r9"}}, wantErr: "missing"},
		{name: "smoke red", report: EvaluationReport{SmokeRed: 2}, wantErr: "smoke red"},
		{name: "bug finding", report: EvaluationReport{BugFindings: 1}, wantErr: "bug finding"},
		{name: "balance only", report: EvaluationReport{BalanceRed: 3, BalanceFindings: 2, DataFindings: 4}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := EvaluationGateError(tc.report)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// TestWriteEvaluationIncludesPenetrationMatrix 验证穿甲场景会汇总进评测的穿甲矩阵。
// 场景：suite 里包含一个 F0 penetration 场景（穿深 305mm、参考比值 2.03、最大可穿 400mm、9 个 AP 点），
// 这类 run 没有 shots/hit%，如果只留在 runs 表里会全是横杠，定稿穿甲数值时无法横向比较。
// 预期：EvaluationReport.Penetration 有 1 行且字段与 metrics 一致；
// Markdown 含「穿甲矩阵」表头与数值，HTML 同样包含矩阵小节。
// 覆盖：临时 reports 目录写 penetration run 的 result.json + suite summary，调用 WriteEvaluation 后断言 JSON/md/html。
func TestWriteEvaluationIncludesPenetrationMatrix(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "runs", "p1")
	require.NoError(t, os.MkdirAll(runDir, 0o755))
	result := Result{
		SchemaVersion: 1,
		Scenario:      Scenario{Name: "pen-203", Mode: ModePenetration},
		Metrics: []MetricRow{
			{Group: "穿甲", Name: "penetration_mm", Value: 305},
			{Group: "穿甲", Name: "reference_armor", Value: 150},
			{Group: "穿甲", Name: "reference_ratio", Value: 2.033},
			{Group: "矩阵", Name: "max_penetrated_armor", Value: 400},
			{Group: "矩阵", Name: "ap_penetrated", Value: 9},
		},
	}
	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "result.json"), payload, 0o644))

	suite := SuiteSummary{Name: "pen", Runs: []SuiteRunResult{{RunID: "p1"}}}
	suitePayload, err := json.Marshal(suite)
	require.NoError(t, err)
	summaryPath := filepath.Join(dir, "suites", "pen.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(summaryPath), 0o755))
	require.NoError(t, os.WriteFile(summaryPath, suitePayload, 0o644))

	report, out, err := WriteEvaluation(Paths{Reports: dir}, summaryPath)
	require.NoError(t, err)
	require.Len(t, report.Penetration, 1)
	pen := report.Penetration[0]
	require.Equal(t, "pen-203", pen.Scenario)
	require.InDelta(t, 305, pen.PenetrationMM, 1e-9)
	require.InDelta(t, 150, pen.ReferenceArmor, 1e-9)
	require.InDelta(t, 400, pen.MaxPenetratedArmor, 1e-9)
	require.Equal(t, 9, pen.APPenetrated)

	md, err := os.ReadFile(filepath.Join(dir, "suites", "pen-evaluation.md"))
	require.NoError(t, err)
	require.Contains(t, string(md), "## 穿甲矩阵")
	require.Contains(t, string(md), "| pen-203 | 305.0 | 150.0 | 2.03 | 400.0 | 9 |")
	html, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Contains(t, string(html), "穿甲矩阵")
}
