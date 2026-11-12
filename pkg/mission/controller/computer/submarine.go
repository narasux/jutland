package computer

import (
	"math"

	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

const (
	// subEngageRadius 潜艇发现敌舰后开始下潜接敌的距离
	subEngageRadius = 16.0
	// subRetreatHPRatio 潜艇残血线：被探测或残血时保持深水撤离
	subRetreatHPRatio = 0.35
	// subRetreatDistance 撤离时朝反方向跑的距离
	subRetreatDistance = 12.0
	// aswResponseRadius 反潜舰响应潜艇接触的最大距离
	aswResponseRadius = 24.0
	// aswFallbackEngageRange 未统计到反潜射程时的兜底贴近距离
	aswFallbackEngageRange = 4.0
	// aswSearchMemorySeconds 失去接触后继续搜索最后位置的时间
	aswSearchMemorySeconds = 30.0
)

// subContact 是反潜舰可用的潜艇接触：live 表示当前仍被声呐点亮，
// 否则是搜索窗口内的最后已知位置。
type subContact struct {
	ship *objUnit.BattleShip
	pos  objPos.MapPos
	live bool
}

// subContactFor 读取某玩家对一艘敌潜艇的接触：优先当前点亮，
// 其次读探测记忆里的最后位置（供 AI 反潜搜索，和绘制无关）。
func subContactFor(
	ms *state.MissionState, player faction.Player, ship *objUnit.BattleShip,
) (subContact, bool) {
	if ms == nil || ship == nil || ship.Type != objUnit.ShipTypeSubmarine {
		return subContact{}, false
	}
	detection, ok := ship.Detection(player)
	if !ok {
		return subContact{}, false
	}
	now := ms.Core.SimTick
	if detection.UntilTick > now {
		return subContact{ship: ship, pos: ship.CurPos, live: true}, true
	}
	if now-detection.UntilTick > objUnit.ReloadTicks(aswSearchMemorySeconds) {
		return subContact{}, false
	}
	if detection.LastPos.RX == 0 && detection.LastPos.RY == 0 {
		return subContact{}, false
	}
	return subContact{ship: ship, pos: detection.LastPos}, true
}

// shouldIssueRoute 用路线记忆判断是否需要重新下发移动指令。
// 不复用 shouldIssue：它只认 ShipMove，对实际下发的 ShipMovePath 会每拍重算。
func (h *ComputerDecisionHandler) shouldIssueRoute(
	ship *objUnit.BattleShip, desired shipRoute,
) bool {
	mem, ok := h.routes[ship.Uid]
	if !ok {
		return true
	}
	if desired.targetUID != mem.targetUID {
		return true
	}
	return desired.dest.Distance(mem.dest) > pathRefreshDistance
}

// submarineInstructions 驱动 AI 潜艇：默认保持水下接敌，
// 进鱼雷射程后攻击，被点亮或残血保持深度撤离。
// 现役潜艇都带鱼雷，水下即可完成攻击，不会为了赶路而主动上浮；
// 只有完全没有鱼雷、必须靠甲板炮的潜艇才会在炮击距离内上浮。
func (h *ComputerDecisionHandler) submarineInstructions(
	snap *battleSnapshot, tick int64,
) map[string]instr.Instruction {
	if len(snap.ownSubs) == 0 {
		return nil
	}
	out := map[string]instr.Instruction{}
	for _, sub := range snap.ownSubs {
		if sub.CurHP <= 0 {
			continue
		}
		// 潜艇的猎物包括水面舰与敌方潜艇（水听器点亮后可对潜）
		nearest, distance := nearestTarget(sub.CurPos, snap.enemy, snap.enemySubs)
		detected := sub.DetectedByAny(tick)
		lowHP := sub.TotalHP > 0 && sub.CurHP/sub.TotalHP <= subRetreatHPRatio

		wantDepth := objUnit.DiveStateSubmerged
		targetUID := ""
		dest := objPos.MapPos{}
		hasDest := false

		switch {
		case detected || lowHP:
			// 已被点亮 / 残血：保持深水，朝远离最近敌人的方向撤离
			if nearest != nil && distance > 0 {
				dest = objPos.NewR(
					sub.CurPos.RX+(sub.CurPos.RX-nearest.CurPos.RX)/distance*subRetreatDistance,
					sub.CurPos.RY+(sub.CurPos.RY-nearest.CurPos.RY)/distance*subRetreatDistance,
				)
				hasDest = true
			}
		case nearest != nil && distance <= subEngageRadius:
			// 接敌：保持水下，进不了鱼雷射程就继续压上
			targetUID = nearest.Uid
			if distance > sub.Weapon.MaxToSubmergedRange*0.9 {
				dest = nearest.CurPos
				hasDest = true
			}
		default:
			// 水下巡航：继续向最近敌舰靠拢，但不为了赶路而上浮
			if nearest != nil {
				dest = nearest.CurPos
				hasDest = true
			}
		}

		// 无鱼雷潜艇无法在水下攻击，只有进入甲板炮射程、且有可炮击的水面目标时才上浮；
		// 撤离/残血时仍保持水下，避免把自己送到敌方炮口下。
		if !detected && !lowHP && !sub.Weapon.HasTorpedo && sub.Weapon.HasMainGun &&
			nearest != nil && !nearest.Submerged() && distance <= sub.Weapon.MaxToShipRange {
			wantDepth = objUnit.DiveStateSurface
		}

		if targetUID != "" {
			sub.AttackTarget = targetUID
		} else if !detected {
			sub.AttackTarget = ""
		}

		if next, ok := sub.NextDiveState(); ok && next == wantDepth {
			if wantDepth == objUnit.DiveStateSubmerged {
				di := instr.NewShipDive(sub.Uid)
				out[di.Uid()] = di
			} else {
				di := instr.NewShipSurface(sub.Uid)
				out[di.Uid()] = di
			}
		}

		if !hasDest {
			continue
		}
		desired := shipRoute{dest: dest, targetUID: targetUID}
		desired.dest = navigableDest(snap, sub, desired.dest)
		if h.shouldIssueRoute(sub, desired) {
			h.setRoute(out, sub, desired, snap)
		}
	}
	return out
}

// selectASWHunters 从当前没有编组任务的舰里挑出反潜猎手。
// 有深弹/反潜鱼雷、且 24 格内存在潜艇接触的舰离开水面编队，专职猎潜。
func (h *ComputerDecisionHandler) selectASWHunters(
	snap *battleSnapshot, used map[string]bool,
) map[string]bool {
	if len(snap.subContacts) == 0 {
		return nil
	}
	hunters := map[string]bool{}
	for _, ship := range snap.own {
		if used[ship.Uid] || ship.CurHP <= 0 || !ship.Weapon.HasAntiSubWeapon {
			continue
		}
		if _, ok := nearestSubContact(ship.CurPos, snap.subContacts); !ok {
			continue
		}
		hunters[ship.Uid] = true
	}
	return hunters
}

// aswInstructions 让反潜舰朝接触点（点亮位置或最后已知位置）机动并贴近投弹。
func (h *ComputerDecisionHandler) aswInstructions(
	snap *battleSnapshot,
) map[string]instr.Instruction {
	if len(h.asw) == 0 {
		return nil
	}
	out := map[string]instr.Instruction{}
	for uid := range h.asw {
		ship := snap.byUID[uid]
		if ship == nil || ship.CurHP <= 0 {
			continue
		}
		contact, ok := nearestSubContact(ship.CurPos, snap.subContacts)
		if !ok {
			continue
		}
		if contact.live {
			ship.AttackTarget = contact.ship.Uid
		}
		if ship.CurPos.Distance(contact.pos) <= aswEngageDistance(ship) {
			continue
		}
		desired := shipRoute{dest: contact.pos, targetUID: contact.ship.Uid}
		desired.dest = navigableDest(snap, ship, desired.dest)
		if h.shouldIssueRoute(ship, desired) {
			h.setRoute(out, ship, desired, snap)
		}
	}
	return out
}

// aswEngageDistance 反潜舰贴近到该距离内就停止推进、开始投弹。
// 必须按自身反潜射程折算：深弹架只有 2 格，若固定停在 4 格会永远够不着。
func aswEngageDistance(ship *objUnit.BattleShip) float64 {
	weaponRange := ship.Weapon.MaxToSubmergedRange
	if weaponRange <= 0 {
		return aswFallbackEngageRange
	}
	return max(1.0, weaponRange*0.6)
}

// nearestTarget 在若干候选列表里找距 origin 最近的存活敌舰。
func nearestTarget(from objPos.MapPos, groups ...[]*objUnit.BattleShip) (*objUnit.BattleShip, float64) {
	var best *objUnit.BattleShip
	bestDist := math.MaxFloat64
	for _, group := range groups {
		for _, ship := range group {
			if ship == nil || ship.CurHP <= 0 {
				continue
			}
			if dist := from.Distance(ship.CurPos); dist < bestDist {
				best, bestDist = ship, dist
			}
		}
	}
	return best, bestDist
}

// nearestSubContact 返回距离 origin 最近、且在响应半径内的潜艇接触。
func nearestSubContact(origin objPos.MapPos, contacts []subContact) (subContact, bool) {
	best, bestDist := subContact{}, aswResponseRadius
	found := false
	for _, contact := range contacts {
		dist := origin.Distance(contact.pos)
		if dist > bestDist || (found && dist == bestDist) {
			continue
		}
		best, bestDist, found = contact, dist, true
	}
	return best, found
}
