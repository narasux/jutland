package action

import "testing"

// hoverAxes 把悬停动作拆成水平 / 竖直两个滚屏轴向，供判定带用例断言。
func hoverAxes(h CursorHoverType) (horizontal, vertical bool) {
	switch h {
	case HoverScreenLeft, HoverScreenRight:
		return true, false
	case HoverScreenTop, HoverScreenBottom:
		return false, true
	case HoverScreenTopLeft, HoverScreenTopRight, HoverScreenBottomLeft, HoverScreenBottomRight:
		return true, true
	}
	return false, false
}

// TestHoverTypeAtCornersAndEdges 验证四角同时带回两个滚屏轴向，四边与正中各自命中对应动作。
// 场景：1280x720 战场，玩家把光标推到屏幕最角上时 Ebiten 给出的坐标是 (0,0)、(1279,0)、
// (0,719)、(1279,719)；旧实现用 0 < x 排除左边界，光标一压到左上角就退化成只向上滚屏。
// 预期：四角分别返回 HoverScreenTopLeft/TopRight/BottomLeft/BottomRight，四边中点返回对应
// HoverScreen* 单轴动作，(640,360) 返回 HoverScreenMiddle；失败意味着角落判定又退化成单轴。
// 覆盖：表驱动 9 个代表点（四角、四边中点、正中），逐点断言 CursorHoverType。
func TestHoverTypeAtCornersAndEdges(t *testing.T) {
	const w, h = 1280, 720

	for _, tc := range []struct {
		name string
		x, y int
		want CursorHoverType
	}{
		{name: "top-left corner", x: 0, y: 0, want: HoverScreenTopLeft},
		{name: "top-right corner", x: w - 1, y: 0, want: HoverScreenTopRight},
		{name: "bottom-left corner", x: 0, y: h - 1, want: HoverScreenBottomLeft},
		{name: "bottom-right corner", x: w - 1, y: h - 1, want: HoverScreenBottomRight},
		{name: "top edge", x: w / 2, y: 0, want: HoverScreenTop},
		{name: "bottom edge", x: w / 2, y: h - 1, want: HoverScreenBottom},
		{name: "left edge", x: 0, y: h / 2, want: HoverScreenLeft},
		{name: "right edge", x: w - 1, y: h / 2, want: HoverScreenRight},
		{name: "middle", x: w / 2, y: h / 2, want: HoverScreenMiddle},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := HoverTypeAt(tc.x, tc.y, w, h); got != tc.want {
				t.Fatalf("HoverTypeAt(%d,%d) = %s, want %s", tc.x, tc.y, got, tc.want)
			}
		})
	}
}

// TestHoverTypeAtHasNoDeadBands 验证判定带内不存在让滚屏停住的空档。
// 场景：旧实现用 x < 5 / 5 < x 的严格不等式，光标正好停在 x==5 或 y==5 时返回 DoNothing，
// 贴着左边缘缓慢移出的手会看到相机走到一半突然停住；顶部判定带还只有 5px，极易被手抖甩出。
// 预期：整屏任何一点都返回非 DoNothing，且 x 落在左判定带内时一定带水平分量、
// y 落在上判定带内时一定带竖直分量；失败意味着判定带里又出现死区或斜向/单轴混判。
// 覆盖：1280x720 全屏逐点扫描，断言动作合法、两个判定带内的轴向分量齐全。
func TestHoverTypeAtHasNoDeadBands(t *testing.T) {
	const w, h = 1280, 720

	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			hover := HoverTypeAt(x, y, w, h)
			if hover == DoNothing {
				t.Fatalf("HoverTypeAt(%d,%d) = DoNothing, 判定带内不应有死区", x, y)
			}
			horizontal, vertical := hoverAxes(hover)
			if x < edgeScrollZone && !horizontal {
				t.Fatalf("HoverTypeAt(%d,%d) = %s, 左判定带缺水平分量", x, y, hover)
			}
			if y < edgeScrollZone && !vertical {
				t.Fatalf("HoverTypeAt(%d,%d) = %s, 上判定带缺竖直分量", x, y, hover)
			}
		}
	}
}
