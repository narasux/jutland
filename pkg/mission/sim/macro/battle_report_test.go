package macro

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/sim"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestBuildBattleReportExportsOverviewDetailsAndLog 验证 F3 战报从结算状态导出总览、明细和战况。
// 场景：珍珠港这类长战斗结束后，HTML 需要复刻游戏结算页的双方汇总、舰艇明细和最新战况；
// 战况日志按最新在前，完整日志则保持时间顺序供 jsonl 使用。
// 预期：4 张摘要卡、双方数量行、舰艇/飞机明细、1 条关键战果和 3 条完整战况；
// 最新事件在前，击沉文案含吨位，资金按初始 → 结余展开。
// 覆盖：构造含双方 Stats、3 条 BattleLog、2 个 Timeline 采样的 MissionState，调用 buildBattleReport。
func TestBuildBattleReportExportsOverviewDetailsAndLog(t *testing.T) {
	ms := &state.MissionState{
		Player: state.MissionPlayerState{
			CurPlayer: faction.HumanAlpha,
			CurEnemy:  faction.ComputerAlpha,
		},
		Stats: state.BattleStats{
			Sides: map[faction.Player]*state.BattleSideStats{
				faction.HumanAlpha: {
					ShipsInitial:        2,
					ShipsLost:           1,
					InitialTonnage:      1000,
					SunkTonnage:         500,
					DamageDealt:         1234,
					ShipsInitialByClass: map[string]int{"kaga": 1, "akagi": 1},
					ShipsLostByClass:    map[string]int{"kaga": 1},
					PlanesInitial:       10,
					PlanesLost:          2,
					PlanesLostWithShip:  1,
					PlanesInitialByType: map[string]int{"A6M2": 10},
					PlanesLostByType:    map[string]int{"A6M2": 2},
				},
				faction.ComputerAlpha: {
					ShipsInitial:        1,
					ShipsLost:           1,
					InitialTonnage:      30000,
					SunkTonnage:         30000,
					DamageDealt:         500,
					ShipsInitialByClass: map[string]int{"arizona": 1},
					ShipsLostByClass:    map[string]int{"arizona": 1},
				},
			},
			FundsStart:  100,
			FundsSpent:  20,
			FundsEarned: 5,
			BattleLog: []state.BattleLogEntry{
				{
					Tick: 10, ActorPlayer: faction.HumanAlpha,
					ActorName: "kaga", ActorType: string(objUnit.ShipTypeAircraftCarrier),
					TargetPlayer: faction.ComputerAlpha,
					TargetName:   "arizona", TargetType: string(objUnit.ShipTypeBattleShip),
					Action: state.BattleLogShipSunk, Tonnage: 30000,
					Hit:         objUnit.HitInfo{WeaponType: "torpedo", Diameter: 610, Damage: 1000},
					TotalDamage: 5000,
				},
				{
					Tick: 60, ActorPlayer: faction.ComputerAlpha,
					ActorName: "arizona", ActorType: string(objUnit.ShipTypeBattleShip),
					TargetPlayer: faction.HumanAlpha,
					TargetName:   "kaga", TargetType: string(objUnit.ShipTypeAircraftCarrier),
					Action: state.BattleLogShipHit, Count: 2,
					Hit: objUnit.HitInfo{WeaponType: "shell", Diameter: 356, Damage: 200},
				},
				{
					Tick: 120, TargetPlayer: faction.HumanAlpha,
					TargetName: "A6M2", TargetType: string(objUnit.PlaneTypeFighter), TargetIsPlane: true,
					Action: state.BattleLogPlaneLost,
				},
			},
			Timeline: []state.BattleTimelineSample{
				{
					Tick: 0,
					Sides: map[faction.Player]state.BattleTimelineSide{
						faction.HumanAlpha:    {ShipsRemaining: 2, PlanesRemaining: 10},
						faction.ComputerAlpha: {ShipsRemaining: 1},
					},
				},
				{
					Tick: 60,
					Sides: map[faction.Player]state.BattleTimelineSide{
						faction.HumanAlpha:    {ShipsRemaining: 1, PlanesRemaining: 8, ShipsLost: 1, PlanesLost: 2},
						faction.ComputerAlpha: {ShipsRemaining: 0, ShipsLost: 1},
					},
				},
			},
		},
	}
	report, fullLog := buildBattleReport(ms, "P1", state.MissionSuccess, 120)

	require.NotNil(t, report)
	require.Len(t, report.Cards, 4)
	require.Equal(t, "2 / 0 / 1 / 1", report.Own.ShipsLine)
	require.Equal(t, "10 / 0 / 2 / 8", report.Own.PlanesLine)
	require.Equal(t, "1", report.Own.PlanesWithShip)
	require.Equal(t, "100 → 85 (+5 / -20)", report.Own.Funds)
	require.Len(t, report.OwnShipRows, 2)
	require.Len(t, report.OwnPlaneRows, 1)
	require.NotEmpty(t, report.OwnShipRows[0].Name)
	require.Equal(t, 1, report.KillsTotal)
	require.Len(t, report.Kills, 1)
	require.Contains(t, report.Kills[0].Summary, "击沉")
	require.Contains(t, report.Kills[0].Summary, "30,000 吨")
	require.Equal(t, 3, report.EventsTotal)
	require.Len(t, report.Events, 3)
	require.Contains(t, report.Events[0].Summary, "损失")
	require.Equal(t, int64(120), report.Events[0].Tick)
	require.Len(t, report.Trend, 2)
	require.Equal(t, 1, report.Trend[1].OwnShips)
	require.Equal(t, 0, report.Trend[1].EnemyShips)
	require.Len(t, fullLog, 3)
	require.Equal(t, int64(10), fullLog[0].Tick)
	require.NotEmpty(t, report.Exchange)
	require.NotEmpty(t, report.Survival)
}

