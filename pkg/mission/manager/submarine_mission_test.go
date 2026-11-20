package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestSubmarineTestMissionSimulation 验证潜艇试验场四区能稳定跑真实 Update，且 AI 潜艇保持水下。
// 场景：加载 SubmarineTest（P1）并 Resize(1280,720)，核对西南/东北 2 艘玩家 U-96、敌方 I-168 与 2 艘 Nautilus，随后连续跑 300 拍。
// 预期：5 艘潜艇入场即 DiveStateSubmerged；300 拍内每拍 Update 无错误，敌方潜艇始终不因赶路切到 Surface。
// 覆盖：New/Update 全流程与 Arena.Ships 清点，逐拍 require.NoError 并检查 DiveState（含最终状态）。
func TestSubmarineTestMissionSimulation(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	m := New("SubmarineTest", faction.SideP1)
	m.Resize(1280, 720)

	var playerSubs, enemySubs []*objUnit.BattleShip
	for _, ship := range m.state.Arena.Ships {
		if ship.Type != objUnit.ShipTypeSubmarine {
			continue
		}
		if ship.BelongPlayer == faction.HumanAlpha {
			playerSubs = append(playerSubs, ship)
		} else {
			enemySubs = append(enemySubs, ship)
		}
	}
	require.Len(t, playerSubs, 2, "西南与东北区各有一艘玩家潜艇")
	require.Len(t, enemySubs, 3, "敌方的 I-168 与两艘 Nautilus 都应入场")

	for _, sub := range playerSubs {
		require.Equal(t, objUnit.DiveStateSubmerged, sub.Submarine.DiveState, "玩家潜艇入场即水下")
	}
	for _, sub := range enemySubs {
		require.Equal(t, objUnit.DiveStateSubmerged, sub.Submarine.DiveState, "敌方潜艇入场即水下")
	}

	// 现役潜艇都带鱼雷，AI 在接近阶段不需要上浮；跑一段接近过程确认不会被拉回水面。
	for range 300 {
		_, err := m.Update()
		require.NoError(t, err)
		for _, sub := range enemySubs {
			require.NotEqual(t, objUnit.DiveStateSurface, sub.Submarine.DiveState,
				"AI 潜艇不应为了赶路而上浮")
		}
	}

	for _, sub := range enemySubs {
		require.NotEqual(t, objUnit.DiveStateSurface, sub.Submarine.DiveState, "敌方潜艇应保持水下")
	}
}
