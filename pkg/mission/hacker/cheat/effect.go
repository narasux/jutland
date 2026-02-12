package cheat

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// AngelicaSinensis 当归
type AngelicaSinensis struct{}

func (c *AngelicaSinensis) String() string {
	return "angelica sinensis"
}

func (c *AngelicaSinensis) Desc() string {
	return "DangGui is a traditional Chinese herbal medicine."
}

func (c *AngelicaSinensis) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *AngelicaSinensis) Exec(misState *state.MissionState) string {
	// TODO 公辞八十载，今夕请当归
	return "Eight decades since your parting sigh. This night, return as Angelica nigh."
}

var _ Cheat = (*AngelicaSinensis)(nil)

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

	misState.Ships = map[string]*objUnit.BattleShip{}
	for _, ship := range curShips {
		duck := objUnit.NewShip(
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
	return "Turn every reinforce time cost to 1s."
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

	return "Reinforce ships oncoming!"
}

var _ Cheat = (*WhoIsCallingTheFleet)(nil)

// DoNotDie 不要死 -> 选中的战舰修改为满血
type DoNotDie struct{}

func (c *DoNotDie) String() string {
	return "do not die"
}

func (c *DoNotDie) Desc() string {
	return "Turn every selected ship to full HP."
}

func (c *DoNotDie) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *DoNotDie) Exec(misState *state.MissionState) string {
	for _, shipUid := range misState.SelectedShips {
		// 满血数值其实就是吨位
		if ship, ok := misState.Ships[shipUid]; ok {
			ship.CurHP = ship.Tonnage
		}
	}

	return fmt.Sprintf("Change %d ships to full hp.", len(misState.SelectedShips))
}

var _ Cheat = (*DoNotDie)(nil)

// YouHaveBetrayedTheWorkingClass 你背叛了工人阶级，XXX -> 标记选中的战舰成为敌人
type YouHaveBetrayedTheWorkingClass struct{}

func (c *YouHaveBetrayedTheWorkingClass) String() string {
	// 原剧是德语：Du, Verräter der Arbeiterklasse, Verpfeif dich!
	return "you have betrayed the working class"
}

func (c *YouHaveBetrayedTheWorkingClass) Desc() string {
	return "Mark selected ships as enemy."
}

func (c *YouHaveBetrayedTheWorkingClass) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *YouHaveBetrayedTheWorkingClass) Exec(misState *state.MissionState) string {
	for _, shipUid := range misState.SelectedShips {
		if ship, ok := misState.Ships[shipUid]; ok {
			ship.BelongPlayer = faction.ComputerAlpha
		}
	}
	return "It's time to clean up the house!"
}

var _ Cheat = (*YouHaveBetrayedTheWorkingClass)(nil)

// AbandonDarkness 弃暗投明 -> 视野范围内的敌人变成自己人
type AbandonDarkness struct{}

func (c *AbandonDarkness) String() string {
	return "abandon darkness"
}

func (c *AbandonDarkness) Desc() string {
	return "Turn enemy ship in camera belong to self."
}

func (c *AbandonDarkness) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *AbandonDarkness) Exec(misState *state.MissionState) string {
	for _, ship := range misState.Ships {
		if misState.Camera.Contains(ship.CurPos) {
			ship.BelongPlayer = misState.CurPlayer
		}
	}
	return "Thank you, you are both good man."
}

var _ Cheat = (*AbandonDarkness)(nil)

// Expelliarmus 除你武器
type Expelliarmus struct{}

func (c *Expelliarmus) String() string {
	return "expelliarmus"
}

func (c *Expelliarmus) Desc() string {
	return "Remove all enemy ship's weapons in camera."
}

func (c *Expelliarmus) Match(cmd string) bool {
	return isCommandEqual(c.String(), cmd)
}

func (c *Expelliarmus) Exec(misState *state.MissionState) string {
	for _, ship := range misState.Ships {
		if misState.Camera.Contains(ship.CurPos) && ship.BelongPlayer != misState.CurPlayer {
			ship.Weapon = objUnit.ShipWeapon{}
		}
	}
	return "Expelliarmus! (>.>)-o------(QAQ)"
}

var _ Cheat = (*Expelliarmus)(nil)
