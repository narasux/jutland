package font

import (
	"testing"

	goTextFont "github.com/go-text/typesetting/font"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/stretchr/testify/require"
)

// TestForLanguage 验证 ForLanguage 按语言返回固定正文字体，中文沿用调用方传入的字体。
// 场景：设置页切到 zh-Hans/en/ru/ja 后，正文分别用楷体、JetBrains Mono、Golos、Zen Kaku。
// 预期：中文返回同一个 Kai 指针；英文无论传 Kai 还是 Hang 都返回 JetbrainsMono；
// 俄语返回 GolosText，日语返回 ZenKakuGothicNew；失败表示界面会回退到缺字形的字体。
// 覆盖：五次调用分别用 require.Same 比较返回的 *text.GoTextFaceSource 指针。
func TestForLanguage(t *testing.T) {
	require.Same(t, Kai, ForLanguage(i18n.LanguageZhHans, Kai))
	require.Same(t, JetbrainsMono, ForLanguage(i18n.LanguageEnglish, Kai))
	require.Same(t, JetbrainsMono, ForLanguage(i18n.LanguageEnglish, Hang))
	require.Same(t, GolosText, ForLanguage(i18n.LanguageRussian, Kai))
	require.Same(t, ZenKakuGothicNew, ForLanguage(i18n.LanguageJapanese, Kai))
}

// TestForTextFallsBackForCJKContent 验证按实际文本字形选字体，避免混排出现豆腐块。
// 场景：英文界面里出现舰名「初春」、日文「零式艦上戦闘機」、俄文「Русский」等非拉丁内容。
// 预期：含汉字且首选 JetbrainsMono 时回退 Kai；首选 Hang 时汉字仍用 Hang；俄文→GolosText；
// 纯英文保留 JetbrainsMono；含假名的「日本語インターフェース 艦戦闘機」→ZenKakuGothicNew。
// 覆盖：逐条调用 ForText；最后切到 ja 并注册 Cleanup 复位原语言，再断言假名混排用例。
func TestForTextFallsBackForCJKContent(t *testing.T) {
	require.Same(t, Kai, ForText("Ship: 初春", JetbrainsMono))
	require.Same(t, Kai, ForText("零式艦上戦闘機", JetbrainsMono))
	require.Same(t, JetbrainsMono, ForText("Ship: Hatsuharu", JetbrainsMono))
	require.Same(t, Hang, ForText("初春", Hang))
	require.Same(t, GolosText, ForText("Русский", JetbrainsMono))
	previousLanguage := i18n.CurrentLanguage()
	i18n.SetLanguage(string(i18n.LanguageJapanese))
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })
	require.Same(t, ZenKakuGothicNew, ForText("日本語インターフェース 艦戦闘機", JetbrainsMono))
}

// TestForLanguageCoversNativeNames 验证四种语言自称都能被各自界面字体完整覆盖。
// 场景：语言下拉每项用自己的字体渲染；若字体缺字（如楷体缺「語」），单词内会回退混排。
// 预期：中文由 Kai 覆盖、英文由 JetBrains Mono、俄文由 Golos、日文由 Zen Kaku 覆盖，
// 失败说明语言下拉会出现缺字或同词混排。
// 覆盖：遍历 SupportedLanguages，取 ForLanguage(lang, Kai) 的内部 Cmap 查 NativeName 的每个 rune。
func TestForLanguageCoversNativeNames(t *testing.T) {
	for _, lang := range i18n.SupportedLanguages() {
		source := ForLanguage(lang, Kai)
		face := source.UnsafeInternal().(*goTextFont.Face)
		for _, r := range lang.NativeName() {
			_, ok := face.Cmap.Lookup(r)
			require.Truef(t, ok, "%s is missing %q in %q", FontToStr(source), r, lang.NativeName())
		}
	}
}

// TestLocalizedUIFontsContainDropdownArrow 验证四种界面字体都含下拉箭头 U+2193（↓）。
// 场景：箭头由字体渲染而非贴图；俄语/日语字体缺该字形时按钮上会出现豆腐块。
// 预期：Kai、JetbrainsMono、GolosText、ZenKakuGothicNew 的 Cmap 都能查到 '↓'；
// 失败信息用 FontToStr 指出缺字形的字体。
// 覆盖：遍历四个 GoTextFaceSource，UnsafeInternal 转 *goTextFont.Face 后 Lookup 箭头。
func TestLocalizedUIFontsContainDropdownArrow(t *testing.T) {
	for _, source := range []*text.GoTextFaceSource{
		Kai,
		JetbrainsMono,
		GolosText,
		ZenKakuGothicNew,
	} {
		face := source.UnsafeInternal().(*goTextFont.Face)
		_, ok := face.Cmap.Lookup('↓')
		require.Truef(t, ok, "%s is missing the dropdown arrow", FontToStr(source))
	}
}
