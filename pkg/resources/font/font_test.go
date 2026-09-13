package font

import (
	"testing"

	goTextFont "github.com/go-text/typesetting/font"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/stretchr/testify/require"
)

func TestForLanguage(t *testing.T) {
	require.Same(t, Kai, ForLanguage(i18n.LanguageZhHans, Kai))
	require.Same(t, JetbrainsMono, ForLanguage(i18n.LanguageEnglish, Kai))
	require.Same(t, JetbrainsMono, ForLanguage(i18n.LanguageEnglish, Hang))
	require.Same(t, GolosText, ForLanguage(i18n.LanguageRussian, Kai))
	require.Same(t, ZenKakuGothicNew, ForLanguage(i18n.LanguageJapanese, Kai))
}

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
