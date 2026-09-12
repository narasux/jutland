package unit

import (
	"slices"
	"time"

	"github.com/narasux/jutland/pkg/mission/object"
)

// ShipAircraft 战舰上的飞机，也能算是武器吧 :D
type ShipAircraft struct {
	// TakeOffTime 起飞冷却（单位：秒），作为所有起飞点的默认冷却
	TakeOffTime float64 `json:"takeOffTime"`
	// Deck 起降甲板模板名（configs/carrier_decks.json5），搭载舰载机时必填
	Deck string `json:"deck"`
	// Groups 战机分组
	Groups []PlaneGroup `json:"groups"`

	// 是否禁用舰载机
	Disable bool
	// 是否拥有舰载机
	HasPlane bool
	// 最近起飞时间（毫秒时间戳)
	LatestTakeOffAt int64

	// deck 解析后的甲板模板（init 阶段填充，不参与序列化）
	deck *CarrierDeck
	// takeoffPointTimes 各起飞点的最近起飞时间（毫秒时间戳），与模板起飞点一一对应
	takeoffPointTimes []int64
	// 各目标类型的下一编组游标，避免同类型第一个机型长期抢占出击位。
	nextGroupCursor map[object.Type]int

	// 以下字段只服务于局内回收调度，不参与配置序列化。
	landingSlots map[string]int
}

// ResolveDeck 绑定解析后的甲板模板，并初始化各起飞点的冷却时间戳。
// 模板为 init 阶段加载的只读配置的拷贝，各舰实例互不影响；deck 为 nil 时清除绑定。
func (sa *ShipAircraft) ResolveDeck(deck *CarrierDeck) {
	sa.deck = deck
	if deck == nil {
		sa.takeoffPointTimes = nil
		return
	}
	sa.takeoffPointTimes = make([]int64, len(deck.TakeoffPoints))
}

// ensureTakeoffPointTimes 保证起飞点冷却时间戳与模板起飞点数量一致。
func (sa *ShipAircraft) ensureTakeoffPointTimes() {
	points := 0
	if sa.deck != nil {
		points = len(sa.deck.TakeoffPoints)
	}
	if len(sa.takeoffPointTimes) != points {
		sa.takeoffPointTimes = make([]int64, points)
	}
}

// takeOffCooldown 返回指定起飞点的实际冷却秒数（点覆盖 > 舰船默认）。
func (sa *ShipAircraft) takeOffCooldown(point TakeoffPoint) float64 {
	if point.TakeOffTime > 0 {
		return point.TakeOffTime
	}
	return sa.TakeOffTime
}

// matchingGroupIdx 返回该起飞点可用的编组下标：目标类型匹配、有库存、
// 机型在白名单内，并且运行时航程足够飞到目标。
func (sa *ShipAircraft) matchingGroupIdx(
	point TakeoffPoint,
	targetObjType object.Type,
	requiredRange float64,
) int {
	if len(sa.Groups) == 0 {
		return -1
	}
	start := 0
	if sa.nextGroupCursor != nil {
		start = sa.nextGroupCursor[targetObjType] % len(sa.Groups)
	}
	for offset := 0; offset < len(sa.Groups); offset++ {
		idx := (start + offset) % len(sa.Groups)
		g := sa.Groups[idx]
		if g.TargetType != targetObjType || g.CurCount <= 0 {
			continue
		}
		if len(point.PlaneTypes) > 0 && !slices.Contains(point.PlaneTypes, planeTypeOf(g.Name)) {
			continue
		}
		if requiredRange > 0 {
			planeRange := planeRangeOf(g.Name)
			if planeRange > 0 && planeRange < requiredRange {
				continue
			}
		}
		return idx
	}
	return -1
}

// advanceGroupCursor 记录目标类型下一次应优先检查的编组下标。
func (sa *ShipAircraft) advanceGroupCursor(targetObjType object.Type, groupIdx int) {
	if len(sa.Groups) == 0 {
		return
	}
	if sa.nextGroupCursor == nil {
		sa.nextGroupCursor = map[object.Type]int{}
	}
	sa.nextGroupCursor[targetObjType] = (groupIdx + 1) % len(sa.Groups)
}

// planeTypeOf 查询飞机模板的机型；未知飞机返回空值（不匹配任何白名单）。
func planeTypeOf(name string) PlaneType {
	if p, ok := PlaneMap[name]; ok {
		return p.Type
	}
	return ""
}

// planeRangeOf 返回运行时飞机模板的单程总航程；模板缺失时返回 0。
func planeRangeOf(name string) float64 {
	if p, ok := PlaneMap[name]; ok {
		return p.Range
	}
	return 0
}

