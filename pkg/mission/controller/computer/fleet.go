package computer

import (
	"fmt"
	"math"
	"sort"

	"github.com/narasux/jutland/pkg/config"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

type battleSnapshot struct {
	own    []*objUnit.BattleShip
	enemy  []*objUnit.BattleShip
	byUID  map[string]*objUnit.BattleShip
	points []*objBuilding.ReinforcePoint
	// misState 这一拍的战场状态，迷雾下靠它查视野图和接触。
	misState *state.MissionState
}

func (h *ComputerDecisionHandler) snapshot(misState *state.MissionState) *battleSnapshot {
	snap := &battleSnapshot{byUID: map[string]*objUnit.BattleShip{}, misState: misState}
	for _, ship := range misState.Arena.Ships {
		if ship == nil || ship.CurHP <= 0 {
			continue
		}
		if ship.BelongPlayer == h.player {
			snap.byUID[ship.Uid] = ship
			snap.own = append(snap.own, ship)
		} else if misState.SeenBy(h.player, ship.CurPos.MX, ship.CurPos.MY) {
			// 迷雾关闭时 SeenBy 恒为真，候选和原来一样是全部活着的敌舰。
			// 看不见的敌舰不进 byUID，追击和沉没判定就自然按「跟丢了」处理。
			snap.byUID[ship.Uid] = ship
			snap.enemy = append(snap.enemy, ship)
		}
	}
	for _, point := range misState.Arena.ReinforcePoints {
		if point == nil || point.BelongPlayer != h.player {
			continue
		}
		snap.points = append(snap.points, point)
	}
	sort.Slice(snap.points, func(i, j int) bool {
		return snap.points[i].Uid < snap.points[j].Uid
	})
	// 第一拍记下初始舰队的任务给定站位：没有增援集结点时靠它保持原始阵型。
	if h.stations == nil && len(snap.points) == 0 {
		h.stations = make(map[string]objPos.MapPos, len(snap.own))
		for _, ship := range snap.own {
			h.stations[ship.Uid] = ship.CurPos
		}
	}
	return snap
}

func (s *battleSnapshot) ships(uids []string) []*objUnit.BattleShip {
	out := make([]*objUnit.BattleShip, 0, len(uids))
	for _, uid := range uids {
		ship := s.byUID[uid]
		if ship == nil || ship.CurHP <= 0 {
			continue
		}
		out = append(out, ship)
	}
	return out
}

type groupPower struct {
	antiShip int
	count    int
}

func powerOf(ships []*objUnit.BattleShip) groupPower {
	var power groupPower
	for _, ship := range ships {
		power.antiShip += ship.CombatPower.AntiShip
		power.count++
	}
	return power
}

func powerOfOne(ship *objUnit.BattleShip) groupPower {
	return groupPower{antiShip: ship.CombatPower.AntiShip, count: 1}
}

// antiShipScores 在双方对舰战斗力都是 0 时退回艘数，每艘计 1。
func antiShipScores(own, enemy groupPower) (float64, float64) {
	if own.antiShip == 0 && enemy.antiShip == 0 {
		return float64(own.count), float64(enemy.count)
	}
	return float64(own.antiShip), float64(enemy.antiShip)
}

func scoreBelow(own, enemy groupPower, ratio float64) bool {
	ownScore, enemyScore := antiShipScores(own, enemy)
	return ownScore < ratio*enemyScore
}

func scoreAtLeast(own, enemy groupPower, ratio float64) bool {
	ownScore, enemyScore := antiShipScores(own, enemy)
	return ownScore >= ratio*enemyScore
}

func scoreAbove(own, enemy groupPower, ratio float64) bool {
	ownScore, enemyScore := antiShipScores(own, enemy)
	if enemyScore == 0 {
		return ownScore > 0
	}
	return ownScore > ratio*enemyScore
}

func hpSum(ships []*objUnit.BattleShip) float64 {
	var sum float64
	for _, ship := range ships {
		sum += ship.CurHP
	}
	return sum
}

func hpRatio(ship *objUnit.BattleShip) float64 {
	if ship.TotalHP <= 0 {
		return 0
	}
	return ship.CurHP / ship.TotalHP
}

func (h *ComputerDecisionHandler) regroup(snap *battleSnapshot) map[string]instr.Instruction {
	anchor, anchorOK := h.currentAnchor(snap)
	h.planRetreats(snap, anchor, anchorOK)

	roster := newFleetRoster(snap, h.retreats, anchor, anchorOK)
	summons := roster.planSummons()

	garrisonShips := roster.pickGarrisonShips()
	h.garrison = orderOf(garrisonShips)
	h.lockAnchor(snap, garrisonShips)

	used := map[string]bool{}
	for uid := range h.retreats {
		used[uid] = true
	}
	for _, ship := range garrisonShips {
		used[ship.Uid] = true
	}

	allowed := config.EnabledAIStrategies()
	if allowed.Raid {
		h.raid = h.assignRaid(snap, used)
	} else {
		h.raid = nil
	}
	if h.raid != nil {
		for _, uid := range h.raid.members {
			used[uid] = true
		}
	}
	if allowed.Scout {
		h.scout = h.assignScout(snap, used)
	} else {
		h.scout = nil
	}
	if h.scout != nil {
		for _, uid := range h.scout.members {
			used[uid] = true
		}
	}

	remaining := attackCandidates(snap, used)
	prevAttack := h.attack
	prevCounter := h.counter
	if allowed.Attack {
		h.attack = h.makeAttack(prevAttack, snap, remaining)
		h.counter = nil
	} else {
		h.attack = nil
		if allowed.Counter {
			h.counter = h.makeCounter(prevCounter, snap, remaining)
		} else {
			h.counter = nil
		}
	}
	return summons
}

func (h *ComputerDecisionHandler) currentAnchor(snap *battleSnapshot) (objPos.MapPos, bool) {
	if len(snap.points) > 0 {
		return snap.points[0].RallyPos, true
	}
	if h.anchorFixed {
		return h.anchor, true
	}
	seeds := alwaysGarrisonShips(snap.own, nil)
	if len(seeds) == 0 {
		seeds = snap.own
	}
	if len(seeds) == 0 {
		return objPos.MapPos{}, false
	}
	return centroid(seeds), true
}

func (h *ComputerDecisionHandler) lockAnchor(snap *battleSnapshot, garrison []*objUnit.BattleShip) {
	if len(snap.points) > 0 {
		h.anchor = snap.points[0].RallyPos
		h.anchorFixed = false
		return
	}
	if h.anchorFixed || len(garrison) == 0 {
		return
	}
	h.anchor = centroid(garrison)
	h.anchorFixed = true
}

func (h *ComputerDecisionHandler) planRetreats(snap *battleSnapshot, anchor objPos.MapPos, anchorOK bool) {
	next := map[string]objPos.MapPos{}
	for _, ship := range snap.own {
		nearby := shipsWithin(ship.CurPos, snap.enemy, nearbyRadius)
		if !shouldRetreat(ship, nearby) {
			continue
		}
		next[ship.Uid] = h.retreatAnchorFor(ship, snap, anchor, anchorOK)
	}
	order := make([]string, 0, len(next))
	for uid := range next {
		order = append(order, uid)
	}
	sort.Strings(order)
	h.retreats = next
	h.retreatOrder = order
}

func shouldRetreat(ship *objUnit.BattleShip, nearby []*objUnit.BattleShip) bool {
	ratio := hpRatio(ship)
	if ratio >= healthyHP {
		return false
	}
	if ratio >= criticalHP {
		if len(nearby) == 0 {
			return false
		}
		if !scoreAbove(powerOf(nearby), powerOfOne(ship), 1) {
			return false
		}
		return ship.MaxSpeed <= fastest(nearby)
	}
	if len(nearby) == 0 {
		return true
	}
	for _, enemy := range nearby {
		if hpRatio(enemy) >= criticalHP {
			return true
		}
	}
	if scoreAtLeast(powerOfOne(ship), powerOf(nearby), 1) {
		return false
	}
	return true
}

func (h *ComputerDecisionHandler) retreatAnchorFor(
	ship *objUnit.BattleShip, snap *battleSnapshot, anchor objPos.MapPos, anchorOK bool,
) objPos.MapPos {
	if h.onStation(snap, ship) {
		if len(snap.points) > 0 {
			return snap.points[0].RallyPos
		}
		return h.anchor
	}
	if len(snap.points) > 0 {
		return snap.points[0].RallyPos
	}
	if escort := nearestCapital(snap.own, ship); escort != nil {
		return asternOf(escort, retreatAsternCells)
	}
	if anchorOK {
		return anchor
	}
	return ship.CurPos
}

func (h *ComputerDecisionHandler) onStation(snap *battleSnapshot, ship *objUnit.BattleShip) bool {
	if h.garrison == nil {
		return false
	}
	index := indexOf(h.garrison.members, ship.Uid)
	if index < 0 {
		return false
	}
	slot := h.stationPos(snap, ship, index)
	return ship.CurPos.Near(slot, arrivalDistance) || ship.CurPos.Near(h.anchor, arrivalDistance)
}

func (h *ComputerDecisionHandler) makeCounter(
	prev *fleetOrder, snap *battleSnapshot, members []*objUnit.BattleShip,
) *fleetOrder {
	if len(members) == 0 {
		return nil
	}
	order := &fleetOrder{members: sortedShipUIDs(members)}
	pool := shipsWithin(h.anchor, snap.enemy, nearbyRadius)
	target := chooseAttackTarget(members, pool)
	if target == nil {
		order.pressing = false
		order.fellBack = prev != nil && prev.fellBack
		// 反击是守锚点的策略，看不见敌人就地待命，不主动出去搜。
		return order
	}
	order.targetUID = target.Uid
	local := shipsWithin(target.CurPos, snap.enemy, nearbyRadius)
	order.pressing, order.fellBack = nextPress(
		prev, powerOf(members), powerOf(local), hpSum(members), hpSum(local),
	)
	return order
}

func (h *ComputerDecisionHandler) assignScout(
	snap *battleSnapshot, used map[string]bool,
) *fleetOrder {
	var candidates []*objUnit.BattleShip
	for _, ship := range snap.own {
		if used[ship.Uid] || !isRaidShip(ship.Type) {
			continue
		}
		candidates = append(candidates, ship)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return betterRaidShip(candidates[i], candidates[j])
	})
	if len(candidates) > scoutMaxShip {
		candidates = candidates[:scoutMaxShip]
	}
	if len(candidates) == 0 {
		return nil
	}
	target := chooseScoutTarget(candidates, snap.enemy)
	if target == nil {
		return nil
	}
	return &fleetOrder{members: sortedShipUIDs(candidates), targetUID: target.Uid}
}

