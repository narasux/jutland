package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestSampleBattleTimelineThrottlesAndSnapshots 验证趋势采样按秒节流、快照复制双方累计值。
// 场景：模拟推进时每拍调用 SampleBattleTimeline；0 拍开始，59 拍仍未满 1 秒，60/120 拍各落一个点，
// 期间人类方建造 2 舰、损失 3 舰（初始 5），电脑方损失 3 架；历史点必须保留当时数值。
// 预期：0/59 后只有 1 点；60 后第 2 点 Tick=60 且记录 2 建造 / 3 损失 / 4 在役 / 100 沉没吨位 /
// 1300 在役吨位 / 12.5 伤害；首点仍全零。
// 覆盖：调用 SampleBattleTimeline 于 0/30/60/90/120，直接改 Sides 计数模拟记账，断言点数与各字段快照。
func TestSampleBattleTimelineThrottlesAndSnapshots(t *testing.T) {
	stats := NewBattleStats()
	stats.SampleBattleTimeline(0)
	require.Len(t, stats.Timeline, 1)
	require.Equal(t, int64(0), stats.Timeline[0].Tick)

	own := stats.Sides[faction.HumanAlpha]
	own.ShipsInitial = 5
	own.ShipsBuilt = 2
	own.ShipsLost = 3
	own.InitialTonnage = 1000
	own.BuiltTonnage = 400
	own.SunkTonnage = 100
	own.DamageDealt = 12.5
	stats.Sides[faction.ComputerAlpha].PlanesLost = 3

	stats.SampleBattleTimeline(30)
	require.Len(t, stats.Timeline, 1)
	stats.SampleBattleTimeline(60)
	require.Len(t, stats.Timeline, 2)
	second := stats.Timeline[1]
	require.Equal(t, int64(60), second.Tick)
	require.Equal(t, 2, second.TimelineSide(faction.HumanAlpha).ShipsBuilt)
	require.Equal(t, 3, second.TimelineSide(faction.HumanAlpha).ShipsLost)
	require.Equal(t, 4, second.TimelineSide(faction.HumanAlpha).ShipsRemaining)
	require.Equal(t, 1300.0, second.TimelineSide(faction.HumanAlpha).SurvivingTonnage)
	require.Equal(t, 3, second.TimelineSide(faction.ComputerAlpha).PlanesLost)
	require.Equal(t, 100.0, second.TimelineSide(faction.HumanAlpha).SunkTonnage)
	require.Equal(t, 12.5, second.TimelineSide(faction.HumanAlpha).DamageDealt)

	// 同一秒内继续推进不新增点；跨到下一采样窗才追加。
	stats.SampleBattleTimeline(90)
	require.Len(t, stats.Timeline, 2)
	stats.SampleBattleTimeline(120)
	require.Len(t, stats.Timeline, 3)

	// 首个采样点是独立副本，后续记账不能把它改成非零。
	first := stats.Timeline[0].TimelineSide(faction.HumanAlpha)
	require.Zero(t, first.ShipsBuilt)
	require.Zero(t, first.SunkTonnage)
}

// TestSampleBattleTimelineForceRefreshesSameTick 验证强制采样在同一拍只刷新最新快照、不追加重复 X 坐标。
// 场景：结算与开局都用 force 采样；若同一拍先被普通采样写入，结算补记的随舰飞机损失必须反映到终点。
// 预期：普通采样同拍调用不改变数值；force 调用后点数不变但最新值更新；时间轴 Tick 严格递增。
// 覆盖：0 拍写入首点后修改 PlanesLost=4，先普通同拍采样断言仍是 0，再 sampleTimeline(0,true) 断言变 4。
func TestSampleBattleTimelineForceRefreshesSameTick(t *testing.T) {
	stats := NewBattleStats()
	stats.SampleBattleTimeline(0)
	require.Len(t, stats.Timeline, 1)

	stats.Sides[faction.HumanAlpha].PlanesLost = 4
	stats.SampleBattleTimeline(0)
	require.Len(t, stats.Timeline, 1)
	require.Zero(t, stats.Timeline[0].TimelineSide(faction.HumanAlpha).PlanesLost)

	stats.sampleTimeline(0, true)
	require.Len(t, stats.Timeline, 1)
	require.Equal(t, 4, stats.Timeline[0].TimelineSide(faction.HumanAlpha).PlanesLost)

	stats.SampleBattleTimeline(60)
	require.Len(t, stats.Timeline, 2)
	require.Less(t, stats.Timeline[0].Tick, stats.Timeline[1].Tick)
}

