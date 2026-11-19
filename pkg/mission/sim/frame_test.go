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
