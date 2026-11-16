package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAllMissionsReturnsOrderedCopy 验证 AllMissions 按配置顺序返回副本，外部修改不污染全局顺序。
// 场景：菜单/调试工具遍历全部任务；若直接返回内部切片，调用方排序或改名会改变后续菜单顺序。
// 预期：返回非空、首关为 PearlHarbor1941；修改返回值后再次调用顺序不变。
// 覆盖：调用 AllMissions 两次，比较首关；改第一个元素后重新取并断言已恢复。
func TestAllMissionsReturnsOrderedCopy(t *testing.T) {
	missions := AllMissions()
	require.NotEmpty(t, missions)
	require.Equal(t, "PearlHarbor1941", missions[0])

	originalFirst := missions[0]
	missions[0] = "mutated"
	require.Equal(t, "mutated", missions[0])

	fresh := AllMissions()
	require.Equal(t, originalFirst, fresh[0])
}
