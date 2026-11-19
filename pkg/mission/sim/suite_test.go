package sim

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLoadSuiteDefaultsAndValidation 验证 suite JSON5 的默认值与模式校验。
// 场景：suite 含 shot/duel/macro 三步，runs 缺省；另写一个不支持的 mode。
// 预期：runs 默认 1，LoadSuite 成功；非法 mode 返回错误。
// 覆盖：json5 解析、ApplyDefaults、Validate 两条分支。
func TestLoadSuiteDefaultsAndValidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "suite.json5")
	body := `{
		name: "smoke",
		steps: [
			{ mode: "shot", scenario: "shot-f0", runs: 50 },
			{ mode: "duel", scenario: "duel-jiujiang", seeds: [1, 3] },
			{ mode: "macro", scenario: "macro-pearl", seeds: [1] },
		],
	}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	suite, err := LoadSuite(path)
	require.NoError(t, err)
	require.Equal(t, 3, len(suite.Steps))
	require.Equal(t, 50, suite.Steps[0].Runs)
	require.Equal(t, 1, suite.Steps[1].Runs)

	bad := filepath.Join(dir, "bad.json5")
	require.NoError(t, os.WriteFile(bad, []byte(`{name:"bad",steps:[{mode:"strike",scenario:"x"}]}`), 0o644))
	_, err = LoadSuite(bad)
	require.Error(t, err)
}
