package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestWarmupMapBlocksReportsCurrentViewReadiness 验证真实地图下预热门只有在当前视野全部就绪后才放行。
// 场景：New("Midway1942", SideP1) 载入中途岛任务，把相机放到 (100,137)、视野 30x20，
// 首次检查时地图块缓存必然还有缺口；随后最多轮询 100 帧，模拟开局前逐帧预热的等待。
// 预期：首次 WarmupMapBlocks 返回 false；100 帧内一旦返回 true 即保持 true（ready 为真且再次调用仍为真）。
// 失败分别意味着开局门永远不放行（卡在预热），或者中途反复 true/false 抖动导致战斗提前开始。
// 覆盖：真实任务管理器上调用 WarmupMapBlocks，记录首次结果与循环找到的就绪状态。
func TestWarmupMapBlocksReportsCurrentViewReadiness(t *testing.T) {
	m := New("Midway1942", faction.SideP1)
	m.state.View.Camera.Pos = objPos.New(100, 137)
	m.state.View.Camera.Width = 30
	m.state.View.Camera.Height = 20

	require.False(t, m.WarmupMapBlocks())

	ready := false
	for range 100 {
		if m.WarmupMapBlocks() {
			ready = true
			break
		}
	}
	require.True(t, ready)
	require.True(t, m.WarmupMapBlocks())
}
