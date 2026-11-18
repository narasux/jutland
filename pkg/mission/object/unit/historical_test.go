package unit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/i18n"
)

// TestHistoricalStatusDisplayAndValidation 验证历史属性的取值校验与四语言徽标文案。
// 场景：图鉴会根据 HistoricalStatus 显示史实 / 图纸 / 架空，配置解析出未知字符串时不能静默当成史实。
// 预期：real/incomplete/prototype/limited/paper/fictional 均合法；空字符串与 "legendary" 非法；四种语言下六个状态分别输出对应译名。
// 覆盖：表驱动六个合法状态 × zh-Hans/en/ja/ru，逐项比较 ToDisplay，并单独断言 IsValid 的合法与非法分支。
func TestHistoricalStatusDisplayAndValidation(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	require.True(t, HistoricalStatusReal.IsValid())
	require.True(t, HistoricalStatusIncomplete.IsValid())
	require.True(t, HistoricalStatusPrototype.IsValid())
	require.True(t, HistoricalStatusLimited.IsValid())
	require.True(t, HistoricalStatusPaper.IsValid())
	require.True(t, HistoricalStatusFictional.IsValid())
	require.False(t, HistoricalStatus("").IsValid())
	require.False(t, HistoricalStatus("legendary").IsValid())

	cases := []struct {
		status HistoricalStatus
		lang   i18n.Language
		want   string
	}{
		{HistoricalStatusReal, i18n.LanguageZhHans, "史实"},
		{HistoricalStatusIncomplete, i18n.LanguageZhHans, "未完工"},
		{HistoricalStatusPrototype, i18n.LanguageZhHans, "仅试验机"},
		{HistoricalStatusLimited, i18n.LanguageZhHans, "少量生产"},
		{HistoricalStatusPaper, i18n.LanguageZhHans, "图纸"},
		{HistoricalStatusFictional, i18n.LanguageZhHans, "架空"},
		{HistoricalStatusReal, i18n.LanguageEnglish, "Real"},
		{HistoricalStatusIncomplete, i18n.LanguageEnglish, "Incomplete"},
		{HistoricalStatusPrototype, i18n.LanguageEnglish, "Prototype"},
		{HistoricalStatusLimited, i18n.LanguageEnglish, "Limited production"},
		{HistoricalStatusPaper, i18n.LanguageEnglish, "Paper"},
		{HistoricalStatusFictional, i18n.LanguageEnglish, "Fictional"},
		{HistoricalStatusReal, i18n.LanguageJapanese, "史実"},
		{HistoricalStatusIncomplete, i18n.LanguageJapanese, "未完成"},
		{HistoricalStatusPrototype, i18n.LanguageJapanese, "試作機"},
		{HistoricalStatusLimited, i18n.LanguageJapanese, "少数生産"},
		{HistoricalStatusPaper, i18n.LanguageJapanese, "図面"},
		{HistoricalStatusFictional, i18n.LanguageJapanese, "架空"},
		{HistoricalStatusReal, i18n.LanguageRussian, "Реальный"},
		{HistoricalStatusIncomplete, i18n.LanguageRussian, "Недостроен"},
		{HistoricalStatusPrototype, i18n.LanguageRussian, "Опытный"},
		{HistoricalStatusLimited, i18n.LanguageRussian, "Малая серия"},
		{HistoricalStatusPaper, i18n.LanguageRussian, "Проект"},
		{HistoricalStatusFictional, i18n.LanguageRussian, "Вымысел"},
	}
	for _, tc := range cases {
		i18n.SetLanguage(string(tc.lang))
		require.Equal(t, tc.want, tc.status.ToDisplay(), "%s/%s", tc.status, tc.lang)
	}
}
