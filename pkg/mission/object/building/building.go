package building

import (
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
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
	Pos      objPos.MapPos
	Rotation float64
	// 集结点 FIXME 支持自定义集结点
	RallyPos objPos.MapPos
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

	fundsCost, timeCost := objUnit.GetShipCost(shipName)
	p.OncomingShips = append(p.OncomingShips, &OncomingShip{
		Name: shipName, FundsCost: fundsCost, TimeCost: timeCost,
	})
}

// Update ...
func (p *ReinforcePoint) Update(
	shipUidGenerator *objUnit.ShipUidGenerator, curFunds int64,
) *objUnit.BattleShip {
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
		return objUnit.NewShip(
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
	pos objPos.MapPos,
	rotation float64,
	rallyPos objPos.MapPos,
	belongPlayer faction.Player,
	maxOncomingShip int,
	providedShipNames []string,
) *ReinforcePoint {
	return &ReinforcePoint{
		Uid:               uuid.NewString(),
		Pos:               pos,
		Rotation:          rotation,
		RallyPos:          rallyPos,
		BelongPlayer:      belongPlayer,
		MaxOncomingShip:   maxOncomingShip,
		ProvidedShipNames: providedShipNames,
		OncomingShips:     []*OncomingShip{},
	}
}

// LoadingOilShip 装载石油的货轮
type LoadingOilShip struct {
	CurPos objPos.MapPos
	// 资金产量
	FundYield int
	// 装载耗时
	TimeCost int64
	// 开始的时间戳
	StartedAt int64
	// 进度
	Progress float64
}

// Update ...
func (s *LoadingOilShip) Update() (finished bool) {
	if s.StartedAt == 0 {
		s.StartedAt = time.Now().UnixMilli()
		return false
	}
	s.Progress = float64(time.Now().UnixMilli()-s.StartedAt) / float64(s.TimeCost) / 10

	// 如果进度到达 100，重置并返回 true
	if s.Progress >= 100 {
		s.StartedAt = 0
		s.Progress = 0
		return true
	}
	return false
}

// OilPlatform 油井
type OilPlatform struct {
	Uid             string
	Pos             objPos.MapPos
	Radius          int
	Yield           int
	LoadingOilShips map[string]*LoadingOilShip
}

// AddShip 添加货轮
func (p *OilPlatform) AddShip(ship *objUnit.BattleShip) {
	if _, ok := p.LoadingOilShips[ship.Uid]; !ok {
		// TODO 目前装载耗时固定为 5s
		p.LoadingOilShips[ship.Uid] = &LoadingOilShip{
			CurPos: ship.CurPos, FundYield: p.Yield, TimeCost: 5,
		}
	}
}

// RemoveShip 移除货轮
func (p *OilPlatform) RemoveShip(shipUid string) {
	if _, ok := p.LoadingOilShips[shipUid]; ok {
		delete(p.LoadingOilShips, shipUid)
	}
}

// NewOilPlatform ...
func NewOilPlatform(pos objPos.MapPos, radius int, yield int) *OilPlatform {
	return &OilPlatform{
		Uid:             uuid.NewString(),
		Pos:             pos,
		Radius:          radius,
		Yield:           yield,
		LoadingOilShips: map[string]*LoadingOilShip{},
	}
}
