package computer

import (
	"math"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/state"
)

// searchMinAdvance 搜索推进点离舰队太近时改用最近的前线格，
// 免得全队已经站在未探索重心上还一直不动。
const searchMinAdvance = 6.0

// searchAdvancePoint 迷雾下看不见敌人时，舰队该往哪推：
// 最近的未过期接触优先，其次未探索海面的重心，重心就在脚下时退回最近的前线格。
func searchAdvancePoint(
	vision *state.FactionVision, from objPos.MapPos, now int64,
) (objPos.MapPos, bool) {
	if vision == nil {
		return objPos.MapPos{}, false
	}
	if pos, ok := nearestContact(vision, from, now); ok {
		return pos, true
	}
	if centroid, ok := vision.UnexploredCentroid(); ok && from.Distance(centroid) > searchMinAdvance {
		return centroid, true
	}
	return nearestFrontier(vision, from)
}

// nearestContact 最近一次看见、而且还没过期的敌舰位置。
func nearestContact(vision *state.FactionVision, origin objPos.MapPos, now int64) (objPos.MapPos, bool) {
	best := math.MaxFloat64
	var pos objPos.MapPos
	found := false
	for _, contact := range vision.Contacts {
		if contact.ExpireTick <= now {
			continue
		}
		target := objPos.NewR(contact.RX, contact.RY)
		if dist := origin.Distance(target); dist < best {
			best = dist
			pos = target
			found = true
		}
	}
	return pos, found
}

// nearestFrontier 离起点最近的「已探索且挨着未探索」的格子。
func nearestFrontier(vision *state.FactionVision, origin objPos.MapPos) (objPos.MapPos, bool) {
	limit := vision.Width + vision.Height
	for radius := 1; radius <= limit; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if absInt(dx) != radius && absInt(dy) != radius {
					continue
				}
				x, y := origin.MX+dx, origin.MY+dy
				if !vision.ExploredAt(x, y) || !hasUnexploredNeighbor(vision, x, y) {
					continue
				}
				return objPos.New(x, y), true
			}
		}
	}
	return objPos.MapPos{}, false
}

func hasUnexploredNeighbor(vision *state.FactionVision, x, y int) bool {
	for _, step := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, ny := x+step[0], y+step[1]
		if nx < 0 || ny < 0 || nx >= vision.Width || ny >= vision.Height {
			continue
		}
		if !vision.ExploredAt(nx, ny) {
			return true
		}
	}
	return false
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
