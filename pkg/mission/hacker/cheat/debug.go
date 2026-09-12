package cheat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/narasux/jutland/pkg/config"
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
	misState.UI.DebugFlags = state.DebugFlags{
		DamageColorByTeam:    true,
		ShowCursorPosObjInfo: true,
		ShowPlaneHP:          true,
		ShowHitBoxes:         true,
		ShowAirfieldRunway:   true,
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
	nextState := !misState.UI.DebugFlags.DamageColorByTeam
	misState.UI.DebugFlags.DamageColorByTeam = nextState
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
	nextState := !misState.UI.DebugFlags.ShowCursorPosObjInfo
	misState.UI.DebugFlags.ShowCursorPosObjInfo = nextState
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
	nextState := !misState.UI.DebugFlags.ShowPlaneHP
	misState.UI.DebugFlags.ShowPlaneHP = nextState
	return "Toggled show plane HP: " + lo.Ternary(nextState, "on", "off")
}

var _ Cheat = (*ShowPlaneHP)(nil)

// ShowHitBoxes -> 修改是否显示舰船和飞机的受打击范围
type ShowHitBoxes struct{}

func (c *ShowHitBoxes) String() string {
	return "show hit boxes"
}

func (c *ShowHitBoxes) Desc() string {
	return "switch ship and plane hit boxes on/off"
}

func (c *ShowHitBoxes) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *ShowHitBoxes) Exec(misState *state.MissionState) string {
	nextState := !misState.UI.DebugFlags.ShowHitBoxes
	misState.UI.DebugFlags.ShowHitBoxes = nextState
	return "Toggled hit boxes: " + lo.Ternary(nextState, "on", "off")
}

var _ Cheat = (*ShowHitBoxes)(nil)

// ShowAirfieldRunway -> 修改是否在选中机场时展示跑道方位线（白色带箭头直线）
type ShowAirfieldRunway struct{}

func (c *ShowAirfieldRunway) String() string {
	return "show airfield runway"
}

func (c *ShowAirfieldRunway) Desc() string {
	return "switch airfield runway direction line on/off (drawn when an airfield is selected)"
}

func (c *ShowAirfieldRunway) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *ShowAirfieldRunway) Exec(misState *state.MissionState) string {
	nextState := !misState.UI.DebugFlags.ShowAirfieldRunway
	misState.UI.DebugFlags.ShowAirfieldRunway = nextState
	return "Toggled show airfield runway: " + lo.Ternary(nextState, "on", "off")
}

var _ Cheat = (*ShowAirfieldRunway)(nil)

// DumpMisState 将当前帧的完整 MissionState 写入 debug 目录。
type DumpMisState struct{}

func (c *DumpMisState) String() string {
	return "dump mission state"
}

func (c *DumpMisState) Desc() string {
	return "export the complete mission state as JSON"
}

func (c *DumpMisState) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *DumpMisState) Exec(misState *state.MissionState) string {
	data, err := json.MarshalIndent(misState, "", "  ")
	if err != nil {
		return fmt.Sprintf("dump mission state failed: %v", err)
	}
	dir := filepath.Join(config.BaseDir, "debug")
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Sprintf("dump mission state failed: %v", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("mission-state-%s.json", time.Now().Format(time.RFC3339)))
	if err = os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Sprintf("dump mission state failed: %v", err)
	}
	return "mission state dumped to " + path
}

var _ Cheat = (*DumpMisState)(nil)