func chooseScoutTarget(squad, enemies []*objUnit.BattleShip) *objUnit.BattleShip {
	if len(squad) == 0 || len(enemies) == 0 {
		return nil
	}
	return nearestEnemy(centroid(squad), enemies)
}

func nearestEnemy(from objPos.MapPos, enemies []*objUnit.BattleShip) *objUnit.BattleShip {
	var best *objUnit.BattleShip
	bestDist := 0.0
	for _, enemy := range enemies {
		dist := from.Distance(enemy.CurPos)
		if best == nil || dist < bestDist || (dist == bestDist && enemy.Uid < best.Uid) {
			best = enemy
			bestDist = dist
		}
	}
	return best
}

// standoffPos 停在目标外侧，侦察舰靠近观察但不贴上去打。
func standoffPos(from, enemy objPos.MapPos, gap float64) objPos.MapPos {
	dx := float64(from.MX - enemy.MX)
	dy := float64(from.MY - enemy.MY)
	dist := math.Hypot(dx, dy)
	if dist == 0 {
		return objPos.New(enemy.MX+int(gap), enemy.MY)
	}
	if dist <= gap {
		return from
	}
	scale := gap / dist
	return objPos.New(int(math.Round(float64(enemy.MX)+dx*scale)), int(math.Round(float64(enemy.MY)+dy*scale)))
}

