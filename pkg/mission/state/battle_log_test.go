package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

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

func TestRecordShipLostLogsAttackerAndAircraft(t *testing.T) {
	stats := NewBattleStats()
	ship := &objUnit.BattleShip{
		Name: "Carrier", Tonnage: 20000, BelongPlayer: faction.HumanAlpha,
		Type: objUnit.ShipTypeAircraftCarrier,
		Aircraft: objUnit.ShipAircraft{Groups: []objUnit.PlaneGroup{
			{Name: "F4F", CurCount: 3},
		}},
		LastAttackerName: "Enemy BB", LastAttackerPlayer: faction.ComputerAlpha, SunkByHit: true,
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

func TestRecordPlaneLostLogsShotDownOrLoss(t *testing.T) {
	stats := NewBattleStats()
	plane := &objUnit.Plane{
		Name: "F4F", BelongPlayer: faction.HumanAlpha,
		LastAttackerName: "A6M", LastAttackerPlayer: faction.ComputerAlpha,
		LastAttackerIsPlane: true, ShotDown: true,
	}
	stats.RecordPlaneLost(plane)
	require.Len(t, stats.BattleLog, 1)
	require.Equal(t, BattleLogPlaneShotDown, stats.BattleLog[0].Action)
	require.Equal(t, "A6M", stats.BattleLog[0].ActorName)
	require.Equal(t, "F4F", stats.BattleLog[0].TargetName)
	require.True(t, stats.BattleLog[0].TargetIsPlane)

	crash := &objUnit.Plane{
		Name: "SBD", BelongPlayer: faction.HumanAlpha,
		LastAttackerName: "A6M", LastAttackerPlayer: faction.ComputerAlpha,
	}
	stats.RecordPlaneLost(crash)
	require.Len(t, stats.BattleLog, 2)
	require.Equal(t, BattleLogPlaneLost, stats.BattleLog[1].Action)
	require.Empty(t, stats.BattleLog[1].ActorName)
}

func TestBattleSideStatsShipClassCount(t *testing.T) {
	stats := BattleSideStats{
		ShipsInitialByClass: map[string]int{"A": 1, "B": 1},
		ShipsBuiltByClass:   map[string]int{"B": 1, "C": 1},
		ShipsLostByClass:    map[string]int{"A": 1, "D": 1},
	}
	require.Equal(t, 4, stats.ShipClassCount())
}

func TestBattleSideStatsPlaneTypeCount(t *testing.T) {
	stats := BattleSideStats{
		PlanesInitialByType: map[string]int{"F4F": 1, "SBD": 1},
		PlanesBuiltByType:   map[string]int{"SBD": 1, "TBF": 1},
		PlanesLostByType:    map[string]int{"F4F": 1, "Zero": 1},
	}
	require.Equal(t, 4, stats.PlaneTypeCount())
}

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
