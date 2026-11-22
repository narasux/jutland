package theme

import (
	"image/color"
	"math"
	"sort"
	"testing"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/resources/font"
)

// TestChevronDirectionsAreCongruent 验证四个方向的 V 形箭头折线全等且尖端朝向正确。
// 场景：以 size=12 的 TriangleUp 为基准，比较 TriangleDown/Left/Right 的三点两两距离；DrawChevron 供下拉把手与单位面板导航共用。
// 预期：各方向排序后的三条边长与 up 基准误差 ≤1e-9；up 顶点 y=-size/2、left 顶点 x=-size/2，保证箭头方向不反转。
// 覆盖：分别调用 chevronPoints 取 start/apex/end，经 sortedPairDistances 比较，再单独断言 up/left 的尖端坐标。
func TestChevronDirectionsAreCongruent(t *testing.T) {
	const size = 12.0
	start, apex, end := chevronPoints(0, 0, size, TriangleUp)
	ref := sortedPairDistances([3][2]float64{start, apex, end})
	for _, dir := range []TriangleDir{TriangleDown, TriangleLeft, TriangleRight} {
		s, a, e := chevronPoints(0, 0, size, dir)
		got := sortedPairDistances([3][2]float64{s, a, e})
		for i := range ref {
			if math.Abs(ref[i]-got[i]) > 1e-9 {
				t.Fatalf("dir %d segment lengths %v differ from up %v", dir, got, ref)
			}
		}
	}

	// 箭头尖端方向正确：朝上时顶点在最上方，朝左时顶点在最左侧。
	_, apexUp, _ := chevronPoints(0, 0, size, TriangleUp)
	if apexUp[1] != -size/2 {
		t.Fatalf("up chevron tip y = %v, want %v", apexUp[1], -size/2)
	}
	_, apexLeft, _ := chevronPoints(0, 0, size, TriangleLeft)
	if apexLeft[0] != -size/2 {
		t.Fatalf("left chevron tip x = %v, want %v", apexLeft[0], -size/2)
	}
}

// sortedPairDistances 返回三点两两距离的升序列表，用于比较三角形全等。
func sortedPairDistances(p [3][2]float64) [3]float64 {
	dists := [3]float64{
		dist(p[0], p[1]),
		dist(p[1], p[2]),
		dist(p[0], p[2]),
	}
	sort.Float64s(dists[:])
	return dists
}

func dist(a, b [2]float64) float64 {
	return math.Hypot(a[0]-b[0], a[1]-b[1])
}

// TestPanelTokensAreDistinct 验证共享色板中语义不同的关键 token 不会调成同一个颜色。
// 场景：右侧 sidebar 与底部 unitpanel 共用 PanelBackground/CardBackground、ButtonIdle/Hover/Pressed、HandleFill/Hover、ProgressTrack/Fill、ButtonDisabled/Mixed。
// 预期：PanelBackground×CardBackground、ButtonIdle×{Hover,Pressed}、HandleFill×Hover、ProgressTrack×Fill、ButtonDisabled×Mixed 六对颜色必须全部不相等；任一相等都说明面板层次或按钮状态在视觉上无法区分，调色时误伤。
// 覆盖：对上述六对 color.RGBA token 做 == 比较，归入五个失败断言，每组失败给出对应语义的 Fatal。
func TestPanelTokensAreDistinct(t *testing.T) {
	if PanelBackground == CardBackground {
		t.Fatal("panel background must differ from card background")
	}
	if ButtonIdle == ButtonHover || ButtonIdle == ButtonPressed {
		t.Fatal("button states must be visually distinct")
	}
	if HandleFill == HandleHover {
		t.Fatal("handle states must be visually distinct")
	}
	if ProgressTrack == ProgressFill {
		t.Fatal("progress track must differ from fill")
	}
	if ButtonDisabled == ButtonMixed {
		t.Fatal("disabled and mixed button states must differ")
	}
}

// TestNumericFaceSupportsLatinDigitsForEveryLanguage 验证五种语言下 Numeric() 都返回等宽拉丁字体。
// 场景：依次切换到 zh-Hans、zh-Hant、en、ru、ja，数值列只含拉丁数字与单位，不应依赖 CJK/西里尔字形而缺字。
// 预期：每种语言的 Numeric() 都等于 font.JetbrainsMono；结束后恢复 zh-Hans，避免污染后续测试。
// 覆盖：遍历五个 i18n.Language 调用 SetLanguage 和 Numeric，比较字体并在末尾重置。
func TestNumericFaceSupportsLatinDigitsForEveryLanguage(t *testing.T) {
	languages := []i18n.Language{
		i18n.LanguageZhHans,
		i18n.LanguageZhHant,
		i18n.LanguageEnglish,
		i18n.LanguageRussian,
		i18n.LanguageJapanese,
	}
	for _, lang := range languages {
		_ = i18n.SetLanguage(string(lang))
		face := Numeric()
		if face != font.JetbrainsMono {
			t.Fatalf("Numeric() for %s = %v, want JetbrainsMono", lang, font.FontToStr(face))
		}
	}
	_ = i18n.SetLanguage(string(i18n.LanguageZhHans))
}

// TestColorContrastHandleForArrows 验证把手填充与箭头前景的对比度仍足以辨认箭头。
// 场景：HandleFill (18,52,62,252) 与 HandleArrow (150,175,185,200) 被刻意弱化，但展开/收起箭头必须可读；按 WCAG 相对亮度公式计算对比度。
// 预期：contrastRatio(HandleFill, HandleArrow) ≥ 2.5；失败说明调色后箭头会糊在把手里。
// 覆盖：调用 contrastRatio，内部用 relativeLuminance 对 RGBA 做 sRGB 线性化后比较。
func TestColorContrastHandleForArrows(t *testing.T) {
	if ratio := contrastRatio(HandleFill, HandleArrow); ratio < 2.5 {
		t.Fatalf("handle arrow contrast = %.2f, want at least 2.5", ratio)
	}
}

func contrastRatio(a, b color.Color) float64 {
	left, right := relativeLuminance(a), relativeLuminance(b)
	if left < right {
		left, right = right, left
	}
	return (left + 0.05) / (right + 0.05)
}

func relativeLuminance(value color.Color) float64 {
	r, g, b, _ := value.RGBA()
	linear := func(channel uint32) float64 {
		v := float64(channel) / 65535
		if v <= 0.04045 {
			return v / 12.92
		}
		return ((v + 0.055) / 1.055) * ((v + 0.055) / 1.055)
	}
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
}
