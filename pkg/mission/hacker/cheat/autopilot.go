package cheat

import "github.com/narasux/jutland/pkg/mission/state"

// HelpMeAISama 帮帮我吧 AI 大人 -> 玩家侧交给 AI 托管，再输入一次取消
type HelpMeAISama struct{}

func (c *HelpMeAISama) String() string {
	return "help me ai sama"
}

func (c *HelpMeAISama) Desc() string {
	return "Hand your fleet to AI sama. Enter again to take it back."
}

func (c *HelpMeAISama) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *HelpMeAISama) Exec(misState *state.MissionState) string {
	misState.Core.PlayerAIControl = !misState.Core.PlayerAIControl
	if misState.Core.PlayerAIControl {
		return "AI sama takes the helm. The fleet is in good hands... probably."
	}
	return "AI sama returns to the server room. The fleet is yours again, Admiral."
}

var _ Cheat = (*HelpMeAISama)(nil)
