package action

import (
	"testing"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestSelectedAreaOverlapsIncludesBoundary 验证选区与轴对齐矩形的重叠判定包含边界。
// 场景：鼠标从地图 (20,20) 反向拖到 (10,10)，另给三个矩形：只共享角点、完全包含、完全分离。
// 预期：共享角点与包含都返回 true，分离返回 false；失败意味着框边刚好压住舰体时漏选。
// 覆盖：反向拖动的 SelectedArea 分别与三种矩形求 Overlaps，并用 Contain 断言内点与界外点。
func TestSelectedAreaOverlapsIncludesBoundary(t *testing.T) {
	area := SelectedArea{StartAt: objPos.New(20, 20), CurAt: objPos.New(10, 10)}

	if !area.Overlaps(20, 20, 25, 25) {
		t.Fatal("rect sharing only the corner should overlap")
	}
	if !area.Overlaps(15, 15, 16, 16) {
		t.Fatal("rect inside the area should overlap")
	}
	if area.Overlaps(20.01, 20.01, 25, 25) {
		t.Fatal("rect outside the area should not overlap")
	}
	if !area.Contain(objPos.New(10, 10)) {
		t.Fatal("area corner should be contained")
	}
	if area.Contain(objPos.NewR(9.9, 10)) {
		t.Fatal("point outside the area should not be contained")
	}
}
