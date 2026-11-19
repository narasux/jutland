package sim

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunWatchOnceRefreshesReports 验证 watch --once 能刷新 trend 与 analysis brief。
// 场景：空 reports 目录，Once=true、Limit=20；没有历史 run 也应生成空 trend/简报。
// 预期：trend.html、ai/analysis-brief.md/json 存在，Writer 输出 trend/brief 路径，ctx 未取消。
// 覆盖：temp Paths + bytes.Buffer，调用 RunWatch；断言文件与输出。
func TestRunWatchOnceRefreshesReports(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	err := RunWatch(context.Background(), Paths{Reports: dir}, WatchOptions{
		Once: true, Limit: 20, Writer: &out,
	})
	require.NoError(t, err)
	for _, path := range []string{
		filepath.Join(dir, "trend.html"),
		filepath.Join(dir, "ai", "analysis-brief.md"),
		filepath.Join(dir, "ai", "analysis-brief.json"),
	} {
		_, err := os.Stat(path)
		require.NoError(t, err)
	}
	require.Contains(t, out.String(), "trend:")
	require.Contains(t, out.String(), "brief:")
}
