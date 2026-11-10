package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestSubmarineTestMissionSimulation 在真实关卡里跑若干拍，确认四区布置不炸、
// 敌方潜艇会按 AI 逻辑下潜接敌。
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

	for range 300 {
		_, err := m.Update()
		require.NoError(t, err)
	}

	for _, sub := range enemySubs {
		require.NotEqual(t, objUnit.DiveStateSurface, sub.DiveState, "敌方潜艇接敌后应下潜")
	}
}
