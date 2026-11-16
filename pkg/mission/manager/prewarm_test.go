package manager

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/state"
	mapBlockImg "github.com/narasux/jutland/pkg/resources/images/mapblock"
)

// TestPrewarmSkipsSettledViewAndReportsGaps 验证地图块预热只在视野/缩放变化后扫描，并如实报告缺口。
// 场景：清空 SceneBlockCache 后，相机 4x2、Zoom=4 的空视野先预热一次；随后在 (1,1) 放入 8x8 底图块
// 但不动相机；最后清掉 mapBlockPrewarmSettled、把 prewarmZoom 归零、相机宽度扩到 40，并铺 30 块陆地。
// 预期：空视野返回 true 且队列为 0；相机未变时再次调用直接返回 true、不排任务，
// 因此“稳定后才补进来的” (1,1) 在 zoom 4 下仍取不到图；扩视野后返回 false 且队列非空。
// 失败分别意味着看门逻辑会反复全量扫描，或漏报未预热的陆地就放行开局。
// 覆盖：ebiten.NewImage(8,8) 造底图，直接操作 SceneBlockCache 的 PutBaseBlock/GetZoom/PrewarmQueueLen。
func TestPrewarmSkipsSettledViewAndReportsGaps(t *testing.T) {
	saved := mapBlockImg.SceneBlockCache
	t.Cleanup(func() {
		mapBlockImg.SceneBlockCache = saved
	})
	mapBlockImg.SceneBlockCache.Clear()

	manager := &MissionManager{
		state: &state.MissionState{
			View: state.MissionViewState{
				Camera: state.Camera{Width: 4, Height: 2},
			},
			UI: state.MissionUIState{
				GameOpts: state.GameOptions{Zoom: 4},
			},
		},
	}

	if !manager.WarmupMapBlocks() {
		t.Fatal("empty view should be ready")
	}
	if mapBlockImg.SceneBlockCache.PrewarmQueueLen() != 0 {
		t.Fatalf("queue = %d, want 0", mapBlockImg.SceneBlockCache.PrewarmQueueLen())
	}

	img := ebiten.NewImage(8, 8)
	mapBlockImg.SceneBlockCache.PutBaseBlock(1, 1, img)
	if !manager.WarmupMapBlocks() {
		t.Fatal("settled camera should return without scanning")
	}
	if mapBlockImg.SceneBlockCache.PrewarmQueueLen() != 0 {
		t.Fatal("settled prewarm enqueued work")
	}
	if mapBlockImg.SceneBlockCache.GetZoom(1, 1, 4) != nil {
		t.Fatal("land added after settle was warmed anyway")
	}

	manager.mapBlockPrewarmSettled = false
	manager.mapBlockPrewarmZoom = 0
	manager.state.View.Camera.Width = 40
	for x := 0; x < 30; x++ {
		mapBlockImg.SceneBlockCache.PutBaseBlock(x, 0, img)
	}
	if manager.WarmupMapBlocks() {
		t.Fatal("view with unwarmed land reported ready")
	}
	if mapBlockImg.SceneBlockCache.PrewarmQueueLen() == 0 {
		t.Fatal("remaining land was not left on the queue")
	}
}
