package manager

import (
	"math"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

const combatCellSize = 8

// 舰体对角线的一半不超过这个值（地图格）。命中查询按弹速再加上它，避免漏掉桶边缘的船。
const maxHullHitRadius = 3

// combatIndex 是只活一拍的均匀格子网。桶键由格子坐标拼成，不给每艘船单独分配对象。
type combatIndex struct {
	ships  map[uint64][]*objUnit.BattleShip
	planes map[uint64][]*objUnit.Plane
}

func combatBucket(v float64) int {
	return int(math.Floor(v / combatCellSize))
}

func combatCellKey(x, y int) uint64 {
	return uint64(uint32(int32(x)))<<32 | uint64(uint32(int32(y)))
}

func bucketRings(radius float64) int {
	if radius <= 0 {
		return 0
	}
	return int(math.Ceil(radius / combatCellSize))
}

func (idx *combatIndex) rebuild(ships map[string]*objUnit.BattleShip, planes map[string]*objUnit.Plane) {
	if idx.ships == nil {
		idx.ships = map[uint64][]*objUnit.BattleShip{}
		idx.planes = map[uint64][]*objUnit.Plane{}
	}
	for key, list := range idx.ships {
		if len(list) > 0 {
			idx.ships[key] = list[:0]
		}
	}
	for key, list := range idx.planes {
		if len(list) > 0 {
			idx.planes[key] = list[:0]
		}
	}
	for _, ship := range ships {
		key := combatCellKey(combatBucket(ship.CurPos.RX), combatBucket(ship.CurPos.RY))
		idx.ships[key] = append(idx.ships[key], ship)
	}
	for _, plane := range planes {
		key := combatCellKey(combatBucket(plane.CurPos.RX), combatBucket(plane.CurPos.RY))
		idx.planes[key] = append(idx.planes[key], plane)
	}
}

// eachShip 访问可能落在半径内的船。回调返回 true 时停止。
func (idx *combatIndex) eachShip(x, y, radius float64, fn func(*objUnit.BattleShip) bool) {
	eachBucket(idx.ships, x, y, radius, fn)
}

// eachPlane 访问可能落在半径内的飞机。回调返回 true 时停止。
func (idx *combatIndex) eachPlane(x, y, radius float64, fn func(*objUnit.Plane) bool) {
	eachBucket(idx.planes, x, y, radius, fn)
}

func eachBucket[T any](buckets map[uint64][]T, x, y, radius float64, fn func(T) bool) {
	if radius <= 0 || buckets == nil {
		return
	}
	rings := bucketRings(radius)
	bx, by := combatBucket(x), combatBucket(y)
	for iy := by - rings; iy <= by+rings; iy++ {
		for ix := bx - rings; ix <= bx+rings; ix++ {
			for _, item := range buckets[combatCellKey(ix, iy)] {
				if fn(item) {
					return
				}
			}
		}
	}
}
