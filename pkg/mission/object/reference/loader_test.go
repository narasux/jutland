package reference

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/stretchr/testify/require"
)

func TestValidateLocalesKeepsStableReferenceShape(t *testing.T) {
	locales := map[i18n.Language][]Reference{}
	for _, lang := range i18n.SupportedLanguages() {
		locales[lang] = []Reference{{
			Name:        "test_ship",
			DisplayName: string(lang),
			Type:        "Battleship",
			Armaments:   []InfoItem{{Label: "Main Battery", Value: "1x2 380mm"}},
			Links:       []Link{{Name: "History", URL: "https://example.com/test_ship"}},
		}}
	}

	require.NoError(t, ValidateLocales(locales))
}

func TestValidateLocalesRejectsLinkURLDrift(t *testing.T) {
	locales := map[i18n.Language][]Reference{}
	for _, lang := range i18n.SupportedLanguages() {
		locales[lang] = []Reference{{
			Name:        "test_ship",
			DisplayName: string(lang),
			Type:        "Battleship",
			Links:       []Link{{Name: "History", URL: "https://example.com/test_ship"}},
		}}
	}
	locales[i18n.LanguageJapanese][0].Links[0].URL = "https://example.com/different"

	err := ValidateLocales(locales)
	require.Error(t, err)
	require.Contains(t, err.Error(), "link 0 URL differs")
}

func TestValidateLocalesRequiresEveryFormalLanguage(t *testing.T) {
	locales := map[i18n.Language][]Reference{
		i18n.LanguageZhHans:  {{Name: "test", DisplayName: "测试", Links: []Link{{Name: "来源", URL: "https://example.com/test"}}}},
		i18n.LanguageEnglish: {{Name: "test", DisplayName: "Test", Links: []Link{{Name: "Source", URL: "https://example.com/test"}}}},
	}

	err := ValidateLocales(locales)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing reference locale")
}

func TestJapaneseReferencesUseJapaneseYearTypeNames(t *testing.T) {
	references, err := Load(filepath.Join("..", "..", "..", "..", "configs", "references.ja.json5"))
	require.NoError(t, err)

	byName := make(map[string]Reference, len(references))
	for _, ref := range references {
		byName[ref.Name] = ref
		require.NotContains(t, ref.DisplayName, "Year Type", ref.Name)
		for _, armament := range ref.Armaments {
			require.NotContains(t, armament.Value, "Year Type", ref.Name)
		}
	}

	require.Equal(t, "三年式 410mm 連装砲", byName["JP/410/45/2/Type3"].DisplayName)
	require.Equal(t, "三年式二号 203mm 連装砲", byName["JP/203/50/2/Type3"].DisplayName)
	require.Equal(t, "八九式 127mm 連装高角砲", byName["JP/127/40/2/Type89/DP"].DisplayName)

	nagato := byName["nagato"]
	require.NotEmpty(t, nagato.Armaments)
	require.Equal(t, "4x2 410mm/45 三年式", nagato.Armaments[0].Value)
	for _, armament := range nagato.Armaments {
		require.NotContains(t, armament.Value, "Type ")
	}
}

func TestLocalizedReferencesUseCompleteDescriptions(t *testing.T) {
	loadLocale := func(t *testing.T, locale string) map[string]Reference {
		t.Helper()
		references, err := Load(filepath.Join("..", "..", "..", "..", "configs", "references."+locale+".json5"))
		require.NoError(t, err)

		byName := make(map[string]Reference, len(references))
		for _, ref := range references {
			byName[ref.Name] = ref
		}
		return byName
	}

	english := loadLocale(t, "en")
	japanese := loadLocale(t, "ja")
	russian := loadLocale(t, "ru")
	require.Len(t, japanese, len(english))
	require.Len(t, russian, len(english))

	for name, enRef := range english {
		for locale, ref := range map[string]Reference{
			"ja": japanese[name],
			"ru": russian[name],
		} {
			require.Equal(
				t,
				strings.TrimSpace(enRef.Description) != "",
				strings.TrimSpace(ref.Description) != "",
				"%s reference %q description presence differs from English",
				locale,
				name,
			)
			require.NotContains(t, ref.Description, "に関する参考情報です", name)
			require.NotContains(t, ref.Description, "以下の出典を参照してください", name)
			require.NotContains(t, ref.Description, "Справочная информация", name)
			require.NotContains(t, ref.Description, "Исторические сведения и характеристики приведены", name)
		}
	}

	require.Contains(t, japanese["hakuryu"].Description, "G-15")
	require.Contains(t, japanese["hakuryu"].Description, "大鳳")
	require.Contains(t, russian["hakuryu"].Description, "G-15")
	require.Contains(t, russian["hakuryu"].Description, "Taihō")

	for _, ref := range japanese {
		for _, link := range ref.Links {
			require.NotContains(t, link.Name, "Wikipedia —", ref.Name)
		}
	}
}
