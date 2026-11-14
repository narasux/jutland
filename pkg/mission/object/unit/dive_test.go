package unit

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
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
	// 重复下潜不会重启计时，但随时可以反向（以最后一次指令为准）
	require.False(t, ship.StartDive())
	next, canToggle := ship.NextDiveState()
	require.True(t, canToggle)
	require.Equal(t, DiveStateSurface, next)
	require.Equal(t, i18n.Text(i18n.MsgShipSurfaceAction), ship.DiveActionText())

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

// 过渡中反向切换：状态立即翻转，深度进度连续，剩余时间按同一速度折算。
func TestSubmarineReversesMidTransition(t *testing.T) {
	ship := &BattleShip{
		Type:        ShipTypeSubmarine,
		CurHP:       100,
		DiveTime:    1,
		SurfaceTime: 1,
	}
	require.True(t, ship.StartDive())
	total := ship.diveTotalTicks

	for i := int64(0); i < total/2; i++ {
		ship.AdvanceDive()
	}
	progress := ship.DiveProgress()
	require.InDelta(t, 0.5, progress, 1.0/float64(total))

	// 下潜中按下上浮：立刻反向，深度进度不跳变
	require.True(t, ship.StartSurface())
	require.Equal(t, DiveStateSurfacing, ship.DiveState)
	require.InDelta(t, progress, ship.DiveProgress(), 1.0/float64(total))
	// 剩余时间按完整上浮耗时折算，切换速度不变
	expectTicks := max(1, int64(math.Round(float64(ReloadTicks(ship.surfaceTimeSeconds()))*progress)))
	require.Equal(t, expectTicks, ship.diveTicksLeft)

	for i := int64(0); i < total; i++ {
		ship.AdvanceDive()
	}
	require.Equal(t, DiveStateSurface, ship.DiveState)
	require.Zero(t, ship.DiveProgress())

	// 上浮中再按下潜：同样反向，并回到水下
	require.True(t, ship.StartDive())
	for i := int64(0); i < total; i++ {
		ship.AdvanceDive()
	}
	require.Equal(t, DiveStateSubmerged, ship.DiveState)

	require.True(t, ship.StartSurface())
	for i := int64(0); i < total/2; i++ {
		ship.AdvanceDive()
	}
	require.True(t, ship.StartDive())
	require.Equal(t, DiveStateDiving, ship.DiveState)
	require.InDelta(t, 0.5, ship.DiveProgress(), 1.0/float64(total))
	for i := int64(0); i < total; i++ {
		ship.AdvanceDive()
	}
	require.Equal(t, DiveStateSubmerged, ship.DiveState)
	require.InDelta(t, 1.0, ship.DiveProgress(), 1e-9)
}

func TestOnlySubmarinesCanChangeDiveState(t *testing.T) {
	gunship := &BattleShip{Type: ShipTypeDestroyer, CurHP: 100}
	require.False(t, gunship.StartDive())
	require.False(t, gunship.StartSurface())
	_, canToggle := gunship.NextDiveState()
	require.False(t, canToggle)
	require.InDelta(t, 1.0, gunship.DiveSpeedRatio(), 1e-9)
}

// TestSubmarineSpeedDisplayUsesCurrentDiveLimit 锁定潜艇航速展示口径：
// 水下航速高于水面的艇（如 XXI 型）在水下时，展示上限必须取水下航速，
// 否则单位面板会出现“当前航速 > 最大航速”。
func TestSubmarineSpeedDisplayUsesCurrentDiveLimit(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })
	config.G = &config.GameSettings{SpeedMultiplier: 2}

	ship := &BattleShip{
		Type:                ShipTypeSubmarine,
		MaxSpeed:            9.7 / ShipSpeedScale,
		SubmergedSpeedRatio: 1.29,
		DiveState:           DiveStateSubmerged,
	}
	// CurSpeed 是世界系速度，已乘全局倍率。
	ship.CurSpeed = ship.CruiseSpeed()

	require.InDelta(t, 9.7, ship.MaxSpeedKnots(), 1e-9)
	require.InDelta(t, 9.7*1.29, ship.SubmergedMaxSpeedKnots(), 1e-9)
	require.InDelta(t, 9.7*1.29, ship.CruiseSpeedKnots(), 1e-9)
	require.InDelta(t, 9.7*1.29, ship.CurSpeedKnots(), 1e-9)
	require.LessOrEqual(t, ship.CurSpeedKnots(), ship.CruiseSpeedKnots()+1e-9)

	ship.DiveState = DiveStateSurface
	require.InDelta(t, 9.7, ship.CruiseSpeedKnots(), 1e-9)

	// 非潜艇没有独立的水下档，静态档案回退到水面航速。
	gunship := &BattleShip{MaxSpeed: 31.0 / ShipSpeedScale}
	require.InDelta(t, 31.0, gunship.SubmergedMaxSpeedKnots(), 1e-9)
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

