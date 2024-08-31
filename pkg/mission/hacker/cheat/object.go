package cheat

import (
	"github.com/narasux/jutland/pkg/mission/action"
	"github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
)

// ShowMeTheDuck -> 在鼠标当前位置初始化一个小黄鸭
type ShowMeTheDuck struct{}

func (c *ShowMeTheDuck) String() string {
	return "show me the duck"
}

func (c *ShowMeTheDuck) Desc() string {
	return "Create a duck at current cursor position"
}

func (c *ShowMeTheDuck) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *ShowMeTheDuck) Exec(misState *state.MissionState) string {
	// 如果鼠标位置是海洋，则成功，否则失败
	pos := action.DetectCursorPosOnMap(misState)
	if misState.MissionMD.MapCfg.Map.IsLand(pos.MX, pos.MY) {
		return "Current cursor position on land, can't create duck"
	}
	uidGenerator := misState.ShipUidGenerators[misState.CurPlayer]
	ship := object.NewShip(uidGenerator, "duck", *pos, 0, misState.CurPlayer)
	misState.Ships[ship.Uid] = ship
	return "Battle ship duck ready at " + ship.CurPos.String()
}

var _ Cheat = (*ShowMeTheDuck)(nil)

// ShowMeTheWaterdrop -> 在鼠标当前位置初始化一个水滴
type ShowMeTheWaterdrop struct{}

func (c *ShowMeTheWaterdrop) String() string {
	return "show me the waterdrop"
}

func (c *ShowMeTheWaterdrop) Desc() string {
	return "Create a waterdrop at current cursor position"
}

func (c *ShowMeTheWaterdrop) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *ShowMeTheWaterdrop) Exec(misState *state.MissionState) string {
	uidGenerator := misState.ShipUidGenerators[misState.CurPlayer]
	pos := action.DetectCursorPosOnMap(misState)
	ship := object.NewShip(uidGenerator, "waterdrop", *pos, 0, misState.CurPlayer)
	misState.Ships[ship.Uid] = ship
	return "Battle ship waterdrop ready at " + ship.CurPos.String()
}

var _ Cheat = (*ShowMeTheWaterdrop)(nil)
