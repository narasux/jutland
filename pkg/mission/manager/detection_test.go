package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func newDetectionState(sub, detector *objUnit.BattleShip) *state.MissionState {
	ships := map[string]*objUnit.BattleShip{sub.Uid: sub}
	if detector != nil {
		ships[detector.Uid] = detector
	}
	return &state.MissionState{
		Core:  state.MissionCoreState{SimTick: 100},
		Arena: state.MissionArenaState{Ships: ships},
	}
}

func TestSubmarineDetectionByEscortSonar(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 100, Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged, CurPos: objPos.NewR(10, 10),
	}
	dd := &objUnit.BattleShip{
		Uid: "dd", CurHP: 100, Type: objUnit.ShipTypeDestroyer,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(16.9, 10),
	}
	ms := newDetectionState(sub, dd)
	m := &MissionManager{state: ms}

	m.updateSubmarineDetection()
	require.True(t, ms.CanSeeShip(faction.ComputerAlpha, sub), "7 格内驱逐舰应点亮潜艇")
	require.False(t, ms.CanSeeShip(faction.HumanAlpha, sub), "另一方不应共享探测")

	// 驱逐舰拉开到 7 格外：记忆期内仍可见，到期后消失
	dd.CurPos = objPos.NewR(20, 10)
	m.updateSubmarineDetection()
	require.True(t, ms.CanSeeShip(faction.ComputerAlpha, sub), "8 秒记忆内仍应可见")
	ms.Core.SimTick += objUnit.ReloadTicks(objUnit.SubmarineDetectionMemorySeconds) + 1
	require.False(t, ms.CanSeeShip(faction.ComputerAlpha, sub), "记忆到期应重新隐身")
}

func TestSubmarineDetectionRadiusByShipType(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 100, Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged, CurPos: objPos.NewR(10, 10),
	}
	// 巡洋舰声呐 4 格，6 格处探不到
	ca := &objUnit.BattleShip{
		Uid: "ca", CurHP: 100, Type: objUnit.ShipTypeCruiser,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(16, 10),
	}
	ms := newDetectionState(sub, ca)
	m := &MissionManager{state: ms}
	m.updateSubmarineDetection()
	require.False(t, ms.CanSeeShip(faction.ComputerAlpha, sub))

	ca.CurPos = objPos.NewR(13.9, 10)
	m.updateSubmarineDetection()
	require.True(t, ms.CanSeeShip(faction.ComputerAlpha, sub))
}

func TestSubmarineDetectionByPlaneAndSurfacingClears(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 100, Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged, CurPos: objPos.NewR(10, 10),
	}
	ms := newDetectionState(sub, nil)
	ms.Arena.Planes = map[string]*objUnit.Plane{
		"p": {Uid: "p", CurHP: 10, BelongPlayer: faction.ComputerAlpha,
			CurPos: objPos.NewR(11.9, 10)},
	}
	m := &MissionManager{state: ms}
	m.updateSubmarineDetection()
	require.True(t, ms.CanSeeShip(faction.ComputerAlpha, sub), "2 格内飞机目视应点亮")

	// 上浮后清空所有探测记忆
	sub.DiveState = objUnit.DiveStateSurface
	m.updateSubmarineDetection()
	require.False(t, sub.DetectedByAny(ms.Core.SimTick))
}
