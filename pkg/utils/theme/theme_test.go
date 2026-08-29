package theme

import (
	"image/color"
	"math"
	"sort"
	"testing"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/resources/font"
)

// TestChevronDirectionsAreCongruent 校验四个方向的 V 形箭头折线形状一致，
// 保证下拉把手与单位面板导航按钮使用同一套几何，不会出现形状差异。
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

// TestPanelTokensAreDistinctEnsures共享色板的几组关键 token 语义不同，避免调色时误伤。
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

// TestNumericFaceSupportsLatinDigitsForEveryLanguage保证数值在四种语言下都能用
// 等宽字体渲染拉丁数字与单位，不依赖 CJK 字形，从而避免缺字。
func TestNumericFaceSupportsLatinDigitsForEveryLanguage(t *testing.T) {
	languages := []i18n.Language{
		i18n.LanguageZhHans,
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

// TestColorContrastHandleForArrows校验把手填充与箭头前景的对比度，确保箭头仍可辨认。
// 把手刻意弱化，但对比度仍需满足可读阈值。
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
