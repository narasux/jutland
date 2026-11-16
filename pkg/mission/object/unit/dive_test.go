package unit

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestSubmarineDiveCycle 验证潜艇水面→下潜→水下→上浮→水面的完整循环与航速档位。
// 场景：玩家命令潜艇下潜/上浮，过渡期间应降速、重复命令不应重启计时，且可随时反向。
// 预期：状态按 Diving/Submerged/Surfacing/Surface 迁移；过渡速度比固定；水下速度比 = 配置值；
// 动作文案随状态切换。
// 覆盖：从水面开始 StartDive，推进到 Submerged，再 StartSurface 推进回 Surface，逐步断言状态/速度比/文案。
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

// TestSubmarineReversesMidTransition 验证过渡中反向切换：状态立即翻转，深度进度连续，剩余时间按同一速度折算。
// 场景：潜艇 DiveTime=1s、SurfaceTime=1s（一轮完整过渡 ReloadTicks(1)=60 拍），从水面 StartDive 后
// 推进 total/2=30 拍，DiveProgress≈0.5；此时玩家按上浮，之后又在上浮中按下潜、下潜中按上浮。
// 预期：StartSurface 立即把状态切成 Surfacing，DiveProgress 仍≈0.5（不跳变）；
// diveTicksLeft = round(完整上浮 60 拍 × 0.5)=30；推进后回到 Surface 且进度 0；
// 再次反向可依次回到 Submerged（进度 1）、Diving（进度 0.5）并最终停在 Submerged。
// 覆盖：单艘潜艇反复调用 StartDive/StartSurface/AdvanceDive，断言状态、剩余拍数与 DiveProgress（容差 1/total）。
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

// TestOnlySubmarinesCanChangeDiveState 验证非潜艇不能进入下潜/上浮流程。
// 场景：单位面板或 AI 误对驱逐舰下发下潜指令；若未拦截，舰船会获得水下状态并破坏索敌/绘制。
// 预期：StartDive/StartSurface 返回 false，NextDiveState 的 canToggle 为 false，DiveSpeedRatio 保持 1。
// 覆盖：用驱逐舰逐项调用四个接口并断言。
func TestOnlySubmarinesCanChangeDiveState(t *testing.T) {
	gunship := &BattleShip{Type: ShipTypeDestroyer, CurHP: 100}
	require.False(t, gunship.StartDive())
	require.False(t, gunship.StartSurface())
	_, canToggle := gunship.NextDiveState()
	require.False(t, canToggle)
	require.InDelta(t, 1.0, gunship.DiveSpeedRatio(), 1e-9)
}

// TestSubmarineSpeedDisplayUsesCurrentDiveLimit 验证潜艇航速展示口径随当前下潜状态切换：
// 水下时展示上限必须取水下航速。
// 场景：全局 SpeedMultiplier=2；潜艇（如 XXI 型）MaxSpeed=9.7 节、SubmergedSpeedRatio=1.29、
// DiveState=Submerged，CurSpeed 取 CruiseSpeed（世界系速度，已乘倍率）；
// 水下航速高于水面的艇若仍按水面 9.7 节展示，单位面板会出现“当前航速 > 最大航速”。
// 预期：MaxSpeedKnots=9.7；SubmergedMaxSpeedKnots=CruiseSpeedKnots=CurSpeedKnots=9.7×1.29=12.513；
// CurSpeedKnots ≤ CruiseSpeedKnots；切回 Surface 后 CruiseSpeedKnots 回到 9.7；
// 非潜艇（31 节普通舰）没有独立水下档，SubmergedMaxSpeedKnots 回退为 31。
// 覆盖：构造水下/水面潜艇与驱逐舰，断言 MaxSpeed/Submerged/Cruise/Cur 四个展示口径。
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

// TestSubmergedShipDeceleratesToUnderwaterCruiseSpeed 验证下潜后舰船减速到水下巡航速度。
// 场景：潜艇已处于 Submerged 状态，仍按水面全速向目标移动；若不减速，水下单位会以水面速度暴露/逃脱。
// 预期：移动 30 帧后 CurSpeed 不超过 MaxSpeed × SubmergedSpeedRatio × 全局速度倍率。
// 覆盖：构造水下潜艇和海岸地图，循环 MoveTo 后断言速度上限。
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

