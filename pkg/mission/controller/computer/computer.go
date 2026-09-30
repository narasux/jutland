package computer

import (
	"github.com/narasux/jutland/pkg/mission/controller"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

const (
	// 逻辑帧 60。三档错开，避免整队同时编组并挤进只有 2 个工人的寻路。
	checkInterval   int64 = 8
	pathInterval    int64 = 30
	regroupInterval int64 = 60

	nearbyRadius        = 20.0
	clusterRadius       = 12.0
	chaseAnchorRadius   = 12.0
	isolatedRadius      = 15.0
	pathRefreshDistance = 8.0
	arrivalDistance     = 1.0

	healthyHP   = 0.7
	criticalHP  = 0.35
	garrisonMin = 1.5
	garrisonCap = 2.0
	pressMin    = 1.0
	pressStop   = 0.85
	pressResume = 1.15
	raidAbandon = 1.5
	raidMaxShip = 3

	retreatAsternCells = 6
	formationStep      = 2
	scoutMaxShip       = 2
	scoutStandoff      = isolatedRadius
)

// fleetOrder 是驻守、进攻或袭击之一。成员按 uid 排序，序号即寻路相位。
type fleetOrder struct {
	members       []string
	targetUID     string
	needsReselect bool
	pressing      bool
	fellBack      bool
}

// routeMemory 记录这艘舰上次下发的目的地，用来判断目标是否已经挪过 8 格。
type routeMemory struct {
	dest      objPos.MapPos
	targetUID string
}

// ComputerDecisionHandler 电脑决策处理器。编组记在处理器上，不写入任务状态。
type ComputerDecisionHandler struct {
	player faction.Player

	garrison *fleetOrder
	attack   *fleetOrder
	counter  *fleetOrder
	raid     *fleetOrder
	scout    *fleetOrder

	retreats     map[string]objPos.MapPos
	retreatOrder []string

	anchor      objPos.MapPos
	anchorFixed bool

	routes map[string]routeMemory
}

// NewHandler ...
func NewHandler(player faction.Player) *ComputerDecisionHandler {
	return &ComputerDecisionHandler{
		player:   player,
		retreats: map[string]objPos.MapPos{},
		routes:   map[string]routeMemory{},
	}
}

var _ controller.InputHandler = (*ComputerDecisionHandler)(nil)

// Handle 按 8 / 30 / 60 帧处理电脑舰队。返回的指令由调用方合并进指令集。
func (h *ComputerDecisionHandler) Handle(
	curInstructions map[string]instr.Instruction, misState *state.MissionState,
) map[string]instr.Instruction {
	tick := misState.Core.SimTick
	regrouping := tick%regroupInterval == 0
	checking := tick%checkInterval == 0
	if !checking && !regrouping && !h.anyPathDue(tick) {
		return nil
	}

	snap := h.snapshot(misState)
	out := map[string]instr.Instruction{}
	if checking {
		h.markSunkTargets(snap)
	}
	if regrouping {
		for uid, ins := range h.regroup(snap) {
			out[uid] = ins
		}
	}
	for uid, ins := range h.maintainPaths(tick, curInstructions, snap) {
		out[uid] = ins
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (h *ComputerDecisionHandler) anyPathDue(tick int64) bool {
	if h.garrison != nil && groupDue(tick, len(h.garrison.members)) {
		return true
	}
	if h.attack != nil && groupDue(tick, len(h.attack.members)) {
		return true
	}
	if h.counter != nil && groupDue(tick, len(h.counter.members)) {
		return true
	}
	if h.raid != nil && groupDue(tick, len(h.raid.members)) {
		return true
	}
	if h.scout != nil && groupDue(tick, len(h.scout.members)) {
		return true
	}
	return groupDue(tick, len(h.retreatOrder))
}

func (h *ComputerDecisionHandler) markSunkTargets(snap *battleSnapshot) {
	h.markSunk(h.attack, snap)
	h.markSunk(h.counter, snap)
	h.markSunk(h.raid, snap)
	h.markSunk(h.scout, snap)
}

func (h *ComputerDecisionHandler) markSunk(order *fleetOrder, snap *battleSnapshot) {
	if order == nil || order.targetUID == "" {
		return
	}
	ship := snap.byUID[order.targetUID]
	if ship == nil || ship.CurHP <= 0 {
		order.needsReselect = true
	}
}

func (h *ComputerDecisionHandler) maintainPaths(
	tick int64, cur map[string]instr.Instruction, snap *battleSnapshot,
) map[string]instr.Instruction {
	h.reselectSharedTargets(tick, snap)
	out := map[string]instr.Instruction{}
	if h.garrison != nil {
		h.issueMembers(out, tick, cur, snap, h.garrison, routeGarrison)
	}
	if h.attack != nil {
		h.issueMembers(out, tick, cur, snap, h.attack, routeAttack)
	}
	if h.counter != nil {
		h.issueMembers(out, tick, cur, snap, h.counter, routeCounter)
	}
	if h.raid != nil {
		h.issueMembers(out, tick, cur, snap, h.raid, routeRaid)
	}
	if h.scout != nil {
		h.issueMembers(out, tick, cur, snap, h.scout, routeScout)
	}
	h.issueRetreats(out, tick, cur, snap)
	return out
}

func (h *ComputerDecisionHandler) reselectSharedTargets(tick int64, snap *battleSnapshot) {
	if h.attack != nil && h.attack.needsReselect && groupDue(tick, len(h.attack.members)) {
		members := snap.ships(h.attack.members)
		if target := chooseAttackTarget(members, snap.enemy); target != nil {
			h.attack.targetUID = target.Uid
		} else {
			h.attack.targetUID = ""
		}
		h.attack.needsReselect = false
	}
	if h.counter != nil && h.counter.needsReselect && groupDue(tick, len(h.counter.members)) {
		members := snap.ships(h.counter.members)
		pool := shipsWithin(h.anchor, snap.enemy, nearbyRadius)
		if target := chooseAttackTarget(members, pool); target != nil {
			h.counter.targetUID = target.Uid
		} else {
			h.counter.targetUID = ""
		}
		h.counter.needsReselect = false
	}
	if h.raid != nil && h.raid.needsReselect && groupDue(tick, len(h.raid.members)) {
		members := snap.ships(h.raid.members)
		if target := chooseRaidTarget(members, snap.enemy); target != nil {
			h.raid.targetUID = target.Uid
		} else {
			h.raid.targetUID = ""
		}
		h.raid.needsReselect = false
	}
	if h.scout != nil && h.scout.needsReselect && groupDue(tick, len(h.scout.members)) {
		members := snap.ships(h.scout.members)
		if target := chooseScoutTarget(members, snap.enemy); target != nil {
			h.scout.targetUID = target.Uid
		} else {
			h.scout.targetUID = ""
		}
		h.scout.needsReselect = false
	}
}

type routeKind int

const (
	routeGarrison routeKind = iota
	routeAttack
	routeCounter
	routeRaid
	routeScout
)

type shipRoute struct {
	dest      objPos.MapPos
	targetUID string
}

func (h *ComputerDecisionHandler) issueMembers(
	out map[string]instr.Instruction,
	tick int64,
	cur map[string]instr.Instruction,
	snap *battleSnapshot,
	order *fleetOrder,
	kind routeKind,
) {
	for index, uid := range order.members {
		if !phaseDue(tick, index) {
			continue
		}
		ship := h.ownShip(snap, uid)
		if ship == nil {
			continue
		}
		desired, ok := h.desiredRoute(ship, index, order, kind, snap)
		if !ok || !h.shouldIssue(cur, ship, desired) {
			continue
		}
		h.setRoute(out, ship, desired)
	}
}

func (h *ComputerDecisionHandler) issueRetreats(
	out map[string]instr.Instruction,
	tick int64,
	cur map[string]instr.Instruction,
	snap *battleSnapshot,
) {
	for index, uid := range h.retreatOrder {
		if !phaseDue(tick, index) {
			continue
		}
		ship := h.ownShip(snap, uid)
		if ship == nil {
			continue
		}
		slot := h.retreatSlot(uid)
		if ship.CurPos.Near(slot, arrivalDistance) {
			continue
		}
		desired := shipRoute{dest: slot}
		if !h.shouldIssue(cur, ship, desired) {
			continue
		}
		h.setRoute(out, ship, desired)
	}
}

func (h *ComputerDecisionHandler) desiredRoute(
	ship *objUnit.BattleShip, index int, order *fleetOrder, kind routeKind, snap *battleSnapshot,
) (shipRoute, bool) {
	switch kind {
	case routeAttack:
		if order.pressing {
			if route, ok := h.pursue(order, snap); ok {
				return route, true
			}
		}
		slot := formationPos(h.anchor, h.garrisonCount()+index)
		return h.anchorRoute(ship, slot, snap)
	case routeCounter:
		if order.pressing {
			if route, ok := h.pursue(order, snap); ok {
				return route, true
			}
		}
		slot := formationPos(h.anchor, h.garrisonCount()+index)
		return h.anchorRoute(ship, slot, snap)
	case routeRaid:
		if route, ok := h.pursue(order, snap); ok {
			return route, true
		}
		slot := formationPos(h.anchor, h.anchorOccupants()+index)
		if ship.CurPos.Near(slot, arrivalDistance) {
			return shipRoute{}, false
		}
		return shipRoute{dest: slot}, true
	case routeScout:
		if route, ok := h.scoutRoute(ship, order, snap); ok {
			return route, true
		}
		slot := formationPos(h.anchor, h.anchorOccupants()+index)
		if ship.CurPos.Near(slot, arrivalDistance) {
			return shipRoute{}, false
		}
		return shipRoute{dest: slot}, true
	default:
		slot := formationPos(h.anchor, index)
		return h.anchorRoute(ship, slot, snap)
	}
}

func (h *ComputerDecisionHandler) scoutRoute(
	ship *objUnit.BattleShip, order *fleetOrder, snap *battleSnapshot,
) (shipRoute, bool) {
	target := snap.byUID[order.targetUID]
	if target == nil || target.CurHP <= 0 {
		return shipRoute{}, false
	}
	dest := standoffPos(centroid(snap.ships(order.members)), target.CurPos, scoutStandoff)
	if ship.CurPos.Near(dest, arrivalDistance) {
		return shipRoute{}, false
	}
	return shipRoute{dest: dest, targetUID: target.Uid}, true
}

func (h *ComputerDecisionHandler) pursue(order *fleetOrder, snap *battleSnapshot) (shipRoute, bool) {
	if order.targetUID == "" {
		return shipRoute{}, false
	}
	target := snap.byUID[order.targetUID]
	if target == nil || target.CurHP <= 0 {
		return shipRoute{}, false
	}
	return shipRoute{dest: target.CurPos, targetUID: target.Uid}, true
}

// anchorRoute 是驻守，以及进攻停下后的行为：锚点附近才短距离追击，否则回阵位。
func (h *ComputerDecisionHandler) anchorRoute(
	ship *objUnit.BattleShip, slot objPos.MapPos, snap *battleSnapshot,
) (shipRoute, bool) {
	if foe := chaseTarget(ship, snap.enemy, h.anchor); foe != nil {
		return shipRoute{dest: foe.CurPos, targetUID: foe.Uid}, true
	}
	if ship.CurPos.Near(slot, arrivalDistance) {
		return shipRoute{}, false
	}
	return shipRoute{dest: slot}, true
}

func (h *ComputerDecisionHandler) shouldIssue(
	cur map[string]instr.Instruction, ship *objUnit.BattleShip, desired shipRoute,
) bool {
	if _, moving := cur[instr.GenInstrUid(instr.NameShipMove, ship.Uid)]; !moving {
		return true
	}
	mem, ok := h.routes[ship.Uid]
	if !ok {
		return true
	}
	if desired.targetUID != mem.targetUID {
		return true
	}
	return desired.dest.Distance(mem.dest) > pathRefreshDistance
}

func (h *ComputerDecisionHandler) setRoute(
	out map[string]instr.Instruction, ship *objUnit.BattleShip, desired shipRoute,
) {
	h.routes[ship.Uid] = routeMemory{dest: desired.dest, targetUID: desired.targetUID}
	var ins instr.Instruction
	if ship.CanOnLand() {
		ins = instr.NewShipMove(ship.Uid, desired.dest)
	} else {
		ins = instr.NewShipMovePath(ship.Uid, ship.CurPos, desired.dest, ship.CurSpeed)
	}
	out[ins.Uid()] = ins
}

func (h *ComputerDecisionHandler) ownShip(snap *battleSnapshot, uid string) *objUnit.BattleShip {
	ship := snap.byUID[uid]
	if ship == nil || ship.CurHP <= 0 || ship.BelongPlayer != h.player {
		return nil
	}
	return ship
}

func (h *ComputerDecisionHandler) garrisonCount() int {
	if h.garrison == nil {
		return 0
	}
	return len(h.garrison.members)
}

func (h *ComputerDecisionHandler) anchorOccupants() int {
	n := h.garrisonCount()
	if h.attack != nil && !h.attack.pressing {
		n += len(h.attack.members)
	}
	if h.counter != nil && !h.counter.pressing {
		n += len(h.counter.members)
	}
	return n
}

func (h *ComputerDecisionHandler) retreatSlot(uid string) objPos.MapPos {
	anchor := h.retreats[uid]
	index := 0
	for _, id := range h.retreatOrder {
		if id == uid {
			break
		}
		if samePos(h.retreats[id], anchor) {
			index++
		}
	}
	base := 0
	if samePos(anchor, h.anchor) {
		base = h.anchorOccupants()
		if h.raid != nil && h.raid.targetUID == "" {
			base += len(h.raid.members)
		}
	}
	return formationPos(anchor, base+index)
}

func phaseDue(tick int64, index int) bool {
	return (tick+int64(index))%pathInterval == 0
}

func groupDue(tick int64, n int) bool {
	if n <= 0 {
		return false
	}
	residue := int((pathInterval - tick%pathInterval) % pathInterval)
	return residue < n
}
