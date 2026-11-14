package drawer

import (
	"testing"

	"github.com/stretchr/testify/require"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

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

func TestCalcAbbrSidePanelsSkipsWhenNoRoom(t *testing.T) {
	require.Zero(t, calcAbbrSidePanels(800, 900, 800).Left.W, "地图占满屏幕时不画态势栏")
}
