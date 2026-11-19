package damage

import (
	"math"
	"testing"
)

// TestResolvePenetrationAngleAndOvermatch 验证纯几何穿深随入射角放大有效装甲并判定过穿。
// 场景：120mm 穿深对 100mm 装甲；0° 时应穿透但不过穿，60° 时有效装甲 200mm 应不穿。
// 预期：0° ratio=1.2、Penetrated=true、Overmatch=false；60° effective=200、ratio=0.6、Penetrated=false。
// 覆盖：垂直/斜入射两个边界，断言有效装甲、比率与布尔结果；浮点用容差。
func TestResolvePenetrationAngleAndOvermatch(t *testing.T) {
	flat := ResolvePenetration(120, 100, 0)
	if !flat.Penetrated || flat.Overmatch {
		t.Fatalf("flat = %+v, want penetrated not overmatch", flat)
	}
	if math.Abs(flat.Ratio-1.2) > 1e-9 {
		t.Fatalf("flat ratio = %v", flat.Ratio)
	}
	oblique := ResolvePenetration(120, 100, 60)
	if oblique.Penetrated || oblique.Overmatch {
		t.Fatalf("oblique = %+v, want no penetration", oblique)
	}
	if math.Abs(oblique.EffectiveArmorMM-200) > 1e-6 {
		t.Fatalf("effective armor = %v", oblique.EffectiveArmorMM)
	}
	if math.Abs(oblique.Ratio-0.6) > 1e-9 {
		t.Fatalf("oblique ratio = %v", oblique.Ratio)
	}
}

// TestResolvePenetrationZeroArmor 验证零装甲在正穿深下按无穷比率处理。
// 场景：无装甲目标被 50mm 穿深命中；避免除零并让分析表显示 inf。
// 预期：Ratio=+Inf、Penetrated/Overmatch=true、EffectiveArmor=0。
// 覆盖：armor=0 分支；断言 Inf 与布尔值。
func TestResolvePenetrationZeroArmor(t *testing.T) {
	result := ResolvePenetration(50, 0, 30)
	if !math.IsInf(result.Ratio, 1) || !result.Penetrated || !result.Overmatch {
		t.Fatalf("zero armor = %+v", result)
	}
}

// TestPenetrationMultiplier 验证实验穿甲倍率的未穿/穿透/过穿边界。
// 场景：ratio=0（下限）、0.5（部分穿）、1（刚好穿）、3（过穿封顶）。
// 预期：0.2、0.6、1.0、1.35；失败说明伤害映射发生跳变或没有封顶。
// 覆盖：四个比例点，浮点容差断言。
func TestPenetrationMultiplier(t *testing.T) {
	cases := []struct {
		ratio float64
		want  float64
	}{
		{0, 0.2},
		{0.5, 0.6},
		{1, 1},
		{3, 1.35},
	}
	for _, tc := range cases {
		if got := PenetrationMultiplier(tc.ratio); math.Abs(got-tc.want) > 1e-9 {
			t.Fatalf("ratio %v = %v, want %v", tc.ratio, got, tc.want)
		}
	}
}
