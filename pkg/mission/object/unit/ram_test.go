package unit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
)

// TestRamConfigNormalizeDefaultsAndClamps 验证撞击配置缺省值与非法值夹取。
// 场景：全零配置、负伤害、过大自伤比、过小/过大舰艏角、非正冷却，均为 JSON 可能写出的值。
// 预期：冷却补 3、目标自伤比夹到 1、舰艏角补 60 或夹到 15~90、负伤害归零；
// 合法配置不被改写；失败说明缺字段会变成零冷却或零角度，导致每拍结算或永远撞不中。
// 覆盖：表驱动四组配置，逐字段断言 Normalize 后的值。
func TestRamConfigNormalizeDefaultsAndClamps(t *testing.T) {
	cases := []struct {
		name string
		in   RamConfig
		want RamConfig
	}{
		{
			name: "zero-values",
			in:   RamConfig{},
			want: RamConfig{Cooldown: 3, SelfDamageRatio: 0, BowArc: 60},
		},
		{
			name: "clamps",
			in:   RamConfig{Damage: -1, Cooldown: 0.1, SelfDamageRatio: 2, MinSpeed: -3, BowArc: 200},
			want: RamConfig{Damage: 0, Cooldown: 0.5, SelfDamageRatio: 1, MinSpeed: 0, BowArc: 90},
		},
		{
			name: "small-bow-arc",
			in:   RamConfig{Damage: 1800, Cooldown: 4, SelfDamageRatio: 0.25, BowArc: 5},
			want: RamConfig{Damage: 1800, Cooldown: 4, SelfDamageRatio: 0.25, BowArc: 15},
		},
		{
			name: "custom-valid",
			in:   RamConfig{Damage: 300000, Cooldown: 2, SelfDamageRatio: 0, MinSpeed: 10, BowArc: 45, CanHitPlanes: true, AlwaysOn: true},
			want: RamConfig{Damage: 300000, Cooldown: 2, SelfDamageRatio: 0, MinSpeed: 10, BowArc: 45, CanHitPlanes: true, AlwaysOn: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in
			got.Normalize()
			require.Equal(t, tc.want, got)
		})
	}
}

// TestRamCooldownIsPerTarget 验证撞击冷却按目标分别记录。
// 场景：同一艘撞击舰先后撞 A、B 两个目标，默认设置下 ReloadTicks(4)=240 拍。
// 预期：撞 A 后 A 进入冷却、B 仍可立即结算；失败说明冷却被错误地做成全舰共享。
// 覆盖：MarkRamHit/RamReady 两个目标分支与 SimTick 推进边界。
func TestRamCooldownIsPerTarget(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		SetSimTick(0)
	})

	ship := &BattleShip{Ram: RamConfig{Cooldown: 4}}
	require.True(t, ship.RamReady("a"))
	ship.MarkRamHit("a")
	require.False(t, ship.RamReady("a"))
	require.True(t, ship.RamReady("b"))
	SetSimTick(ReloadTicks(4))
	require.True(t, ship.RamReady("a"))
}

// TestDefaultRamProfileScalesWithTypeTonnageAndArmor 验证按舰种推导的默认撞击能力。
// 场景：同为 10000 HP 的战列舰 / 巡洋舰 / 驱逐舰 / 护卫舰，水平/垂直减伤 0.4/0.2；
// 另测航母、货轮、潜艇、鱼雷艇与 0 HP 不生成默认撞击；再用 0.1 与 0.8 装甲比较重甲收益。
// 预期：四种战舰都能推导出 Derived=true 的配置，战列舰基础伤害最高；
// 装甲从 0.1 提到 0.8 后 damage 上升、SelfDamageRatio 下降；非战斗舰种 ok=false。
// 覆盖：表驱动四舰种 + 排除舰种 + 装甲单调性，断言 damage/自伤/冷却/Derived；
// 失败说明默认撞击没有覆盖普通战舰，或重甲反而受伤更重。
func TestDefaultRamProfileScalesWithTypeTonnageAndArmor(t *testing.T) {
	base, ok := DefaultRamProfile(ShipTypeBattleShip, 10000, 0.4, 0.2)
	require.True(t, ok)
	require.True(t, base.Derived)
	require.InDelta(t, 10000*0.30*1.2, base.Damage, 1e-9)
	require.InDelta(t, 0.15*(1-0.14), base.SelfDamageRatio, 1e-9)
	require.Equal(t, 4.0, base.Cooldown)

	cases := []struct {
		shipType ShipType
		wantOK   bool
		wantDmg  float64
	}{
		{ShipTypeBattleShip, true, 10000 * 0.30 * 1.2},
		{ShipTypeCruiser, true, 10000 * 0.24 * 1.2},
		{ShipTypeDestroyer, true, 10000 * 0.30 * 1.2},
		{ShipTypeFrigate, true, 10000 * 0.26 * 1.2},
		{ShipTypeAircraftCarrier, false, 0},
		{ShipTypeCargo, false, 0},
		{ShipTypeSubmarine, false, 0},
		{ShipTypeTorpedoBoat, false, 0},
	}
	for _, tc := range cases {
		got, ok := DefaultRamProfile(tc.shipType, 10000, 0.4, 0.2)
		require.Equal(t, tc.wantOK, ok, tc.shipType)
		if tc.wantOK {
			require.InDelta(t, tc.wantDmg, got.Damage, 1e-9, tc.shipType)
		}
	}

	light, ok := DefaultRamProfile(ShipTypeBattleShip, 10000, 0.1, 0.1)
	require.True(t, ok)
	heavy, ok := DefaultRamProfile(ShipTypeBattleShip, 10000, 0.8, 0.8)
	require.True(t, ok)
	require.Greater(t, heavy.Damage, light.Damage)
	require.Less(t, heavy.SelfDamageRatio, light.SelfDamageRatio)

	_, ok = DefaultRamProfile(ShipTypeBattleShip, 0, 0.5, 0.5)
	require.False(t, ok)
}
