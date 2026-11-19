package sim

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestBuildFrameReportPercentiles 验证渲染帧 p50/p95/p99/max 与预算判定。
// 场景：100 帧，1..100ms 递增；预算 99ms 应通过，98ms 应失败。
// 预期：p50≈50、p95=95、p99=99、max=100；WriteFrameReport 写出合法 JSON。
// 覆盖：BuildFrameReport 两种预算 + WriteFrameReport 临时文件。
func TestBuildFrameReportPercentiles(t *testing.T) {
	times := make([]time.Duration, 100)
	for i := range times {
		times[i] = time.Duration(i+1) * time.Millisecond
	}
	pass := BuildFrameReport(times, 99)
	require.Equal(t, 100, pass.Frames)
	require.InDelta(t, 50, pass.P50MS, 0.001)
	require.InDelta(t, 95, pass.P95MS, 0.001)
	require.InDelta(t, 99, pass.P99MS, 0.001)
	require.InDelta(t, 100, pass.MaxMS, 0.001)
	require.True(t, pass.Pass)
	fail := BuildFrameReport(times, 98)
	require.False(t, fail.Pass)
	path := filepath.Join(t.TempDir(), "frame.json")
	require.NoError(t, WriteFrameReport(path, pass))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.NotZero(t, info.Size())
}

// TestFrameHistoryRoundTrip 验证渲染帧门禁结果能按时间顺序追加并读回，坏行被跳过。
// 场景：同一次 lab 里跑两次门禁（pass 与 fail），中间夹一条手工坏行；
// trend 需要按时间升序读取，不能被单行损坏中断。
// 预期：AppendFrameHistory 写出 frame/history.jsonl；Load 返回 2 条且按 RecordedAt 升序，
// 第二条 Pass=false、P99MS=20，Commit/Dirty 原样保留。
// 覆盖：NewFrameHistoryEntry 转换 FrameReport，临时目录 append 两条 + 一段乱码，Load 后断言顺序与字段。
func TestFrameHistoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	first := NewFrameHistoryEntry(
		FrameReport{Frames: 60, P50MS: 1, P95MS: 3, P99MS: 4, MaxMS: 6, Budget: 16.7, Pass: true},
		"abc1234", true, time.Date(2026, 11, 20, 6, 0, 0, 0, time.UTC),
	)
	second := NewFrameHistoryEntry(
		FrameReport{Frames: 120, P50MS: 2, P95MS: 9, P99MS: 20, MaxMS: 30, Budget: 16.7, Pass: false},
		"def5678", false, time.Date(2026, 11, 20, 7, 0, 0, 0, time.UTC),
	)
	require.NoError(t, AppendFrameHistory(dir, second))
	require.NoError(t, AppendFrameHistory(dir, first))
	path := FrameHistoryPath(dir)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, append(raw, []byte("not-json\n")...), 0o644))

	entries, err := LoadFrameHistory(dir)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "abc1234", entries[0].Commit)
	require.True(t, entries[0].Dirty)
	require.Equal(t, "def5678", entries[1].Commit)
	require.False(t, entries[1].Pass)
	require.InDelta(t, 20, entries[1].P99MS, 1e-9)
}

// TestBuildFrameTrendAggregates 验证渲染帧 p99 历史能汇总成趋势摘要。
// 场景：三次门禁 p99 分别 10/20/30ms，预算 16.7ms，最新一次超预算；
// trend 卡片需要最新值、均值、最差值和通过状态。
// 预期：samples=3、latest=30 且 LatestPass=false、avg=20、worst=30、budget=16.7；空输入返回零值。
// 覆盖：直接构造 FrameHistoryEntry 切片调用 BuildFrameTrend，断言四个统计字段与空输入分支。
func TestBuildFrameTrendAggregates(t *testing.T) {
	entries := []FrameHistoryEntry{
		{RecordedAt: time.Date(2026, 11, 20, 6, 0, 0, 0, time.UTC), P99MS: 10, BudgetMS: 16.7, Pass: true},
		{RecordedAt: time.Date(2026, 11, 20, 7, 0, 0, 0, time.UTC), P99MS: 20, BudgetMS: 16.7, Pass: false},
		{RecordedAt: time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC), P99MS: 30, BudgetMS: 16.7, Pass: false},
	}
	trend := BuildFrameTrend(entries)
	require.Equal(t, 3, trend.Samples)
	require.InDelta(t, 30, trend.LatestP99MS, 1e-9)
	require.False(t, trend.LatestPass)
	require.InDelta(t, 20, trend.AvgP99MS, 1e-9)
	require.InDelta(t, 30, trend.WorstP99MS, 1e-9)
	require.InDelta(t, 16.7, trend.BudgetMS, 1e-9)

	empty := BuildFrameTrend(nil)
	require.Zero(t, empty.Samples)
}
