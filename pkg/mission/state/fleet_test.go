package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

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

func TestContactFleetMatchesFleetWhenFogOff(t *testing.T) {
	ms := foggedState()
	ms.Core.FogOfWar = false
	ms.Arena.Ships = map[string]*objUnit.BattleShip{
		"a": {Uid: "a", Name: "akagi", BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(5.5, 5.5)},
		"b": {Uid: "b", Name: "yamato", BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(9.5, 9.5)},
	}

	require.Equal(t, 2, ms.ContactFleet(faction.ComputerAlpha).Total)
}

func TestContactFleetHonorsIgnoreFogCheat(t *testing.T) {
	ms := foggedState()
	ms.Player.IgnoreFog = true
	ms.Arena.Ships = map[string]*objUnit.BattleShip{
		"a": {Uid: "a", Name: "akagi", BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(9.5, 9.5)},
	}

	require.Equal(t, 1, ms.ContactFleet(faction.ComputerAlpha).Total)
}

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