func (h *ComputerDecisionHandler) assignRaid(
	snap *battleSnapshot, used map[string]bool,
) *fleetOrder {
	var candidates []*objUnit.BattleShip
	for _, ship := range snap.own {
		if used[ship.Uid] || !isRaidShip(ship.Type) {
			continue
		}
		candidates = append(candidates, ship)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return betterRaidShip(candidates[i], candidates[j])
	})
	if len(candidates) > raidMaxShip {
		candidates = candidates[:raidMaxShip]
	}
	if len(candidates) == 0 {
		return nil
	}
	target := chooseRaidTarget(candidates, snap.enemy)
	if target == nil || raidAbandoned(candidates, snap.enemy, target) {
		return nil
	}
	return &fleetOrder{members: sortedShipUIDs(candidates), targetUID: target.Uid}
}

func (h *ComputerDecisionHandler) makeAttack(
	prev *fleetOrder, snap *battleSnapshot, members []*objUnit.BattleShip,
) *fleetOrder {
	if len(members) == 0 {
		return nil
	}
	order := &fleetOrder{members: sortedShipUIDs(members)}
	target := chooseAttackTarget(members, snap.enemy)
	if target == nil {
		order.pressing = false
		order.fellBack = prev != nil && prev.fellBack
		// 看不见敌人时别再回锚点罚站，整队转去搜索推进。
		order.searching = true
		return order
	}
	order.targetUID = target.Uid
	local := shipsWithin(target.CurPos, snap.enemy, nearbyRadius)
	order.pressing, order.fellBack = nextPress(
		prev, powerOf(members), powerOf(local), hpSum(members), hpSum(local),
	)
	return order
}

