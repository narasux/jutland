package reference

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/stretchr/testify/require"
	"github.com/yosuke-furukawa/json5/encoding/json5"
)

// TestValidateLocalesKeepsStableReferenceShape 验证各语言同一条目结构一致时校验通过。
// 场景：遍历 SupportedLanguages，为每种语言构造 test_ship：Battleship、1 条武备、1 条链接。
// 预期：ValidateLocales 返回 nil；失败意味着跨语言条目数、武备数或链接数被误判。
// 覆盖：仅 DisplayName 随语言变化，其余字段一致；断言 NoError，覆盖全语言齐备分支。
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

// TestValidateLocalesAllowsLocalizedSourceURLs 验证各语言可使用不同来源 URL。
// 场景：每语言两条链接，把 ja/ru 的 History 链接换成 ja./ru.example.com 域名。
// 预期：校验仍通过；失败意味着本地化来源域名被判成结构不一致。
// 覆盖：先构造全语言一致条目，再替换两条 URL，断言 NoError，锁死只比较链接数量。
func TestValidateLocalesAllowsLocalizedSourceURLs(t *testing.T) {
	locales := map[i18n.Language][]Reference{}
	for _, lang := range i18n.SupportedLanguages() {
		locales[lang] = []Reference{{
			Name:        "test_ship",
			DisplayName: string(lang),
			Type:        "Battleship",
			Links: []Link{
				{Name: "Artwork", URL: "https://example.com/artwork"},
				{Name: "History", URL: "https://example.com/test_ship"},
			},
		}}
	}
	locales[i18n.LanguageJapanese][0].Links[1].URL = "https://ja.example.com/test_ship"
	locales[i18n.LanguageRussian][0].Links[1].URL = "https://ru.example.com/test_ship"

	require.NoError(t, ValidateLocales(locales))
}

// TestValidateLocalesRequiresEveryFormalLanguage 验证缺正式语言时校验必须报错。
// 场景：只提供 zh-Hans 与 en 两份 test 图鉴，各含 1 条来源链接，缺 ja/ru。
// 预期：返回错误且含 missing reference locale；失败意味着缺语言包能通过校验。
// 覆盖：构造两语言 map，断言 Error 与 ErrorContains，覆盖 SupportedLanguages 缺失分支。
func TestValidateLocalesRequiresEveryFormalLanguage(t *testing.T) {
	locales := map[i18n.Language][]Reference{
		i18n.LanguageZhHans: {
			{Name: "test", DisplayName: "测试", Links: []Link{{Name: "来源", URL: "https://example.com/test"}}},
		},
		i18n.LanguageEnglish: {
			{Name: "test", DisplayName: "Test", Links: []Link{{Name: "Source", URL: "https://example.com/test"}}},
		},
	}

	err := ValidateLocales(locales)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing reference locale")
}

// TestJapaneseReferencesUseJapaneseYearTypeNames 验证日文图鉴不使用英文 Year Type 命名。
// 场景：Load references.ja.json5，扫描全部 DisplayName 与 Armaments[].Value，并核对重点炮名。
// 预期：无 Year Type；410/203mm 用三年式、127mm 用八九式；nagato 首项为 4x2 410mm/45 三年式。
// 覆盖：全量 NotContains + 三条炮名 Equal + nagato 武备值断言。
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

// TestJapaneseCarrierTypesUseShortNavalLabels 验证日文舰种标签使用短称谓而非完整航空母艦。
// 场景：Load references.ja.json5，扫描所有 Type，并核对 yorktown/taiho/shinano/queen。
// 预期：Type 不含 航空母艦；四舰依次为空母/装甲空母/重装甲空母/護衛空母，正文仍可含全称。
// 覆盖：全量 NotContains + 四条 Equal + yorktown 描述 Contains，锁定标签与正文口径分离。
func TestJapaneseCarrierTypesUseShortNavalLabels(t *testing.T) {
	references, err := Load(filepath.Join("..", "..", "..", "..", "configs", "references.ja.json5"))
	require.NoError(t, err)

	byName := make(map[string]Reference, len(references))
	for _, ref := range references {
		byName[ref.Name] = ref
		require.NotContains(t, ref.Type, "航空母艦", ref.Name)
	}

	require.Equal(t, "空母", byName["yorktown"].Type)
	require.Equal(t, "装甲空母", byName["taiho"].Type)
	require.Equal(t, "重装甲空母", byName["shinano"].Type)
	require.Equal(t, "護衛空母", byName["queen"].Type)
	require.Contains(t, byName["yorktown"].Description, "航空母艦")
}

