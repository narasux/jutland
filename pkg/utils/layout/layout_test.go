package layout

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/resources/font"
)

// TestWrapWordsPreservesWordBoundaries 验证按词换行只在空格处断行，并用实测像素宽判界。
// 场景：英文/俄文界面用 JetbrainsMono 18px；maxWidth 取 "alpha beta" 实测宽减 1，整行差 1px 放不下。
// 预期：返回 ["alpha","beta"]，不会把 alpha 拆成 "alph"/"a"；失败表示英文换行会切断单词。
// 覆盖：以 JetbrainsMono、size=18、CalcTextWidth("alpha beta")-1 为上限，全等比较 wrapWords 结果。
func TestWrapWordsPreservesWordBoundaries(t *testing.T) {
	source := font.JetbrainsMono
	size := 18.0
	maxWidth := CalcTextWidth("alpha beta", size, source) - 1

	require.Equal(t, []string{"alpha", "beta"}, wrapWords("alpha beta", maxWidth, size, source))
}

// TestWrapRunesUsesMeasuredWidth 验证逐 rune 换行按实测宽度断行，放不下的字落到下一行。
// 场景：中文/日文无空格，只能逐字断行；maxWidth 取 "中文" 实测宽加 1，第三个字放不下。
// 预期：返回 ["中文","换行"]；若把宽度当字符数，会得到 ["中","文换","行"] 之类的切分。
// 覆盖：以 Kai、size=18、CalcTextWidth("中文")+1 为上限，全等比较 wrapRunes("中文换行") 结果。
func TestWrapRunesUsesMeasuredWidth(t *testing.T) {
	source := font.Kai
	size := 18.0
	maxWidth := CalcTextWidth("中文", size, source) + 1

	require.Equal(t, []string{"中文", "换行"}, wrapRunes("中文换行", maxWidth, size, source))
}
