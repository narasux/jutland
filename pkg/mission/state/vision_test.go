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
		vision.FinishFrame()
	}

	rw, rh := vision.renderSize()
	for py := 0; py < rh; py++ {
		for px := 0; px < rw; px++ {
			cx, cy := px/visionRenderScale, py/visionRenderScale
			// 四角都已探索的像素必须压到 shadeExplored，不能更黑。
			if fullExploredAround(vision, cx, cy) {
				if alpha := int(vision.Shade[(py*rw+px)*4+3]); alpha > shadeExplored {
					t.Fatalf("已探索格子在第 %d,%d 个渲染像素被画出 %d 的暗度", px, py, alpha)
				}
			}
			// 反过来，从没探索过的海面不能被画得比未知更亮。
			if alpha := int(vision.Shade[(py*rw+px)*4+3]); alpha > shadeUnknown {
				t.Fatalf("第 %d,%d 个渲染像素暗度 %d 超过未知海面", px, py, alpha)
			}
		}
	}
}

// fullExploredAround 判断渲染像素四角的格子是否全部已探索。
func fullExploredAround(v *FactionVision, cx, cy int) bool {
	for _, d := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		x, y := cx+d[0], cy+d[1]
		if x >= v.Width {
			x = v.Width - 1
		}
		if y >= v.Height {
			y = v.Height - 1
		}
		if !v.ExploredAt(x, y) {
			return false
		}
	}
	return true
}

// 蒙层的三个基准状态不能串：看得见完全透明，看过但这一拍看不见是 shadeExplored，
// 从没看过是 shadeUnknown。查表合成后这三个值必须精确落在格心像素上。
func TestShadeBaseLevels(t *testing.T) {
	vision := NewFactionVision(32, 32)
	vision.BeginFrame()
	vision.Stamp(8, 8, 3)
	vision.FinishFrame()
	// 第二拍只照亮别处，让 (8,8) 变成「看过但现在看不见」。
	vision.BeginFrame()
	vision.Stamp(24, 24, 3)
	vision.FinishFrame()

	center := func(cx, cy int) int {
		px := cx*visionRenderScale + visionRenderScale/2
		py := cy*visionRenderScale + visionRenderScale/2
		rw, _ := vision.renderSize()
		return int(vision.Shade[(py*rw+px)*4+3])
	}
	if got := center(8, 8); got != shadeExplored {
		t.Fatalf("已探索但当前不可见的海面暗度 = %d, want %d", got, shadeExplored)
	}
	if got := center(24, 24); got != 0 {
		t.Fatalf("当前可见的海面暗度 = %d, want 0", got)
	}
	if got := center(16, 16); got != shadeUnknown {
		t.Fatalf("从没探索过的海面暗度 = %d, want %d", got, shadeUnknown)
	}
}
