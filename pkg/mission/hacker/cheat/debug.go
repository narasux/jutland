package cheat

import (
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/state"
)

// DebugAll -> 开启所有的 DebugFlags
type DebugAll struct{}

func (c *DebugAll) String() string {
	return "debug all"
}

func (c *DebugAll) Desc() string {
	return "enable all debug flags"
}

func (c *DebugAll) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *DebugAll) Exec(misState *state.MissionState) string {
	misState.DebugFlags = state.DebugFlags{
		DamageColorByTeam:    true,
		ShowCursorPosObjInfo: true,
		ShowPlaneHP:          true,
	}
	return "Enabled all debug flags"
}

var _ Cheat = (*DebugAll)(nil)

// DamageColorByTeam -> 修改是否区分敌我伤害颜色
type DamageColorByTeam struct{}

func (c *DamageColorByTeam) String() string {
	return "damage color by team"
}

func (c *DamageColorByTeam) Desc() string {
	return "switch damage color by team on/off"
}

func (c *DamageColorByTeam) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *DamageColorByTeam) Exec(misState *state.MissionState) string {
	nextState := !misState.DebugFlags.DamageColorByTeam
	misState.DebugFlags.DamageColorByTeam = nextState
	return "Toggled damage color by team: " + lo.Ternary(nextState, "on", "off")
}

var _ Cheat = (*DamageColorByTeam)(nil)

// ShowCursorPosObjInfo -> 修改是否显示光标悬停对象信息
type ShowCursorPosObjInfo struct{}

func (c *ShowCursorPosObjInfo) String() string {
	return "show cursor pos obj info"
}

func (c *ShowCursorPosObjInfo) Desc() string {
	return "switch show cursor position object info on/off"
}

func (c *ShowCursorPosObjInfo) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *ShowCursorPosObjInfo) Exec(misState *state.MissionState) string {
	nextState := !misState.DebugFlags.ShowCursorPosObjInfo
	misState.DebugFlags.ShowCursorPosObjInfo = nextState
	return "Toggled show cursor position object info: " + lo.Ternary(nextState, "on", "off")
}

var _ Cheat = (*ShowCursorPosObjInfo)(nil)

// ShowPlaneHP -> 修改是否显示飞机生命值
type ShowPlaneHP struct{}

func (c *ShowPlaneHP) String() string {
	return "show plane hp"
}

func (c *ShowPlaneHP) Desc() string {
	return "switch show plane HP on/off (display current HP / total HP above planes)"
}

func (c *ShowPlaneHP) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *ShowPlaneHP) Exec(misState *state.MissionState) string {
	nextState := !misState.DebugFlags.ShowPlaneHP
	misState.DebugFlags.ShowPlaneHP = nextState
	return "Toggled show plane HP: " + lo.Ternary(nextState, "on", "off")
}

var _ Cheat = (*ShowPlaneHP)(nil)
