package drawer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestBuildBattleShipDetailRowsSortsByTonnage 验证舰艇明细按吨位降序，并合并初始/建造/损失计算在役数。
// 场景：战报结算要优先列重舰；DetailHeavy 4 万吨、DetailLight 1200 吨，两舰都有建造或损失记录。
// 预期：Heavy 在前（Initial1/Built0/Lost1/Remaining0），Light 在后（2/2/0/4）；失败意味着排序或合并公式错。
// 覆盖：临时向 ShipMap 注入两舰，构造三张统计表，断言 2 行及每行四计数，Cleanup 删除注入项。
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

// TestBuildBattlePlaneDetailRowsSortsAllTypes 验证 12 个机型全部保留并按损失数降序，不再折叠为「其他」。
// 场景：战报飞机页遇到 P00–P11 共 12 型，损失从 12 递减到 1；旧实现可能截断长尾或聚合。
// 预期：返回 12 行，P00 第一（Lost=12），P11 最后（Lost=1），每行 Name 非空；失败意味着机型丢失或乱序。
// 覆盖：循环填 PlanesBuiltByType[Pxx]=idx、PlanesLostByType[Pxx]=12-idx，断言长度、首尾与全部名称非空。
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

// TestBuildBattlePlaneDetailRowsDropsZeroCountRows 验证三表均为 0 的机型不会生成空行。
// 场景：统计初始化会为未参战机型写 0（Empty/BuiltZero/LostZero），不过滤会撑大表格和滚动范围。
// 预期：只返回 Real 一行；失败意味着空机型进入明细，合计与滚动条被虚增。
// 覆盖：构造 initial{Real:1,Empty:0}、built{BuiltZero:0}、lost{LostZero:0}，断言长度 1 且 Name=Real。
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

// TestBattleReportPlaneTypeAbbr 验证机型缩写最多 2 字符，Other 固定为 O，未知机型返回空串。
// 场景：战报名称前列宽固定，配置里可能出现超长缩写（SCOUT）和特殊类型 Other。
// 预期：Fighter→"F"、SCOUT→"SC"、Other→"O"、PlaneMap 缺失→""；失败意味着标签列错位或空框。
// 覆盖：临时注入 TagFighter/TagLong/TagOther 三个 Plane，逐个断言 battleReportPlaneTypeAbbr，Cleanup 删除。
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

// TestDetailRowsMergeInitialBuiltAndLost 验证按名称合并三张表，并用 Initial+Built-Lost 算出在役数。
// 场景：某机型只出现在初始表，另一机型三张表都有；战报每行需要完整的生命周期计数。
// 预期：OnlyInitial 为 3/0/0/3，All 为 2/4/1/5；失败意味着漏合并某张表或用在役公式错。
// 覆盖：构造 Planes 三表（OnlyInitial:3；All:2/4/1），把返回行转成 map[name]row 后逐字段断言。
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

// TestBuildBattleShipDetailRowsKeepsAllClasses 验证 15 个舰级全部出现在明细中，不因面板高度被截断。
// 场景：大型战役参战舰级超过一屏（15 行）；分页/滚动应由外层滚动条处理，而不是丢数据。
// 预期：返回 15 行且每行 Name 非空；失败意味着舰级被静默截断，战报统计不完整。
// 覆盖：循环构造 Ship00–Ship14 初始各 1 艘，断言 rows 长度 15 并检查名称非空。
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

// TestBuildBattleShipDetailRowsDropsZeroCountRows 验证三表均为 0 的舰级不会生成空行。
// 场景：统计初始化会为未参战舰级写 0（Empty/BuiltZero/LostZero），不过滤会多出无效行并影响滚动。
// 预期：只保留 Real 一行；失败意味着空舰级占行，合计与滚动范围被污染。
// 覆盖：构造 ShipsInitialByClass{Real:1,Empty:0}、Built{BuiltZero:0}、Lost{LostZero:0}，断言长度 1。
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

// TestBattleReportDetailScrollStartKeepsScrolledWindow 验证明细表绘制窗口采用滚动偏移，而不是永远从第一行开始。
// 场景：战报舰艇页有滚动条（MaxScroll=12）；玩家把滑块停在 5 或拖到底部 40 时，最大吨位的那批船不该一直占着屏幕。
// 预期：scroll 0→0、5→5、40 被夹紧到 MaxScroll=12；无滚动条时即使状态里残留 5 也从 0 开始；失败意味着滑块到底后列表仍停在第一行。
// 覆盖：battleReportDetailScrollStart 四组调用（0/5/40 有滚动条 + 5 无滚动条），用 require.Equal/require.Zero 断言返回值。
func TestBattleReportDetailScrollStartKeepsScrolledWindow(t *testing.T) {
	scrollbar := state.BattleReportScrollbar{MaxScroll: 12}
	require.Equal(t, 0, battleReportDetailScrollStart(0, scrollbar, true))
	require.Equal(t, 5, battleReportDetailScrollStart(5, scrollbar, true))
	require.Equal(t, 12, battleReportDetailScrollStart(40, scrollbar, true))
	require.Zero(t, battleReportDetailScrollStart(5, scrollbar, false))
}
