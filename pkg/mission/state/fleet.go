package state

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	obj "github.com/narasux/jutland/pkg/mission/object"
)

// ShipClass 同级战舰
type ShipClass struct {
	Total int
	Kind  *obj.BattleShip
}

// Fleet 舰队
type Fleet struct {
	Player  faction.Player
	Total   int
	Classes []ShipClass
}
