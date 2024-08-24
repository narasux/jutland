package object

import (
	"slices"
	"time"

	"github.com/narasux/jutland/pkg/mission/faction"
)

// OncomingShip 增援中的战舰
type OncomingShip struct {
	Name      string
	FundsCost int64
	TimeCost  int64
	// 开始的时间戳
	StartedAt int64
	// 进度
	Process float64
}

// Update ...
func (s *OncomingShip) Update() (finished bool) {
	if s.StartedAt == 0 {
		s.StartedAt = time.Now().UnixMilli()
		return false
	}
	s.Process = float64(time.Now().UnixMilli()-s.StartedAt) / float64(s.TimeCost) * 1e3
	return s.Process >= 100
}

// ReinforcePoint 增援点
type ReinforcePoint struct {
	Pos      MapPos
	Rotation float64
	// 所属阵营（玩家）
	BelongPlayer faction.Player
	// 提供的战舰类型
	ProvidedShipNames []string
	// 最大增援进度数量
	MaxOncomingShip int
	// 增援进度
	OncomingShips []*OncomingShip
	// FIXME 提供占领的功能（独占某个区域？）
}

// Summon 召唤增援
func (p *ReinforcePoint) Summon(shipName string) {
	if len(p.ProvidedShipNames) >= p.MaxOncomingShip {
		return
	}
	if !slices.Contains(p.ProvidedShipNames, shipName) {
		return
	}
	ship := shipMap[shipName]
	p.OncomingShips = append(p.OncomingShips, &OncomingShip{
		Name:      shipName,
		FundsCost: ship.FundsCost,
		TimeCost:  ship.TimeCost,
	})
}

// Update ...
func (p *ReinforcePoint) Update(shipUidGenerator *ShipUidGenerator, curFunds int64) *BattleShip {
	if len(p.OncomingShips) == 0 {
		return nil
	}

	oncomingShip := p.OncomingShips[0]
	// 钱不够就不干活！
	if curFunds < oncomingShip.FundsCost {
		return nil
	}
	// 增援进度计算
	if finished := oncomingShip.Update(); finished {
		p.OncomingShips = p.OncomingShips[1:]
		return NewShip(
			shipUidGenerator,
			oncomingShip.Name,
			// FIXME 支持计算位置而不是固定
			p.Pos,
			p.Rotation,
			p.BelongPlayer,
		)
	}
	return nil
}

// NewReinforcePoint ...
func NewReinforcePoint(
	pos MapPos,
	rotation float64,
	belongPlayer faction.Player,
	maxOncomingShip int,
	providedShipNames []string,
) *ReinforcePoint {
	return &ReinforcePoint{
		Pos:               pos,
		Rotation:          rotation,
		BelongPlayer:      belongPlayer,
		MaxOncomingShip:   maxOncomingShip,
		ProvidedShipNames: providedShipNames,
		OncomingShips:     []*OncomingShip{},
	}
}
