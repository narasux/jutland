package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"

	"github.com/BurntSushi/toml"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

// TestDefaultAndFallbackLanguage 验证语言切换的默认值与非法输入回退。
// 场景：设置页依次选 zh-Hans/en/ru/ja，配置文件也可能写入 "invalid" 这类历史脏值。
// 预期：非法值回退到 zh-Hans；MsgMenuMissionSelect 依次输出 任务选择/Mission Select/
// Выбор миссии/ミッション選択；失败表示 NormalizeLanguage 或默认目录失效。
// 覆盖：SetLanguage 五次调用后断言 CurrentLanguage 与 Text，Cleanup 复位 zh-Hans。
func TestDefaultAndFallbackLanguage(t *testing.T) {
	t.Cleanup(func() {
		SetLanguage(string(LanguageZhHans))
	})
	require.Equal(t, LanguageZhHans, SetLanguage("zh-Hans"))
	require.Equal(t, LanguageZhHans, CurrentLanguage())
	require.Equal(t, LanguageZhHans, SetLanguage("invalid"))
	require.Equal(t, "任务选择", Text(MsgMenuMissionSelect))
	require.Equal(t, "资金: 123", Format(MsgSidebarFunds, map[string]any{"Funds": 123}))
	require.Equal(t, LanguageEnglish, SetLanguage("en"))
	require.Equal(t, "Mission Select", Text(MsgMenuMissionSelect))
	require.Equal(t, LanguageRussian, SetLanguage("ru"))
	require.Equal(t, "Выбор миссии", Text(MsgMenuMissionSelect))
	require.Equal(t, LanguageJapanese, SetLanguage("ja"))
	require.Equal(t, "ミッション選択", Text(MsgMenuMissionSelect))
}

// TestFallbackLanguageOrder 验证各语言的查找顺序为 自身 → 英文 → 简体中文，且无重复。
// 场景：俄语/日语目录缺词时先用英文兜底，英文也缺才落到中文；中文只需查自己。
// 预期：ru→[ru,en,zh-Hans]、ja→[ja,en,zh-Hans]、en→[en,zh-Hans]、zh-Hans→[zh-Hans]。
// 覆盖：对四种 Language 调用 FallbackLanguages，用 require.Equal 比较完整切片的元素与顺序。
func TestFallbackLanguageOrder(t *testing.T) {
	require.Equal(t, []Language{LanguageRussian, LanguageEnglish, LanguageZhHans}, FallbackLanguages(LanguageRussian))
	require.Equal(
		t,
		[]Language{LanguageJapanese, LanguageEnglish, LanguageZhHans},
		FallbackLanguages(LanguageJapanese),
	)
	require.Equal(t, []Language{LanguageEnglish, LanguageZhHans}, FallbackLanguages(LanguageEnglish))
	require.Equal(t, []Language{LanguageZhHans}, FallbackLanguages(LanguageZhHans))
}

// TestRussianPluralForms 验证俄语复数按 CLDR 规则在 one/few/many 三形间切换。
// 场景：收藏页统计 1、2、5、21 架飞机；俄语 21 与 1 同属 one，2..4 为 few，5..20 为 many。
// 预期：1→"Всего 1 самолёт"、2→"Всего 2 самолёта"、5→"Всего 5 самолётов"、
// 21→"Всего 21 самолёт"；失败表示 PluralCount 没透传或 TOML 复数形缺失。
// 覆盖：切到 ru 后以 Count=n 调用 Plural(MsgCollectionPlaneCount,n,...)，Cleanup 复位 zh-Hans。
func TestRussianPluralForms(t *testing.T) {
	t.Cleanup(func() { SetLanguage(string(LanguageZhHans)) })
	SetLanguage(string(LanguageRussian))
	require.Equal(t, "Всего 1 самолёт", Plural(MsgCollectionPlaneCount, 1, map[string]any{"Count": 1}))
	require.Equal(t, "Всего 2 самолёта", Plural(MsgCollectionPlaneCount, 2, map[string]any{"Count": 2}))
	require.Equal(t, "Всего 5 самолётов", Plural(MsgCollectionPlaneCount, 5, map[string]any{"Count": 5}))
	require.Equal(t, "Всего 21 самолёт", Plural(MsgCollectionPlaneCount, 21, map[string]any{"Count": 21}))
}

