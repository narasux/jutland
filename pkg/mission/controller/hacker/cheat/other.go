package cheat

import "github.com/narasux/jutland/pkg/mission/state"

// NotEffect 无效秘籍（兜底）
type NotEffect struct{}

func (c *NotEffect) String() string {
	return ""
}

func (c *NotEffect) Match(_ string) bool {
	return true
}

func (c *NotEffect) Exec(_ *state.MissionState) string {
	return "Not Effect"
}

var _ Cheat = (*NotEffect)(nil)
