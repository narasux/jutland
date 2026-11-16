package unit

import "testing"

// TestResolvePlaneSightScoutOutrangesFighter 验证侦察机视距大于战斗机，且各自命中配置常量。
// 场景：开迷雾时舰队靠飞机点亮海域；若侦察机视距不大于战斗机，专职侦察机型就失去意义。
// 预期：ResolvePlaneSight 返回 SightRangeScout/SightRangeFighter，且 scout > fighter。
// 覆盖：分别调用两种机型，比较大小并断言等于常量。
func TestResolvePlaneSightScoutOutrangesFighter(t *testing.T) {
	scout := ResolvePlaneSight(PlaneTypeScout, 0)
	fighter := ResolvePlaneSight(PlaneTypeFighter, 0)
	if scout <= fighter {
		t.Fatalf("scout sight %v, fighter sight %v", scout, fighter)
	}
	if scout != SightRangeScout || fighter != SightRangeFighter {
		t.Fatalf("scout %v fighter %v", scout, fighter)
	}
}

// TestResolveShipSightCapsAtMainGun 验证未显式配置视距时，战舰视距不超过主炮射程。
// 场景：战列舰主炮射程 17.5 格，配置没有 sightRange；若默认视距大于射程，会出现“看得见但打不到”的迷雾漏洞。
// 预期：ResolveShipSight 返回主炮射程 17.5。
// 覆盖：传入 ShipTypeBattleShip、显式视距 0、主炮射程 17.5，断言结果。
func TestResolveShipSightCapsAtMainGun(t *testing.T) {
	got := ResolveShipSight(ShipTypeBattleShip, 0, 17.5)
	if got != 17.5 {
		t.Fatalf("capped sight = %v, want 17.5", got)
	}
}

// TestResolveShipSightKeepsExplicitOverride 验证显式 sightRange 优先于主炮射程。
// 场景：历史任务给特定战舰配置更远/更近的视距（雷达、侦察能力差异）；配置值不能被主炮自动封顶覆盖。
// 预期：显式 10 格视距原样返回，即使主炮射程为 17.5。
// 覆盖：传入显式 10 与主炮 17.5，断言返回 10。
func TestResolveShipSightKeepsExplicitOverride(t *testing.T) {
	got := ResolveShipSight(ShipTypeBattleShip, 10, 17.5)
	if got != 10 {
		t.Fatalf("override sight = %v, want 10", got)
	}
}

// TestResolveShipSightGivesSpecialShipsSight 验证 default 舰种的特殊船固定使用 SightRangeSpecial=16 格，
// 不按主炮射程封顶，且显式配置仍然优先。
// 场景：小黄鸭、水滴这类特殊船不在舰种视距表里；水滴的撞击炮射程极短（1 格），
// 若按“不超过主炮射程”封顶，开迷雾时它们周围一格都不亮。
// 预期：主炮射程 64 和 1 都返回 SightRangeSpecial；显式 sightRange=7 时返回 7。
// 覆盖：三次调用 ResolveShipSight(ShipTypeDefault, configured, mainGunRange)，分别断言 16 / 16 / 7。
func TestResolveShipSightGivesSpecialShipsSight(t *testing.T) {
	if got := ResolveShipSight(ShipTypeDefault, 0, 64); got != SightRangeSpecial {
		t.Fatalf("default type sight = %v, want %v", got, SightRangeSpecial)
	}
	// 水滴的撞击炮射程极短，不能按主炮封顶，否则等于看不见。
	if got := ResolveShipSight(ShipTypeDefault, 0, 1); got != SightRangeSpecial {
		t.Fatalf("default type sight with a stub gun = %v, want %v", got, SightRangeSpecial)
	}
	// 显式配了视距仍然优先。
	if got := ResolveShipSight(ShipTypeDefault, 7, 64); got != 7 {
		t.Fatalf("default type override sight = %v, want 7", got)
	}
}

// TestResolveShipSightWithoutMainGun 验证没有主炮的辅助舰仍有基础视距。
// 场景：医院船、补给舰没有主炮，若视距按主炮封顶会变成 0，开迷雾时自身周围永远全黑。
// 预期：ResolveShipSight 返回该舰种的 SightRangeHospital。
// 覆盖：传入 ShipTypeHospital、显式视距 0、主炮 0，断言结果。
func TestResolveShipSightWithoutMainGun(t *testing.T) {
	got := ResolveShipSight(ShipTypeHospital, 0, 0)
	if got != SightRangeHospital {
		t.Fatalf("hospital sight = %v, want %v", got, SightRangeHospital)
	}
}
