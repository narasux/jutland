package instruction

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// registerSummonTestShip 注册一艘增援/召唤测试舰，避免依赖特定关卡配置。
func registerSummonTestShip(t *testing.T, name string) {
	t.Helper()
	old, had := objUnit.ShipMap[name]
	objUnit.ShipMap[name] = &objUnit.BattleShip{
		Name: name, TypeAbbr: "TS", Type: objUnit.ShipTypeDestroyer,
		FundsCost: 10, TimeCost: 1, TotalHP: 10, CurHP: 10,
	}
	t.Cleanup(func() {
		if had {
			objUnit.ShipMap[name] = old
		} else {
			delete(objUnit.ShipMap, name)
		}
	})
}

// TestShipSummonExecValidatesPointLimitAndResolvesName 验证增援指令的缺失点/上限/随机选舰三条分支。
// 场景：AI 对增援点下 ShipSummon；点已销毁或队列已满时应返回错误，未指定舰名时应从白名单随机取一艘。
// 预期：缺失 uid 报 "not found"；已满报 "limit reached"；未指定名称成功入队并 Executed=true。
// 覆盖：构造单个增援点，分别执行缺失/满队/空舰名三次 Exec，断言错误、队列与状态。
func TestShipSummonExecValidatesPointLimitAndResolvesName(t *testing.T) {
	const shipName = "test-summon-ship"
	registerSummonTestShip(t, shipName)

	rp := objBuilding.NewReinforcePoint(
		objPos.New(5, 5), 0, objPos.New(6, 6), faction.HumanAlpha, 1, []string{shipName},
	)
	ms := &state.MissionState{Arena: state.MissionArenaState{
		ReinforcePoints: map[string]*objBuilding.ReinforcePoint{rp.Uid: rp},
	}}

	missing := NewShipSummon("missing", shipName)
	require.ErrorContains(t, missing.Exec(ms), "not found")
	require.False(t, missing.Executed())

	rp.Summon(shipName)
	require.Len(t, rp.OncomingShips, 1)
	full := NewShipSummon(rp.Uid, shipName)
	require.ErrorContains(t, full.Exec(ms), "limit reached")
	require.False(t, full.Executed())

	rp.OncomingShips = nil
	random := NewShipSummon(rp.Uid, "")
	require.NoError(t, random.Exec(ms))
	require.True(t, random.Executed())
	require.Len(t, rp.OncomingShips, 1)
	require.Equal(t, shipName, rp.OncomingShips[0].Name)
	require.Equal(t, GenInstrUid(NameShipSummon, rp.Uid), random.Uid())
	require.Contains(t, random.String(), shipName)
}
