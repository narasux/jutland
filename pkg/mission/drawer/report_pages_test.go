package drawer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func TestBuildBattleShipDetailRowsSortsByTonnage(t *testing.T) {
	objUnit.ShipMap["DetailHeavy"] = &objUnit.BattleShip{Name: "DetailHeavy", Tonnage: 40000}
	objUnit.ShipMap["DetailLight"] = &objUnit.BattleShip{Name: "DetailLight", Tonnage: 1200}
	t.Cleanup(func() {
		delete(objUnit.ShipMap, "DetailHeavy")
		delete(objUnit.ShipMap, "DetailLight")
	})

	stats := state.BattleSideStats{
		ShipsInitialByClass: map[string]int{"DetailHeavy": 1, "DetailLight": 2},
		ShipsBuiltByClass:   map[string]int{"DetailLight": 2},
		ShipsLostByClass:    map[string]int{"DetailHeavy": 1},
	}
	rows := buildBattleShipDetailRows(stats)
	require.Len(t, rows, 2)
	require.Equal(t, "DetailHeavy", rows[0].Name)
	require.Equal(t, 1, rows[0].Initial)
	require.Equal(t, 0, rows[0].Built)
	require.Equal(t, 1, rows[0].Lost)
	require.Equal(t, 0, rows[0].Remaining)
	require.Equal(t, "DetailLight", rows[1].Name)
	require.Equal(t, 2, rows[1].Initial)
	require.Equal(t, 2, rows[1].Built)
	require.Equal(t, 4, rows[1].Remaining)
}

func TestBuildBattlePlaneDetailRowsSortsAllTypes(t *testing.T) {
	stats := state.BattleSideStats{
		PlanesInitialByType: map[string]int{},
		PlanesBuiltByType:   map[string]int{},
		PlanesLostByType:    map[string]int{},
	}
	for idx := 0; idx < 12; idx++ {
		name := fmt.Sprintf("P%02d", idx)
		stats.PlanesBuiltByType[name] = idx
		stats.PlanesLostByType[name] = 12 - idx
	}
	rows := buildBattlePlaneDetailRows(stats)
	require.Len(t, rows, 12)
	require.Equal(t, "P00", rows[0].Name)
	require.Equal(t, 12, rows[0].Lost)
	require.Equal(t, "P11", rows[11].Name)
	require.Equal(t, 1, rows[11].Lost)
	// 机型不再聚合到「其他」行，每一行都保留真实机型名。
	for _, row := range rows {
		require.NotEmpty(t, row.Name)
	}
}

func TestBuildBattlePlaneDetailRowsDropsZeroCountRows(t *testing.T) {
	stats := state.BattleSideStats{
		PlanesInitialByType: map[string]int{"Real": 1, "Empty": 0},
		PlanesBuiltByType:   map[string]int{"BuiltZero": 0},
		PlanesLostByType:    map[string]int{"LostZero": 0},
	}
	rows := buildBattlePlaneDetailRows(stats)
	require.Len(t, rows, 1)
	require.Equal(t, "Real", rows[0].Name)
}

func TestBattleReportPlaneTypeAbbr(t *testing.T) {
	objUnit.PlaneMap["TagFighter"] = &objUnit.Plane{
		Name: "TagFighter", Type: objUnit.PlaneTypeFighter, TypeAbbr: "F",
	}
	objUnit.PlaneMap["TagLong"] = &objUnit.Plane{
		Name: "TagLong", Type: objUnit.PlaneTypeScout, TypeAbbr: "SCOUT",
	}
	objUnit.PlaneMap["TagOther"] = &objUnit.Plane{
		Name: "TagOther", Type: objUnit.PlaneTypeOther, TypeAbbr: "OTHER",
	}
	t.Cleanup(func() {
		delete(objUnit.PlaneMap, "TagFighter")
		delete(objUnit.PlaneMap, "TagLong")
		delete(objUnit.PlaneMap, "TagOther")
	})

	require.Equal(t, "F", battleReportPlaneTypeAbbr("TagFighter"))
	require.Equal(t, "SC", battleReportPlaneTypeAbbr("TagLong"))
	require.Equal(t, "O", battleReportPlaneTypeAbbr("TagOther"))
	require.Empty(t, battleReportPlaneTypeAbbr("MissingPlane"))
}

func TestDetailRowsMergeInitialBuiltAndLost(t *testing.T) {
	stats := state.BattleSideStats{
		PlanesInitialByType: map[string]int{"OnlyInitial": 3, "All": 2},
		PlanesBuiltByType:   map[string]int{"All": 4},
		PlanesLostByType:    map[string]int{"All": 1},
	}
	rows := buildBattlePlaneDetailRows(stats)
	require.Len(t, rows, 2)
	byName := map[string]battleReportDetailRow{}
	for _, row := range rows {
		byName[row.Name] = row
	}
	require.Equal(t, 3, byName["OnlyInitial"].Initial)
	require.Equal(t, 3, byName["OnlyInitial"].Remaining)
	require.Equal(t, 2, byName["All"].Initial)
	require.Equal(t, 4, byName["All"].Built)
	require.Equal(t, 1, byName["All"].Lost)
	require.Equal(t, 5, byName["All"].Remaining)
}

func TestBuildBattleShipDetailRowsKeepsAllClasses(t *testing.T) {
	stats := state.BattleSideStats{ShipsInitialByClass: map[string]int{}}
	for idx := 0; idx < 15; idx++ {
		stats.ShipsInitialByClass[fmt.Sprintf("Ship%02d", idx)] = 1
	}
	rows := buildBattleShipDetailRows(stats)
	require.Len(t, rows, 15)
	for _, row := range rows {
		require.NotEmpty(t, row.Name)
	}
}

func TestBuildBattleShipDetailRowsDropsZeroCountRows(t *testing.T) {
	stats := state.BattleSideStats{
		ShipsInitialByClass: map[string]int{"Real": 1, "Empty": 0},
		ShipsBuiltByClass:   map[string]int{"BuiltZero": 0},
		ShipsLostByClass:    map[string]int{"LostZero": 0},
	}
	rows := buildBattleShipDetailRows(stats)
	require.Len(t, rows, 1)
	require.Equal(t, "Real", rows[0].Name)
}

// 有滚动条时，绘制窗口必须使用状态里的偏移；否则滑块到了底部，
// 列表仍然从第一行开始画（最大吨位的那批船一直占着屏幕）。
func TestBattleReportDetailScrollStartKeepsScrolledWindow(t *testing.T) {
	scrollbar := state.BattleReportScrollbar{MaxScroll: 12}
	require.Equal(t, 0, battleReportDetailScrollStart(0, scrollbar, true))
	require.Equal(t, 5, battleReportDetailScrollStart(5, scrollbar, true))
	require.Equal(t, 12, battleReportDetailScrollStart(40, scrollbar, true))
	require.Zero(t, battleReportDetailScrollStart(5, scrollbar, false))
}
