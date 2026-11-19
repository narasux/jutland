package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestWriteReportEmbedsPreviousDiff 验证第二次同场景报告会嵌入与上次的 diff。
// 场景：同一 demo 场景先生成 10:00 run，再生成 11:00 run；history 只有前一条。
// 预期：第二次 summary 含“与上次对比”，result.json.previousDiff 非空且 ARun=第一次 runId。
// 覆盖：WriteReport→PreviousComparison→BuildComparison→模板/Markdown 嵌入。
func TestWriteReportEmbedsPreviousDiff(t *testing.T) {
	home := t.TempDir()
	paths := Paths{
		Home: home, Scenarios: filepath.Join(home, "scenarios"),
		Golden: filepath.Join(home, "golden"), Baselines: filepath.Join(home, "baselines"),
		Reports: filepath.Join(home, "reports"), Notes: filepath.Join(home, "notes"),
	}
	require.NoError(t, EnsureLayout(paths))
	sc := DefaultDemoScenario()
	firstAt := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	first, err := WriteReport(paths, GenerateDemoResult(&sc, firstAt), "built-in:diff", "first")
	require.NoError(t, err)
	second, err := WriteReport(paths, GenerateDemoResult(&sc, firstAt.Add(time.Hour)), "built-in:diff", "second")
	require.NoError(t, err)

	summary, err := os.ReadFile(second.Summary)
	require.NoError(t, err)
	require.Contains(t, string(summary), "与上次对比")
	html, err := os.ReadFile(second.ReportHTML)
	require.NoError(t, err)
	require.Contains(t, string(html), "与上次对比")

	var result Result
	payload, err := os.ReadFile(filepath.Join(second.Dir, "result.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(payload, &result))
	require.NotNil(t, result.PreviousDiff)
	require.Equal(t, first.RunID, result.PreviousDiff.ARun)
	require.True(t, strings.Contains(result.PreviousDiff.BRun, "this-run"))
}
