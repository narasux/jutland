package manager

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// 舰对空命中率按弹药口径分级：小口径速射防空炮最高，中口径次之，
// 133~155mm 大口径高平两用炮再次，>155mm 主炮客串防空直击概率极低；
// 俯冲轰炸机统一再降为三分之一。
func TestAaHitChanceCaliberTiers(t *testing.T) {
	small := aaHitChance(20, objUnit.PlaneTypeLevelBomber)
	medium := aaHitChance(127, objUnit.PlaneTypeLevelBomber)
	bigDP := aaHitChance(152, objUnit.PlaneTypeLevelBomber)
	large := aaHitChance(203, objUnit.PlaneTypeLevelBomber)

	if !(small > medium && medium > bigDP && bigDP > large) {
		t.Fatalf(
			"hit chance should decrease with caliber: small=%.4f medium=%.4f bigDP=%.4f large=%.4f",
			small,
			medium,
			bigDP,
			large,
		)
	}
	if small != 1.0/6 {
		t.Fatalf("small caliber (<=40mm) hit chance = %.4f, want %.4f", small, 1.0/6)
	}
	if medium != 1.0/12 {
		t.Fatalf("medium caliber (<=130mm) hit chance = %.4f, want %.4f", medium, 1.0/12)
	}
	if bigDP != 1.0/16 {
		t.Fatalf("big DP caliber (<=155mm) hit chance = %.4f, want %.4f", bigDP, 1.0/16)
	}
	if large != 1.0/24 {
		t.Fatalf("large caliber (>155mm) hit chance = %.4f, want %.4f", large, 1.0/24)
	}
	// 边界值：40mm 归小口径，130mm 归中口径，152/155mm 归大口径高平两用炮
	if aaHitChance(40, objUnit.PlaneTypeLevelBomber) != small {
		t.Fatal("40mm should use the small caliber tier")
	}
	if aaHitChance(130, objUnit.PlaneTypeLevelBomber) != medium {
		t.Fatal("130mm should use the medium caliber tier")
	}
	for _, diameter := range []int{131, 152, 155} {
		if aaHitChance(diameter, objUnit.PlaneTypeLevelBomber) != bigDP {
			t.Fatalf("%dmm should use the big DP caliber tier", diameter)
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
	diveDP := aaHitChance(152, objUnit.PlaneTypeDiveBomber)
	if diveDP*3 != bigDP {
		t.Fatalf("dive bomber DP hit chance = %.4f, want bigDP/3 = %.4f", diveDP, bigDP/3)
	}
}
