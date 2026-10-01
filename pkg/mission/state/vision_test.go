package state

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
)

func TestStampLeavesCellsOutsideRadiusDark(t *testing.T) {
	vision := NewFactionVision(16, 16)
	vision.Stamp(5.2, 5.2, 2)
	if !vision.VisibleAt(5, 5) {
		t.Fatal("cell under the unit should be visible")
	}
	if vision.VisibleAt(9, 5) {
		t.Fatal("cell outside the radius should stay dark")
	}
	vision.CommitExplored()
	vision.ClearVisible()
	if vision.VisibleAt(5, 5) {
		t.Fatal("visible should clear each tick")
	}
	if !vision.ExploredAt(5, 5) {
		t.Fatal("explored should keep the lit cell")
	}
	if vision.ExploredAt(9, 5) {
		t.Fatal("unlit cell should stay unexplored")
	}
}

func TestAllocateVisionsSkipsWhenFogDisabled(t *testing.T) {
	ms := &MissionState{}
	ms.allocateVisions(8, 8)
	if ms.Player.Visions != nil {
		t.Fatal("disabled fog should not allocate vision maps")
	}

	ms.Core.FogOfWar = true
	ms.allocateVisions(8, 8)
	if ms.Player.Visions[faction.HumanAlpha] == nil || ms.Player.Visions[faction.ComputerAlpha] == nil {
		t.Fatal("enabled fog should allocate both sides")
	}
}

// 未知海面的边界必须按渲染分辨率渐变。按格读 Explored 会让边界在格线上
// 一步跳 80 的暗度，主视图上就是一圈像被啃过的锯齿阶梯。
func TestShadeEdgeIsSmoothAcrossCells(t *testing.T) {
	vision := NewFactionVision(32, 32)
	vision.BeginFrame()
	vision.Stamp(16, 16, 5)
	vision.StampLight(16, 16, 5)
	vision.FinishFrame()
	// 第二拍不再照亮，只看已探索留下的边界。
	vision.BeginFrame()
	vision.FinishFrame()

	rw, _ := vision.renderSize()
	row := 16 * visionRenderScale
	worst, worstAt := 0, 0
	prev := int(vision.Shade[(row*rw)*4+3])
	for px := 1; px < rw; px++ {
		cur := int(vision.Shade[(row*rw+px)*4+3])
		diff := cur - prev
		if diff < 0 {
			diff = -diff
		}
		if diff > worst {
			worst, worstAt = diff, px
		}
		prev = cur
	}
	// 一格的 80 点差值要摊到四个渲染像素上，单像素只该跨它的四分之一上下；
	// 按格跳变时一整格会在一个像素里跳完。
	if worst > (shadeUnknown-shadeExplored)/2 {
		t.Fatalf("已探索边界在第 %d 个渲染像素跳变 %d，仍是按格跳变的硬边", worstAt, worst)
	}
}

// 已探索的海面重新被照亮时，光圈的软边不能比已探索的暗度更黑，
// 否则视野边缘会多出一圈比周围更暗的黑环。
func TestShadeKeepsExploredSeaFromGoingDarker(t *testing.T) {
	vision := NewFactionVision(32, 32)
	// 先停着照一片，再往前挪一格，制造出软边压在已探索海面上的情形。
	for _, x := range []float64{16, 17} {
		vision.BeginFrame()
		vision.Stamp(x, 16, 5)
		vision.StampLight(x, 16, 5)
		vision.FinishFrame()
	}

	for i := range vision.Light {
		if vision.ExploredLight[i] < 255 {
			continue
		}
		if alpha := int(vision.Shade[i*4+3]); alpha > shadeExplored {
			t.Fatalf("已探索格子被画出 %d 的暗度，视野边缘会出现黑环", alpha)
		}
	}
}
