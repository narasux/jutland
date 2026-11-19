package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWriteLLMPackageExcludesHTML 验证 LLM package 只收 AI 文本/JSON，不收 HTML/CSV。
// 场景：run 目录有 result.json、ai/* 和 report.html；package 写入 reports/llm/r1。
// 预期：analysis-context/findings/metrics/diagnostics/changes/llm-prompt 存在，report.html 不存在，package.json 有 token 估算。
// 覆盖：WriteLLMPackage 的复制、prompt 兜底和 package manifest。
func TestWriteLLMPackageExcludesHTML(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "runs", "r1")
	require.NoError(t, os.MkdirAll(filepath.Join(runDir, "ai"), 0o755))
	result := Result{Scenario: Scenario{Name: "s"}, Fidelity: "F1"}
	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "result.json"), payload, 0o644))
	for _, name := range []string{"analysis-context.md", "findings.json", "metrics.json", "diagnostics.json", "changes.json", "llm-prompt.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(runDir, "ai", name), []byte("x"), 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "report.html"), []byte("html"), 0o644))
	out, err := WriteLLMPackage(Paths{Reports: dir}, "r1")
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(out, "analysis-context.md"))
	require.FileExists(t, filepath.Join(out, "package.json"))
	require.NoFileExists(t, filepath.Join(out, "report.html"))
	var pkg LLMPackage
	raw, err := os.ReadFile(filepath.Join(out, "package.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &pkg))
	require.Greater(t, pkg.TotalBytes, int64(0))
	require.NotZero(t, pkg.EstimatedTokens)
}