func nextPress(prev *fleetOrder, own, enemy groupPower, ownHP, enemyHP float64) (bool, bool) {
	wasPressing := prev != nil && prev.pressing
	fellBack := prev != nil && prev.fellBack
	if wasPressing {
		if scoreBelow(own, enemy, pressStop) || ownHP < pressStop*enemyHP {
			return false, true
		}
		return true, false
	}
	if fellBack {
		if scoreAbove(own, enemy, pressResume) && ownHP > pressResume*enemyHP {
			return true, false
		}
		return false, true
	}
	if scoreAtLeast(own, enemy, pressMin) && ownHP >= enemyHP {
		return true, false
	}
	return false, false
}

func attackCandidates(snap *battleSnapshot, used map[string]bool) []*objUnit.BattleShip {
	var out []*objUnit.BattleShip
	for _, ship := range snap.own {
		if used[ship.Uid] || !isCombat(ship.Type) {
			continue
		}
		out = append(out, ship)
	}
	return out
}

func chooseAttackTarget(squad, enemies []*objUnit.BattleShip) *objUnit.BattleShip {
	if len(squad) == 0 || len(enemies) == 0 {
		return nil
	}
	center := centroid(squad)
	nearest := enemies[0]
	nearestDist := center.Distance(nearest.CurPos)
	for _, enemy := range enemies[1:] {
		dist := center.Distance(enemy.CurPos)
		if dist < nearestDist || (dist == nearestDist && enemy.Uid < nearest.Uid) {
			nearest = enemy
			nearestDist = dist
		}
	}
	cluster := make([]*objUnit.BattleShip, 0, len(enemies))
	for _, enemy := range enemies {
		if enemy.Uid == nearest.Uid || enemy.CurPos.Distance(nearest.CurPos) <= clusterRadius {
			cluster = append(cluster, enemy)
		}
	}
	sort.Slice(cluster, func(i, j int) bool {
		left, right := hpRatio(cluster[i]), hpRatio(cluster[j])
		if left != right {
			return left < right
		}
		if cluster[i].CombatPower.AntiShip != cluster[j].CombatPower.AntiShip {
			return cluster[i].CombatPower.AntiShip > cluster[j].CombatPower.AntiShip
		}
		return cluster[i].Uid < cluster[j].Uid
	})
	return cluster[0]
}

const (
	raidNone = iota
	raidIsolated
	raidWounded
	raidCarrier
)

func chooseRaidTarget(squad, enemies []*objUnit.BattleShip) *objUnit.BattleShip {
	if len(squad) == 0 || len(enemies) == 0 {
		return nil
	}
	center := centroid(squad)
	var best *objUnit.BattleShip
	bestPriority := raidNone
	bestDist := 0.0
	for _, enemy := range enemies {
		priority := raidPriority(enemy, enemies)
		if priority == raidNone {
			continue
		}
		dist := center.Distance(enemy.CurPos)
		if best == nil || priority > bestPriority ||
			(priority == bestPriority && (dist < bestDist || (dist == bestDist && enemy.Uid < best.Uid))) {
			best = enemy
			bestPriority = priority
			bestDist = dist
		}
	}
	return best
}

func raidPriority(target *objUnit.BattleShip, enemies []*objUnit.BattleShip) int {
	best := raidNone
	if lightlyEscortedCarrier(target, enemies) {
		best = raidCarrier
	}
	if hpRatio(target) < criticalHP && best < raidWounded {
		best = raidWounded
	}
	if isolatedShip(target, enemies) && best < raidIsolated {
		best = raidIsolated
	}
	return best
}

func hasRaidTarget(enemies []*objUnit.BattleShip) bool {
	for _, enemy := range enemies {
		if raidPriority(enemy, enemies) > raidNone {
			return true
		}
	}
	return false
}

func lightlyEscortedCarrier(target *objUnit.BattleShip, enemies []*objUnit.BattleShip) bool {
	if target.Type != objUnit.ShipTypeAircraftCarrier {
		return false
	}
	var others []*objUnit.BattleShip
	for _, enemy := range enemies {
		if enemy.Uid == target.Uid {
			continue
		}
		if enemy.CurPos.Distance(target.CurPos) <= isolatedRadius {
			others = append(others, enemy)
		}
	}
	if len(others) == 0 {
		return true
	}
	return powerOf(others).antiShip < target.CombatPower.AntiShip
}

func isolatedShip(target *objUnit.BattleShip, enemies []*objUnit.BattleShip) bool {
	for _, enemy := range enemies {
		if enemy.Uid == target.Uid {
			continue
		}
		if enemy.CurPos.Distance(target.CurPos) <= isolatedRadius {
			return false
		}
	}
	return true
}

