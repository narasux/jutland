package unit

import "testing"

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

func TestResolveShipSightCapsAtMainGun(t *testing.T) {
	got := ResolveShipSight(ShipTypeBattleShip, 0, 17.5)
	if got != 17.5 {
		t.Fatalf("capped sight = %v, want 17.5", got)
	}
}

func TestResolveShipSightKeepsExplicitOverride(t *testing.T) {
	got := ResolveShipSight(ShipTypeBattleShip, 10, 17.5)
	if got != 10 {
		t.Fatalf("override sight = %v, want 10", got)
	}
}

// 小黄鸭、水滴这类特殊船也要有视距，否则开迷雾时它们不照亮任何格子。
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

func TestResolveShipSightWithoutMainGun(t *testing.T) {
	got := ResolveShipSight(ShipTypeHospital, 0, 0)
	if got != SightRangeHospital {
		t.Fatalf("hospital sight = %v, want %v", got, SightRangeHospital)
	}
}
