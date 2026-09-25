package manager

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// 舰对空命中率按弹药口径分级：小口径速射防空炮最高，中口径次之，
// >155mm 主炮客串防空直击概率最低；俯冲轰炸机统一再降为三分之一。
func TestAaHitChanceCaliberTiers(t *testing.T) {
	small := aaHitChance(20, objUnit.PlaneTypeLevelBomber)
	medium := aaHitChance(127, objUnit.PlaneTypeLevelBomber)
	large := aaHitChance(203, objUnit.PlaneTypeLevelBomber)

	if !(small > medium && medium > large) {
		t.Fatalf(
			"hit chance should decrease with caliber: small=%.4f medium=%.4f large=%.4f",
			small,
			medium,
			large,
		)
	}
	if small != 1.0/3 {
		t.Fatalf("small caliber (<=40mm) hit chance = %.4f, want %.4f", small, 1.0/3)
	}
	if medium != 1.0/6 {
		t.Fatalf("medium caliber (<=155mm) hit chance = %.4f, want %.4f", medium, 1.0/6)
	}
	if large != 1.0/9 {
		t.Fatalf("large caliber (>155mm) hit chance = %.4f, want %.4f", large, 1.0/9)
	}
	// 边界值：40mm 归小口径，155mm 归中口径，156mm 起归大口径。
	if aaHitChance(40, objUnit.PlaneTypeLevelBomber) != small {
		t.Fatal("40mm should use the small caliber tier")
	}
	for _, diameter := range []int{41, 127, 155} {
		if aaHitChance(diameter, objUnit.PlaneTypeLevelBomber) != medium {
			t.Fatalf("%dmm should use the medium caliber tier", diameter)
		}
	}
	if aaHitChance(156, objUnit.PlaneTypeLevelBomber) != large {
		t.Fatal("156mm should use the large caliber tier")
	}
	// 俯冲轰炸机命中率减为三分之一，与口径无关
	dive := aaHitChance(127, objUnit.PlaneTypeDiveBomber)
	if dive*3 != medium {
		t.Fatalf("dive bomber hit chance = %.4f, want medium/3 = %.4f", dive, medium/3)
	}
	diveLarge := aaHitChance(203, objUnit.PlaneTypeDiveBomber)
	if diveLarge*3 != large {
		t.Fatalf("dive bomber large-caliber hit chance = %.4f, want large/3 = %.4f", diveLarge, large/3)
	}
}
