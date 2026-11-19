package unit

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
)

// TestHurtByPenetrationIsOptIn 验证穿甲伤害开关默认关闭，开启后才应用倍率。
// 场景：100 伤害、120mm 穿深、100mm 装甲；关闭时伤害 100，开启时 ratio=1.2 → x1.05。
// 预期：HP 900（关闭）与 895（开启）；恢复 config.G，避免污染其他测试。
// 覆盖：同一 BattleShip 两次 HurtBy，构造 Bullet 字段，断言 HP 差。
func TestHurtByPenetrationIsOptIn(t *testing.T) {
	old := config.G
	config.G = config.NewDefaultGameSettings()
	defer func() { config.G = old }()

	makeShip := func() *BattleShip {
		return &BattleShip{
			CurHP: 1000, TotalHP: 1000, ArmorMM: 100,
			HorizontalDamageReduction: 0, VerticalDamageReduction: 0,
		}
	}
	bullet := &objBullet.Bullet{Damage: 100, PenetrationMM: 120, CriticalRate: 0}
	off := makeShip()
	off.HurtBy(bullet)
	if off.CurHP != 900 {
		t.Fatalf("disabled HP = %v, want 900", off.CurHP)
	}
	config.G.EnablePenetration = true
	on := makeShip()
	on.HurtBy(bullet)
	if on.CurHP != 895 {
		t.Fatalf("enabled HP = %v, want 895", on.CurHP)
	}
}
