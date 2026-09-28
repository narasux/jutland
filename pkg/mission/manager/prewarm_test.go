package manager

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/state"
	mapBlockImg "github.com/narasux/jutland/pkg/resources/images/mapblock"
)

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
