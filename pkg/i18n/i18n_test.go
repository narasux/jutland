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

func TestDefaultAndFallbackLanguage(t *testing.T) {
	require.Equal(t, LanguageZhHans, CurrentLanguage())
	require.Equal(t, LanguageZhHans, SetLanguage("invalid"))
	require.Equal(t, "任务选择", Text(MsgMenuMissionSelect))
}

func TestFormat(t *testing.T) {
	got := Format(MsgSidebarFunds, map[string]any{"Funds": 123})
	require.Equal(t, "资金: 123", got)
}

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
