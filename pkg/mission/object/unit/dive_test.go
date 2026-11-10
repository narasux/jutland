package unit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

func TestSubmarineDiveCycle(t *testing.T) {
	ship := &BattleShip{
		Type:                ShipTypeSubmarine,
		TotalHP:             100,
		CurHP:               100,
		MaxSpeed:            0.03,
		DiveTime:            1,
		SurfaceTime:         1,
		SubmergedSpeedRatio: 0.35,
	}
	require.Equal(t, DiveStateSurface, ship.DiveState)
	require.InDelta(t, 1.0, ship.DiveSpeedRatio(), 1e-9)
	require.False(t, ship.Submerged())

	require.True(t, ship.StartDive())
	require.Equal(t, DiveStateDiving, ship.DiveState)
	require.True(t, ship.DiveTransitioning())
	require.InDelta(t, diveTransitionSpeedRatio, ship.DiveSpeedRatio(), 1e-9)
	// 过渡中不允许再次切换，避免计时器被重置
	_, canToggle := ship.NextDiveState()
	require.False(t, canToggle)
	require.False(t, ship.StartDive())

	for i := 0; i < 200 && ship.DiveState == DiveStateDiving; i++ {
		ship.AdvanceDive()
	}
	require.Equal(t, DiveStateSubmerged, ship.DiveState)
	require.True(t, ship.Submerged())
	require.InDelta(t, 0.35, ship.DiveSpeedRatio(), 1e-9)

	require.True(t, ship.StartSurface())
	require.Equal(t, DiveStateSurfacing, ship.DiveState)
	require.InDelta(t, diveTransitionSpeedRatio, ship.DiveSpeedRatio(), 1e-9)
	for i := 0; i < 200 && ship.DiveState == DiveStateSurfacing; i++ {
		ship.AdvanceDive()
	}
	require.Equal(t, DiveStateSurface, ship.DiveState)
	require.InDelta(t, 1.0, ship.DiveSpeedRatio(), 1e-9)
}

func TestOnlySubmarinesCanChangeDiveState(t *testing.T) {
	gunship := &BattleShip{Type: ShipTypeDestroyer, CurHP: 100}
	require.False(t, gunship.StartDive())
	require.False(t, gunship.StartSurface())
	_, canToggle := gunship.NextDiveState()
	require.False(t, canToggle)
	require.InDelta(t, 1.0, gunship.DiveSpeedRatio(), 1e-9)
}

func TestSubmergedShipDeceleratesToUnderwaterCruiseSpeed(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	mapCfg := newCoastMapCfg()
	ship := &BattleShip{
		Type:                ShipTypeSubmarine,
		CurHP:               100,
		CurPos:              objPos.NewR(0.5, 0.5),
		CurRotation:         90,
		CurSpeed:            0.03,
		MaxSpeed:            0.03,
		Acceleration:        0.001,
		RotateSpeed:         10,
		DiveState:           DiveStateSubmerged,
		SubmergedSpeedRatio: 0.35,
	}
	target := objPos.NewR(7.5, 0.5)
	for i := 0; i < 30; i++ {
		arrive, blocked := ship.MoveTo(mapCfg, target, false)
		require.False(t, arrive)
		require.False(t, blocked)
	}
	want := 0.03 * 0.35 * config.G.SpeedMultiplier
	require.LessOrEqual(t, ship.CurSpeed, want+1e-9, "下潜后必须减速到水下巡航速度")
}
