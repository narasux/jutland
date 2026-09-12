package manager

import (
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/targeting"
)

// markTargetingDirty 标记目标计划需要重算，实际计算会合并到下一次后台调度。
func (m *MissionManager) markTargetingDirty() {
	m.targetingDirty = true
}

// updateTargetPlanning 在低频模拟帧提交后台计划，并优先消费最新结果。
// 主线程只做快照复制和 O(1) 结果切换，不在每帧执行候选扫描或评分。
func (m *MissionManager) updateTargetPlanning() {
	if m.targetingResults == nil {
		m.targetingResults = make(chan targeting.Plan, 1)
	}
	if m.targetingCursors == nil {
		m.targetingCursors = map[string]map[object.Type]int{}
	}

	select {
	case plan := <-m.targetingResults:
		if plan.Revision >= m.targetingPlan.Revision {
			m.targetingPlan = plan
			m.clampTargetingCursors()
		}
		m.targetingBusy = false
	default:
	}

	if m.targetingBusy {
		return
	}
	intervalReady := m.targetingLastRequestTick == 0 ||
		m.simTick-m.targetingLastRequestTick >= targeting.ReplanIntervalTicks
	planExpired := m.targetingPlan.ComputedTick == 0 ||
		m.simTick-m.targetingPlan.ComputedTick >= targeting.PlanTTLTicks
	if (!m.targetingDirty && !planExpired) || !intervalReady {
		return
	}

	snapshot := m.buildTargetingSnapshot()
	m.targetingBusy = true
	m.targetingDirty = false
	m.targetingLastRequestTick = m.simTick
	results := m.targetingResults
	go func() {
		plan := targeting.BuildPlan(snapshot)
		select {
		case results <- plan:
		default:
		}
	}()
}

// buildTargetingSnapshot 复制主线程状态，生成后台规划所需的不可变输入。
func (m *MissionManager) buildTargetingSnapshot() targeting.Snapshot {
	snapshot := targeting.Snapshot{
		Tick:        m.simTick,
		Bases:       make([]targeting.Base, 0, len(m.state.Arena.Airfields)+len(m.state.Arena.Ships)),
		Planes:      make([]targeting.Plane, 0, len(m.state.Arena.Planes)),
		EnemyShips:  make([]targeting.EnemyShip, 0, len(m.state.Arena.Ships)),
		EnemyPlanes: make([]targeting.EnemyPlane, 0, len(m.state.Arena.Planes)),
	}

	for _, airfield := range m.state.Arena.Airfields {
		if !airfield.Aircraft.HasPlane {
			continue
		}
		groups := snapshotPlaneGroups(airfield.Aircraft.Groups)
		if airfield.Disabled {
			for idx := range groups {
				groups[idx].Available = 0
			}
		}
		snapshot.Bases = append(snapshot.Bases, targeting.Base{
			UID:    airfield.Uid,
			Player: airfield.BelongPlayer,
			Pos:    targeting.Point{X: airfield.Pos.RX, Y: airfield.Pos.RY},
			Groups: groups,
		})
	}
	for _, ship := range m.state.Arena.Ships {
		if ship.CurHP <= 0 || !ship.Aircraft.HasPlane {
			continue
		}
		snapshot.Bases = append(snapshot.Bases, targeting.Base{
			UID:    ship.Uid,
			Player: ship.BelongPlayer,
			Pos:    targeting.Point{X: ship.CurPos.RX, Y: ship.CurPos.RY},
			Groups: snapshotPlaneGroups(ship.Aircraft.Groups),
		})
	}

	for _, plane := range m.state.Arena.Planes {
		if plane.CurHP <= 0 {
			continue
		}
		if plane.IsCruising() {
			threat := float64(plane.CombatPower.Total) / 100
			if threat <= 0 {
				threat = 0.5
			}
			snapshot.EnemyPlanes = append(snapshot.EnemyPlanes, targeting.EnemyPlane{
				UID:      plane.Uid,
				Player:   plane.BelongPlayer,
				Pos:      targeting.Point{X: plane.CurPos.RX, Y: plane.CurPos.RY},
				Threat:   threat,
				Airborne: true,
			})
		}
		targetType := snapshotPlaneTargetType(plane)
		if targetType == object.TypeNone {
			continue
		}
		snapshot.Planes = append(snapshot.Planes, targeting.Plane{
			UID:        plane.Uid,
			BaseUID:    plane.BelongShip,
			Player:     plane.BelongPlayer,
			TargetType: targetType,
			TargetUID:  plane.CurAttackTarget,
			Pos:        targeting.Point{X: plane.CurPos.RX, Y: plane.CurPos.RY},
		})
	}
	for _, ship := range m.state.Arena.Ships {
		if ship.CurHP <= 0 {
			continue
		}
		snapshot.EnemyShips = append(snapshot.EnemyShips, targeting.EnemyShip{
			UID:    ship.Uid,
			Player: ship.BelongPlayer,
			Pos:    targeting.Point{X: ship.CurPos.RX, Y: ship.CurPos.RY},
			Value:  shipTargetingValue(ship),
		})
	}
	return snapshot
}

