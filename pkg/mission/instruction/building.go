package instruction

import (
	"fmt"
	"math/rand"

	"github.com/pkg/errors"

	"github.com/narasux/jutland/pkg/mission/state"
)

// ShipSummon 召唤增援（仅 AI 使用）
type ShipSummon struct {
	reinforcePointUid string
	shipName          string
	status            InstrStatus
}

// NewShipSummon ...
func NewShipSummon(reinforcePointUid string, shipName string) *ShipSummon {
	return &ShipSummon{reinforcePointUid: reinforcePointUid, shipName: shipName, status: Ready}
}

var _ Instruction = (*ShipSummon)(nil)

// Exec 执行战舰增援指令
func (i *ShipSummon) Exec(s *state.MissionState) error {
	rp, ok := s.ReinforcePoints[i.reinforcePointUid]
	if !ok {
		return errors.Errorf("reinforce point %s not found", i.reinforcePointUid)
	}
	if len(rp.OncomingShips) >= rp.MaxOncomingShip {
		return errors.Errorf("reinforce point %s oncoming ship limit reached", i.reinforcePointUid)
	}
	// 如果没指定，就随机来一个
	if i.shipName == "" {
		idx := rand.Intn(len(rp.ProvidedShipNames))
		i.shipName = rp.ProvidedShipNames[idx]
	}
	rp.Summon(i.shipName)
	i.status = Executed
	return nil
}

// Executed 判断指令是否执行完成
func (i *ShipSummon) Executed() bool {
	return i.status == Executed
}

// Uid 指令 uid
func (i *ShipSummon) Uid() string {
	return GenInstrUid(NameShipSummon, i.reinforcePointUid)
}

// String 返回指令字符串表示
func (i *ShipSummon) String() string {
	return fmt.Sprintf("summon %s from reinforce point %s", i.shipName, i.reinforcePointUid)
}
