package unit

import "testing"

// TestDefaultAntiTorpedoProfileByShipClass 验证反鱼雷火控按舰种分档：小舰更快更准，主力舰更慢更差。
// 场景：驱逐舰 / 护卫舰 / 鱼雷艇、巡洋舰、战列舰 / 航母、以及未标注舰种各取一次。
// 预期：驱逐档 FireInterval=0.5s、HitChance=0.08；巡洋档 1.0s / 0.05；
// 主力舰档 1.5s / 0.03；未标注回落到 1.0s / 0.05。失败意味着驱逐舰不再有反鱼雷优势，
// 或者主力舰重新变成鱼雷免疫。
// 覆盖：表驱动四类输入，断言两个字段的具体数值，并比较驱逐档间隔 < 主力舰档、概率 > 主力舰档。
func TestDefaultAntiTorpedoProfileByShipClass(t *testing.T) {
	cases := []struct {
		name     string
		shipType ShipType
		interval float64
		chance   float64
	}{
		{name: "destroyer", shipType: ShipTypeDestroyer, interval: 0.5, chance: 0.08},
		{name: "frigate", shipType: ShipTypeFrigate, interval: 0.5, chance: 0.08},
		{name: "torpedo_boat", shipType: ShipTypeTorpedoBoat, interval: 0.5, chance: 0.08},
		{name: "cruiser", shipType: ShipTypeCruiser, interval: 1.0, chance: 0.05},
		{name: "battleship", shipType: ShipTypeBattleShip, interval: 1.5, chance: 0.03},
		{name: "aircraft_carrier", shipType: ShipTypeAircraftCarrier, interval: 1.5, chance: 0.03},
		{name: "default", shipType: ShipTypeDefault, interval: 1.0, chance: 0.05},
	}
	for _, tc := range cases {
		t.Run(string(tc.shipType), func(t *testing.T) {
			got := DefaultAntiTorpedoProfile(tc.shipType)
			if got.FireInterval != tc.interval || got.HitChance != tc.chance {
				t.Fatalf("profile(%s) = {%v, %v}, want {%v, %v}",
					tc.shipType, got.FireInterval, got.HitChance, tc.interval, tc.chance)
			}
		})
	}

	fast := DefaultAntiTorpedoProfile(ShipTypeDestroyer)
	capital := DefaultAntiTorpedoProfile(ShipTypeBattleShip)
	if fast.FireInterval >= capital.FireInterval || fast.HitChance <= capital.HitChance {
		t.Fatalf("驱逐档 %+v 没有比主力舰档 %+v 更快更准", fast, capital)
	}
}