func raidAbandoned(squad, enemies []*objUnit.BattleShip, target *objUnit.BattleShip) bool {
	if target == nil || len(squad) == 0 {
		return true
	}
	local := shipsWithin(target.CurPos, enemies, nearbyRadius)
	return scoreAbove(powerOf(local), powerOf(squad), raidAbandon)
}

type fleetRoster struct {
	field      []*objUnit.BattleShip
	queued     []*objUnit.BattleShip
	enemies    []*objUnit.BattleShip
	retreat    map[string]bool
	anchor     objPos.MapPos
	anchorOK   bool
	points     []*objBuilding.ReinforcePoint
	strategies config.AIStrategies
}

func newFleetRoster(
	snap *battleSnapshot, retreats map[string]objPos.MapPos, anchor objPos.MapPos, anchorOK bool,
) *fleetRoster {
	roster := &fleetRoster{
		field:      snap.own,
		enemies:    snap.enemy,
		retreat:    map[string]bool{},
		anchor:     anchor,
		anchorOK:   anchorOK,
		points:     snap.points,
		strategies: config.EnabledAIStrategies(),
	}
	for uid := range retreats {
		roster.retreat[uid] = true
	}
	for _, point := range snap.points {
		for i, oncoming := range point.OncomingShips {
			stub := shipStub(oncoming.Name, fmt.Sprintf("%s:queue:%d", point.Uid, i))
			if stub == nil {
				continue
			}
			roster.queued = append(roster.queued, stub)
		}
	}
	return roster
}

func (r *fleetRoster) planSummons() map[string]instr.Instruction {
	out := map[string]instr.Instruction{}
	for _, point := range r.points {
		if point.MaxOncomingShip <= 0 || len(point.OncomingShips) >= 1 ||
			len(point.OncomingShips) >= point.MaxOncomingShip {
			continue
		}
		name := r.chooseSummon(point)
		stub := shipStub(name, "virtual:"+point.Uid)
		if name == "" || stub == nil {
			continue
		}
		r.queued = append(r.queued, stub)
		ins := instr.NewShipSummon(point.Uid, name)
		out[ins.Uid()] = ins
	}
	return out
}

func (r *fleetRoster) chooseSummon(point *objBuilding.ReinforcePoint) string {
	options := buildOptions(point)
	if r.strategies.Defend && r.garrisonShort() {
		if name := pickGarrisonBuild(options); name != "" {
			return name
		}
	}
	if r.strategies.Defend && r.countType(objUnit.ShipTypeAircraftCarrier) == 0 {
		if name := pickType(options, objUnit.ShipTypeAircraftCarrier, r.enemiesNearAnchor()); name != "" {
			return name
		}
	}
	if r.strategies.Raid &&
		r.countTypes(objUnit.ShipTypeCruiser, objUnit.ShipTypeDestroyer) < raidMaxShip &&
		hasRaidTarget(r.enemies) {
		if name := pickRaidBuild(options); name != "" {
			return name
		}
	}
	if r.strategies.Scout && r.scoutShort() && len(r.enemies) > 0 {
		if name := pickRaidBuild(options); name != "" {
			return name
		}
	}
	if r.strategies.Attack || (r.strategies.Counter && r.enemiesNearAnchor()) {
		return pickAttackBuild(options, r.countType(objUnit.ShipTypeBattleShip), r.enemyBattleships())
	}
	return ""
}

func (r *fleetRoster) scoutShort() bool {
	limit := scoutMaxShip
	if r.strategies.Raid {
		limit += raidMaxShip
	}
	return r.countTypes(objUnit.ShipTypeCruiser, objUnit.ShipTypeDestroyer) < limit
}

func (r *fleetRoster) garrisonShort() bool {
	if !r.anchorOK {
		return false
	}
	foes := shipsWithin(r.anchor, r.enemies, nearbyRadius)
	if len(foes) == 0 {
		return false
	}
	always, combat := r.garrisonPools(true)
	filled := takeGarrison(always, combat, powerOf(foes))
	return scoreBelow(powerOf(filled), powerOf(foes), garrisonMin)
}

func (r *fleetRoster) pickGarrisonShips() []*objUnit.BattleShip {
	if !r.strategies.Defend {
		return nil
	}
	always, combat := r.garrisonPools(false)
	if !r.anchorOK {
		return always
	}
	foes := shipsWithin(r.anchor, r.enemies, nearbyRadius)
	if len(foes) == 0 {
		return always
	}
	return takeGarrison(always, combat, powerOf(foes))
}

