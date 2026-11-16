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

// TestSubmarineDetectionByEscortSonar 验证驱逐舰声呐点亮下潜潜艇、阵营隔离与 8 秒记忆到期。
// 场景：SimTick=100，潜艇在 (10,10)，ComputerAlpha 驱逐舰先在 6.9 格、再拉到 10 格；声呐 7 格、记忆 8 秒。
// 预期：首次探测后仅 ComputerAlpha 可见；拉远后记忆内仍可见，SimTick 超过 untilTick 后重新隐身。
// 覆盖：newDetectionState 放入 sub/dd，跑探测后断言 CanSeeShip 两个阵营及记忆到期前后。
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

// TestSubmarineDetectionRadiusByShipType 验证巡洋舰对下潜潜艇的声呐半径固定为 4 格。
// 场景：潜艇在 (10,10)，巡洋舰先在相距 6 格的 (16,10)，再靠近到相距 3.9 格的 (13.9,10)。
// 预期：6 格时超出 ShipSonarRangeCapital=4，不可见；3.9 格进入半径后 updateSubmarineDetection 点亮潜艇。
// 覆盖：newDetectionState 放 sub/ca，两次调用 updateSubmarineDetection 并断言 CanSeeShip 由 false 变 true。
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

// TestSubmarineDetectionByPlaneAndSurfacingClears 验证飞机目视 2 格点亮下潜潜艇，且上浮会清空探测记忆。
// 场景：潜艇在 (10,10) 且无舰船探测者，ComputerAlpha 飞机在 (11.9,10)；随后潜艇改为水面状态。
// 预期：首次探测后飞机方可见潜艇；上浮后再跑探测会清空记忆，DetectedByAny 返回 false。
// 覆盖：手工构造 2 格内的飞机，断言 CanSeeShip(ComputerAlpha) 与 DetectedByAny(SimTick)。
func TestSubmarineDetectionByPlaneAndSurfacingClears(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 100, Type: objUnit.ShipTypeSubmarine,
		DiveState: objUnit.DiveStateSubmerged, CurPos: objPos.NewR(10, 10),
	}
	ms := newDetectionState(sub, nil)
	ms.Arena.Planes = map[string]*objUnit.Plane{
		"p": {
			Uid: "p", CurHP: 10, BelongPlayer: faction.ComputerAlpha,
			CurPos: objPos.NewR(11.9, 10),
		},
	}
	m := &MissionManager{state: ms}
	m.updateSubmarineDetection()
	require.True(t, ms.CanSeeShip(faction.ComputerAlpha, sub), "2 格内飞机目视应点亮")

	// 上浮后清空所有探测记忆
	sub.DiveState = objUnit.DiveStateSurface
	m.updateSubmarineDetection()
	require.False(t, sub.DetectedByAny(ms.Core.SimTick))
}
