package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestContactFleetOnlyCountsVisibleEnemies 验证 ContactFleet 只统计 CanSeeShip 通过的敌舰，己方舰队恒可见。
// 场景：foggedState 只在 (5,5) 点亮；敌 akagi 在 (5.5,5.5)、yamato 在 (9.5,9.5)，己方 iowa 在 (1.5,1.5)。
// 预期：ComputerAlpha Total=1、Classes 仅 akagi；HumanAlpha 恒为 1；Fleet=2 说明底层未被过滤。
// 覆盖：三艘 BattleShip 放入 Arena.Ships，手动置 Visible[5*16+5]=1，断言 Total、Classes 与 Fleet 三处。
func TestContactFleetOnlyCountsVisibleEnemies(t *testing.T) {
	ms := foggedState()
	own := &objUnit.BattleShip{
		Uid: "own-1", Name: "iowa", BelongPlayer: faction.HumanAlpha,
		CurPos: objPos.NewR(1.5, 1.5),
	}
	visible := &objUnit.BattleShip{
		Uid: "enemy-1", Name: "akagi", BelongPlayer: faction.ComputerAlpha,
		CurPos: objPos.NewR(5.5, 5.5),
	}
	hidden := &objUnit.BattleShip{
		Uid: "enemy-2", Name: "yamato", BelongPlayer: faction.ComputerAlpha,
		CurPos: objPos.NewR(9.5, 9.5),
	}
	ms.Arena.Ships = map[string]*objUnit.BattleShip{
		own.Uid: own, visible.Uid: visible, hidden.Uid: hidden,
	}
	ms.Player.Visions[faction.HumanAlpha].Visible[5*16+5] = 1

	got := ms.ContactFleet(faction.ComputerAlpha)
	require.Equal(t, 1, got.Total)
	require.Len(t, got.Classes, 1)
	require.Equal(t, "akagi", got.Classes[0].Kind.Name)

	require.Equal(t, 1, ms.ContactFleet(faction.HumanAlpha).Total, "己方舰队始终可见")
	require.Equal(t, 2, ms.Fleet(faction.ComputerAlpha).Total, "Fleet 保持不过滤，供非 UI 逻辑使用")
}

// TestContactFleetMatchesFleetWhenFogOff 验证迷雾关闭时 ContactFleet 与 Fleet 等价，不再做可见性过滤。
// 场景：部分任务或调试关掉 FogOfWar，此时不应读取视野图，敌我全部可见；若仍走 CanSeeShip 会清空敌情。
// 预期：FogOfWar=false 时 ComputerAlpha 的 Total=2（akagi 与 yamato），失败意味着关雾局面敌情面板为空。
// 覆盖：关闭 foggedState 的 FogOfWar，放入两艘位于 (5.5,5.5)/(9.5,9.5) 的敌舰，断言 ContactFleet Total=2。
func TestContactFleetMatchesFleetWhenFogOff(t *testing.T) {
	ms := foggedState()
	ms.Core.FogOfWar = false
	ms.Arena.Ships = map[string]*objUnit.BattleShip{
		"a": {Uid: "a", Name: "akagi", BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(5.5, 5.5)},
		"b": {Uid: "b", Name: "yamato", BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(9.5, 9.5)},
	}

	require.Equal(t, 2, ms.ContactFleet(faction.ComputerAlpha).Total)
}

// TestContactFleetHonorsIgnoreFogCheat 验证 IgnoreFog 秘籍对当前玩家生效：视野外敌舰也计入 ContactFleet。
// 场景：调试/观战开启 IgnoreFog 全图可见；akagi 位于 (9.5,9.5)，视野图全空，正常规则下应不可见。
// 预期：ContactFleet(ComputerAlpha).Total=1，失败意味着秘籍只影响绘制，敌情面板仍是黑屏。
// 覆盖：在 foggedState 上置 Player.IgnoreFog=true，放入一艘视野外敌舰，断言 Total=1。
func TestContactFleetHonorsIgnoreFogCheat(t *testing.T) {
	ms := foggedState()
	ms.Player.IgnoreFog = true
	ms.Arena.Ships = map[string]*objUnit.BattleShip{
		"a": {Uid: "a", Name: "akagi", BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(9.5, 9.5)},
	}

	require.Equal(t, 1, ms.ContactFleet(faction.ComputerAlpha).Total)
}

// TestContactFleetSkipsUndetectedSubmarine 验证下潜潜艇即使所在格视野全亮也不计入接触，只认未过期的探测记忆。
// 场景：u-boat 在 (5.5,5.5) 下潜且视野格被点亮；SimTick=100 时标记探测至 110，随后推进到 120。
// 预期：探测前 Total=0、标记后 1、SimTick=120 时归 0；失败意味着潜艇被“透视”或探测不过期。
// 覆盖：ShipTypeSubmarine、DiveStateSubmerged 敌舰，置 Visible[5*16+5]=1，三次断言 ContactFleet。
func TestContactFleetSkipsUndetectedSubmarine(t *testing.T) {
	ms := foggedState()
	sub := &objUnit.BattleShip{
		Uid: "sub-1", Name: "u-boat", BelongPlayer: faction.ComputerAlpha,
		CurPos: objPos.NewR(5.5, 5.5), Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged,
	}
	ms.Arena.Ships = map[string]*objUnit.BattleShip{sub.Uid: sub}
	// 视野格全亮：水面舰会可见，但下潜且未探测的潜艇不应计入接触。
	ms.Player.Visions[faction.HumanAlpha].Visible[5*16+5] = 1
	require.Zero(t, ms.ContactFleet(faction.ComputerAlpha).Total)

	sub.MarkDetected(faction.HumanAlpha, ms.Core.SimTick+10, sub.CurPos)
	require.Equal(t, 1, ms.ContactFleet(faction.ComputerAlpha).Total)

	ms.Core.SimTick += 20
	require.Zero(t, ms.ContactFleet(faction.ComputerAlpha).Total, "探测到期后接触消失")
}

// TestContactFleetGroupsSameClass 验证 ContactFleet 按 Name 聚合同级舰：两艘 fletcher 合成一个 ShipClass。
// 场景：编成面板按舰级显示“dd ×2”；同型舰 Uid 不同但 Name 相同，若按 Uid 分组会展开成两行。
// 预期：Total=2、len(Classes)=1、Classes[0].Total=2，失败意味着同型舰在编成面板重复列行。
// 覆盖：关闭迷雾后放入两艘 Name="fletcher"、坐标 (1.5,1.5)/(2.5,2.5) 的敌舰，断言总数与分组数。
func TestContactFleetGroupsSameClass(t *testing.T) {
	ms := foggedState()
	ms.Core.FogOfWar = false
	ms.Arena.Ships = map[string]*objUnit.BattleShip{
		"a": {Uid: "a", Name: "fletcher", BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(1.5, 1.5)},
		"b": {Uid: "b", Name: "fletcher", BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(2.5, 2.5)},
	}

	got := ms.ContactFleet(faction.ComputerAlpha)
	require.Equal(t, 2, got.Total)
	require.Len(t, got.Classes, 1)
	require.Equal(t, 2, got.Classes[0].Total)
}
