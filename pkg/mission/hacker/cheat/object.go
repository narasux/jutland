package cheat

import (
	"fmt"
	"math/rand"

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
	return "Create a duck at current cursor position."
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
	return "Duck ready at " + ship.CurPos.String()
}

var _ Cheat = (*ShowMeTheDuck)(nil)

// ShowMeTheWaterdrop -> 在鼠标当前位置初始化一个水滴
type ShowMeTheWaterdrop struct{}

func (c *ShowMeTheWaterdrop) String() string {
	return "show me the waterdrop"
}

func (c *ShowMeTheWaterdrop) Desc() string {
	return "Create a waterdrop at current cursor position."
}

func (c *ShowMeTheWaterdrop) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *ShowMeTheWaterdrop) Exec(misState *state.MissionState) string {
	uidGenerator := misState.ShipUidGenerators[misState.CurPlayer]
	pos := action.DetectCursorPosOnMap(misState)
	ship := object.NewShip(uidGenerator, "waterdrop", *pos, 0, misState.CurPlayer)
	misState.Ships[ship.Uid] = ship
	return "Waterdrop ready at " + ship.CurPos.String()
}

var _ Cheat = (*ShowMeTheWaterdrop)(nil)

// ShowMeTheMolaMola -> 在鼠标当前位置初始化一个翻车鱼
type ShowMeTheMolaMola struct{}

func (c *ShowMeTheMolaMola) String() string {
	return "show me the molamola"
}

func (c *ShowMeTheMolaMola) Desc() string {
	return "Create a molamola at current cursor position."
}

func (c *ShowMeTheMolaMola) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *ShowMeTheMolaMola) Exec(misState *state.MissionState) string {
	uidGenerator := misState.ShipUidGenerators[misState.CurPlayer]
	pos := action.DetectCursorPosOnMap(misState)
	if misState.MissionMD.MapCfg.Map.IsLand(pos.MX, pos.MY) {
		return "Current cursor position on land, can't create molamola"
	}
	ship := object.NewShip(uidGenerator, "molamola", *pos, 0, misState.CurPlayer)
	misState.Ships[ship.Uid] = ship
	return "MolaMola ready at " + ship.CurPos.String()
}

var _ Cheat = (*ShowMeTheMolaMola)(nil)

// BlackGoldRush -> 在指定地点生成一个油井
type BlackGoldRush struct{}

func (c *BlackGoldRush) String() string {
	return "black gold rush"
}

func (c *BlackGoldRush) Desc() string {
	return "Create a oil platform at current cursor position."
}

func (c *BlackGoldRush) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *BlackGoldRush) Exec(misState *state.MissionState) string {
	pos := action.DetectCursorPosOnMap(misState)
	if misState.MissionMD.MapCfg.Map.IsLand(pos.MX, pos.MY) {
		return "Current cursor position on land, can't create oil platform"
	}
	// 油井范围，生成量随机
	radius := 3 + rand.Intn(4)
	yield := 50 + rand.Intn(100)
	op := object.NewOilPlatform(*pos, radius, yield)
	misState.OilPlatforms[op.Uid] = op
	return fmt.Sprintf("Oil platform created at %s. Be careful, oil can breed mold!", pos.String())
}

var _ Cheat = (*BlackGoldRush)(nil)
