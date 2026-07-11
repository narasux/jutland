package unit

import (
	"time"

	"github.com/narasux/jutland/pkg/mission/object"
)

// ShipAircraft 战舰上的飞机，也能算是武器吧 :D
type ShipAircraft struct {
	// TakeOffTime 起飞耗时（单位：秒）
	TakeOffTime float64 `json:"takeOffTime"`
	// Groups 战机分组
	Groups []PlaneGroup `json:"groups"`

	// 是否禁用舰载机
	Disable bool
	// 是否拥有舰载机
	HasPlane bool
	// 最近起飞时间（毫秒时间戳)
	LatestTakeOffAt int64

	// 以下字段只服务于局内回收调度，不参与配置序列化。
	landingSlots map[string]int
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

// TakeOff 起飞战机（不区分飞机种类，只看打击对象类型）
func (sa *ShipAircraft) TakeOff(ship *BattleShip, targetObjType object.Type) *Plane {
	// 判断起飞冷却，冷却中不允许起飞
	if sa.LatestTakeOffAt+int64(sa.TakeOffTime*1e3) > time.Now().UnixMilli() {
		return nil
	}

	for idx, g := range sa.Groups {
		if g.TargetType != targetObjType {
			continue
		}
		if g.CurCount <= 0 {
			continue
		}
		// 非指针需要通过索引修改
		sa.Groups[idx].CurCount--
		sa.LatestTakeOffAt = time.Now().UnixMilli()
		plane := NewPlane(g.Name, ship.CurPos, ship.CurRotation, ship.Uid, ship.BelongPlayer)
		plane.StartTakeoff(ship)
		return plane
	}
	return nil
}

// Recovery 回收飞机
func (sa *ShipAircraft) Recovery(plane *Plane) {
	sa.CancelLanding(plane.Uid)
	// 飞机血量低于 15% 时，没有回收价值
	if plane.CurHP/plane.TotalHP < 0.15 {
		return
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
		return
	}
}
