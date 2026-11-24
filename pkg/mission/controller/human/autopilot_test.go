package human

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestHandleSuppressedWhenPlayerAIControl 验证 AI 托管期间人类输入处理器不产生任何指令。
// 场景：任务处于 MissionRunning，但 Core.PlayerAIControl 为 true；玩家仍可能按键/点右键。
// 预期：Handle 返回空 map，且不会读取 Ebiten 输入（测试不需要图形窗口）。
// 覆盖：构造只带 Core 的 MissionState，调用 NewHandler(HA).Handle，断言结果为空。
func TestHandleSuppressedWhenPlayerAIControl(t *testing.T) {
	ms := &state.MissionState{
		Core: state.MissionCoreState{
			MissionStatus:   state.MissionRunning,
			PlayerAIControl: true,
		},
	}
	handler := NewHandler(faction.HumanAlpha)

	instructions := handler.Handle(nil, ms)
	require.Empty(t, instructions)
}
