package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/tutorial"
)

// TestTutorialMissionSkipsWinLoss 验证只有玩家舰队的教学任务不会被自动判胜利。
// 场景：教学状态里只有一艘 HA 舰船、没有 P2 舰船；正常任务在这里会返回 MissionSuccess。
// 预期：updateMissionStatus 后状态仍为 MissionRunning，TutorialActive 为 true；
// 失败意味着教学关会在第一拍直接通关，玩家看不到教学卡。
// 覆盖：cameraTestState + TutorialBasicControls 控制器，只调用 updateMissionStatus 并断言状态。
func TestTutorialMissionSkipsWinLoss(t *testing.T) {
	ms := cameraTestState()
	ms.Core.MissionStatus = state.MissionRunning
	ms.Player.CurPlayer = faction.HumanAlpha
	ms.Arena.Ships = map[string]*objUnit.BattleShip{
		"HA/DD-1": {
			Uid:          "HA/DD-1",
			Name:         "johnston",
			Type:         objUnit.ShipTypeDestroyer,
			CurHP:        100,
			CurPos:       objPos.New(20, 36),
			BelongPlayer: faction.HumanAlpha,
		},
	}

	controller, err := tutorial.New("TutorialBasicControls", ms)
	require.NoError(t, err)
	manager := &MissionManager{state: ms, tutorial: controller}

	manager.updateMissionStatus()

	require.True(t, manager.TutorialActive())
	require.False(t, manager.TutorialExitRequested())
	require.Equal(t, state.MissionRunning, ms.Core.MissionStatus)
}
