package manager

import (
	"testing"
)

// 舰对空命中率按弹药口径分级：小口径速射防空炮最高，中口径次之，
// >155mm 主炮客串防空直击概率最低。机种差异不再硬编码在本函数里，
// 而是乘上飞机自身的 FlakVulnerability，故这里统一以常规脆弱度 1 为基准。
func TestAaHitChanceCaliberTiers(t *testing.T) {
	small := aaHitChance(20, 1)
	medium := aaHitChance(127, 1)
	large := aaHitChance(203, 1)

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
	if aaHitChance(40, 1) != small {
		t.Fatal("40mm should use the small caliber tier")
	}
	for _, diameter := range []int{41, 127, 155} {
		if aaHitChance(diameter, 1) != medium {
			t.Fatalf("%dmm should use the medium caliber tier", diameter)
		}
	}
	if aaHitChance(156, 1) != large {
		t.Fatal("156mm should use the large caliber tier")
	}
}

// 脆弱度作为命中率乘数线性生效；零值与负数（未解析默认值的手工构造实例）
// 退化回常规，避免出现“永远打不中”。
func TestAaHitChanceScalesWithVulnerability(t *testing.T) {
	base := aaHitChance(127, 1)

	if got := aaHitChance(127, 1.0/3); got != base/3 {
		t.Fatalf("vulnerability 1/3 hit chance = %.6f, want base/3 = %.6f", got, base/3)
	}
	if got := aaHitChance(127, 2); got != base*2 {
		t.Fatalf("vulnerability 2 hit chance = %.6f, want base*2 = %.6f", got, base*2)
	}
	for _, vulnerability := range []float64{0, -0.5} {
		if got := aaHitChance(127, vulnerability); got != base {
			t.Fatalf(
				"vulnerability %v should fall back to normal, got %.6f want %.6f",
				vulnerability, got, base,
			)
		}
	}
}
