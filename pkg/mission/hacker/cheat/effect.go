package cheat

import (
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
)

// BlackSheepWall 黑羊之墙 -> 地图全开（目前没用）
type BlackSheepWall struct{}

func (c *BlackSheepWall) String() string {
	return "black sheep wall"
}

func (c *BlackSheepWall) Desc() string {
	return "Removing the fog of war and allowing players to see all enemy units."
}

func (c *BlackSheepWall) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *BlackSheepWall) Exec(_ *state.MissionState) string {
	return "Not Implemented"
}

var _ Cheat = (*BlackSheepWall)(nil)

// BathtubWar 澡盆战争 -> 地图上的所有战舰都变成小黄鸭
type BathtubWar struct{}

func (c *BathtubWar) String() string {
	return "bathtub war"
}

func (c *BathtubWar) Desc() string {
	return "Turn every ship into a duck for a quacking good time!"
}

func (c *BathtubWar) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *BathtubWar) Exec(misState *state.MissionState) string {
	curShips := lo.Values(misState.Ships)

	misState.Ships = map[string]*object.BattleShip{}
	for _, ship := range curShips {
		duck := object.NewShip(
			misState.ShipUidGenerators[ship.BelongPlayer],
			"duck",
			ship.CurPos,
			ship.CurRotation,
			ship.BelongPlayer,
		)
		misState.Ships[duck.Uid] = duck
	}
	return "Congratulations! All the battle ships on map become duck, enjoy it!"
}

var _ Cheat = (*BathtubWar)(nil)

// WhoIsCallingTheFleet 谁在呼叫舰队 -> 所有增援点队列耗时修改为 1s（但是没钱就没办法）
type WhoIsCallingTheFleet struct{}

func (c *WhoIsCallingTheFleet) String() string {
	return "who is calling the fleet"
}

func (c *WhoIsCallingTheFleet) Desc() string {
	return "Turn every reinforce time cost to 1s"
}

func (c *WhoIsCallingTheFleet) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *WhoIsCallingTheFleet) Exec(misState *state.MissionState) string {
	for _, rfp := range misState.ReinforcePoints {
		// 只有自己的会生效
		if rfp.BelongPlayer != misState.CurPlayer {
			continue
		}
		for _, ship := range rfp.OncomingShips {
			ship.TimeCost = 1
		}
	}

	return "Reinforce ships oncoming! Who is calling the fleet!"
}

var _ Cheat = (*WhoIsCallingTheFleet)(nil)
