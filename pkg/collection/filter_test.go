package collection

import (
	"testing"

	"github.com/stretchr/testify/require"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// withTestShipCatalog 临时替换全局舰船表，返回后恢复，避免污染其他图鉴测试。
func withTestShipCatalog(t *testing.T) {
	t.Helper()
	previousMap, previousNames := objUnit.ShipMap, objUnit.AllShipNames
	objUnit.ShipMap = map[string]*objUnit.BattleShip{
		"us-bb": {Name: "us-bb", Nation: objUnit.NationUS, Type: objUnit.ShipTypeBattleShip},
		"us-cv": {Name: "us-cv", Nation: objUnit.NationUS, Type: objUnit.ShipTypeAircraftCarrier},
		"jp-bb": {Name: "jp-bb", Nation: objUnit.NationJP, Type: objUnit.ShipTypeBattleShip},
	}
	objUnit.AllShipNames = []string{"us-bb", "us-cv", "jp-bb"}
	t.Cleanup(func() {
		objUnit.ShipMap, objUnit.AllShipNames = previousMap, previousNames
	})
}

// TestFilteredShipsMatchesNationAndClass 验证图鉴列表同时按国籍和舰种过滤，顺序保持配置顺序。
// 场景：玩家在收藏页选择 US + 战列舰、JP + 全部、全部 + 全部；列表不能混入其他国籍或舰种。
// 预期：US/Battleship → [us-bb]；JP/Battleship → [jp-bb]；All/All → [us-bb, us-cv, jp-bb]。
// 覆盖：替换 ShipMap/AllShipNames 后设置 CollectionUI 的 shipNation/shipClass，调用 filteredShips 比对名称。
func TestFilteredShipsMatchesNationAndClass(t *testing.T) {
	withTestShipCatalog(t)

	cases := []struct {
		name   string
		nation objUnit.Nation
		class  shipClassFilter
		want   []string
	}{
		{name: "US battleship", nation: objUnit.NationUS, class: shipClassBattleship, want: []string{"us-bb"}},
		{name: "JP battleship", nation: objUnit.NationJP, class: shipClassBattleship, want: []string{"jp-bb"}},
		{name: "all ships", nation: objUnit.NationAll, class: shipClassAll, want: []string{"us-bb", "us-cv", "jp-bb"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ui := &CollectionUI{shipNation: tc.nation, shipClass: tc.class}
			got := ui.filteredShips()
			names := make([]string, len(got))
			for i, ship := range got {
				names[i] = ship.Name
			}
			require.Equal(t, tc.want, names)
		})
	}
}

// TestMoveShipWrapsWithinFilteredList 验证左右切换在当前筛选结果内循环，并跳过不在列表中的当前舰。
// 场景：当前 us-bb，按右键到 us-cv、再按右键回到 us-bb；筛选切到 JP 后当前舰不在列表，
// 导航应从列表第一艘重新开始，而不是越界或停在空卡片。
// 预期：moveShip(1)/moveShip(1) 在 US 列表循环；切到 JP 后 moveShip(1) 落到 jp-bb；
// 空列表直接返回，curShip 保持原值。
// 覆盖：复用测试舰船表，直接调用 moveShip 并断言 curShip 与 pendingRebuild。
func TestMoveShipWrapsWithinFilteredList(t *testing.T) {
	withTestShipCatalog(t)
	ui := &CollectionUI{shipNation: objUnit.NationUS, shipClass: shipClassAll, curShip: "us-bb"}

	ui.moveShip(1)
	require.Equal(t, "us-cv", ui.curShip)
	ui.moveShip(1)
	require.Equal(t, "us-bb", ui.curShip)
	require.True(t, ui.pendingRebuild)

	ui.shipNation = objUnit.NationJP
	ui.moveShip(1)
	require.Equal(t, "jp-bb", ui.curShip)

	empty := &CollectionUI{shipNation: objUnit.NationUS, shipClass: shipClassSubmarine}
	empty.curShip = "us-bb"
	empty.moveShip(1)
	require.Equal(t, "us-bb", empty.curShip, "empty filtered list should keep the current ship")
}
