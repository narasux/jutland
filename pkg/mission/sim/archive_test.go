package sim

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestArchiveRunsKeepsNewestAndTrimsHistory 验证 archive 保留最新 run、移动旧 run 并裁剪 history。
// 场景：runs/r1..r4，keep=2；history 四条；dry-run 首次不改盘。
// 预期：r3/r4 移到 archive/<ts>/runs，r1/r2 保留；reports/history 只剩 r1/r2，archive history 有 r3/r4。
// 覆盖：ArchiveRuns dry-run + 实际执行，断言目录、history 行数与 manifest。
func TestArchiveRunsKeepsNewestAndTrimsHistory(t *testing.T) {
	dir := t.TempDir()
	reports := filepath.Join(dir, "reports")
	for _, run := range []string{"r1", "r2", "r3", "r4"} {
		require.NoError(t, os.MkdirAll(filepath.Join(reports, "runs", run), 0o755))
	}
	history := `{"runId":"r1","startedAt":"2026-11-19 10:00:00"}
{"runId":"r2","startedAt":"2026-11-19 11:00:00"}
{"runId":"r3","startedAt":"2026-11-19 12:00:00"}
{"runId":"r4","startedAt":"2026-11-19 13:00:00"}
`
	require.NoError(t, os.WriteFile(filepath.Join(reports, "history.jsonl"), []byte(history), 0o644))
	paths := Paths{Reports: reports}
	dry, err := ArchiveRuns(paths, ArchiveOptions{Keep: 2, DryRun: true})
	require.NoError(t, err)
	require.Equal(t, []string{"r4", "r3"}, dry.Kept)
	require.Equal(t, []string{"r2", "r1"}, dry.Moved)
	require.DirExists(t, filepath.Join(reports, "runs", "r4"))

	result, err := ArchiveRuns(paths, ArchiveOptions{Keep: 2})
	require.NoError(t, err)
	require.Equal(t, []string{"r2", "r1"}, result.Moved)
	require.DirExists(t, filepath.Join(result.ArchiveDir, "runs", "r2"))
	require.NoDirExists(t, filepath.Join(reports, "runs", "r2"))
	kept, err := LoadHistory(reports)
	require.NoError(t, err)
	require.Len(t, kept, 2)
	archived, err := LoadHistory(result.ArchiveDir)
	require.NoError(t, err)
	require.Len(t, archived, 2)
	require.FileExists(t, filepath.Join(result.ArchiveDir, "manifest.json"))
}
