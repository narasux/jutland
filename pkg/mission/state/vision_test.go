package state

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
)

// TestStampLeavesCellsOutsideRadiusDark 验证 Stamp 只点亮半径内格子，Visible 每拍清空但 Explored 保留。
// 场景：16×16 图上以 (5.2,5.2) 半径 2 照亮，(5,5) 在圆内、(9,5) 在圆外；帧末 CommitExplored 再 ClearVisible。
// 预期：Stamp 后 (5,5) 可见、(9,5) 不可见；清空后 (5,5) 不可见但已探索，(9,5) 仍未被探索。
// 覆盖：NewFactionVision(16,16) 上依次调用 Stamp/CommitExplored/ClearVisible，断言 VisibleAt 与 ExploredAt。
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

// TestAllocateVisionsSkipsWhenFogDisabled 验证 allocateVisions 只在开启迷雾时建图，关闭时 Visions 保持 nil。
// 场景：NewMissionState 在地图配置就绪后按 MapCfg 宽高调用；无雾任务不应产生任何视野分配与每拍扫描。
// 预期：默认调用后 Player.Visions 为 nil；置 Core.FogOfWar=true 后 HumanAlpha/ComputerAlpha 均非 nil。
// 覆盖：同一个空 MissionState 上前后两次 allocateVisions(8,8)，断言 map 的 nil 与两侧条目。
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

// TestShadeEdgeIsSmoothAcrossCells 验证已探索/未知海面的蒙层边界按渲染分辨率渐变，而不是按格跳变。
// 场景：视野边缘横跨一格时，若按格读取 Explored，边界会在格线上一步跳 80 点暗度，主视图出现锯齿硬边。
// 预期：同一行相邻渲染像素的最大暗度差不超过 (shadeUnknown-shadeExplored)/2。
// 覆盖：32x32 视野照亮 (16,16) 半径 5，第二拍只提交探索；扫描第 16 行渲染像素并记录最大差值。
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

// TestShadeKeepsExploredSeaFromGoingDarker 验证已探索海面重新被照亮时，软边不会比已探索基准更黑。
// 场景：视野从 (16,16) 前移到 (17,16)，光圈软边压在已探索海面上；
// 若插值把 alpha 推高，边缘会出现更暗的黑环。
// 预期：四角都已探索的渲染像素 alpha <= shadeExplored；任何像素 alpha <= shadeUnknown。
// 覆盖：连续两拍 Stamp 制造软边，遍历全部渲染像素，用 fullExploredAround 判断四角并断言 alpha。
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

// TestShadeBaseLevels 验证蒙层三个基准状态（可见/已探索不可见/未知）精确落在格心像素上。
// 场景：某一格当前可见应完全透明，曾经看过但本拍不可见应为 shadeExplored，从未探索应为 shadeUnknown；
// 三个值串了会让已探索区域变黑或未知区域透出地图。
// 预期：格心像素 alpha 分别等于 0、shadeExplored、shadeUnknown。
// 覆盖：先照亮 (8,8)，第二拍改照 (24,24)，读取三格中心像素断言。
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
