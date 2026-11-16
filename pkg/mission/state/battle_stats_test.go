package state

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

func newBattleTestShip(
	uid, name string,
	player faction.Player,
	tonnage float64,
	groups ...objUnit.PlaneGroup,
) *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Uid:          uid,
		Name:         name,
		BelongPlayer: player,
		Tonnage:      tonnage,
		Aircraft:     objUnit.ShipAircraft{Groups: groups},
	}
}

// TestBattleStatsCarrierLossCountsHangarPlanesOnce 验证沉没航母的机库存量计入损失并清零，重复调用不二次记账。
// 场景：Akagi 沉没时机库剩 A6M 5 架、B5N 3 架（MaxCount 20/18），沉船处理可能被多个调用点重入。
// 预期：ShipsLost=1、SunkTonnage=36500、ShipsLostByClass["Akagi"]=1、PlanesLost/WithShip=8、机型 5/3。
// 覆盖：先 RecordInitialShip 再两次 RecordShipLost，断言各计数与两组 CurCount 归零，PlanesLost 仍为 8。
func TestBattleStatsCarrierLossCountsHangarPlanesOnce(t *testing.T) {
	stats := NewBattleStats()
	carrier := newBattleTestShip(
		"carrier", "Akagi", faction.HumanAlpha, 36500,
		objUnit.PlaneGroup{Name: "A6M", MaxCount: 20, CurCount: 5},
		objUnit.PlaneGroup{Name: "B5N", MaxCount: 18, CurCount: 3},
	)
	stats.RecordInitialShip(carrier)
	stats.RecordShipLost(carrier)

	own := stats.Side(faction.HumanAlpha)
	require.Equal(t, 1, own.ShipsLost)
	require.Equal(t, 36500.0, own.SunkTonnage)
	require.Equal(t, 1, own.ShipsLostByClass["Akagi"])
	require.Equal(t, 8, own.PlanesLost)
	require.Equal(t, 8, own.PlanesLostWithShip)
	require.Equal(t, 5, own.PlanesLostByType["A6M"])
	require.Equal(t, 3, own.PlanesLostByType["B5N"])
	require.Zero(t, carrier.Aircraft.Groups[0].CurCount)
	require.Zero(t, carrier.Aircraft.Groups[1].CurCount)

	// 机库已清空，重复调用不会把随舰飞机再记一遍。
	stats.RecordShipLost(carrier)
	require.Equal(t, 8, stats.Side(faction.HumanAlpha).PlanesLost)
}

// TestBattleStatsFinalizeIsIdempotentAndCountsStrandedPlanes 验证结算幂等，并补记失去基地的在空飞机。
// 场景：SimTick=120 时取胜，A6M stranded 的母舰 sunk-carrier 已沉，safe 仍挂靠 base，两次结算不能互相覆盖。
// 预期：Finalized、EndTick=120、EndReason=victory、PlanesLost=1；第二次以 defeat/999 结算后三者不变。
// 覆盖：手工组装 MissionState 的 Ships/Planes，调 FinalizeBattleStats 两次，断言结束时间、原因与损失。
func TestBattleStatsFinalizeIsIdempotentAndCountsStrandedPlanes(t *testing.T) {
	base := &objUnit.BattleShip{Uid: "base", CurHP: 100, BelongPlayer: faction.HumanAlpha}
	ms := &MissionState{
		Core: MissionCoreState{SimTick: 120},
		Arena: MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{base.Uid: base},
			Planes: map[string]*objUnit.Plane{
				"stranded": {
					Uid: "stranded", Name: "A6M", BelongPlayer: faction.HumanAlpha,
					BelongShip: "sunk-carrier", CurHP: 10,
				},
				"safe": {
					Uid: "safe", Name: "A6M", BelongPlayer: faction.HumanAlpha,
					BelongShip: base.Uid, CurHP: 10,
				},
			},
		},
	}
	ms.Stats = NewBattleStats()
	ms.FinalizeBattleStats(BattleEndVictory)

	require.True(t, ms.Stats.Finalized)
	require.Equal(t, int64(120), ms.Stats.EndTick)
	require.Equal(t, BattleEndVictory, ms.Stats.EndReason)
	own := ms.Stats.Side(faction.HumanAlpha)
	require.Equal(t, 1, own.PlanesLost)
	require.Equal(t, 1, own.PlanesLostByType["A6M"])

	// 再次结算不得覆盖结束时间 / 原因，也不得重复补记飞机。
	ms.Core.SimTick = 999
	ms.FinalizeBattleStats(BattleEndDefeat)
	require.Equal(t, int64(120), ms.Stats.EndTick)
	require.Equal(t, BattleEndVictory, ms.Stats.EndReason)
	require.Equal(t, 1, ms.Stats.Side(faction.HumanAlpha).PlanesLost)
}

