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
