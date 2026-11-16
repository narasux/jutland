package unit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetShipCostIncludesAircraftAndHullCostExcludesIt 验证战舰总成本包含舰载机，舰体成本不包含。
// 场景：增援界面按总价扣款，拆解/维修按舰体价结算；若两者混用，航母会被重复收取舰载机费用。
// 预期：总成本 = 舰体 FundsCost + Σ(MaxCount × 飞机 FundsCost)，TimeCost 取舰体；舰体成本只返回舰体值；
// 未知舰名返回 (0,0)。
// 覆盖：临时注册战舰 100/5、舰载机 10，MaxCount=2，断言 GetShipCost/GetShipHullCost 与未知舰。
func TestGetShipCostIncludesAircraftAndHullCostExcludesIt(t *testing.T) {
	const shipName = "test-cost-carrier"
	const planeName = "test-cost-plane"
	oldShip, hadShip := ShipMap[shipName]
	oldPlane, hadPlane := PlaneMap[planeName]
	t.Cleanup(func() {
		if hadShip {
			ShipMap[shipName] = oldShip
		} else {
			delete(ShipMap, shipName)
		}
		if hadPlane {
			PlaneMap[planeName] = oldPlane
		} else {
			delete(PlaneMap, planeName)
		}
	})

	ShipMap[shipName] = &BattleShip{
		Name: shipName, FundsCost: 100, TimeCost: 5,
		Aircraft: ShipAircraft{Groups: []PlaneGroup{{Name: planeName, MaxCount: 2}}},
	}
	PlaneMap[planeName] = &Plane{Name: planeName, FundsCost: 10}

	funds, timeCost := GetShipCost(shipName)
	require.Equal(t, int64(120), funds)
	require.Equal(t, int64(5), timeCost)

	hullFunds, hullTime := GetShipHullCost(shipName)
	require.Equal(t, int64(100), hullFunds)
	require.Equal(t, int64(5), hullTime)

	unknownFunds, unknownTime := GetShipCost("missing-ship")
	require.Zero(t, unknownFunds)
	require.Zero(t, unknownTime)
}

// TestEnableWeaponRestoresEveryWeaponGroup 验证 EnableWeapon(All) 清掉所有分组禁用标记和单体 Disable。
// 场景：玩家在单位面板点“启用全部武器”，此前可能分别禁用了主炮/副炮/防空/鱼雷/火箭；
// 只清分组标记而漏清单体标记，面板会显示启用但实际仍不开火。
// 预期：五个分组禁用标记全为 false，所有 Gun/Launcher 的 Disable 全为 false。
// 覆盖：构造含五类武器且全部 Disable 的战舰，调用 EnableWeapon(WeaponTypeAll) 后逐项断言。
func TestEnableWeaponRestoresEveryWeaponGroup(t *testing.T) {
	ship := &BattleShip{Weapon: ShipWeapon{
		MainGuns:                []*Gun{{Disable: true}},
		SecondaryGuns:           []*Gun{{Disable: true}},
		AntiAircraftGuns:        []*Gun{{Disable: true}},
		Torpedoes:               []*TorpedoLauncher{{Disable: true}},
		Rockets:                 []*RocketLauncher{{Disable: true}},
		MainGunDisabled:         true,
		SecondaryGunDisabled:    true,
		AntiAircraftGunDisabled: true,
		TorpedoDisabled:         true,
		RocketDisabled:          true,
	}}

	ship.EnableWeapon(WeaponTypeAll)

	require.False(t, ship.Weapon.MainGunDisabled)
	require.False(t, ship.Weapon.SecondaryGunDisabled)
	require.False(t, ship.Weapon.AntiAircraftGunDisabled)
	require.False(t, ship.Weapon.TorpedoDisabled)
	require.False(t, ship.Weapon.RocketDisabled)
	require.False(t, ship.Weapon.MainGuns[0].Disable)
	require.False(t, ship.Weapon.SecondaryGuns[0].Disable)
	require.False(t, ship.Weapon.AntiAircraftGuns[0].Disable)
	require.False(t, ship.Weapon.Torpedoes[0].Disable)
	require.False(t, ship.Weapon.Rockets[0].Disable)
}

// TestPlaneWeaponAnyReloadedCoversGunsAndOrdnance 验证飞机 AnyReloaded 能识别机炮、未投放弹药和火箭。
// 场景：AI 判断是否继续攻击/返航；若只检查机炮而漏掉炸弹/鱼雷/火箭，挂载充足的攻击机会提前返航。
// 预期：无武器 false；机炮就绪 true；炸弹/鱼雷未投放 true；全部投放且机炮装填中 false；
// 火箭就绪 true。
// 覆盖：逐步构造 PlaneWeapon 的 Guns/Bombs/Torpedoes/Rockets，调用 AnyReloaded 断言。
func TestPlaneWeaponAnyReloadedCoversGunsAndOrdnance(t *testing.T) {
	require.False(t, (&PlaneWeapon{}).AnyReloaded())

	require.True(t, (&PlaneWeapon{
		Guns: []*Gun{{ReloadStartAt: 0, ReloadTime: 1}},
	}).AnyReloaded())

	require.True(t, (&PlaneWeapon{
		Bombs: []*Releaser{{Released: false}},
	}).AnyReloaded())

	require.True(t, (&PlaneWeapon{
		Torpedoes: []*Releaser{{Released: false}},
	}).AnyReloaded())

	require.False(t, (&PlaneWeapon{
		Guns:      []*Gun{{ReloadStartTick: SimTick() + 1, ReloadTime: 100}},
		Bombs:     []*Releaser{{Released: true}},
		Torpedoes: []*Releaser{{Released: true}},
	}).AnyReloaded())

	require.True(t, (&PlaneWeapon{
		Rockets: []*PlaneRocketLauncher{{RocketCount: 1}},
	}).AnyReloaded())
}