// TestRecordInitialBattleStatsCollectsShipsAndAirfields 验证开局统计采集舰船与机场库存，并保存开局资金。
// 场景：NewMissionState 初始化后调用；Akagi 机库 4 架 A6M、敌机场 af 有 2 架 B5N，战后又建 Shokaku。
// 预期：FundsStart=5000、初始 1 舰 4 机、敌方 2 架 B5N；建造侧 1 舰 18 机、RemainingPlanes=22。
// 覆盖：调 recordInitialBattleStats 后 RecordShipBuilt，断言 FundsStart、初始/建造计数与 RemainingPlanes。
func TestRecordInitialBattleStatsCollectsShipsAndAirfields(t *testing.T) {
	carrier := newBattleTestShip(
		"carrier", "Akagi", faction.HumanAlpha, 36500,
		objUnit.PlaneGroup{Name: "A6M", MaxCount: 4, CurCount: 4},
	)
	airfield := &objBuilding.Airfield{
		Uid:          "af",
		BelongPlayer: faction.ComputerAlpha,
		Aircraft: objUnit.ShipAircraft{Groups: []objUnit.PlaneGroup{
			{Name: "B5N", MaxCount: 4, CurCount: 2},
		}},
	}
	ms := &MissionState{
		Player: MissionPlayerState{CurFunds: 5000},
		Arena: MissionArenaState{
			Ships:     map[string]*objUnit.BattleShip{carrier.Uid: carrier},
			Airfields: map[string]*objBuilding.Airfield{airfield.Uid: airfield},
		},
	}
	ms.recordInitialBattleStats()
	ms.Stats.RecordShipBuilt(newBattleTestShip(
		"built", "Shokaku", faction.HumanAlpha, 32000,
		objUnit.PlaneGroup{Name: "A6M", MaxCount: 18, CurCount: 18},
	))

	require.Equal(t, int64(5000), ms.Stats.FundsStart)
	own := ms.Stats.Side(faction.HumanAlpha)
	require.Equal(t, 1, own.ShipsInitial)
	require.Equal(t, 4, own.PlanesInitial)
	require.Equal(t, 4, own.PlanesInitialByType["A6M"])
	require.Equal(t, 1, own.ShipsInitialByClass["Akagi"])
	require.Equal(t, 2, ms.Stats.Side(faction.ComputerAlpha).PlanesInitial)
	require.Equal(t, 2, ms.Stats.Side(faction.ComputerAlpha).PlanesInitialByType["B5N"])
	require.Equal(t, 1, own.ShipsBuilt)
	require.Equal(t, 32000.0, own.BuiltTonnage)
	require.Equal(t, 1, own.ShipsBuiltByClass["Shokaku"])
	require.Equal(t, 18, own.PlanesBuilt)
	require.Equal(t, 22, own.RemainingPlanes())
}

// TestBattleSideStatsDerivedValuesClampAtZero 验证剩余舰机、吨位在三项差值为负时夹到 0，保存率可回退。
// 场景：损失多于投入（2+1-5 舰、10+2-20 机、1000+500-2000 吨），以及吨位未记录的测试单位。
// 预期：三项剩余均夹到 0；全零吨位有损失 rate=0；仅数量 4-1→0.75、混合 2-1→0.5；空记录 ok=false。
// 覆盖：四组 BattleSideStats 分别断言 RemainingShips/Planes、SurvivingTonnage 与 SurvivalRate 的 (rate, ok)。
func TestBattleSideStatsDerivedValuesClampAtZero(t *testing.T) {
	side := BattleSideStats{
		ShipsInitial: 2, ShipsBuilt: 1, ShipsLost: 5,
		PlanesInitial: 10, PlanesBuilt: 2, PlanesLost: 20,
		InitialTonnage: 1000, BuiltTonnage: 500, SunkTonnage: 2000,
	}
	require.Zero(t, side.RemainingShips())
	require.Zero(t, side.RemainingPlanes())
	require.Zero(t, side.SurvivingTonnage())

	rate, ok := side.SurvivalRate()
	require.True(t, ok)
	require.Zero(t, rate)

	countOnly := BattleSideStats{ShipsInitial: 4, ShipsLost: 1}
	rate, ok = countOnly.SurvivalRate()
	require.True(t, ok)
	require.InDelta(t, 0.75, rate, 1e-9)

	// 有损失但没记到吨位（测试对象 / 特殊单位）时同样回退按数量。
	mixed := BattleSideStats{ShipsInitial: 2, ShipsLost: 1, InitialTonnage: 1000}
	rate, ok = mixed.SurvivalRate()
	require.True(t, ok)
	require.InDelta(t, 0.5, rate, 1e-9)

	_, ok = (BattleSideStats{}).SurvivalRate()
	require.False(t, ok)
}

