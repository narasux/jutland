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

// TestWriteTrendBuildsHistoryViews 验证 history.jsonl 能生成 trend.html/trend.md/ai/trend.json。
// 场景：两条 run 记录，其中一条乱码行；绿色/红色/新 finding/耗时各不同。
// 预期：LoadHistory 跳过乱码并按 StartedAt 升序；BuildTrend 汇总 green/red/new；WriteTrend 三份文件存在且 HTML 内嵌 SVG。
// 覆盖：临时 reports 目录写 JSONL，调用 LoadHistory/BuildTrend/WriteTrend，断言汇总和文件内容。
func TestWriteTrendBuildsHistoryViews(t *testing.T) {
	dir := t.TempDir()
	history := `{"runId":"r1","startedAt":"2026-11-19 10:00:00","fidelity":"F1","scorecard":{"green":3,"red":1},"findings":{"new":2},"runtimeMs":10}
not-json
{"runId":"r2","startedAt":"2026-11-19 11:00:00","fidelity":"F3","scorecard":{"green":5,"red":0},"findings":{"new":1},"runtimeMs":30}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "history.jsonl"), []byte(history), 0o644))
	entries, err := LoadHistory(dir)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "r1", entries[0].RunID)
	data := BuildTrend(entries)
	require.Equal(t, 2, data.Summary.Runs)
	require.Equal(t, 8, data.Summary.Green)
	require.Equal(t, 1, data.Summary.Red)
	require.Equal(t, 3, data.Summary.NewFindings)
	require.InDelta(t, 20, data.Summary.AvgRuntime, 1e-9)
	out, err := WriteTrend(Paths{Reports: dir}, 10)
	require.NoError(t, err)
	for _, path := range []string{out.HTML, out.MD, out.AIJSON} {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.NotZero(t, info.Size())
	}
	html, err := os.ReadFile(out.HTML)
	require.NoError(t, err)
	require.True(t, strings.Contains(string(html), "<svg"))
	require.False(t, strings.Contains(string(html), "&lt;svg"))
	md, err := os.ReadFile(out.MD)
	require.NoError(t, err)
	require.Contains(t, string(md), "runs: 2")
}

// TestWriteTrendIncludesFrameGate 验证 frame/history.jsonl 会同时进入 trend.md/html/ai JSON。
// 场景：reports 目录既有一次仿真 run，又有一条渲染帧门禁记录（p99 15ms、预算 16.7ms 通过）；
// 定期运行需要把仿真趋势与真实渲染帧门禁放在同一份 trend 里看。
// 预期：trend.md 含 "frame gate: latest p99 15.000 ms (pass"，trend.html 含「渲染帧门禁」，
// ai/trend.json 的 frame.samples=1、latestP99Ms=15、latestPass=true。
// 覆盖：写 history.jsonl + AppendFrameHistory 后调用 WriteTrend，分别读 md/html/json 断言。
func TestWriteTrendIncludesFrameGate(t *testing.T) {
	dir := t.TempDir()
	history := `{"runId":"r1","startedAt":"2026-11-19 10:00:00","fidelity":"F1","scorecard":{"green":3,"red":1},"findings":{"new":0},"runtimeMs":10}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "history.jsonl"), []byte(history), 0o644))
	entry := NewFrameHistoryEntry(
		FrameReport{Frames: 60, P99MS: 15, Budget: 16.7, Pass: true},
		"abc", false, time.Date(2026, 11, 20, 6, 0, 0, 0, time.UTC),
	)
	require.NoError(t, AppendFrameHistory(dir, entry))

	out, err := WriteTrend(Paths{Reports: dir}, 10)
	require.NoError(t, err)
	md, err := os.ReadFile(out.MD)
	require.NoError(t, err)
	require.Contains(t, string(md), "frame gate: latest p99 15.000 ms (pass")
	html, err := os.ReadFile(out.HTML)
	require.NoError(t, err)
	require.Contains(t, string(html), "渲染帧门禁")
	payload, err := os.ReadFile(out.AIJSON)
	require.NoError(t, err)
	var trend TrendData
	require.NoError(t, json.Unmarshal(payload, &trend))
	require.Equal(t, 1, trend.Frame.Samples)
	require.InDelta(t, 15, trend.Frame.LatestP99MS, 1e-9)
	require.True(t, trend.Frame.LatestPass)
}