// TestBattleTimelineDecimatesWhenFull 验证采样点达到上限后抽稀旧点，且保留首尾与严格递增的时间轴。
// 场景：长时间战斗按秒采样会产生数千个点；不抽稀会让内存随游戏时长增长，抽稀则只能降低旧段分辨率。
// 预期：点数始终不超过 battleTimelineMaxSamples；首点 Tick=0、末点等于最后一次采样拍，Tick 严格递增。
// 覆盖：连续采样 1441 个点（上限两倍以上），断言长度上限、首尾 Tick，并逐个比较相邻 Tick。
func TestBattleTimelineDecimatesWhenFull(t *testing.T) {
	stats := NewBattleStats()
	total := battleTimelineMaxSamples*2 + 1
	lastTick := int64(0)
	for idx := 0; idx < total; idx++ {
		lastTick = int64(idx) * battleTimelineSampleTicks
		stats.SampleBattleTimeline(lastTick)
	}
	require.LessOrEqual(t, len(stats.Timeline), battleTimelineMaxSamples)
	require.Greater(t, len(stats.Timeline), battleTimelineMaxSamples/4)
	require.Equal(t, int64(0), stats.Timeline[0].Tick)
	require.Equal(t, lastTick, stats.Timeline[len(stats.Timeline)-1].Tick)
	for idx := 1; idx < len(stats.Timeline); idx++ {
		require.Less(t, stats.Timeline[idx-1].Tick, stats.Timeline[idx].Tick)
	}
}

// TestFinalizeBattleStatsAppendsTimelineEndPoint 验证结算补记终点采样，且重复结算不会追加第二个终点。
// 场景：战斗在 600 拍结束，最后一次周期采样可能停在 540 拍；结算页曲线必须延伸到 600 拍。
// 预期：结算后最后采样 Tick=600、EndTick=600；再次结算不改变点数与终点。
// 覆盖：先 SampleBattleTimeline(0)/(540)，再 FinalizeBattleStats，断言点数 3、末点 600；二次调用长度不变。
func TestFinalizeBattleStatsAppendsTimelineEndPoint(t *testing.T) {
	ms := &MissionState{
		Core:  MissionCoreState{SimTick: 540},
		Arena: MissionArenaState{},
	}
	ms.Stats = NewBattleStats()
	ms.Stats.SampleBattleTimeline(0)
	ms.Stats.SampleBattleTimeline(540)

	ms.Core.SimTick = 600
	ms.FinalizeBattleStats(BattleEndVictory)
	require.Len(t, ms.Stats.Timeline, 3)
	require.Equal(t, int64(600), ms.Stats.Timeline[2].Tick)
	require.Equal(t, int64(600), ms.Stats.EndTick)

	ms.FinalizeBattleStats(BattleEndDefeat)
	require.Len(t, ms.Stats.Timeline, 3)
	require.Equal(t, int64(600), ms.Stats.Timeline[2].Tick)
}

// TestRecordInitialBattleStatsSeedsTimelineAtZero 验证开局统计写入 0 拍采样点，曲线从原点开始。
// 场景：NewMissionState 完成对象初始化后调用 recordInitialBattleStats；如果首点不是 0，
// 曲线会从第一个事件的时间开始，玩家看不到开局到首次交火之间的平台段。
// 预期：Timeline 长度为 1、Tick=0，双方累计建造/损失均为 0。
// 覆盖：构造空舰队与空机场的 MissionState，调用 recordInitialBattleStats 后断言采样点。
func TestRecordInitialBattleStatsSeedsTimelineAtZero(t *testing.T) {
	ms := &MissionState{
		Arena: MissionArenaState{
			Ships:     map[string]*objUnit.BattleShip{},
			Airfields: map[string]*objBuilding.Airfield{},
		},
	}
	ms.recordInitialBattleStats()
	require.Len(t, ms.Stats.Timeline, 1)
	require.Equal(t, int64(0), ms.Stats.Timeline[0].Tick)
	require.Zero(t, ms.Stats.Timeline[0].TimelineSide(faction.HumanAlpha).ShipsBuilt)
	require.Zero(t, ms.Stats.Timeline[0].TimelineSide(faction.ComputerAlpha).PlanesLost)
}