// TestEveryConfiguredGunHasLocalizedReference 验证 guns.json5 每门炮在全部语言图鉴中都有条目。
// 场景：解析 guns.json5 的全部 name；逐语言 Load references.<lang>.json5，zh-Hans 用 references.json5。
// 预期：每门炮都能在该语言 byName 命中；失败意味着图鉴缺项会让侧栏引用为空。
// 覆盖：guns × SupportedLanguages 双层循环，Containsf 带语言和炮名定位缺失项。
func TestEveryConfiguredGunHasLocalizedReference(t *testing.T) {
	configDir := filepath.Join("..", "..", "..", "..", "configs")
	data, err := os.ReadFile(filepath.Join(configDir, "guns.json5"))
	require.NoError(t, err)
	var guns []struct {
		Name string `json:"name"`
	}
	require.NoError(t, json5.Unmarshal(data, &guns))

	for _, lang := range i18n.SupportedLanguages() {
		fileName := "references." + string(lang) + ".json5"
		if lang == i18n.LanguageZhHans {
			fileName = "references.json5"
		}
		references, err := Load(filepath.Join(configDir, fileName))
		require.NoError(t, err)
		byName := make(map[string]Reference, len(references))
		for _, ref := range references {
			byName[ref.Name] = ref
		}
		for _, gun := range guns {
			require.Containsf(t, byName, gun.Name, "%s is missing gun reference %q", lang, gun.Name)
		}
	}
}

// TestEveryConfiguredGermanBombHasLocalizedReference 验证德制炸弹在所有语言图鉴中都有条目。
// 场景：解析 releasers.json5，只筛 Name 前缀为 GER/BB/ 的投放器，再逐语言加载图鉴。
// 预期：每个德制炸弹名在每种语言中都存在；失败意味着德舰挂载缺少本地化来源与说明。
// 覆盖：strings.HasPrefix 过滤 + 每语言 Containsf；非 GER/BB/ 投放器不参与断言。
func TestEveryConfiguredGermanBombHasLocalizedReference(t *testing.T) {
	configDir := filepath.Join("..", "..", "..", "..", "configs")
	data, err := os.ReadFile(filepath.Join(configDir, "releasers.json5"))
	require.NoError(t, err)
	var releasers []struct {
		Name string `json:"name"`
	}
	require.NoError(t, json5.Unmarshal(data, &releasers))

	for _, lang := range i18n.SupportedLanguages() {
		fileName := "references." + string(lang) + ".json5"
		if lang == i18n.LanguageZhHans {
			fileName = "references.json5"
		}
		references, err := Load(filepath.Join(configDir, fileName))
		require.NoError(t, err)
		byName := make(map[string]Reference, len(references))
		for _, ref := range references {
			byName[ref.Name] = ref
		}
		for _, releaser := range releasers {
			if !strings.HasPrefix(releaser.Name, "GER/BB/") {
				continue
			}
			require.Containsf(
				t, byName, releaser.Name, "%s is missing German bomb reference %q",
				lang, releaser.Name,
			)
		}
	}
}

// TestLocalizedReferencesUseCompleteDescriptions 验证 ja/ru 图鉴描述与英文完整度一致且无模板残留。
// 场景：加载 en/ja/ru 三份图鉴，逐条比较描述是否存在，并核对 hakuryu 与日文链接名。
// 预期：三语条目数相同，ja/ru 描述有无与 en 一致；无「に関する参考情報です」等模板套话。
// 覆盖：hakuryu 日文含 G-15/大鳳、俄文含 G-15/Taihō；日文链接名不含 "Wikipedia —"。
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
