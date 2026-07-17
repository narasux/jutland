package reference

import (
	"fmt"
	"net/url"
	"os"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/yosuke-furukawa/json5/encoding/json5"
)

// Load 从 JSON5 文件读取图鉴引用配置。
func Load(path string) ([]Reference, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var references []Reference
	if err := json5.Unmarshal(data, &references); err != nil {
		return nil, err
	}
	return references, nil
}

// ValidateLocales 校验所有图鉴语言的对象和稳定数据保持同步。
func ValidateLocales(locales map[i18n.Language][]Reference) error {
	validated := make(map[i18n.Language]map[string]Reference, len(locales))
	for lang, references := range locales {
		byName, err := validateReferences(references, string(lang))
		if err != nil {
			return err
		}
		validated[lang] = byName
	}
	chinese := validated[i18n.LanguageZhHans]
	english := validated[i18n.LanguageEnglish]
	if chinese == nil || english == nil {
		return fmt.Errorf("zh-Hans and en reference locales are required")
	}
	for _, lang := range i18n.SupportedLanguages() {
		localized := validated[lang]
		if localized == nil {
			return fmt.Errorf("missing reference locale %s", lang)
		}
		if len(localized) != len(chinese) {
			return fmt.Errorf(
				"locale object counts differ: zh-Hans=%d %s=%d",
				len(chinese), lang, len(localized),
			)
		}
		for name, zhRef := range chinese {
			ref, ok := localized[name]
			if !ok {
				return fmt.Errorf("%s is missing reference %q", lang, name)
			}
			if lang == i18n.LanguageEnglish && ref.Type != "Weapon" && ref.Type != "武器" && len(ref.Links) == 0 {
				return fmt.Errorf("en reference %q has no source links", name)
			}
			if len(zhRef.Armaments) != len(ref.Armaments) {
				return fmt.Errorf(
					"reference %q armament counts differ: zh-Hans=%d %s=%d",
					name, len(zhRef.Armaments), lang, len(ref.Armaments),
				)
			}
			if lang == i18n.LanguageRussian || lang == i18n.LanguageJapanese {
				enRef := english[name]
				if len(ref.Links) != len(enRef.Links) {
					return fmt.Errorf(
						"reference %q link counts differ: en=%d %s=%d",
						name, len(enRef.Links), lang, len(ref.Links),
					)
				}
			}
		}
	}
	return nil
}

func validateReferences(references []Reference, locale string) (map[string]Reference, error) {
	byName := make(map[string]Reference, len(references))
	for idx, ref := range references {
		if ref.Name == "" {
			return nil, fmt.Errorf("%s reference at index %d has no name", locale, idx)
		}
		if _, exists := byName[ref.Name]; exists {
			return nil, fmt.Errorf("%s contains duplicate reference %q", locale, ref.Name)
		}
		if ref.DisplayName == "" {
			return nil, fmt.Errorf("%s reference %q has no displayName", locale, ref.Name)
		}
		for linkIdx, link := range ref.Links {
			if link.Name == "" {
				return nil, fmt.Errorf("%s reference %q link %d has no name", locale, ref.Name, linkIdx)
			}
			parsed, err := url.ParseRequestURI(link.URL)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return nil, fmt.Errorf("%s reference %q link %d has invalid URL %q", locale, ref.Name, linkIdx, link.URL)
			}
		}
		byName[ref.Name] = ref
	}
	return byName, nil
}
