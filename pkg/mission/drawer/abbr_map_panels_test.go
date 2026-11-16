package drawer

import (
	"testing"

	"github.com/stretchr/testify/require"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestFleetTypeRowsAggregateAndOrderByShipType 验证编成按舰种聚合，并按主力舰优先的权重排序。
// 场景：态势栏过窄，不能逐舰级展开；同一舰种被拆在多个舰级条目里（两个驱逐舰条目各 2/1 艘）。
// 预期：3 行依次为战列舰 1、巡洋舰 2、驱逐舰 3；失败意味着聚合漏加或排序权重错。
// 覆盖：构造 Total=6、4 个 ShipClass（驱逐 2、战列 1、巡洋 2、驱逐 1），断言 []fleetTypeRow 完全相等。
func TestFleetTypeRowsAggregateAndOrderByShipType(t *testing.T) {
	fleet := state.Fleet{
		Total: 6,
		Classes: []state.ShipClass{
			{Total: 2, Kind: &objUnit.BattleShip{Type: objUnit.ShipTypeDestroyer}},
			{Total: 1, Kind: &objUnit.BattleShip{Type: objUnit.ShipTypeBattleShip}},
			{Total: 2, Kind: &objUnit.BattleShip{Type: objUnit.ShipTypeCruiser}},
			{Total: 1, Kind: &objUnit.BattleShip{Type: objUnit.ShipTypeDestroyer}},
		},
	}

	require.Equal(t, []fleetTypeRow{
		{shipType: objUnit.ShipTypeBattleShip, count: 1},
		{shipType: objUnit.ShipTypeCruiser, count: 2},
		{shipType: objUnit.ShipTypeDestroyer, count: 3},
	}, fleetTypeRows(fleet))
}

// TestCalcAbbrSidePanelsStayInsideScreenAndOutsideMap 验证左右态势栏落在各自留白内，不压地图也不出屏。
// 场景：全屏缩略地图居中，两侧留白随分辨率和地图宽度变化；留白不足时需降级为 compact 布局。
// 预期：左栏宽 >0，左右外边距对称，左栏右缘 ≤ colW、右栏左缘 ≥ w-colW、右栏右缘 ≤ w，两栏同 Y 同宽。
// 覆盖：表驱动 6 组 2560x1440 至 1024x768（含 abbrW 768/512），断言 Compact、对称、高度与边界。
func TestCalcAbbrSidePanelsStayInsideScreenAndOutsideMap(t *testing.T) {
	cases := []struct {
		name    string
		w, h    int
		abbrW   int
		compact bool
	}{
		{name: "2560x1440", w: 2560, h: 1440, abbrW: 1440},
		{name: "1920x1080", w: 1920, h: 1080, abbrW: 1080},
		{name: "1366x768", w: 1366, h: 768, abbrW: 768},
		{name: "1280x720", w: 1280, h: 720, abbrW: 720},
		{name: "1024x768_square", w: 1024, h: 768, abbrW: 768, compact: true},
		{name: "1024x768_portrait", w: 1024, h: 768, abbrW: 512},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			panels := calcAbbrSidePanels(tc.w, tc.h, tc.abbrW)
			require.Equal(t, tc.compact, panels.Compact)
			require.Positive(t, panels.Left.W)

			colW := float64(tc.w-tc.abbrW) / 2
			require.GreaterOrEqual(t, panels.Left.X, abbrPanelMargin-0.001)
			require.InDelta(t, panels.Left.X, float64(tc.w)-(panels.Right.X+panels.Right.W), 0.001, "左右留白对称")
			require.LessOrEqual(t, panels.Left.X+panels.Left.W, colW+0.001, "左栏不能压进地图")
			require.GreaterOrEqual(t, panels.Right.X, float64(tc.w)-colW-0.001, "右栏不能压进地图")
			require.LessOrEqual(t, panels.Right.X+panels.Right.W, float64(tc.w)+0.001, "右栏不能出屏")
			require.InDelta(t, float64(tc.h)-abbrPanelTop-abbrPanelBottom, panels.Left.H, 0.001)
			require.Equal(t, panels.Left.Y, panels.Right.Y)
			require.Equal(t, panels.Left.W, panels.Right.W)
		})
	}
}

// TestCalcAbbrSidePanelsSkipsWhenNoRoom 验证地图占满屏幕宽度时返回零布局，调用方跳过绘制。
// 场景：800 宽窗口加载 800 宽缩略地图，两侧留白 colW=0，panelW=-32，低于最小宽 80。
// 预期：Left.W=0（零布局）；失败意味着会画出负宽面板或压住地图。
// 覆盖：单次调用 calcAbbrSidePanels(800,900,800)，require.Zero 左栏宽度并带「地图占满屏幕」消息。
func TestCalcAbbrSidePanelsSkipsWhenNoRoom(t *testing.T) {
	require.Zero(t, calcAbbrSidePanels(800, 900, 800).Left.W, "地图占满屏幕时不画态势栏")
}
