package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestBattleLogCoalescesRepeatedHits 验证同一窗口内的连续命中合并为一条战报，
// 避免高射速武器在战报列表里刷屏；超出窗口后必须另起一条。
// 场景：多门主炮/防空炮在同一秒内连续命中同一目标，每发都调用 RecordBattleLog；
// 若逐发追加，一局战报会膨胀到数万条，详情页无法阅读。
// 预期：合并条目的 Count 累加、Tick 取最后一次命中、版本号每次记录都递增；
// 间隔超过 60 tick（1 秒）后保留两条独立记录。
// 覆盖：先记 5 条 0..40 tick 的命中，再记 500 tick 的命中，断言条数/Count/Tick/版本。
func TestBattleLogCoalescesRepeatedHits(t *testing.T) {
	stats := NewBattleStats()
	for idx := 0; idx < 5; idx++ {
		stats.RecordBattleLog(BattleLogEntry{
			Tick: int64(idx * 10), ActorName: "A", TargetName: "B", Action: BattleLogShipHit,
		})
	}
	require.Len(t, stats.BattleLog, 1)
	require.Equal(t, 5, stats.BattleLog[0].Count)
	require.Equal(t, int64(40), stats.BattleLog[0].Tick)
	require.Equal(t, uint64(5), stats.BattleLogVersion)

	// 超出合并窗口后另起一条。
	stats.RecordBattleLog(BattleLogEntry{
		Tick: 500, ActorName: "A", TargetName: "B", Action: BattleLogShipHit,
	})
	require.Len(t, stats.BattleLog, 2)
	require.Equal(t, 1, stats.BattleLog[1].Count)
	require.Equal(t, uint64(6), stats.BattleLogVersion)
}

// TestRecordShipLostLogsAttackerAndAircraft 验证沉船战报同时记录攻击者、吨位与随舰飞机数量。
// 场景：航母 Carrier（20000 吨）被 Enemy BB 击沉，机库剩 F4F 3 架未起飞，结算页需分开显示。
// 预期：BattleLog 两条：ShipSunk 带 Actor=Enemy BB/ComputerAlpha 与 Tonnage=20000，PlaneWithShip Count=3。
// 覆盖：Damage.KilledByHit、Damage.LastAttackerName/Player 齐全的 BattleShip 调 RecordShipLost，逐字段断言两条记录。
func TestRecordShipLostLogsAttackerAndAircraft(t *testing.T) {
	stats := NewBattleStats()
	ship := &objUnit.BattleShip{
		Name: "Carrier", Tonnage: 20000, BelongPlayer: faction.HumanAlpha,
		Type: objUnit.ShipTypeAircraftCarrier,
		Aircraft: objUnit.ShipAircraft{Groups: []objUnit.PlaneGroup{
			{Name: "F4F", CurCount: 3},
		}},
		Damage: objUnit.DamageState{
			LastAttackerName: "Enemy BB", LastAttackerPlayer: faction.ComputerAlpha, KilledByHit: true,
		},
	}
	stats.RecordShipLost(ship)

	require.Len(t, stats.BattleLog, 2)
	sink := stats.BattleLog[0]
	require.Equal(t, BattleLogShipSunk, sink.Action)
	require.Equal(t, "Enemy BB", sink.ActorName)
	require.Equal(t, faction.ComputerAlpha, sink.ActorPlayer)
	require.Equal(t, "Carrier", sink.TargetName)
	require.Equal(t, 20000.0, sink.Tonnage)
	withShip := stats.BattleLog[1]
	require.Equal(t, BattleLogPlaneWithShip, withShip.Action)
	require.Equal(t, 3, withShip.Count)
}

// TestRecordPlaneLostLogsShotDownOrLoss 验证 RecordPlaneLost 区分被击落与坠毁，只在击落时记录攻击者。
// 场景：F4F 被 A6M 击落（Damage.KilledByHit、Damage.LastAttackerIsPlane），SBD 无攻击者坠毁，两种损失都会进日志。
// 预期：首条 Action=PlaneShotDown、Actor=A6M、Target=F4F；次条 Action=PlaneLost 且 Actor 为空。
// 覆盖：两架 Plane 依次调 RecordPlaneLost，断言两条日志的 Action、ActorName、TargetName 与 TargetIsPlane。
func TestRecordPlaneLostLogsShotDownOrLoss(t *testing.T) {
	stats := NewBattleStats()
	plane := &objUnit.Plane{
		Name: "F4F", BelongPlayer: faction.HumanAlpha,
		Damage: objUnit.DamageState{
			LastAttackerName: "A6M", LastAttackerPlayer: faction.ComputerAlpha,
			LastAttackerIsPlane: true, KilledByHit: true,
		},
	}
	stats.RecordPlaneLost(plane)
	require.Len(t, stats.BattleLog, 1)
	require.Equal(t, BattleLogPlaneShotDown, stats.BattleLog[0].Action)
	require.Equal(t, "A6M", stats.BattleLog[0].ActorName)
	require.Equal(t, "F4F", stats.BattleLog[0].TargetName)
	require.True(t, stats.BattleLog[0].TargetIsPlane)

	crash := &objUnit.Plane{
		Name: "SBD", BelongPlayer: faction.HumanAlpha,
		Damage: objUnit.DamageState{
			LastAttackerName: "A6M", LastAttackerPlayer: faction.ComputerAlpha,
		},
	}
	stats.RecordPlaneLost(crash)
	require.Len(t, stats.BattleLog, 2)
	require.Equal(t, BattleLogPlaneLost, stats.BattleLog[1].Action)
	require.Empty(t, stats.BattleLog[1].ActorName)
}