// RequestLanding 为返航飞机分配稳定的回收槽位。
func (sa *ShipAircraft) RequestLanding(planeUID string) int {
	if slot, exists := sa.landingSlots[planeUID]; exists {
		return slot
	}

	usedSlots := make([]bool, len(sa.landingSlots)+1)
	for _, slot := range sa.landingSlots {
		if slot < len(usedSlots) {
			usedSlots[slot] = true
		}
	}
	slot := 0
	for usedSlots[slot] {
		slot++
	}
	if sa.landingSlots == nil {
		sa.landingSlots = map[string]int{}
	}
	sa.landingSlots[planeUID] = slot
	return slot
}

// CancelLanding 将飞机从回收槽位中移除。
func (sa *ShipAircraft) CancelLanding(planeUID string) {
	delete(sa.landingSlots, planeUID)
}

// CanTakeOffWithinRange 判断基地是否有库存中的机型可以到达指定距离。
// 这里不检查起飞点冷却，目标规划只需要知道机型能力。
func (sa *ShipAircraft) CanTakeOffWithinRange(targetObjType object.Type, requiredRange float64) bool {
	if sa == nil || sa.Disable || sa.deck == nil {
		return false
	}
	for _, point := range sa.deck.TakeoffPoints {
		if sa.matchingGroupIdx(point, targetObjType, requiredRange) >= 0 {
			return true
		}
	}
	return false
}

// TakeOff 起飞战机（不区分飞机种类，只看打击对象类型）。
// 多弹射器并行且独享：各起飞点独立计时冷却，同一时刻一个点只服务一架；
// 点按配置顺序选用，planeTypes 白名单实现「长起飞点专供轰炸机 / 鱼雷机」。
func (sa *ShipAircraft) TakeOff(base AircraftBase, targetObjType object.Type) *Plane {
	return sa.takeOff(base, targetObjType, 0)
}

// TakeOffWithinRange 起飞一架能够到达指定距离的飞机。
func (sa *ShipAircraft) TakeOffWithinRange(
	base AircraftBase,
	targetObjType object.Type,
	requiredRange float64,
) *Plane {
	return sa.takeOff(base, targetObjType, requiredRange)
}

// takeOff 是普通起飞和航程约束起飞的共同实现。
// requiredRange 大于 0 时，只考虑运行时航程足够到达目标的机型。
func (sa *ShipAircraft) takeOff(
	base AircraftBase,
	targetObjType object.Type,
	requiredRange float64,
) *Plane {
	// 禁止起飞只阻止新飞机离舰，不影响已经出击飞机继续作战或返航。
	if sa.Disable {
		return nil
	}
	if sa.deck == nil || len(sa.deck.TakeoffPoints) == 0 {
		return nil
	}
	sa.ensureTakeoffPointTimes()

	timeNow := time.Now().UnixMilli()
	for idx, point := range sa.deck.TakeoffPoints {
		// 判断起飞冷却，冷却中该弹射器被占用
		cooldown := sa.takeOffCooldown(point)
		if cooldown > 0 &&
			sa.takeoffPointTimes[idx]+int64(cooldown*1e3/gameSpeedMultiplier()) > timeNow {
			continue
		}
		groupIdx := sa.matchingGroupIdx(point, targetObjType, requiredRange)
		if groupIdx < 0 {
			continue
		}
		// 非指针需要通过索引修改
		sa.advanceGroupCursor(targetObjType, groupIdx)
		sa.Groups[groupIdx].CurCount--
		sa.takeoffPointTimes[idx] = timeNow
		sa.LatestTakeOffAt = timeNow
		plane := NewPlane(
			sa.Groups[groupIdx].Name,
			base.BasePos(), base.BaseRotation(),
			base.BaseUid(), base.BaseBelongPlayer(),
		)
		plane.StartTakeoff(base, point)
		return plane
	}
	return nil
}

// Recovery 回收飞机，返回是否恢复库存（飞机血量低于 15% 时无回收价值）。
func (sa *ShipAircraft) Recovery(plane *Plane) bool {
	sa.CancelLanding(plane.Uid)
	// 飞机血量低于 15% 时，没有回收价值
	if plane.CurHP/plane.TotalHP < 0.15 {
		return false
	}
	// 逐个组按名称匹配
	for idx, g := range sa.Groups {
		if g.Name != plane.Name {
			continue
		}
		if g.CurCount >= g.MaxCount {
			continue
		}
		// 添加库存数量（非指针需要通过索引修改）
		sa.Groups[idx].CurCount++
		return true
	}
	return false
}
