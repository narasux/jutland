package building

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// registerTestShip 注册一艘最小战舰模板，NewShip/GetShipCost 依赖 ShipMap 中存在该名称。
func registerTestShip(t *testing.T, name string, fundsCost, timeCost int64) {
	t.Helper()
	old, had := objUnit.ShipMap[name]
	objUnit.ShipMap[name] = &objUnit.BattleShip{
		Name: name, TypeAbbr: "TS", Type: objUnit.ShipTypeBattleShip,
		FundsCost: fundsCost, TimeCost: timeCost,
		TotalHP: 100, CurHP: 100, Length: 128, Width: 20, MaxSpeed: 0.03,
	}
	t.Cleanup(func() {
		if had {
			objUnit.ShipMap[name] = old
		} else {
			delete(objUnit.ShipMap, name)
		}
	})
}

// TestReinforcePointSummonEnforcesLimitAndWhitelist 验证增援点只接受白名单舰型，且不超过在途上限。
// 场景：玩家/机场增援界面连续点选舰型；未提供的舰型或已排满队列时不能继续追加。
// 预期：未知舰型不入队；白名单舰型最多 MaxOncomingShip 艘；每艘记录 GetShipCost 的资金/时间。
// 覆盖：MaxOncomingShip=2 的增援点依次 Summon 未知/白名单/白名单/白名单，断言队列长度与费用。
func TestReinforcePointSummonEnforcesLimitAndWhitelist(t *testing.T) {
	const shipName = "test-reinforce-bb"
	registerTestShip(t, shipName, 100, 5)

	rp := NewReinforcePoint(
		objPos.New(10, 10), 0, objPos.New(12, 12),
		faction.HumanAlpha, 2, []string{shipName},
	)

	rp.Summon("unknown-ship")
	require.Empty(t, rp.OncomingShips)

	rp.Summon(shipName)
	rp.Summon(shipName)
	require.Len(t, rp.OncomingShips, 2)
	require.Equal(t, int64(100), rp.OncomingShips[0].FundsCost)
	require.Equal(t, int64(5), rp.OncomingShips[0].TimeCost)

	rp.Summon(shipName)
	require.Len(t, rp.OncomingShips, 2, "queue should stay at MaxOncomingShip")
}

// TestReinforcePointUpdateWaitsForFundsThenSpawns 验证增援点等资金、等计时完成后才出舰并出队。
// 场景：玩家资金不足时增援进度暂停；资金到位且计时结束后，应在集结点生成战舰并从队列移除。
// 预期：资金 99 时 Update 返回 nil 且队列不变；资金 100 且 StartedAt 已过 TimeCost 秒后返回战舰，
// 舰名/位置/朝向/阵营正确，队列长度减 1。
// 覆盖：Summon 后手动把 StartedAt 拨到过去，分别用 99/100 资金调用 Update。
func TestReinforcePointUpdateWaitsForFundsThenSpawns(t *testing.T) {
	const shipName = "test-reinforce-spawn-bb"
	registerTestShip(t, shipName, 100, 5)

	rp := NewReinforcePoint(
		objPos.New(10, 10), 45, objPos.New(12, 12),
		faction.HumanAlpha, 2, []string{shipName},
	)
	rp.Summon(shipName)

	uidGen := objUnit.NewShipUidGenerator(faction.HumanAlpha)
	require.Nil(t, rp.Update(uidGen, 99), "not enough funds should not spawn")
	require.Len(t, rp.OncomingShips, 1)

	rp.OncomingShips[0].StartedAt = time.Now().UnixMilli() - 6_000
	ship := rp.Update(uidGen, 100)
	require.NotNil(t, ship)
	require.Equal(t, shipName, ship.Name)
	require.Equal(t, objPos.New(10, 10), ship.CurPos)
	require.Equal(t, 45.0, ship.CurRotation)
	require.Equal(t, faction.HumanAlpha, ship.BelongPlayer)
	require.Empty(t, rp.OncomingShips)
}

// TestReinforcePointProgressClamps 验证增援进度在无在途舰时显示 0，超过 100 时截断到 100。
// 场景：单位面板/侧栏用 Progress 画进度条；未开始增援或计时异常时不能把进度画到 100 以外。
// 预期：空队列返回 0；Progress=150 返回 100；Progress=42.7 返回 42。
// 覆盖：构造空增援点与带单个 OncomingShip 的增援点，分别改 Progress 后断言。
func TestReinforcePointProgressClamps(t *testing.T) {
	rp := NewReinforcePoint(objPos.New(0, 0), 0, objPos.New(0, 0), faction.HumanAlpha, 1, nil)
	require.Zero(t, rp.Progress())

	rp.OncomingShips = []*OncomingShip{{Progress: 150}}
	require.Equal(t, 100, rp.Progress())

	rp.OncomingShips[0].Progress = 42.7
	require.Equal(t, 42, rp.Progress())
}

// TestOilPlatformAddRemoveAndLoadingCycle 验证油井货轮装卸幂等，以及装载完成后的进度重置。
// 场景：货轮进入/离开油井半径时由 updateBuildings 反复调用 AddShip/RemoveShip；
// 装载完成后油井应重新开始计时，而不是永久停在 100%。
// 预期：重复 AddShip 只保留一条记录；RemoveShip 删除记录；LoadingOilShip 首次 Update 设置 StartedAt，
// 到点后返回 true 并把 StartedAt/Progress 清零。
// 覆盖：同一货轮连续 AddShip 两次、RemoveShip 两次；再单独驱动 LoadingOilShip 的首次/完成两段。
func TestOilPlatformAddRemoveAndLoadingCycle(t *testing.T) {
	op := NewOilPlatform(objPos.New(5, 5), 2, 7)
	ship := &objUnit.BattleShip{Uid: "cargo", CurPos: objPos.New(5, 5)}

	op.AddShip(ship)
	op.AddShip(ship)
	require.Len(t, op.LoadingOilShips, 1)
	require.Equal(t, 7, op.LoadingOilShips[ship.Uid].FundYield)
	require.Equal(t, int64(5), op.LoadingOilShips[ship.Uid].TimeCost)

	op.RemoveShip(ship.Uid)
	require.Empty(t, op.LoadingOilShips)
	op.RemoveShip("missing")
	require.Empty(t, op.LoadingOilShips)

	loading := &LoadingOilShip{FundYield: 7, TimeCost: 5}
	require.False(t, loading.Update(), "first update only starts the timer")
	require.NotZero(t, loading.StartedAt)

	loading.StartedAt = time.Now().UnixMilli() - 6_000
	require.True(t, loading.Update(), "elapsed beyond TimeCost should finish")
	require.Zero(t, loading.StartedAt)
	require.Zero(t, loading.Progress)
}
