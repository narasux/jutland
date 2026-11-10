package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// foggedState 构造一个开启迷雾、双方视野图都为空（初始不可见）的任务状态。
func foggedState() *MissionState {
	ms := &MissionState{
		Core:   MissionCoreState{SimTick: 100, FogOfWar: true},
		Player: MissionPlayerState{CurPlayer: faction.HumanAlpha},
	}
	ms.Player.Visions = map[faction.Player]*FactionVision{
		faction.HumanAlpha:    NewFactionVision(16, 16),
		faction.ComputerAlpha: NewFactionVision(16, 16),
	}
	return ms
}

func TestCanSeeShipSurfaceFollowsFog(t *testing.T) {
	ms := foggedState()
	enemy := &objUnit.BattleShip{
		BelongPlayer: faction.ComputerAlpha, CurHP: 100, CurPos: objPos.NewR(5.5, 5.5),
	}
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, enemy), "迷雾图上没盖到就看不见")

	ms.Player.Visions[faction.HumanAlpha].Visible[5*16+5] = 1
	require.True(t, ms.CanSeeShip(faction.HumanAlpha, enemy))

	// 己方恒可见，且不受迷雾影响
	own := &objUnit.BattleShip{BelongPlayer: faction.HumanAlpha, CurHP: 100, CurPos: objPos.NewR(9.5, 9.5)}
	require.True(t, ms.CanSeeShip(faction.HumanAlpha, own))
}

func TestCanSeeShipSubmergedRequiresDetection(t *testing.T) {
	ms := foggedState()
	ms.Core.SimTick = 100
	sub := &objUnit.BattleShip{
		BelongPlayer: faction.ComputerAlpha, CurHP: 100,
		CurPos: objPos.NewR(5.5, 5.5), Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged,
	}
	// 视野格全亮也不该看见水下潜艇
	ms.Player.Visions[faction.HumanAlpha].Visible[5*16+5] = 1
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, sub))

	sub.MarkDetected(faction.HumanAlpha, 150, sub.CurPos)
	require.True(t, ms.CanSeeShip(faction.HumanAlpha, sub))
	// 到期即失效
	ms.Core.SimTick = 150
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, sub))
}

func TestCanSeeShipIgnoreFogRevealsAll(t *testing.T) {
	ms := foggedState()
	ms.Player.IgnoreFog = true
	// 第三方潜艇：既不是当前玩家的，也不是电脑的
	sub := &objUnit.BattleShip{
		BelongPlayer: faction.HumanBeta, CurHP: 100,
		CurPos: objPos.NewR(1.5, 1.5), Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged,
	}
	require.True(t, ms.CanSeeShip(faction.HumanAlpha, sub))
	// 秘籍只揭开当前玩家，电脑仍要靠探测
	require.False(t, ms.CanSeeShip(faction.ComputerAlpha, sub))
}

func TestRevealsSubmarineIsVisualOnly(t *testing.T) {
	ms := foggedState()
	ms.Core.FogOfWar = false
	sub := &objUnit.BattleShip{
		BelongPlayer: faction.ComputerAlpha, CurHP: 100,
		CurPos: objPos.NewR(4.5, 4.5), Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged,
	}

	// 秘籍没开：不可见、也不暴露
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, sub))
	require.False(t, ms.RevealsSubmarine(faction.HumanAlpha, sub))

	ms.Player.RevealSubmarines = true
	require.True(t, ms.RevealsSubmarine(faction.HumanAlpha, sub))
	// 只是绘制层暴露，战斗可见性必须保持原样
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, sub))

	// 己方潜艇、水面/未下潜潜艇不需要红框
	own := &objUnit.BattleShip{
		BelongPlayer: faction.HumanAlpha, CurHP: 100, Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged,
	}
	require.False(t, ms.RevealsSubmarine(faction.HumanAlpha, own))

	surfaced := &objUnit.BattleShip{
		BelongPlayer: faction.ComputerAlpha, CurHP: 100,
		CurPos: objPos.NewR(6.5, 6.5), Type: objUnit.ShipTypeSubmarine,
	}
	require.True(t, ms.CanSeeShip(faction.HumanAlpha, surfaced))
	require.False(t, ms.RevealsSubmarine(faction.HumanAlpha, surfaced), "已经可见的潜艇不用再加红框")
}
