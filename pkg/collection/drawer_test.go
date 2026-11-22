package collection

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/resources/font"
)

// TestCollectionCardSeparatorClearsFontGlyphs 验证图鉴卡片标题分隔线位于字形下缘之下。
// 场景：标题以 card.Y+18 为顶端绘制，不同语言字体的 ascent/descent 不同；日文 Zen Kaku 的
// 字底明显低于简体楷体、繁体粉圆和拉丁字体，旧版只给日文下移 8px。
// 预期：日文分隔线为 card.Y+52，其他语言为 card.Y+44，且都大于标题顶端+ascent+descent；
// 失败说明分隔线压到标题或字体分支漏判。
// 覆盖：表驱动 zh-Hans/zh-Hant/ja/en，用 ForLanguage 实际字体 Metrics 计算字底，断言偏移与竖直间距。
func TestCollectionCardSeparatorClearsFontGlyphs(t *testing.T) {
	const (
		cardY = 100.0
		scale = 1.0
	)
	for _, tc := range []struct {
		lang       i18n.Language
		wantOffset float64
	}{
		{i18n.LanguageZhHans, 44},
		{i18n.LanguageZhHant, 44},
		{i18n.LanguageJapanese, 52},
		{i18n.LanguageEnglish, 44},
	} {
		titleSize := 22.0
		if tc.lang.UsesLatinVisualScale() {
			titleSize *= 0.82
		}
		face := &text.GoTextFace{Source: font.ForLanguage(tc.lang, font.Kai), Size: titleSize}
		metrics := face.Metrics()
		textBottom := cardY + 18*scale + metrics.HAscent + metrics.HDescent

		separatorY := collectionCardSeparatorY(cardY, scale, tc.lang)
		require.Equal(t, cardY+tc.wantOffset*scale, separatorY)
		require.Greaterf(
			t, separatorY, textBottom,
			"%s separator %.2f crosses text bottom %.2f", tc.lang, separatorY, textBottom,
		)
	}
}