// TestRussianCollectionDropdownLabelLeavesArrowToUI 验证俄语舰级标签不内嵌下拉箭头。
// 场景：收藏页下拉框文案只提供 "Класс: <舰级>"，箭头 ↓ 由控件单独绘制；译名自带箭头会重复。
// 预期：Format(MsgCollectionShipClass,{"Value":"Авианосец"}) 精确等于 "Класс: Авианосец"，
// 末尾没有 ↓；失败表示目录文案与 UI 绘制职责重叠，下拉框会出现两个箭头。
// 覆盖：切到 ru 后格式化一次并全等比较，Cleanup 复位 zh-Hans。
func TestRussianCollectionDropdownLabelLeavesArrowToUI(t *testing.T) {
	t.Cleanup(func() { SetLanguage(string(LanguageZhHans)) })
	SetLanguage(string(LanguageRussian))

	require.Equal(
		t,
		"Класс: Авианосец",
		Format(MsgCollectionShipClass, map[string]any{"Value": "Авианосец"}),
	)
}

// TestEnglishPluralWithTestCatalog 验证 localizeWith 会把 pluralCount 交给 go-i18n 选复数形。
// 场景：用独立 Bundle 注册临时 TOML（Ships 的 one/other），排除生产目录缺词对结论的干扰。
// 预期：Count=1 得 "1 ship"，Count=2 得 "2 ships"；失败表示 pluralCount 被忽略或模板数据未绑定。
// 覆盖：ParseMessageFileBytes 建测试目录，NewLocalizer("en") 后分别用 count 1/2 调 localizeWith。
func TestEnglishPluralWithTestCatalog(t *testing.T) {
	testBundle := goi18n.NewBundle(language.English)
	testBundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	_, err := testBundle.ParseMessageFileBytes([]byte(`
[Ships]
one = "{{.Count}} ship"
other = "{{.Count}} ships"
`), "active.en.toml")
	require.NoError(t, err)
	localizer := goi18n.NewLocalizer(testBundle, "en")

	one, err := localizeWith(localizer, "Ships", map[string]any{"Count": 1}, 1)
	require.NoError(t, err)
	require.Equal(t, "1 ship", one)
	other, err := localizeWith(localizer, "Ships", map[string]any{"Count": 2}, 2)
	require.NoError(t, err)
	require.Equal(t, "2 ships", other)
}

// TestEnabledCatalogsMatchDeclaredMessageIDs 验证四语言 TOML 与 messages.go 的消息 ID 双向一致。
// 场景：新增/改名文案时容易只补中文或漏删旧键，四个目录必须覆盖代码引用的全部 ID。
// 预期：每个 SupportedLanguages 目录既不缺 declared 的 ID，也不含未声明的键；失败指出缺失/多余消息。
// 覆盖：go/ast 扫描 messages.go 收集字符串常量，逐语言解码 locales/active.<lang>.toml 后双向比对。
func TestEnabledCatalogsMatchDeclaredMessageIDs(t *testing.T) {
	declared := declaredMessageIDs(t)
	for _, lang := range SupportedLanguages() {
		path := "locales/active." + string(lang) + ".toml"
		bytes, err := localeFiles.ReadFile(path)
		require.NoError(t, err)
		catalog := map[string]map[string]any{}
		_, err = toml.Decode(string(bytes), &catalog)
		require.NoError(t, err)

		for id := range declared {
			_, ok := catalog[id]
			require.Truef(t, ok, "message %s is missing from %s catalog", id, lang)
		}
		for id := range catalog {
			_, ok := declared[id]
			require.Truef(t, ok, "%s catalog contains unknown message %s", lang, id)
		}
	}
}

func declaredMessageIDs(t *testing.T) map[string]struct{} {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "messages.go", nil, 0)
	require.NoError(t, err)
	ids := map[string]struct{}{}
	ast.Inspect(file, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok || len(spec.Values) != 1 {
			return true
		}
		literal, ok := spec.Values[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		require.NoError(t, err)
		ids[value] = struct{}{}
		return true
	})
	return ids
}
