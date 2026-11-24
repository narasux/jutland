package cheat

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/state"
)

// TestHelpMeAISamaTogglesPlayerAIControl 验证同一句秘籍在开/关之间翻转玩家侧托管，并返回不同文本。
// 场景：终端输入 "help me ai sama"/"HELP  ME  AI  SAMA"；第一次开启托管，第二次关闭；
// 错误命令 "help me ai" 不能误触发，迷雾/潜艇标记等既有状态不能被顺带修改。
// 预期：Match 对两种合法写法为 true、对短句为 false；Exec 后 PlayerAIControl 依次为 true/false，
// 两次返回文本不同且分别包含 "takes"/"yours"；Player.IgnoreFog 与 RevealSubmarines 保持 false。
// 覆盖：构造空 MissionState，连续 Exec 两次并断言开关、文本与旁路字段。
func TestHelpMeAISamaTogglesPlayerAIControl(t *testing.T) {
	ms := &state.MissionState{}
	help := &HelpMeAISama{}

	require.True(t, help.Match("help me ai sama"))
	require.True(t, help.Match("HELP  ME  AI  SAMA"))
	require.False(t, help.Match("help me ai"))

	onText := help.Exec(ms)
	require.True(t, ms.Core.PlayerAIControl)
	require.Contains(t, onText, "takes")

	offText := help.Exec(ms)
	require.False(t, ms.Core.PlayerAIControl)
	require.NotEqual(t, onText, offText)
	require.Contains(t, offText, "yours")

	require.False(t, ms.Player.IgnoreFog)
	require.False(t, ms.Player.RevealSubmarines)
}
