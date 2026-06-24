package reference

import (
	"fmt"
	"net/url"
	"os"

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

// ValidateLocales 校验中英文图鉴配置的对象和稳定数据保持同步。
func ValidateLocales(chinese, english []Reference) error {
	chineseByName, err := validateReferences(chinese, "zh-Hans")
	if err != nil {
		return err
	}
	englishByName, err := validateReferences(english, "en")
	if err != nil {
		return err
	}
	if len(chineseByName) != len(englishByName) {
		return fmt.Errorf("locale object counts differ: zh-Hans=%d en=%d", len(chineseByName), len(englishByName))
	}
	for name, zhRef := range chineseByName {
		enRef, ok := englishByName[name]
		if !ok {
			return fmt.Errorf("en is missing reference %q", name)
		}
		if len(zhRef.Armaments) != len(enRef.Armaments) {
			return fmt.Errorf(
				"reference %q armament counts differ: zh-Hans=%d en=%d",
				name, len(zhRef.Armaments), len(enRef.Armaments),
			)
		}
		if len(enRef.Links) == 0 {
			return fmt.Errorf("en reference %q has no source links", name)
		}
	}
	for name := range englishByName {
		if _, ok := chineseByName[name]; !ok {
			return fmt.Errorf("en contains unknown reference %q", name)
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
