package state

import (
	"slices"
	"strings"

	"github.com/samber/lo"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

func shipUIDOrder(a, b *objUnit.BattleShip) int {
	return strings.Compare(a.Uid, b.Uid)
}

func planeUIDOrder(a, b *objUnit.Plane) int {
	return strings.Compare(a.Uid, b.Uid)
}

// PutShip 放入或替换一艘舰，并维持按 UID 排序的绘制名单。
func (a *MissionArenaState) PutShip(ship *objUnit.BattleShip) {
	if ship == nil {
		return
	}
	if a.Ships == nil {
		a.Ships = map[string]*objUnit.BattleShip{}
	}
	if _, exists := a.Ships[ship.Uid]; exists {
		a.Ships[ship.Uid] = ship
		for i, existing := range a.orderedShips {
			if existing != nil && existing.Uid == ship.Uid {
				a.orderedShips[i] = ship
				return
			}
		}
	}
	a.Ships[ship.Uid] = ship
	if len(a.orderedShips) != len(a.Ships)-1 {
		a.orderedShips = nil
		return
	}
	index, _ := slices.BinarySearchFunc(a.orderedShips, ship, shipUIDOrder)
	a.orderedShips = slices.Insert(a.orderedShips, index, ship)
}

// RemoveShip 从战场移除一艘舰。
func (a *MissionArenaState) RemoveShip(uid string) {
	delete(a.Ships, uid)
	index, found := slices.BinarySearchFunc(a.orderedShips, &objUnit.BattleShip{Uid: uid}, shipUIDOrder)
	if !found {
		a.orderedShips = nil
		return
	}
	a.orderedShips = slices.Delete(a.orderedShips, index, index+1)
}

// OrderedShips 返回按 UID 排好的舰船。名单和字典不一致时才重排。
func (a *MissionArenaState) OrderedShips() []*objUnit.BattleShip {
	if len(a.orderedShips) != len(a.Ships) {
		a.orderedShips = lo.Values(a.Ships)
		slices.SortFunc(a.orderedShips, shipUIDOrder)
	}
	return a.orderedShips
}

// PutPlane 放入或替换一架飞机，并维持按 UID 排序的绘制名单。
func (a *MissionArenaState) PutPlane(plane *objUnit.Plane) {
	if plane == nil {
		return
	}
	if a.Planes == nil {
		a.Planes = map[string]*objUnit.Plane{}
	}
	if _, exists := a.Planes[plane.Uid]; exists {
		a.Planes[plane.Uid] = plane
		for i, existing := range a.orderedPlanes {
			if existing != nil && existing.Uid == plane.Uid {
				a.orderedPlanes[i] = plane
				return
			}
		}
	}
	a.Planes[plane.Uid] = plane
	if len(a.orderedPlanes) != len(a.Planes)-1 {
		a.orderedPlanes = nil
		return
	}
	index, _ := slices.BinarySearchFunc(a.orderedPlanes, plane, planeUIDOrder)
	a.orderedPlanes = slices.Insert(a.orderedPlanes, index, plane)
}

// RemovePlane 从战场移除一架飞机。
func (a *MissionArenaState) RemovePlane(uid string) {
	delete(a.Planes, uid)
	index, found := slices.BinarySearchFunc(a.orderedPlanes, &objUnit.Plane{Uid: uid}, planeUIDOrder)
	if !found {
		a.orderedPlanes = nil
		return
	}
	a.orderedPlanes = slices.Delete(a.orderedPlanes, index, index+1)
}

// OrderedPlanes 返回按 UID 排好的飞机。名单和字典不一致时才重排。
func (a *MissionArenaState) OrderedPlanes() []*objUnit.Plane {
	if len(a.orderedPlanes) != len(a.Planes) {
		a.orderedPlanes = lo.Values(a.Planes)
		slices.SortFunc(a.orderedPlanes, planeUIDOrder)
	}
	return a.orderedPlanes
}