// snapshotPlaneTargetType 返回飞机运行时使用的攻击目标类型。
func snapshotPlaneTargetType(plane *objUnit.Plane) object.Type {
	if plane.Name != "" {
		if _, ok := objUnit.PlaneMap[plane.Name]; ok {
			return plane.AttackObjType()
		}
	}
	switch plane.Type {
	case objUnit.PlaneTypeFighter:
		return object.TypePlane
	case objUnit.PlaneTypeDiveBomber, objUnit.PlaneTypeLevelBomber, objUnit.PlaneTypeTorpedoBomber:
		return object.TypeShip
	default:
		return object.TypeNone
	}
}

// snapshotPlaneGroups 把机库编组转换成规划器使用的机型范围和库存信息。
func snapshotPlaneGroups(groups []objUnit.PlaneGroup) []targeting.Group {
	result := make([]targeting.Group, 0, len(groups))
	for _, group := range groups {
		if group.TargetType == object.TypeNone {
			continue
		}
		planeRange := 0.0
		if plane, ok := objUnit.PlaneMap[group.Name]; ok {
			planeRange = plane.Range
		}
		result = append(result, targeting.Group{
			Name:       group.Name,
			TargetType: group.TargetType,
			Available:  group.CurCount,
			Range:      planeRange,
		})
	}
	return result
}

// shipTargetingValue 按舰种返回对舰目标的基础价值。
func shipTargetingValue(ship *objUnit.BattleShip) float64 {
	switch ship.Type {
	case objUnit.ShipTypeAircraftCarrier:
		return 1
	case objUnit.ShipTypeBattleShip:
		return 0.85
	case objUnit.ShipTypeCruiser:
		return 0.60
	case objUnit.ShipTypeDestroyer, objUnit.ShipTypeFrigate:
		return 0.35
	case objUnit.ShipTypeTorpedoBoat:
		return 0.20
	default:
		return 0.10
	}
}

// peekTargetUID 查看基地游标处第一个有效目标，但不推进游标。
func (m *MissionManager) peekTargetUID(baseUID string, targetType object.Type) (string, bool) {
	idx, uid, ok := m.findTargetAtCursor(baseUID, targetType)
	if ok {
		m.setTargetCursor(baseUID, targetType, idx)
	}
	return uid, ok
}

// peekTargetUIDWhere 查看游标后第一个满足条件的有效目标，但不推进游标。
func (m *MissionManager) peekTargetUIDWhere(
	baseUID string,
	targetType object.Type,
	accept func(string) bool,
) (string, bool) {
	idx, uid, ok := m.findTargetAtCursorWhere(baseUID, targetType, accept)
	if ok {
		m.setTargetCursor(baseUID, targetType, idx)
	}
	return uid, ok
}

