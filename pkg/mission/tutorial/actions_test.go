package tutorial

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResumeStepHintUsesPauseKey 验证“继续”步骤的文案与键帽都指向暂停面板真实可用的 Esc。
// 场景：基础操作第 8 步是 guided + resumed，卡片只渲染 ActionResume 键帽；暂停面板在
// MissionPaused 下只响应 Esc、Q 和鼠标点击，Enter 不会恢复任务。
// 预期：该步骤五种语言文案都包含 "Esc"，hint 含 resume 动作且键帽为 Esc；失败意味着
// 玩家照着提示按键无法继续，文案或键帽又退回了无效的 Enter。
// 覆盖：从 TutorialBasicControls 脚本找出 CompletionResumed 步骤，遍历 Body.values
// 断言均含 Esc，再用 TokensFor(ActionResume) 断言单个 TokenKey 键帽且标签为 Esc。
func TestResumeStepHintUsesPauseKey(t *testing.T) {
	script, ok := ScriptForMission("TutorialBasicControls")
	require.True(t, ok)

	var resumeStep *Step
	for idx := range script.Steps {
		if script.Steps[idx].Completion.Kind == CompletionResumed {
			resumeStep = &script.Steps[idx]
			break
		}
	}
	require.NotNil(t, resumeStep, "基础操作缺少 resumed 步骤")

	for lang, body := range resumeStep.Body.values {
		require.Containsf(t, body, "Esc", "语言 %s 的继续提示未写 Esc", lang)
	}
	require.Contains(t, resumeStep.Hint.Actions, ActionResume)

	spec, ok := TokensFor(ActionResume)
	require.True(t, ok)
	require.Len(t, spec.Tokens, 1)
	require.Equal(t, TokenKey, spec.Tokens[0].Kind)
	require.Equal(t, "Esc", spec.Tokens[0].Label)
}