// TestBattleReportDetailViewportFitsThirteenRows 验证 1.9 倍缩放下明细面板恰好完整显示 13 行。
// 场景：设计行高 26、面板高 480*1.9=912，13 行是常见窗口的下限，多出一行才该出现滚动条。
// 预期：VisibleRows=13；total=13 时 ok=false；total=14 时 ok=true 且 MaxScroll=1，失败即末行被遮住。
// 覆盖：BattleReportDetailVisibleRows(912,1.9) 与 DetailScrollbar(590*1.9 宽面板, 13/14 行) 的返回值。
func TestBattleReportDetailViewportFitsThirteenRows(t *testing.T) {
	scale := 1.9
	panelH := 480 * scale
	require.Equal(t, 13, BattleReportDetailVisibleRows(panelH, scale))

	panel := BattleReportRect{X: 100, Y: 100, W: 590 * scale, H: panelH}
	_, ok := BattleReportDetailScrollbar(panel, 13, 0, scale)
	require.False(t, ok, "13 行应完整显示，不出现滚动条")

	scrollbar, ok := BattleReportDetailScrollbar(panel, 14, 0, scale)
	require.True(t, ok)
	require.Equal(t, 1, scrollbar.MaxScroll)
}

// TestFilterBattleLogKills 验证“只看击杀”筛选保留击沉与击落，滤掉命中、坠毁与随舰损失。
// 场景：战况页签有全部/击杀两种视图，日志里五种 Action 各一条，玩家想直接看到战果。
// 预期：All 保留 5 条；Kills 只剩 ShipSunk、PlaneShotDown 两条且顺序不变，计数函数同为 2。
// 覆盖：构造五种 Action 各一条的 BattleLogEntry 切片，断言两种筛选结果与 FilteredBattleLogCount。
func TestFilterBattleLogKills(t *testing.T) {
	log := []BattleLogEntry{
		{Action: BattleLogShipHit},
		{Action: BattleLogShipSunk},
		{Action: BattleLogPlaneShotDown},
		{Action: BattleLogPlaneLost},
		{Action: BattleLogPlaneWithShip},
	}
	require.Len(t, FilterBattleLog(log, BattleReportEventFilterAll), 5)
	kills := FilterBattleLog(log, BattleReportEventFilterKills)
	require.Len(t, kills, 2)
	require.Equal(t, BattleLogShipSunk, kills[0].Action)
	require.Equal(t, BattleLogPlaneShotDown, kills[1].Action)
	require.Equal(t, 2, FilteredBattleLogCount(log, BattleReportEventFilterKills))
}

// TestBattleLogCapTrimsInBatches 验证战报日志只在攒满 50000+4096 条后成批裁剪，保留最新 50000 条。
// 场景：长时间炮战每发命中都 RecordBattleLog，逐条整体复制会卡帧，因此按 battleLogTrimBatch 分批裁。
// 预期：首轮裁剪后长度 50000 且首条 Tick=4096；再记一条只增到 50001，不触发第二次复制。
// 覆盖：预置 54095 条按 idx 标记 Tick 的日志，两次 RecordBattleLog 后断言长度与 BattleLog[0].Tick。
func TestBattleLogCapTrimsInBatches(t *testing.T) {
	stats := NewBattleStats()
	stats.BattleLog = make([]BattleLogEntry, battleLogMaxEntries+battleLogTrimBatch-1)
	for idx := range stats.BattleLog {
		stats.BattleLog[idx].Tick = int64(idx)
	}

	stats.RecordBattleLog(BattleLogEntry{
		Tick: int64(len(stats.BattleLog)), ActorName: "A", TargetName: "B", Action: BattleLogShipHit,
	})
	require.Len(t, stats.BattleLog, battleLogMaxEntries)
	require.Equal(t, int64(battleLogTrimBatch), stats.BattleLog[0].Tick)

	// 下一发不会再次复制整份日志，等再攒满一批才裁剪。
	stats.RecordBattleLog(BattleLogEntry{
		Tick: int64(battleLogMaxEntries + battleLogTrimBatch), ActorName: "A", TargetName: "C",
		Action: BattleLogShipHit,
	})
	require.Len(t, stats.BattleLog, battleLogMaxEntries+1)
}

// TestBattleSideStatsCountsDistinctRecordedNames 验证舰级/机型计数取初始、建造、损失三张表的并集去重。
// 场景：A 同时出现在初始与损失表、B 同时在初始与建造表、C 只在建造、D 只在损失；
// 战报页签数量必须等于真实出现过的名称数，不能把三张表的计数相加。
// 预期：舰级 4、机型 4；若把表值累加会得到 6，页签与明细行数会不一致。
// 覆盖：表驱动分别构造舰级/机型三张 map，调用 ShipClassCount/PlaneTypeCount 断言结果。
func TestBattleSideStatsCountsDistinctRecordedNames(t *testing.T) {
	classStats := BattleSideStats{
		ShipsInitialByClass: map[string]int{"A": 1, "B": 1},
		ShipsBuiltByClass:   map[string]int{"B": 1, "C": 1},
		ShipsLostByClass:    map[string]int{"A": 1, "D": 1},
	}
	planeStats := BattleSideStats{
		PlanesInitialByType: map[string]int{"F4F": 1, "SBD": 1},
		PlanesBuiltByType:   map[string]int{"SBD": 1, "TBF": 1},
		PlanesLostByType:    map[string]int{"F4F": 1, "Zero": 1},
	}

	for _, tc := range []struct {
		name string
		got  int
		want int
	}{
		{name: "ship classes", got: classStats.ShipClassCount(), want: 4},
		{name: "plane types", got: planeStats.PlaneTypeCount(), want: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.got)
		})
	}
}
