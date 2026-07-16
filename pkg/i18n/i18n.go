package i18n

import (
	"embed"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/BurntSushi/toml"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// Language 游戏界面语言。
type Language string

const (
	LanguageZhHans   Language = "zh-Hans"
	LanguageEnglish  Language = "en"
	LanguageRussian  Language = "ru"
	LanguageJapanese Language = "ja"
)

//go:embed locales/active.*.toml
var localeFiles embed.FS

type localizerState struct {
	language  Language
	localizer *goi18n.Localizer
}

var (
	enabledLanguages = []Language{
		LanguageZhHans,
		LanguageEnglish,
		LanguageRussian,
		LanguageJapanese,
	}
	bundle       = newBundle()
	currentState atomic.Pointer[localizerState]
	missingOnce  sync.Map
)

func init() {
	setCurrentLanguage(LanguageZhHans)
}

func newBundle() *goi18n.Bundle {
	b := goi18n.NewBundle(language.SimplifiedChinese)
	b.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	for _, lang := range enabledLanguages {
		path := fmt.Sprintf("locales/active.%s.toml", lang)
		if _, err := b.LoadMessageFileFS(localeFiles, path); err != nil {
			panic(fmt.Sprintf("load %s messages: %v", lang, err))
		}
	}
	return b
}

// SupportedLanguages 返回已经具备完整正式目录的语言。
func SupportedLanguages() []Language {
	return append([]Language(nil), enabledLanguages...)
}

// NativeName 返回语言的自称。
func (l Language) NativeName() string {
	switch l {
	case LanguageEnglish:
		return "English"
	case LanguageRussian:
		return "Русский"
	case LanguageJapanese:
		return "日本語"
	default:
		return "中文"
	}
}

// FallbackLanguages 返回指定语言的查找顺序。
func FallbackLanguages(lang Language) []Language {
	lang = NormalizeLanguage(string(lang))
	switch lang {
	case LanguageEnglish:
		return []Language{LanguageEnglish, LanguageZhHans}
	case LanguageRussian:
		return []Language{LanguageRussian, LanguageEnglish, LanguageZhHans}
	case LanguageJapanese:
		return []Language{LanguageJapanese, LanguageEnglish, LanguageZhHans}
	default:
		return []Language{LanguageZhHans}
	}
}

// LocalizedValue 按目标语言、英文、简体中文的顺序读取本地化值。
func LocalizedValue(values map[Language]string) string {
	for _, lang := range FallbackLanguages(CurrentLanguage()) {
		if value := values[lang]; value != "" {
			return value
		}
	}
	return ""
}

// UsesWordWrapping 表示该语言优先按空格分词换行。
func (l Language) UsesWordWrapping() bool {
	return l == LanguageEnglish || l == LanguageRussian
}

// UsesLatinVisualScale 表示该语言使用拉丁/西里尔字体的字号补偿。
func (l Language) UsesLatinVisualScale() bool {
	return l == LanguageEnglish || l == LanguageRussian
}

// NormalizeLanguage 将配置值修正为已支持语言。
func NormalizeLanguage(value string) Language {
	lang := Language(value)
	for _, supported := range SupportedLanguages() {
		if lang == supported {
			return lang
		}
	}
	return LanguageZhHans
}

// SetLanguage 切换当前语言；未启用语言回退到简体中文。
func SetLanguage(value string) Language {
	lang := NormalizeLanguage(value)
	setCurrentLanguage(lang)
	return lang
}

func setCurrentLanguage(lang Language) {
	preferences := FallbackLanguages(lang)
	tags := make([]string, len(preferences))
	for idx := range preferences {
		tags[idx] = string(preferences[idx])
	}
	currentState.Store(&localizerState{
		language:  lang,
		localizer: goi18n.NewLocalizer(bundle, tags...),
	})
}

// CurrentLanguage 返回当前界面语言。
func CurrentLanguage() Language {
	return currentState.Load().language
}

// Text 查询无参数消息。
func Text(id MessageID) string {
	return localize(id, nil, nil)
}

// Format 使用命名参数格式化消息。
func Format(id MessageID, data any) string {
	return localize(id, data, nil)
}

// Plural 根据 count 选择复数形式并使用命名参数格式化消息。
func Plural(id MessageID, count any, data any) string {
	return localize(id, data, count)
}

func localize(id MessageID, data, pluralCount any) string {
	state := currentState.Load()
	result, err := localizeWith(state.localizer, id, data, pluralCount)
	if err == nil {
		return result
	}
	if _, loaded := missingOnce.LoadOrStore(id, struct{}{}); !loaded {
		log.Printf("[ERROR] missing i18n message %q: %v", id, err)
	}
	return "[" + string(id) + "]"
}

func localizeWith(localizer *goi18n.Localizer, id MessageID, data, pluralCount any) (string, error) {
	return localizer.Localize(&goi18n.LocalizeConfig{
		MessageID:    string(id),
		TemplateData: data,
		PluralCount:  pluralCount,
	})
}
