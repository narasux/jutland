package unit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/i18n"
	objRef "github.com/narasux/jutland/pkg/mission/object/reference"
)

func TestGetPlaneTitleCombinesCodeAndDisplayName(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })
	i18n.SetLanguage(string(i18n.LanguageZhHans))

	// 图鉴标题需要同时展示编号和可读名称，编号来自 reference 的 name（即配置中的 name）。
	objRef.SetReference(i18n.LanguageZhHans, "X-9", &objRef.Reference{DisplayName: "野猫"})
	require.Equal(t, "X-9 野猫", GetPlaneTitle("X-9"))

	// 编号里没有数字的条目只是占位名称（如测试机 F / B / A），不拼接编号。
	objRef.SetReference(i18n.LanguageZhHans, "F", &objRef.Reference{DisplayName: "歼击机"})
	require.Equal(t, "歼击机", GetPlaneTitle("F"))

	// 老数据里 displayName 已包含编号时不能重复拼接，空格和连字符写法差异也要识别。
	objRef.SetReference(i18n.LanguageZhHans, "Fw190A", &objRef.Reference{DisplayName: "Fw 190A 百舌鸟"})
	require.Equal(t, "Fw 190A 百舌鸟", GetPlaneTitle("Fw190A"))
	objRef.SetReference(i18n.LanguageZhHans, "N1K2", &objRef.Reference{DisplayName: "N1K2-J 紫电改二一型"})
	require.Equal(t, "N1K2-J 紫电改二一型", GetPlaneTitle("N1K2"))
	// 编号本身就是可读名称的一部分时不重复拼接。
	objRef.SetReference(i18n.LanguageZhHans, "Mosquito", &objRef.Reference{DisplayName: "Sea Mosquito TR.33"})
	require.Equal(t, "Sea Mosquito TR.33", GetPlaneTitle("Mosquito"))
	// 编号是基型（配置名带变体后缀）时不重复拼接。
	objRef.SetReference(i18n.LanguageZhHans, "Seafire", &objRef.Reference{DisplayName: "Seafire Mk III"})
	require.Equal(t, "Seafire Mk III", GetPlaneTitle("Seafire"))

	// 没有 reference 时退回配置名，保持与 GetPlaneDisplayName 一致的兜底行为。
	require.Equal(t, "unknown-plane", GetPlaneTitle("unknown-plane"))
}

// 日文、俄文图鉴里厂商名写在编号之前，拼接时要把编号插到厂商名之后。
func TestGetPlaneTitleKeepsManufacturerBeforeCode(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	objRef.SetReference(i18n.LanguageJapanese, "F4U-4", &objRef.Reference{DisplayName: "ヴォート F4U-4 コルセア（米海軍）"})
	objRef.SetReference(i18n.LanguageRussian, "F4U-4", &objRef.Reference{DisplayName: "Воут F4U-4 Корсар (ВМС США)"})
	objRef.SetReference(i18n.LanguageJapanese, "F2A-3", &objRef.Reference{DisplayName: "バッファロー"})

	i18n.SetLanguage(string(i18n.LanguageJapanese))
	require.Equal(t, "ヴォート F4U-4 コルセア（米海軍）", GetPlaneTitle("F4U-4"))
	require.Equal(t, "F2A-3 バッファロー", GetPlaneTitle("F2A-3"))
	i18n.SetLanguage(string(i18n.LanguageRussian))
	require.Equal(t, "Воут F4U-4 Корсар (ВМС США)", GetPlaneTitle("F4U-4"))
}

func TestGetPlaneTitleFollowsCurrentLanguage(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	objRef.SetReference(i18n.LanguageZhHans, "F4F-3-test", &objRef.Reference{DisplayName: "野猫"})
	objRef.SetReference(i18n.LanguageJapanese, "F4F-3-test", &objRef.Reference{DisplayName: "ワイルドキャット"})

	i18n.SetLanguage(string(i18n.LanguageZhHans))
	require.Equal(t, "F4F-3-test 野猫", GetPlaneTitle("F4F-3-test"))
	i18n.SetLanguage(string(i18n.LanguageJapanese))
	require.Equal(t, "F4F-3-test ワイルドキャット", GetPlaneTitle("F4F-3-test"))
}