func TestDiveProgressTracksTransition(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	ship := &BattleShip{
		Type:        ShipTypeSubmarine,
		CurHP:       100,
		DiveTime:    1,
		SurfaceTime: 1,
	}
	require.Zero(t, ship.DiveProgress(), "水面时应为 0")

	require.True(t, ship.StartDive())
	require.Zero(t, ship.DiveProgress(), "刚开始下潜时应仍为 0")
	total := ship.diveTotalTicks
	require.Positive(t, total)

	for i := int64(0); i < total/2; i++ {
		ship.AdvanceDive()
	}
	require.InDelta(t, 0.5, ship.DiveProgress(), 1.0/float64(total), "下潜中应随计时平滑增长")

	for i := int64(0); i < total-total/2; i++ {
		ship.AdvanceDive()
	}
	require.Equal(t, DiveStateSubmerged, ship.DiveState)
	require.InDelta(t, 1.0, ship.DiveProgress(), 1e-9)

	require.True(t, ship.StartSurface())
	require.InDelta(t, 1.0, ship.DiveProgress(), 1e-9, "刚开始上浮时应仍为 1")
	total = ship.diveTotalTicks
	require.Positive(t, total)

	for i := int64(0); i < total/2; i++ {
		ship.AdvanceDive()
	}
	require.InDelta(t, 0.5, ship.DiveProgress(), 1.0/float64(total), "上浮中应随计时平滑减退")

	for i := int64(0); i < total-total/2; i++ {
		ship.AdvanceDive()
	}
	require.Equal(t, DiveStateSurface, ship.DiveState)
	require.Zero(t, ship.DiveProgress())
}

func TestDiveProgressOnlyForSubmarines(t *testing.T) {
	gunship := &BattleShip{Type: ShipTypeDestroyer, CurHP: 100, DiveState: DiveStateSubmerged}
	require.Zero(t, gunship.DiveProgress())
}

func TestResolveDiveConfigDefaultsToSubmerged(t *testing.T) {
	sub := &BattleShip{Type: ShipTypeSubmarine, CurHP: 100}
	require.NoError(t, sub.ResolveDiveConfig())
	require.Equal(t, DiveStateSubmerged, sub.DiveState, "未配置 defaultDiveState 时应默认水下")
	require.InDelta(t, defaultDiveTimeSeconds, sub.DiveTime, 1e-9)
	require.InDelta(t, defaultSurfaceTimeSeconds, sub.SurfaceTime, 1e-9)
	require.InDelta(t, defaultSubmergedSpeedRatio, sub.SubmergedSpeedRatio, 1e-9)
}

func TestResolveDiveConfigKeepsExplicitSurface(t *testing.T) {
	sub := &BattleShip{Type: ShipTypeSubmarine, CurHP: 100, DefaultDiveState: "surface"}
	require.NoError(t, sub.ResolveDiveConfig())
	require.Equal(t, DiveStateSurface, sub.DiveState, "显式 surface 仍应按水面开局")
}

func TestResolveDiveConfigRejectsUnknownState(t *testing.T) {
	sub := &BattleShip{Type: ShipTypeSubmarine, CurHP: 100, DefaultDiveState: "periscope"}
	require.Error(t, sub.ResolveDiveConfig())
}