func (r *fleetRoster) garrisonPools(includeQueued bool) (always, combat []*objUnit.BattleShip) {
	always = alwaysGarrisonShips(r.field, r.retreat)
	for _, ship := range r.field {
		if r.retreat[ship.Uid] || !isCombat(ship.Type) {
			continue
		}
		combat = append(combat, ship)
	}
	if includeQueued {
		for _, ship := range r.queued {
			if isAlwaysGarrison(ship.Type) {
				always = append(always, ship)
				continue
			}
			if isCombat(ship.Type) {
				combat = append(combat, ship)
			}
		}
	}
	sort.Slice(combat, func(i, j int) bool {
		return garrisonBefore(combat[i], combat[j])
	})
	return always, combat
}

func (r *fleetRoster) countType(shipType objUnit.ShipType) int {
	return r.countTypes(shipType)
}

func (r *fleetRoster) countTypes(types ...objUnit.ShipType) int {
	match := func(shipType objUnit.ShipType) bool {
		for _, candidate := range types {
			if shipType == candidate {
				return true
			}
		}
		return false
	}
	n := 0
	for _, ship := range r.field {
		if match(ship.Type) {
			n++
		}
	}
	for _, ship := range r.queued {
		if match(ship.Type) {
			n++
		}
	}
	return n
}

func (r *fleetRoster) enemyBattleships() int {
	n := 0
	for _, ship := range r.enemies {
		if ship.Type == objUnit.ShipTypeBattleShip {
			n++
		}
	}
	return n
}

func (r *fleetRoster) enemiesNearAnchor() bool {
	if !r.anchorOK {
		return false
	}
	return len(shipsWithin(r.anchor, r.enemies, nearbyRadius)) > 0
}

// takeGarrison 先补到 1.5 倍，富余的战列舰可以继续补，但不超过 2 倍。
func takeGarrison(base, candidates []*objUnit.BattleShip, enemy groupPower) []*objUnit.BattleShip {
	selected := append([]*objUnit.BattleShip{}, base...)
	current := powerOf(selected)
	index := 0
	for index < len(candidates) && scoreBelow(current, enemy, garrisonMin) {
		selected = append(selected, candidates[index])
		current = powerOf(selected)
		index++
	}
	for index < len(candidates) && candidates[index].Type == objUnit.ShipTypeBattleShip {
		next := powerOf(append(append([]*objUnit.BattleShip{}, selected...), candidates[index]))
		if scoreAbove(next, enemy, garrisonCap) {
			break
		}
		selected = append(selected, candidates[index])
		current = next
		index++
	}
	return selected
}

func buildOptions(point *objBuilding.ReinforcePoint) []*objUnit.BattleShip {
	var options []*objUnit.BattleShip
	seen := map[string]bool{}
	for _, name := range point.ProvidedShipNames {
		if seen[name] {
			continue
		}
		seen[name] = true
		stub := shipStub(name, name)
		if stub == nil || !canSummonType(stub.Type) {
			continue
		}
		options = append(options, stub)
	}
	return options
}

func pickGarrisonBuild(options []*objUnit.BattleShip) string {
	var battleships []*objUnit.BattleShip
	for _, ship := range options {
		if ship.Type == objUnit.ShipTypeBattleShip {
			battleships = append(battleships, ship)
		}
	}
	if best := preferBuild(battleships, true); best != nil {
		return best.Name
	}
	bestAA := -1
	for _, ship := range options {
		if !isSummonCombat(ship.Type) {
			continue
		}
		if ship.CombatPower.AntiAir > bestAA {
			bestAA = ship.CombatPower.AntiAir
		}
	}
	if bestAA < 0 {
		return ""
	}
	var same []*objUnit.BattleShip
	for _, ship := range options {
		if isSummonCombat(ship.Type) && ship.CombatPower.AntiAir == bestAA {
			same = append(same, ship)
		}
	}
	if best := preferBuild(same, true); best != nil {
		return best.Name
	}
	return ""
}

