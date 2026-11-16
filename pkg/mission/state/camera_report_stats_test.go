package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestCameraContainsClosedBounds 验证相机命中测试包含四边，越界一格即返回 false。
// 场景：绘制/选择单位前用 Camera.Contains 裁剪；边界应算可见，否则屏幕最右/最下一列单位会被漏画。
// 预期：(10,20) 到 (110,70) 的相机包含四角与内部点；左/右/上/下越界一格均 false。
// 覆盖：Rect 边界与内部/外部共 7 个坐标调用 Contains。
func TestCameraContainsClosedBounds(t *testing.T) {
	camera := &Camera{Pos: objPos.New(10, 20), Width: 100, Height: 50}
	for _, tc := range []struct {
		name string
		pos  objPos.MapPos
		want bool
	}{
		{name: "top left", pos: objPos.New(10, 20), want: true},
		{name: "bottom right", pos: objPos.New(110, 70), want: true},
		{name: "inside", pos: objPos.New(60, 45), want: true},
		{name: "left outside", pos: objPos.New(9, 45), want: false},
		{name: "right outside", pos: objPos.New(111, 45), want: false},
		{name: "top outside", pos: objPos.New(60, 19), want: false},
		{name: "bottom outside", pos: objPos.New(60, 71), want: false},
	} {
		require.Equal(t, tc.want, camera.Contains(tc.pos), tc.name)
	}
}

// TestBattleReportPageShiftWrapsAndValidates 验证战报页签前后循环与非法页回退。
// 场景：玩家用 Tab/Shift+Tab 翻页；非法页（-1/4）应回到总览再移动，不能取模成负下标。
// 预期：Overview 后退→Events，后退再前进→Overview；Ships 前进 3→Overview；
// Valid 对 0..3 为 true，-1/4 为 false。
// 覆盖：表驱动 Shift(+1/-1/+3/0) 与 Valid 边界。
func TestBattleReportPageShiftWrapsAndValidates(t *testing.T) {
	require.Equal(t, BattleReportPageEvents, BattleReportPageOverview.Shift(-1))
	require.Equal(t, BattleReportPageOverview, BattleReportPageEvents.Shift(1))
	require.Equal(t, BattleReportPageOverview, BattleReportPageShips.Shift(3))
	require.Equal(t, BattleReportPageShips, BattleReportPage(-1).Shift(1))
	require.Equal(t, BattleReportPageEvents, BattleReportPage(4).Shift(-1))

	for _, page := range []BattleReportPage{BattleReportPageOverview, BattleReportPageShips, BattleReportPagePlanes, BattleReportPageEvents} {
		require.True(t, page.Valid(), page)
	}
	require.False(t, BattleReportPage(-1).Valid())
	require.False(t, BattleReportPage(4).Valid())
}

// TestRecordPlaneBuiltAggregatesBySideAndType 验证飞机完工统计按阵营累计总数与机型计数。
// 场景：战报“建造”列统计战时补机；同一机型多架、双方各自生产时必须分开累计。
// 预期：HumanAlpha 造 2 架 F4F、ComputerAlpha 造 1 架 Zero；双方 PlanesBuilt 与 ByType 独立。
// 覆盖：空 BattleStats（Sides 未初始化）上调用三次 RecordPlaneBuilt，断言两侧字段。
func TestRecordPlaneBuiltAggregatesBySideAndType(t *testing.T) {
	stats := &BattleStats{}
	stats.RecordPlaneBuilt(faction.HumanAlpha, "F4F")
	stats.RecordPlaneBuilt(faction.HumanAlpha, "F4F")
	stats.RecordPlaneBuilt(faction.ComputerAlpha, "Zero")

	human := stats.Sides[faction.HumanAlpha]
	require.Equal(t, 2, human.PlanesBuilt)
	require.Equal(t, 2, human.PlanesBuiltByType["F4F"])

	computer := stats.Sides[faction.ComputerAlpha]
	require.Equal(t, 1, computer.PlanesBuilt)
	require.Equal(t, 1, computer.PlanesBuiltByType["Zero"])
}

// TestBattleReportEventScrollbarClampsAndMovesThumb 验证战况滚动条在记录多一行时出现，并把滑块从顶移到底。
// 场景：战况页总共只显示 BattleReportEventRows 行；多 1 条记录时 MaxScroll=1，滚到 99 也必须夹到 1。
// 预期：total=EventRows 时 ok=false；total=EventRows+1 时 MaxScroll=1，scroll=0 滑块在轨道顶，
// scroll=99 滑块在轨道底；scale<=0 返回 false。
// 覆盖：固定 600×500 面板，分别用三种 total/scroll/scale 调用 BattleReportEventScrollbar。
func TestBattleReportEventScrollbarClampsAndMovesThumb(t *testing.T) {
	panel := BattleReportRect{X: 100, Y: 50, W: 600, H: 500}
	if _, ok := BattleReportEventScrollbar(panel, BattleReportEventRows, 0, 1); ok {
		t.Fatal("exactly one screen of events should not show a scrollbar")
	}
	if _, ok := BattleReportEventScrollbar(panel, BattleReportEventRows+1, 0, 0); ok {
		t.Fatal("non-positive scale should not show a scrollbar")
	}

	top, ok := BattleReportEventScrollbar(panel, BattleReportEventRows+1, 0, 1)
	require.True(t, ok)
	require.Equal(t, 1, top.MaxScroll)
	require.InDelta(t, top.TrackY, top.ThumbY, 1e-9)
	require.InDelta(t, top.TrackH*float64(BattleReportEventRows)/float64(BattleReportEventRows+1), top.ThumbH, 1e-9)

	bottom, ok := BattleReportEventScrollbar(panel, BattleReportEventRows+1, 99, 1)
	require.True(t, ok)
	require.InDelta(t, bottom.TrackY+bottom.TrackH-bottom.ThumbH, bottom.ThumbY, 1e-9)
}
