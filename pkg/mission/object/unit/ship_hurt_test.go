package unit

import (
	"math"
	"testing"

	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
)

// TestHurtByBlendsArmorByPlunge 验证装甲减免按弹道落角连续过渡：平射吃水平装甲带，吊射吃垂直装甲带，
// 中间落角取两者的加权平均，不允许在某条射程上跳变。
// 场景：1000 HP 战舰，水平装甲减免 0.5、垂直装甲减免 0.3；100 伤害弹且 CriticalRate=0 排除暴击，
// 落角 0 表示平射、1 表示吊射。
// 预期：平射伤害 100×(1-0.5)=50；吊射 100×(1-0.3)=70；
// 落角 0.5 取加权平均 100×(1-(0.5×0.5+0.3×0.5))=60；落角 0→1 每 0.1 采样伤害单调不减。
// 覆盖：每次新建同型舰调用 HurtBy，用 1000-CurHP 读实际伤害，并循环 11 个落角验证单调性。
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
