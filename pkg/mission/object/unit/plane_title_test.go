package unit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/i18n"
	objRef "github.com/narasux/jutland/pkg/mission/object/reference"
)

// TestGetPlaneTitleUsesNicknameThenCode 验证图鉴标题“昵称（编号）”的拼接/去重/兜底规则。
// 场景：有昵称和数字编号（X-9）、纯占位名（F）、昵称已含编号（Ar 196/FF.33e/Fw 190A）、无 reference。
// 预期：数字编号追加括号；占位名不追加；昵称已含编号不重复；无 reference 退回配置名。
// 覆盖：逐条设置 reference 后调用 GetPlaneTitle 断言字符串。
func TestGetPlaneTitleUsesNicknameThenCode(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })
	i18n.SetLanguage(string(i18n.LanguageZhHans))

	// 图鉴标题统一为“昵称（编号）”，编号来自 reference 的 name（即配置中的 name）。
	objRef.SetReference(i18n.LanguageZhHans, "X-9", &objRef.Reference{DisplayName: "野猫"})
	require.Equal(t, "野猫（X-9）", GetPlaneTitle("X-9"))

	// 编号里没有数字的条目只是占位名称（如测试机 F / B / A），不拼接编号。
	objRef.SetReference(i18n.LanguageZhHans, "F", &objRef.Reference{DisplayName: "歼击机"})
	require.Equal(t, "歼击机", GetPlaneTitle("F"))

	// 昵称里已经包含编号时不再追加括号，避免“腓特烈港 FF.33e（FF33E）”这类重复。
	objRef.SetReference(i18n.LanguageZhHans, "Ar196", &objRef.Reference{DisplayName: "Ar 196"})
	require.Equal(t, "Ar 196", GetPlaneTitle("Ar196"))
	objRef.SetReference(i18n.LanguageZhHans, "FF33E", &objRef.Reference{DisplayName: "腓特烈港 FF.33e"})
	require.Equal(t, "腓特烈港 FF.33e", GetPlaneTitle("FF33E"))
	objRef.SetReference(i18n.LanguageZhHans, "Fw190A", &objRef.Reference{DisplayName: "Fw 190A"})
	require.Equal(t, "Fw 190A", GetPlaneTitle("Fw190A"))

	// 没有 reference 时退回配置名，保持与 GetPlaneDisplayName 一致的兜底行为。
	require.Equal(t, "unknown-plane", GetPlaneTitle("unknown-plane"))
}

// TestGetPlaneTitleFollowsCurrentLanguage 验证图鉴标题跟随当前界面语言切换。
// 场景：玩家在中文/日文之间切换，同一架 F4F-3 应显示对应语言的昵称，编号保持不变。
// 预期：zh-Hans 返回“野猫（F4F-3-test）”，ja 返回“ワイルドキャット（F4F-3-test）”。
// 覆盖：分别 SetLanguage 后调用 GetPlaneTitle，断言两种语言结果。
func TestGetPlaneTitleFollowsCurrentLanguage(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	objRef.SetReference(i18n.LanguageZhHans, "F4F-3-test", &objRef.Reference{DisplayName: "野猫"})
	objRef.SetReference(i18n.LanguageJapanese, "F4F-3-test", &objRef.Reference{DisplayName: "ワイルドキャット"})

	i18n.SetLanguage(string(i18n.LanguageZhHans))
	require.Equal(t, "野猫（F4F-3-test）", GetPlaneTitle("F4F-3-test"))
	i18n.SetLanguage(string(i18n.LanguageJapanese))
	require.Equal(t, "ワイルドキャット（F4F-3-test）", GetPlaneTitle("F4F-3-test"))
}
