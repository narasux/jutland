package object

import (
	"slices"
	"time"

	"github.com/google/uuid"

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
	Progress float64
}

// Update ...
func (s *OncomingShip) Update() (finished bool) {
	if s.StartedAt == 0 {
		s.StartedAt = time.Now().UnixMilli()
		return false
	}
	s.Progress = float64(time.Now().UnixMilli()-s.StartedAt) / float64(s.TimeCost) / 10
	return s.Progress >= 100
}

// ReinforcePoint 增援点
type ReinforcePoint struct {
	Uid      string
	Pos      MapPos
	Rotation float64
	// 所属阵营（玩家）
	BelongPlayer faction.Player
	// 当前被选中的战舰索引
	CurSelectedShipIndex int
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
	if len(p.OncomingShips) >= p.MaxOncomingShip {
		return
	}
	if !slices.Contains(p.ProvidedShipNames, shipName) {
		return
	}

	fundsCost, timeCost := GetShipCost(shipName)
	p.OncomingShips = append(p.OncomingShips, &OncomingShip{
		Name: shipName, FundsCost: fundsCost, TimeCost: timeCost,
	})
}

// Update ...
func (p *ReinforcePoint) Update(
	shipUidGenerator *ShipUidGenerator, curFunds int64,
) *BattleShip {
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

// Progress 获取进度
func (p *ReinforcePoint) Progress() int {
	if len(p.OncomingShips) == 0 {
		return 0
	}
	return min(int(p.OncomingShips[0].Progress), 100)
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
		Uid:               uuid.NewString(),
		Pos:               pos,
		Rotation:          rotation,
		BelongPlayer:      belongPlayer,
		MaxOncomingShip:   maxOncomingShip,
		ProvidedShipNames: providedShipNames,
		OncomingShips:     []*OncomingShip{},
	}
}