// TestDiveProgressTracksTransition 验证下潜/上浮进度按过渡时间线性变化。
// 场景：绘制层用 DiveProgress 插值舰体透明度和缩放；进度跳变会导致模型突然消失或出现。
// 预期：水面 0；下潜中从 0 平滑增到 1；上浮中从 1 平滑减到 0。
// 覆盖：推进半个过渡时长断言 0.5（含一 tick 容差），再跑完断言端点值。
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

// TestDiveProgressOnlyForSubmarines 验证非潜艇即使带着 DiveState 也不会返回下潜进度。
// 场景：配置错误或调试代码把驱逐舰的 DiveState 设成 Submerged；绘制/UI 不应因此把舰体画成半透明。
// 预期：DiveProgress 返回 0。
// 覆盖：构造带 Submerged 状态的驱逐舰，直接断言返回值为 0。
func TestDiveProgressOnlyForSubmarines(t *testing.T) {
	gunship := &BattleShip{Type: ShipTypeDestroyer, CurHP: 100, DiveState: DiveStateSubmerged}
	require.Zero(t, gunship.DiveProgress())
}

// TestResolveDiveConfigDefaultsToSubmerged 验证潜艇未配置下潜字段时使用默认值。
// 场景：老配置/新潜艇只写了类型，没有 defaultDiveState、下潜时间、上浮时间或速度比。
// 预期：默认水下开局，DiveTime/SurfaceTime/SubmergedSpeedRatio 都填充为默认常量。
// 覆盖：空配置潜艇调用 ResolveDiveConfig，逐字段断言默认值。
func TestResolveDiveConfigDefaultsToSubmerged(t *testing.T) {
	sub := &BattleShip{Type: ShipTypeSubmarine, CurHP: 100}
	require.NoError(t, sub.ResolveDiveConfig())
	require.Equal(t, DiveStateSubmerged, sub.DiveState, "未配置 defaultDiveState 时应默认水下")
	require.InDelta(t, defaultDiveTimeSeconds, sub.DiveTime, 1e-9)
	require.InDelta(t, defaultSurfaceTimeSeconds, sub.SurfaceTime, 1e-9)
	require.InDelta(t, defaultSubmergedSpeedRatio, sub.SubmergedSpeedRatio, 1e-9)
}

// TestResolveDiveConfigKeepsExplicitSurface 验证配置显式写 surface 时按水面开局。
// 场景：需要水面航行的潜艇（例如历史任务中的布雷/侦察）在 ships.json5 写 defaultDiveState: "surface"。
// 预期：ResolveDiveConfig 不覆盖显式值，DiveState 保持 Surface。
// 覆盖：构造 DefaultDiveState="surface" 的潜艇，调用后断言状态。
func TestResolveDiveConfigKeepsExplicitSurface(t *testing.T) {
	sub := &BattleShip{Type: ShipTypeSubmarine, CurHP: 100, DefaultDiveState: "surface"}
	require.NoError(t, sub.ResolveDiveConfig())
	require.Equal(t, DiveStateSurface, sub.DiveState, "显式 surface 仍应按水面开局")
}

// TestResolveDiveConfigRejectsUnknownState 验证非法 defaultDiveState 在启动期报错。
// 场景：配置写成 "periscope" 等未实现状态；若静默接受，运行时会落到未定义分支。
// 预期：ResolveDiveConfig 返回 error，由初始化流程终止并提示配置错误。
// 覆盖：构造 DefaultDiveState="periscope" 的潜艇，断言返回错误。
func TestResolveDiveConfigRejectsUnknownState(t *testing.T) {
	sub := &BattleShip{Type: ShipTypeSubmarine, CurHP: 100, DefaultDiveState: "periscope"}
	require.Error(t, sub.ResolveDiveConfig())
}