// nextTargetUID 取出基地游标处的有效目标，并推进游标。
func (m *MissionManager) nextTargetUID(baseUID string, targetType object.Type) (string, bool) {
	idx, uid, ok := m.findTargetAtCursor(baseUID, targetType)
	if !ok {
		return "", false
	}
	m.advanceTargetCursor(baseUID, targetType, idx)
	return uid, true
}

// nextTargetUIDForPlane 为已起飞飞机选择当前 RemainRange 足够到达的下一个目标。
func (m *MissionManager) nextTargetUIDForPlane(plane *objUnit.Plane) (string, bool) {
	targetType := plane.AttackObjType()
	idx, uid, ok := m.findTargetAtCursorWhere(
		plane.BelongShip,
		targetType,
		func(uid string) bool { return m.targetReachableByPlane(plane, uid, targetType) },
	)
	if !ok {
		return "", false
	}
	m.advanceTargetCursor(plane.BelongShip, targetType, idx)
	return uid, true
}

// findTargetAtCursor 从基地游标开始查找第一个仍然存在的目标。
func (m *MissionManager) findTargetAtCursor(baseUID string, targetType object.Type) (int, string, bool) {
	return m.findTargetAtCursorWhere(baseUID, targetType, nil)
}

// findTargetAtCursorWhere 从基地游标开始查找满足 accept 条件的第一个有效目标。
// 返回其在队列中的下标；扫描完整圈仍无结果时会将游标归零并标记计划失效。
func (m *MissionManager) findTargetAtCursorWhere(
	baseUID string,
	targetType object.Type,
	accept func(string) bool,
) (int, string, bool) {
	if m.targetingPlan.BaseQueues == nil {
		m.markTargetingDirty()
		return 0, "", false
	}
	queue := m.targetingPlan.BaseQueues[baseUID][targetType]
	if len(queue) == 0 {
		m.markTargetingDirty()
		return 0, "", false
	}

	cursor := m.targetCursor(baseUID, targetType)
	for offset := 0; offset < len(queue); offset++ {
		idx := (cursor + offset) % len(queue)
		uid := queue[idx].UID
		if m.targetExists(uid, targetType) && (accept == nil || accept(uid)) {
			return idx, queue[idx].UID, true
		}
	}
	m.setTargetCursor(baseUID, targetType, 0)
	m.markTargetingDirty()
	return 0, "", false
}

// targetReachableByPlane 判断飞机当前剩余航程是否足够到达目标。
func (m *MissionManager) targetReachableByPlane(plane *objUnit.Plane, uid string, targetType object.Type) bool {
	if plane == nil || plane.RemainRange <= 0 {
		return false
	}
	targetDistance, ok := m.targetDistanceFrom(plane.CurPos, uid, targetType)
	return ok && targetDistance <= plane.RemainRange
}

// targetDistanceFrom 返回指定位置到目标当前位置的距离，并校验目标是否存活。
func (m *MissionManager) targetDistanceFrom(
	from objPos.MapPos,
	uid string,
	targetType object.Type,
) (float64, bool) {
	var targetPos objPos.MapPos
	switch targetType {
	case object.TypePlane:
		target := m.state.Arena.Planes[uid]
		if target == nil || target.CurHP <= 0 {
			return 0, false
		}
		targetPos = target.CurPos
	case object.TypeShip:
		target := m.state.Arena.Ships[uid]
		if target == nil || target.CurHP <= 0 {
			return 0, false
		}
		targetPos = target.CurPos
	default:
		return 0, false
	}
	return from.Distance(targetPos), true
}

// targetExists 判断目标是否仍然存在于任务状态中。
func (m *MissionManager) targetExists(uid string, targetType object.Type) bool {
	switch targetType {
	case object.TypePlane:
		target := m.state.Arena.Planes[uid]
		return target != nil && target.CurHP > 0
	case object.TypeShip:
		target := m.state.Arena.Ships[uid]
		return target != nil && target.CurHP > 0
	default:
		return false
	}
}

