package state

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// ShipClass 同级战舰
type ShipClass struct {
	Total int
	Kind  *objUnit.BattleShip
}

// Fleet 舰队
type Fleet struct {
	Player  faction.Player
	Total   int
	Classes []ShipClass
}
