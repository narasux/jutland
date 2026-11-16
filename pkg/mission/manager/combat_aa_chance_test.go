package manager

import (
	"testing"
)

// TestAaHitChanceCaliberTiers 验证舰对空命中率按弹药口径分三档，且 40/155/156mm 边界归属正确。
// 场景：固定 FlakVulnerability=1，比较 20mm、127mm、203mm，再逐个检查 40、41/127/155、156mm 的档位。
// 预期：20mm=1/5、127mm=1/9、203mm=1/14 且严格递减；40mm 仍小口径，41~155mm 中口径，156mm 起大口径。
// 覆盖：aaHitChance 的 ≤40/≤155/>155 分支与三个命中率常数，精确比较浮点值而不只比大小。
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

// TestAaHitChanceScalesWithVulnerability 验证命中率对飞机 FlakVulnerability 线性缩放，零值与负值回退常规。
// 场景：以 127mm、脆弱度 1 的 1/9 为基准，分别取 1/3、2、0、-0.5 计算 aaHitChance。
// 预期：1/3 精确等于 base/3，2 精确翻倍；0 与 -0.5 都回退成 base，避免手工构造实例永远打不中。
// 覆盖：aaHitChance 的 vulnerability<=0 兜底与 chance*vulnerability 乘法路径，逐值断言。
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
