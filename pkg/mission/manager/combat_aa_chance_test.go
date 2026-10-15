package manager

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// 舰对空命中率按弹药口径分级：小口径速射防空炮最高，中口径次之，
// >155mm 主炮客串防空直击概率最低；俯冲轰炸机与高空轰炸机统一再降为三分之一。
func TestAaHitChanceCaliberTiers(t *testing.T) {
	// 以战斗机为基准机种，避免受轰炸机命中率折扣影响。
	small := aaHitChance(20, objUnit.PlaneTypeFighter)
	medium := aaHitChance(127, objUnit.PlaneTypeFighter)
	large := aaHitChance(203, objUnit.PlaneTypeFighter)

	if !(small > medium && medium > large) {
		t.Fatalf(
			"hit chance should decrease with caliber: small=%.4f medium=%.4f large=%.4f",
			small,
			medium,
			large,
		)
	}
	if small != 1.0/5 {
		t.Fatalf("small caliber (<=40mm) hit chance = %.4f, want %.4f", small, 1.0/5)
	}
	if medium != 1.0/9 {
		t.Fatalf("medium caliber (<=155mm) hit chance = %.4f, want %.4f", medium, 1.0/9)
	}
	if large != 1.0/14 {
		t.Fatalf("large caliber (>155mm) hit chance = %.4f, want %.4f", large, 1.0/14)
	}
	// 边界值：40mm 归小口径，155mm 归中口径，156mm 起归大口径。
	if aaHitChance(40, objUnit.PlaneTypeFighter) != small {
		t.Fatal("40mm should use the small caliber tier")
	}
	for _, diameter := range []int{41, 127, 155} {
		if aaHitChance(diameter, objUnit.PlaneTypeFighter) != medium {
			t.Fatalf("%dmm should use the medium caliber tier", diameter)
		}
	}
	if aaHitChance(156, objUnit.PlaneTypeFighter) != large {
		t.Fatal("156mm should use the large caliber tier")
	}
	// 俯冲轰炸机与高空轰炸机命中率减为三分之一，与口径无关
	for _, planeType := range []objUnit.PlaneType{objUnit.PlaneTypeDiveBomber, objUnit.PlaneTypeLevelBomber} {
		if got := aaHitChance(127, planeType); got != medium/3 {
			t.Fatalf("%s medium-caliber hit chance = %.4f, want medium/3 = %.4f", planeType, got, medium/3)
		}
		if got := aaHitChance(203, planeType); got != large/3 {
			t.Fatalf("%s large-caliber hit chance = %.4f, want large/3 = %.4f", planeType, got, large/3)
		}
	}
}
