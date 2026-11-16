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

// TestLanguageSelectorCoversSettingsText 验证设置页语言选择器的三字体复合覆盖所有选项文字。
// 场景：下拉项含「慢/标准/快」和四种语言名；复合字体缺任一字形就会显示方块。
// 预期：Kai+ZenKakuGothicNew+GolosText 的 Cmap 至少有一个覆盖每个 rune；
// LanguageSelectorFace(22) 返回非 nil。
// 覆盖：逐字符串逐 rune 查三个 GoTextFaceSource 的内部 Face.Cmap，用 require.Truef 报告缺失字形。
func TestLanguageSelectorCoversSettingsText(t *testing.T) {
	sources := []*text.GoTextFaceSource{Kai, ZenKakuGothicNew, GolosText}
	for _, value := range []string{"慢", "标准", "快", "中文", "English", "Русский", "日本語"} {
		for _, r := range value {
			covered := false
			for _, source := range sources {
				face := source.UnsafeInternal().(*goTextFont.Face)
				if _, ok := face.Cmap.Lookup(r); ok {
					covered = true
					break
				}
			}
			require.Truef(t, covered, "no selector font covers %q in %q", r, value)
		}
	}
	require.NotNil(t, LanguageSelectorFace(22))
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