// TestBattleStatsExchangeRatio 验证战损交换比优先按吨位、缺吨位回退舰数，并处理零损失与全零场景。
// 场景：结算页展示“敌方损失/我方损失”；特殊单位沉没时吨位为 0，混编损失会让吨位口径失效。
// 预期：1000 vs 4900→4.9；无损失 ok=false；我方 0 沉→+Inf；2 vs 6 艘→3；1 vs 1 且敌方 500 吨→1。
// 覆盖：表驱动 5 个 BattleStats，断言 ok、math.Inf 分支与 InDelta(1e-9)。
func TestBattleStatsExchangeRatio(t *testing.T) {
	tests := []struct {
		name  string
		own   BattleSideStats
		enemy BattleSideStats
		want  float64
		ok    bool
	}{
		{
			name:  "tonnage",
			own:   BattleSideStats{ShipsLost: 2, SunkTonnage: 1000},
			enemy: BattleSideStats{ShipsLost: 3, SunkTonnage: 4900},
			want:  4.9,
			ok:    true,
		},
		{
			name: "no losses",
			ok:   false,
		},
		{
			name:  "own zero loss is infinite",
			enemy: BattleSideStats{ShipsLost: 1, SunkTonnage: 100},
			want:  math.Inf(1),
			ok:    true,
		},
		{
			name:  "zero tonnage falls back to counts",
			own:   BattleSideStats{ShipsLost: 2},
			enemy: BattleSideStats{ShipsLost: 6},
			want:  3,
			ok:    true,
		},
		{
			name:  "mixed zero tonnage falls back to counts",
			own:   BattleSideStats{ShipsLost: 1},
			enemy: BattleSideStats{ShipsLost: 1, SunkTonnage: 500},
			want:  1,
			ok:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			own, enemy := tt.own, tt.enemy
			stats := BattleStats{Sides: map[faction.Player]*BattleSideStats{
				faction.HumanAlpha:    &own,
				faction.ComputerAlpha: &enemy,
			}}
			ratio, ok := stats.ExchangeRatio(faction.HumanAlpha, faction.ComputerAlpha)
			require.Equal(t, tt.ok, ok)
			if !ok {
				return
			}
			if math.IsInf(tt.want, 1) {
				require.True(t, math.IsInf(ratio, 1))
				return
			}
			require.InDelta(t, tt.want, ratio, 1e-9)
		})
	}
}

// TestBattleStatsFundsAndDamageAccumulate 验证伤害与资金只累计正数，NaN、负数与 0 都被忽略。
// 场景：命中可能传入溢出伤害、未破甲的 0、回滚负值与除零得到的 NaN；资金退款用负数调用。
// 预期：HumanAlpha DamageDealt=10、ComputerAlpha=0；FundsSpent=100、FundsEarned=30，NaN 不得污染总和。
// 覆盖：依次调 RecordDamage(10/-1/NaN/0)、RecordFundsSpent(100/-5)、RecordFundsEarned(30)，断言四字段。
func TestBattleStatsFundsAndDamageAccumulate(t *testing.T) {
	stats := NewBattleStats()
	stats.RecordDamage(faction.HumanAlpha, 10)
	stats.RecordDamage(faction.HumanAlpha, -1)
	stats.RecordDamage(faction.HumanAlpha, math.NaN())
	stats.RecordDamage(faction.ComputerAlpha, 0)
	require.Equal(t, 10.0, stats.Side(faction.HumanAlpha).DamageDealt)
	require.Zero(t, stats.Side(faction.ComputerAlpha).DamageDealt)

	stats.RecordFundsSpent(100)
	stats.RecordFundsSpent(-5)
	stats.RecordFundsEarned(30)
	require.Equal(t, int64(100), stats.FundsSpent)
	require.Equal(t, int64(30), stats.FundsEarned)
}
