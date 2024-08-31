package cheat

import (
	"strconv"

	"github.com/narasux/jutland/pkg/mission/state"
)

// ShowMeTheMoney -> 资金 +10000
type ShowMeTheMoney struct{}

func (c *ShowMeTheMoney) String() string {
	return "show me the money"
}

func (c *ShowMeTheMoney) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *ShowMeTheMoney) Exec(misState *state.MissionState) string {
	misState.CurFunds += 10000
	return "Add 10000 funds, current funds: " + strconv.FormatInt(misState.CurFunds, 10)
}

var _ Cheat = (*ShowMeTheMoney)(nil)
