package state

import (
	"math"

	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// BattleEndReason 战斗结束原因。
type BattleEndReason string

const (
	// BattleEndVictory 任务成功
	BattleEndVictory BattleEndReason = "victory"
	// BattleEndDefeat 任务失败
	BattleEndDefeat BattleEndReason = "defeat"
	// BattleEndAborted 主动放弃 / 调试退出
	BattleEndAborted BattleEndReason = "aborted"
)

// BattleEventKind 战报事件类型。
type BattleEventKind string

const (
	// BattleEventShipLost 舰艇沉没
	BattleEventShipLost BattleEventKind = "shipLost"
	// BattleEventPlaneLost 飞机损失（同一时间窗内会合并计数）
	BattleEventPlaneLost BattleEventKind = "planeLost"
)

// planeLossEventWindowTicks 同一方连续飞机损失合并到一条事件的时间窗（2 秒）。
const planeLossEventWindowTicks = 120

// BattleEvent 本局关键事件（舰沉、飞机损失），供后续战报复盘使用。
type BattleEvent struct {
	Tick    int64
	Player  faction.Player
	Kind    BattleEventKind
	Name    string
	Tonnage float64
	Carrier bool
	Count   int
}

// BattleSideStats 单一阵营在本局内的战果累计。
// 所有字段都在事件发生点累加，绘制层只读，不在结算时扫描战场。
type BattleSideStats struct {
	// 舰艇
	ShipsInitial   int
	ShipsBuilt     int
	ShipsLost      int
	InitialTonnage float64
	BuiltTonnage   float64
	SunkTonnage    float64
	// 按舰名（舰级）统计的初始 / 建造 / 损失
	ShipsInitialByClass map[string]int
	ShipsBuiltByClass   map[string]int
	ShipsLostByClass    map[string]int

	// 飞机
	PlanesInitial       int
	PlanesBuilt         int
	PlanesLost          int
	PlanesLostWithShip  int
	PlanesInitialByType map[string]int
	PlanesBuiltByType   map[string]int
	PlanesLostByType    map[string]int

	// 造成伤害（实际命中后的 RealDamage 累计，含溢出伤害）
	DamageDealt float64
}

// BattleStats 本局战果累计。
type BattleStats struct {
	Sides map[faction.Player]*BattleSideStats

	StartTick   int64
	EndTick     int64
	EndReason   BattleEndReason
	Finalized   bool
	FundsStart  int64
	FundsSpent  int64
	FundsEarned int64
	// Events 关键事件时间轴（舰沉、飞机损失）
	Events []BattleEvent
	// BattleLog 详细战况（命中 / 击沉 / 击落），结算页滚动查看，关闭结算后清空
	BattleLog []BattleLogEntry
	// BattleLogVersion 战况日志版本号；追加或合并时递增，供界面缓存筛选结果
	BattleLogVersion uint64
}

// NewBattleSideStats 创建带初始化计数表的单方战果。
func NewBattleSideStats() *BattleSideStats {
	return &BattleSideStats{
		ShipsInitialByClass: map[string]int{},
		ShipsBuiltByClass:   map[string]int{},
		ShipsLostByClass:    map[string]int{},
		PlanesInitialByType: map[string]int{},
		PlanesBuiltByType:   map[string]int{},
		PlanesLostByType:    map[string]int{},
	}
}

// NewBattleStats 创建双方战果均为零值的本局统计。
func NewBattleStats() BattleStats {
	return BattleStats{
		Sides: map[faction.Player]*BattleSideStats{
			faction.HumanAlpha:    NewBattleSideStats(),
			faction.ComputerAlpha: NewBattleSideStats(),
		},
	}
}

// side 返回指定阵营的可写战果记录，缺失时补建。
func (s *BattleStats) side(player faction.Player) *BattleSideStats {
	if s.Sides == nil {
		s.Sides = map[faction.Player]*BattleSideStats{}
	}
	stats := s.Sides[player]
	if stats == nil {
		stats = NewBattleSideStats()
		s.Sides[player] = stats
	}
	return stats
}

// ShipClassCount 返回有记录的舰级数量（初始 / 建造 / 损失任一表出现即算）。
func (s BattleSideStats) ShipClassCount() int {
	return countRecordedNames(
		s.ShipsInitialByClass, s.ShipsBuiltByClass, s.ShipsLostByClass,
	)
}

// PlaneTypeCount 返回有记录的机型数量（初始 / 建造 / 损失任一表出现即算）。
func (s BattleSideStats) PlaneTypeCount() int {
	return countRecordedNames(
		s.PlanesInitialByType, s.PlanesBuiltByType, s.PlanesLostByType,
	)
}

// countRecordedNames 返回多张计数表中出现过的非零名称数。
func countRecordedNames(tables ...map[string]int) int {
	counts := make(map[string]int)
	for _, table := range tables {
		for name, count := range table {
			counts[name] += count
		}
	}
	total := 0
	for _, count := range counts {
		if count > 0 {
			total++
		}
	}
	return total
}

// Side 返回指定阵营的战果只读快照；未记录时返回零值。
func (s *BattleStats) Side(player faction.Player) BattleSideStats {
	if stats := s.Sides[player]; stats != nil {
		return *stats
	}
	return BattleSideStats{}
}

// RecordInitialShip 记录初始编成中的一艘舰及其随舰舰载机。
func (s *BattleStats) RecordInitialShip(ship *objUnit.BattleShip) {
	if ship == nil {
		return
	}
	stats := s.side(ship.BelongPlayer)
	stats.ShipsInitial++
	stats.ShipsInitialByClass[ship.Name]++
	stats.InitialTonnage += ship.Tonnage
	for _, group := range ship.Aircraft.Groups {
		count := int(group.CurCount)
		if count <= 0 {
			continue
		}
		stats.PlanesInitial += count
		stats.PlanesInitialByType[group.Name] += count
	}
}

// RecordInitialPlanes 记录初始岸基飞机（机场机库库存）。
func (s *BattleStats) RecordInitialPlanes(player faction.Player, name string, count int) {
	if count <= 0 {
		return
	}
	stats := s.side(player)
	stats.PlanesInitial += count
	stats.PlanesInitialByType[name] += count
}

// RecordShipBuilt 记录一艘在战斗中实际完工的舰艇。
// 新建航母按满编交付舰载机，这部分飞机同样计入战时建造。
func (s *BattleStats) RecordShipBuilt(ship *objUnit.BattleShip) {
	if ship == nil {
		return
	}
	stats := s.side(ship.BelongPlayer)
	stats.ShipsBuilt++
	stats.BuiltTonnage += ship.Tonnage
	stats.ShipsBuiltByClass[ship.Name]++
	for _, group := range ship.Aircraft.Groups {
		count := int(group.CurCount)
		if count <= 0 {
			continue
		}
		stats.PlanesBuilt += count
		stats.PlanesBuiltByType[group.Name] += count
	}
}

// RecordShipLost 记录一艘被击沉的舰艇。
// 机库内尚未起飞的舰载机随舰损失，并清零库存避免重复统计；
// 已经起飞的飞机不在此刻计入，失去基地后由正常损失路径记账。
func (s *BattleStats) RecordShipLost(ship *objUnit.BattleShip) {
	if ship == nil {
		return
	}
	stats := s.side(ship.BelongPlayer)
	stats.ShipsLost++
	stats.SunkTonnage += ship.Tonnage
	stats.ShipsLostByClass[ship.Name]++
	lostPlanes := 0
	for idx := range ship.Aircraft.Groups {
		group := &ship.Aircraft.Groups[idx]
		count := int(group.CurCount)
		if count <= 0 {
			continue
		}
		stats.PlanesLost += count
		stats.PlanesLostWithShip += count
		stats.PlanesLostByType[group.Name] += count
		group.CurCount = 0
		lostPlanes += count
	}
	now := objUnit.SimTick()
	s.Events = append(s.Events, BattleEvent{
		Tick:    now,
		Player:  ship.BelongPlayer,
		Kind:    BattleEventShipLost,
		Name:    ship.Name,
		Tonnage: ship.Tonnage,
		Carrier: ship.Type == objUnit.ShipTypeAircraftCarrier,
		Count:   1,
	})
	actorPlayer, actorName, actorType, actorIsPlane := faction.Player(""), "", "", false
	hit := objUnit.HitInfo{}
	totalDamage := 0.0
	if ship.SunkByHit {
		actorPlayer, actorName, actorType = ship.LastAttackerPlayer, ship.LastAttackerName, ship.LastAttackerType
		actorIsPlane = ship.LastAttackerIsPlane
		hit = ship.LastHit
		totalDamage = ship.DamageTaken
	}
	s.RecordBattleLog(BattleLogEntry{
		Tick:         now,
		ActorPlayer:  actorPlayer,
		ActorName:    actorName,
		ActorType:    actorType,
		ActorIsPlane: actorIsPlane,
		TargetPlayer: ship.BelongPlayer,
		TargetName:   ship.Name,
		TargetType:   string(ship.Type),
		Action:       BattleLogShipSunk,
		Tonnage:      ship.Tonnage,
		Hit:          hit,
		TotalDamage:  totalDamage,
	})
	if lostPlanes > 0 {
		s.recordPlaneLossEvent(ship.BelongPlayer, lostPlanes, now)
		s.RecordBattleLog(BattleLogEntry{
			Tick:        now,
			ActorPlayer: ship.BelongPlayer,
			ActorName:   ship.Name,
			ActorType:   string(ship.Type),
			Action:      BattleLogPlaneWithShip,
			Count:       lostPlanes,
		})
	}
}

// RecordPlaneBuilt 记录一架在战斗中实际完工的飞机。
func (s *BattleStats) RecordPlaneBuilt(player faction.Player, name string) {
	stats := s.side(player)
	stats.PlanesBuilt++
	stats.PlanesBuiltByType[name]++
}

// RecordPlaneLost 记录一架被击落 / 坠毁的飞机。
func (s *BattleStats) RecordPlaneLost(plane *objUnit.Plane) {
	if plane == nil {
		return
	}
	stats := s.side(plane.BelongPlayer)
	stats.PlanesLost++
	stats.PlanesLostByType[plane.Name]++
	now := objUnit.SimTick()
	s.recordPlaneLossEvent(plane.BelongPlayer, 1, now)
	actorPlayer, actorName, actorType, actorIsPlane := faction.Player(""), "", "", false
	hit := objUnit.HitInfo{}
	totalDamage := 0.0
	action := BattleLogPlaneLost
	if plane.ShotDown {
		action = BattleLogPlaneShotDown
		actorPlayer, actorName, actorType = plane.LastAttackerPlayer, plane.LastAttackerName, plane.LastAttackerType
		actorIsPlane = plane.LastAttackerIsPlane
		hit = plane.LastHit
		totalDamage = plane.DamageTaken
	}
	s.RecordBattleLog(BattleLogEntry{
		Tick:          now,
		ActorPlayer:   actorPlayer,
		ActorName:     actorName,
		ActorType:     actorType,
		ActorIsPlane:  actorIsPlane,
		TargetPlayer:  plane.BelongPlayer,
		TargetName:    plane.Name,
		TargetType:    string(plane.Type),
		TargetIsPlane: true,
		Action:        action,
		Count:         1,
		Hit:           hit,
		TotalDamage:   totalDamage,
	})
}

// recordPlaneLossEvent 记录飞机损失事件；同一方在短时间窗内的连续损失合并计数，
// 避免时间轴被防空弹幕刷屏。
func (s *BattleStats) recordPlaneLossEvent(player faction.Player, count int, tick int64) {
	if count <= 0 {
		return
	}
	if n := len(s.Events); n > 0 {
		last := &s.Events[n-1]
		if last.Kind == BattleEventPlaneLost && last.Player == player &&
			tick >= last.Tick && tick-last.Tick <= planeLossEventWindowTicks {
			last.Count += count
			return
		}
	}
	s.Events = append(s.Events, BattleEvent{
		Tick: tick, Player: player, Kind: BattleEventPlaneLost, Count: count,
	})
}

// RecordDamage 累计开火方实际造成的伤害。
func (s *BattleStats) RecordDamage(player faction.Player, damage float64) {
	if !(damage > 0) {
		return
	}
	s.side(player).DamageDealt += damage
}

// RecordFundsSpent 累计当前玩家的战时支出。
func (s *BattleStats) RecordFundsSpent(amount int64) {
	if amount > 0 {
		s.FundsSpent += amount
	}
}

// RecordFundsEarned 累计当前玩家的战时收入。
func (s *BattleStats) RecordFundsEarned(amount int64) {
	if amount > 0 {
		s.FundsEarned += amount
	}
}

// RemainingShips 在役舰艇数：初始 + 建造 - 损失，下限夹到 0。
func (s BattleSideStats) RemainingShips() int {
	return max(0, s.ShipsInitial+s.ShipsBuilt-s.ShipsLost)
}

// RemainingPlanes 在役飞机数：初始 + 建造 - 损失，下限夹到 0。
func (s BattleSideStats) RemainingPlanes() int {
	return max(0, s.PlanesInitial+s.PlanesBuilt-s.PlanesLost)
}

// SurvivingTonnage 在役吨位：初始 + 建造 - 损失，下限夹到 0。
func (s BattleSideStats) SurvivingTonnage() float64 {
	return max(0, s.InitialTonnage+s.BuiltTonnage-s.SunkTonnage)
}

// SurvivalRate 舰艇保存率。优先按吨位计算，吨位缺失时回退按舰艇数量计算。
func (s BattleSideStats) SurvivalRate() (float64, bool) {
	totalTonnage := s.InitialTonnage + s.BuiltTonnage
	if totalTonnage > 0 && !(s.ShipsLost > 0 && s.SunkTonnage <= 0) {
		return min(1, s.SurvivingTonnage()/totalTonnage), true
	}
	totalShips := s.ShipsInitial + s.ShipsBuilt
	if totalShips <= 0 {
		return 0, false
	}
	return min(1, float64(s.RemainingShips())/float64(totalShips)), true
}

// ExchangeRatio 战损交换比：敌方损失 / 我方损失。
// 优先按吨位计算；任一方有损失但吨位为 0（测试对象 / 特殊单位）时回退按数量。
// 我方零损失且敌方有损失时返回 +Inf，双方都零损失时 ok 为 false。
func (s *BattleStats) ExchangeRatio(player, enemy faction.Player) (ratio float64, ok bool) {
	own, opp := s.Side(player), s.Side(enemy)
	useTonnage := !(own.ShipsLost > 0 && own.SunkTonnage <= 0) &&
		!(opp.ShipsLost > 0 && opp.SunkTonnage <= 0)
	ownLoss, enemyLoss := float64(own.ShipsLost), float64(opp.ShipsLost)
	if useTonnage {
		ownLoss, enemyLoss = own.SunkTonnage, opp.SunkTonnage
	}
	if ownLoss <= 0 {
		if enemyLoss <= 0 {
			return 0, false
		}
		return math.Inf(1), true
	}
	return enemyLoss / ownLoss, true
}

// recordInitialBattleStats 在 NewMissionState 完成对象初始化后记录初始编成。
// 之后所有战果都靠事件累加，不再回读战场。
func (s *MissionState) recordInitialBattleStats() {
	s.Stats = NewBattleStats()
	s.Stats.StartTick = s.Core.SimTick
	s.Stats.FundsStart = s.Player.CurFunds
	for _, ship := range s.Arena.Ships {
		s.Stats.RecordInitialShip(ship)
	}
	for _, airfield := range s.Arena.Airfields {
		for _, group := range airfield.Aircraft.Groups {
			s.Stats.RecordInitialPlanes(airfield.BelongPlayer, group.Name, int(group.CurCount))
		}
	}
}

// FinalizeBattleStats 结算收尾：写入结束时间与原因，并补记失去基地的在空飞机。
// 幂等：已经结算过直接返回，避免 EndTick 被后续帧覆盖。
func (s *MissionState) FinalizeBattleStats(reason BattleEndReason) {
	if s.Stats.Finalized {
		return
	}
	s.Stats.EndTick = s.Core.SimTick
	s.Stats.EndReason = reason
	s.Stats.Finalized = true
	for _, plane := range s.Arena.Planes {
		if plane == nil || plane.CurHP <= 0 {
			continue
		}
		if _, ok := s.FindAircraftBase(plane.BelongShip); ok {
			continue
		}
		s.Stats.RecordPlaneLost(plane)
	}
}