// targetCursor 返回基地在指定目标类型上的当前队列游标。
func (m *MissionManager) targetCursor(baseUID string, targetType object.Type) int {
	if m.targetingCursors == nil {
		m.targetingCursors = map[string]map[object.Type]int{}
	}
	if m.targetingCursors[baseUID] == nil {
		m.targetingCursors[baseUID] = map[object.Type]int{}
	}
	return m.targetingCursors[baseUID][targetType]
}

// setTargetCursor 设置基地在指定目标类型上的队列游标。
func (m *MissionManager) setTargetCursor(baseUID string, targetType object.Type, cursor int) {
	_ = m.targetCursor(baseUID, targetType)
	m.targetingCursors[baseUID][targetType] = cursor
}

// advanceTargetCursor 将基地游标推进到已消费目标的下一个位置并标记计划需要重算。
func (m *MissionManager) advanceTargetCursor(baseUID string, targetType object.Type, idx int) {
	queue := m.targetingPlan.BaseQueues[baseUID][targetType]
	if len(queue) == 0 {
		m.setTargetCursor(baseUID, targetType, 0)
		return
	}
	m.setTargetCursor(baseUID, targetType, (idx+1)%len(queue))
	m.markTargetingDirty()
}

// clampTargetingCursors 在新计划到达后把旧游标限制到新队列范围内。
func (m *MissionManager) clampTargetingCursors() {
	for baseUID, byType := range m.targetingCursors {
		for targetType, cursor := range byType {
			queue := m.targetingPlan.BaseQueues[baseUID][targetType]
			if len(queue) == 0 {
				byType[targetType] = 0
				continue
			}
			byType[targetType] = cursor % len(queue)
		}
	}
}

// takeoffTargetTypes 返回基地当前库存中可出击的目标类型，顺序按机库编组排列。
func takeoffTargetTypes(aircraft *objUnit.ShipAircraft) []object.Type {
	if aircraft == nil {
		return nil
	}
	result := make([]object.Type, 0, len(aircraft.Groups))
	seen := map[object.Type]bool{}
	for _, group := range aircraft.Groups {
		if group.CurCount <= 0 || group.TargetType == object.TypeNone || seen[group.TargetType] {
			continue
		}
		seen[group.TargetType] = true
		result = append(result, group.TargetType)
	}
	return result
}

// takeOffFromBase 从基地派出一架飞机并领取基地队列目标。
// 目标类型按批次轮转；具体机型必须满足目标距离，起飞后才推进目标游标。
func (m *MissionManager) takeOffFromBase(base objUnit.AircraftBase) (*objUnit.Plane, object.Type, string, bool) {
	if base == nil {
		return nil, object.TypeNone, "", false
	}
	targetTypes := takeoffTargetTypes(base.BaseAircraft())
	if len(targetTypes) == 0 {
		return nil, object.TypeNone, "", false
	}
	baseUID := base.BaseUid()
	if m.takeoffTypeCursors == nil {
		m.takeoffTypeCursors = map[string]int{}
	}
	start := m.takeoffTypeCursors[baseUID] % len(targetTypes)
	for offset := 0; offset < len(targetTypes); offset++ {
		idx := (start + offset) % len(targetTypes)
		targetType := targetTypes[idx]
		aircraft := base.BaseAircraft()
		targetRange := 0.0
		targetUID, ok := m.peekTargetUIDWhere(baseUID, targetType, func(uid string) bool {
			distance, exists := m.targetDistanceFrom(base.BasePos(), uid, targetType)
			if !exists || !aircraft.CanTakeOffWithinRange(targetType, distance) {
				return false
			}
			targetRange = distance
			return true
		})
		if !ok {
			continue
		}
		plane := aircraft.TakeOffWithinRange(base, targetType, targetRange)
		if plane == nil {
			continue
		}
		m.advanceTargetCursor(baseUID, targetType, m.targetCursor(baseUID, targetType))
		m.takeoffTypeCursors[baseUID] = (idx + 1) % len(targetTypes)
		return plane, targetType, targetUID, true
	}
	return nil, object.TypeNone, "", false
}