func pickRaidBuild(options []*objUnit.BattleShip) string {
	var candidates []*objUnit.BattleShip
	for _, ship := range options {
		if isRaidShip(ship.Type) {
			candidates = append(candidates, ship)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	best := candidates[0]
	for _, ship := range candidates[1:] {
		if betterRaidShip(ship, best) {
			best = ship
		}
	}
	return best.Name
}

func pickAttackBuild(options []*objUnit.BattleShip, ownBB, enemyBB int) string {
	tiers := []objUnit.ShipType{
		objUnit.ShipTypeCruiser,
		objUnit.ShipTypeDestroyer,
		objUnit.ShipTypeFrigate,
	}
	if ownBB < enemyBB {
		tiers = append([]objUnit.ShipType{objUnit.ShipTypeBattleShip}, tiers...)
	}
	for _, tier := range tiers {
		if name := pickType(options, tier, false); name != "" {
			return name
		}
	}
	return ""
}

func pickType(options []*objUnit.BattleShip, shipType objUnit.ShipType, enemiesNear bool) string {
	var candidates []*objUnit.BattleShip
	for _, ship := range options {
		if ship.Type == shipType {
			candidates = append(candidates, ship)
		}
	}
	best := preferBuild(candidates, enemiesNear)
	if best == nil {
		return ""
	}
	return best.Name
}

func preferBuild(ships []*objUnit.BattleShip, enemiesNear bool) *objUnit.BattleShip {
	if len(ships) == 0 {
		return nil
	}
	best := ships[0]
	for _, ship := range ships[1:] {
		if betterBuild(ship, best, enemiesNear) {
			best = ship
		}
	}
	return best
}

func betterBuild(left, right *objUnit.BattleShip, enemiesNear bool) bool {
	if enemiesNear {
		if left.TimeCost != right.TimeCost {
			return left.TimeCost < right.TimeCost
		}
		if left.CombatPower.AntiShip != right.CombatPower.AntiShip {
			return left.CombatPower.AntiShip > right.CombatPower.AntiShip
		}
	} else if left.CombatPower.AntiShip != right.CombatPower.AntiShip {
		return left.CombatPower.AntiShip > right.CombatPower.AntiShip
	} else if left.TimeCost != right.TimeCost {
		return left.TimeCost < right.TimeCost
	}
	return left.Name < right.Name
}

func betterRaidShip(left, right *objUnit.BattleShip) bool {
	if left.CombatPower.Mobility != right.CombatPower.Mobility {
		return left.CombatPower.Mobility > right.CombatPower.Mobility
	}
	if left.CombatPower.Projection != right.CombatPower.Projection {
		return left.CombatPower.Projection > right.CombatPower.Projection
	}
	if left.Uid != right.Uid && left.Uid != "" && right.Uid != "" {
		return left.Uid < right.Uid
	}
	return left.Name < right.Name
}

func garrisonBefore(left, right *objUnit.BattleShip) bool {
	leftBB := left.Type == objUnit.ShipTypeBattleShip
	rightBB := right.Type == objUnit.ShipTypeBattleShip
	if leftBB != rightBB {
		return leftBB
	}
	if left.CombatPower.AntiAir != right.CombatPower.AntiAir {
		return left.CombatPower.AntiAir > right.CombatPower.AntiAir
	}
	return left.Uid < right.Uid
}

func shipStub(name, uid string) *objUnit.BattleShip {
	if name == "" {
		return nil
	}
	template := objUnit.ShipMap[name]
	if template == nil {
		return nil
	}
	ship := *template
	ship.Name = name
	ship.Uid = uid
	if ship.TotalHP > 0 && ship.CurHP <= 0 {
		ship.CurHP = ship.TotalHP
	}
	return &ship
}

func orderOf(ships []*objUnit.BattleShip) *fleetOrder {
	if len(ships) == 0 {
		return nil
	}
	return &fleetOrder{members: sortedShipUIDs(ships)}
}

func sortedShipUIDs(ships []*objUnit.BattleShip) []string {
	uids := make([]string, len(ships))
	for i, ship := range ships {
		uids[i] = ship.Uid
	}
	sort.Strings(uids)
	return uids
}

func alwaysGarrisonShips(ships []*objUnit.BattleShip, retreat map[string]bool) []*objUnit.BattleShip {
	var out []*objUnit.BattleShip
	for _, ship := range ships {
		if retreat[ship.Uid] || !isAlwaysGarrison(ship.Type) {
			continue
		}
		out = append(out, ship)
	}
	return out
}

func isAlwaysGarrison(shipType objUnit.ShipType) bool {
	switch shipType {
	case objUnit.ShipTypeAircraftCarrier, objUnit.ShipTypeCargo, objUnit.ShipTypeRepair, objUnit.ShipTypeHospital:
		return true
	default:
		return false
	}
}

func isCombat(shipType objUnit.ShipType) bool {
	switch shipType {
	case objUnit.ShipTypeBattleShip, objUnit.ShipTypeCruiser, objUnit.ShipTypeDestroyer,
		objUnit.ShipTypeFrigate, objUnit.ShipTypeTorpedoBoat:
		return true
	default:
		return false
	}
}

func isSummonCombat(shipType objUnit.ShipType) bool {
	switch shipType {
	case objUnit.ShipTypeBattleShip, objUnit.ShipTypeCruiser, objUnit.ShipTypeDestroyer, objUnit.ShipTypeFrigate:
		return true
	default:
		return false
	}
}

func isRaidShip(shipType objUnit.ShipType) bool {
	return shipType == objUnit.ShipTypeCruiser || shipType == objUnit.ShipTypeDestroyer
}

func canSummonType(shipType objUnit.ShipType) bool {
	switch shipType {
	case objUnit.ShipTypeCargo, objUnit.ShipTypeRepair, objUnit.ShipTypeHospital, objUnit.ShipTypeTorpedoBoat:
		return false
	default:
		return true
	}
}

func shipsWithin(origin objPos.MapPos, ships []*objUnit.BattleShip, radius float64) []*objUnit.BattleShip {
	var out []*objUnit.BattleShip
	for _, ship := range ships {
		if ship.CurPos.Distance(origin) <= radius {
			out = append(out, ship)
		}
	}
	return out
}

func chaseTarget(ship *objUnit.BattleShip, enemies []*objUnit.BattleShip, anchor objPos.MapPos) *objUnit.BattleShip {
	var best *objUnit.BattleShip
	bestDist := 0.0
	for _, enemy := range enemies {
		if enemy.CurPos.Distance(anchor) > chaseAnchorRadius {
			continue
		}
		dist := ship.CurPos.Distance(enemy.CurPos)
		if dist > nearbyRadius {
			continue
		}
		if best == nil || dist < bestDist || (dist == bestDist && enemy.Uid < best.Uid) {
			best = enemy
			bestDist = dist
		}
	}
	return best
}

func centroid(ships []*objUnit.BattleShip) objPos.MapPos {
	if len(ships) == 0 {
		return objPos.MapPos{}
	}
	var sumX, sumY int
	for _, ship := range ships {
		sumX += ship.CurPos.MX
		sumY += ship.CurPos.MY
	}
	return objPos.New(sumX/len(ships), sumY/len(ships))
}

func asternOf(ship *objUnit.BattleShip, cells int) objPos.MapPos {
	rad := ship.CurRotation * math.Pi / 180
	x := float64(ship.CurPos.MX) - math.Sin(rad)*float64(cells)
	y := float64(ship.CurPos.MY) + math.Cos(rad)*float64(cells)
	return objPos.New(int(math.Round(x)), int(math.Round(y)))
}

func nearestCapital(ships []*objUnit.BattleShip, from *objUnit.BattleShip) *objUnit.BattleShip {
	var best *objUnit.BattleShip
	bestDist := 0.0
	for _, ship := range ships {
		if ship.Uid == from.Uid {
			continue
		}
		if ship.Type != objUnit.ShipTypeBattleShip && ship.Type != objUnit.ShipTypeAircraftCarrier {
			continue
		}
		dist := from.CurPos.Distance(ship.CurPos)
		if best == nil || dist < bestDist || (dist == bestDist && ship.Uid < best.Uid) {
			best = ship
			bestDist = dist
		}
	}
	return best
}

func fastest(ships []*objUnit.BattleShip) float64 {
	var speed float64
	for _, ship := range ships {
		if ship.MaxSpeed > speed {
			speed = ship.MaxSpeed
		}
	}
	return speed
}

// formationPos 把成员排在锚点外间距 2 格的方环上，0 号占锚点本身。
func formationPos(anchor objPos.MapPos, index int) objPos.MapPos {
	dx, dy := spiralOffset(index)
	return objPos.New(anchor.MX+dx, anchor.MY+dy)
}

func spiralOffset(index int) (int, int) {
	if index <= 0 {
		return 0, 0
	}
	ring := 1
	remaining := index
	for {
		capacity := 8 * ring
		if remaining <= capacity {
			x, y := ringPoint(ring, remaining-1)
			return x * formationStep, y * formationStep
		}
		remaining -= capacity
		ring++
	}
}

func ringPoint(ring, index int) (int, int) {
	side := ring * 2
	x, y := -ring, -ring
	steps := [][3]int{{1, 0, side}, {0, 1, side}, {-1, 0, side}, {0, -1, side}}
	left := index
	for _, step := range steps {
		if left < step[2] {
			return x + step[0]*left, y + step[1]*left
		}
		x += step[0] * step[2]
		y += step[1] * step[2]
		left -= step[2]
	}
	return 0, 0
}

func samePos(left, right objPos.MapPos) bool {
	return left.MX == right.MX && left.MY == right.MY
}

func indexOf(uids []string, uid string) int {
	for i, candidate := range uids {
		if candidate == uid {
			return i
		}
	}
	return -1
}
