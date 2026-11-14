package instruction

import (
	"fmt"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// ShipDive 下潜
type ShipDive struct {
	shipUid string
	status  InstrStatus
}

// NewShipDive ...
func NewShipDive(shipUid string) *ShipDive {
	return &ShipDive{shipUid: shipUid, status: Ready}
}

var _ Instruction = (*ShipDive)(nil)

// Exec ...
func (i *ShipDive) Exec(s *state.MissionState) error {
	i.status = Executed
	ship, ok := s.Arena.Ships[i.shipUid]
	if !ok {
		return nil
	}
	ship.StartDive()
	return nil
}

// Executed ...
func (i *ShipDive) Executed() bool {
	return i.status == Executed
}

// Uid ...
func (i *ShipDive) Uid() string {
	return GenInstrUid(NameShipDive, i.shipUid)
}

// String ...
func (i *ShipDive) String() string {
	return fmt.Sprintf("Dive ship %s", i.shipUid)
}

// ShipSurface 上浮
type ShipSurface struct {
	shipUid string
	status  InstrStatus
}

// NewShipSurface ...
func NewShipSurface(shipUid string) *ShipSurface {
	return &ShipSurface{shipUid: shipUid, status: Ready}
}

var _ Instruction = (*ShipSurface)(nil)

// Exec ...
func (i *ShipSurface) Exec(s *state.MissionState) error {
	i.status = Executed
	ship, ok := s.Arena.Ships[i.shipUid]
	if !ok {
		return nil
	}
	ship.StartSurface()
	return nil
}

// Executed ...
func (i *ShipSurface) Executed() bool {
	return i.status == Executed
}

// Uid ...
func (i *ShipSurface) Uid() string {
	return GenInstrUid(NameShipSurface, i.shipUid)
}

// String ...
func (i *ShipSurface) String() string {
	return fmt.Sprintf("Surface ship %s", i.shipUid)
}

// NewDiveToggle 按潜艇当前深度（含过渡态）生成下潜/上浮指令，
// 过渡中按“最后一次指令为准”反向切换。非潜艇、已沉没时返回 nil。
func NewDiveToggle(ship *objUnit.BattleShip) Instruction {
	if ship == nil || ship.CurHP <= 0 {
		return nil
	}
	next, ok := ship.NextDiveState()
	if !ok {
		return nil
	}
	if next == objUnit.DiveStateSubmerged {
		return NewShipDive(ship.Uid)
	}
	return NewShipSurface(ship.Uid)
}
