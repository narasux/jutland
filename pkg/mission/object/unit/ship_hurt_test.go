package unit

import (
	"math"
	"testing"

	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
)

// 装甲减免按弹道落角连续过渡：平射吃水平装甲带，吊射吃垂直装甲带，
// 中间落角取两者的加权平均，不允许在某条射程上跳变。
func TestHurtByBlendsArmorByPlunge(t *testing.T) {
	newShip := func() *BattleShip {
		return &BattleShip{
			TotalHP: 1000, CurHP: 1000,
			HorizontalDamageReduction: 0.5,
			VerticalDamageReduction:   0.3,
		}
	}
	damageAt := func(plunge float64) float64 {
		ship := newShip()
		bullet := &objBullet.Bullet{Damage: 100, Plunge: plunge, CriticalRate: 0}
		ship.HurtBy(bullet)
		return 1000 - ship.CurHP
	}

	if got := damageAt(0); math.Abs(got-50) > 1e-9 {
		t.Fatalf("flat fire damage = %v, want 50 (1 - 0.5)", got)
	}
	if got := damageAt(1); math.Abs(got-70) > 1e-9 {
		t.Fatalf("plunging damage = %v, want 70 (1 - 0.3)", got)
	}
	if got := damageAt(0.5); math.Abs(got-60) > 1e-9 {
		t.Fatalf("half plunge damage = %v, want 60 (weighted average)", got)
	}
	// 单调性：落角越大，单发伤害越高（垂直装甲带更薄）
	prev := -1.0
	for plunge := 0.0; plunge <= 1.0; plunge += 0.1 {
		got := damageAt(plunge)
		if got < prev {
			t.Fatalf("damage decreases with plunge at %v: %v < %v", plunge, got, prev)
		}
		prev = got
	}
}