// TestLatestEventsKeepsNewestAndCountsOmitted 验证战报条数上限只保留最新记录并统计省略数。
// 场景：F3 完整战斗可能有数千条命中，HTML 不能全量展开；完整日志另写 jsonl，
// 因此切片必须取尾部、按最新在前返回，并准确报告被省略的旧记录数。
// 预期：5 条保留 2 条时返回 tick 4/3，omitted=3；limit<=0 时返回 nil 且 omitted=总数。
// 覆盖：构造 5 条递增 tick 的事件，调用 latestEvents 断言顺序、长度和省略数。
func TestLatestEventsKeepsNewestAndCountsOmitted(t *testing.T) {
	events := make([]sim.BattleReportEvent, 0, 5)
	for tick := int64(0); tick < 5; tick++ {
		events = append(events, sim.BattleReportEvent{Tick: tick})
	}
	got, omitted := latestEvents(events, 2)
	require.Len(t, got, 2)
	require.Equal(t, int64(4), got[0].Tick)
	require.Equal(t, int64(3), got[1].Tick)
	require.Equal(t, 3, omitted)

	got, omitted = latestEvents(events, 0)
	require.Nil(t, got)
	require.Equal(t, 5, omitted)
}

// TestBattleWeaponAndDamageTextStable 验证战报武器/伤害文案的关键分支。
// 场景：游戏内战报按弹药类型和平台生成「610mm 鱼雷 / 127mm 防空炮」，
// 小伤害保留小数，大伤害加千位分隔；这些文案直接进入 HTML 与 summary。
// 预期：鱼雷取精确口径、防空炮按目标为飞机识别、0.52 保留两位、30000 加逗号。
// 覆盖：构造 torpedo/shell 命中与 0.52/30000 两个伤害值，断言 battleWeaponText/battleDamageText。
func TestBattleWeaponAndDamageTextStable(t *testing.T) {
	torpedo := state.BattleLogEntry{
		Hit: objUnit.HitInfo{WeaponType: "torpedo", Diameter: 610},
	}
	require.Equal(t, "610mm 鱼雷", battleWeaponText(torpedo))
	aa := state.BattleLogEntry{
		TargetIsPlane: true,
		Hit:           objUnit.HitInfo{WeaponType: "shell", Diameter: 127},
	}
	require.Equal(t, "127mm 防空炮", battleWeaponText(aa))
	require.Equal(t, "0.52", battleDamageText(0.52))
	require.Equal(t, "30,000", battleDamageText(30000))
	require.True(t, strings.Contains(battleDamageText(1234.4), "1,234"))
}
