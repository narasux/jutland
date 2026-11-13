package manager

import (
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// sonarQueryRadius 探测查桶半径：覆盖全部舰载声呐与水听器的最大值。
func sonarQueryRadius() float64 {
	return max(objUnit.ShipSonarRangeEscort, objUnit.HydrophoneRange)
}

// updateSubmarineDetection 是独立于迷雾的声呐探测 pass。
// 只遍历水下潜艇，用邻域桶找敌方舰船/飞机；命中的写回 8 秒记忆。
// 必须在 updateVision 之前执行，且先按这一拍的位置重建桶。
func (m *MissionManager) updateSubmarineDetection() {
	hasSubmarine := false
	for _, ship := range m.state.Arena.Ships {
		if ship.Type == objUnit.ShipTypeSubmarine && ship.CurHP > 0 {
			hasSubmarine = true
			break
		}
	}
	if !hasSubmarine {
		return
	}
	m.rebuildCombatBuckets()
	now := m.state.Core.SimTick
	memory := objUnit.ReloadTicks(objUnit.SubmarineDetectionMemorySeconds)

	for _, sub := range m.state.Arena.Ships {
		if sub.Type != objUnit.ShipTypeSubmarine || sub.CurHP <= 0 {
			continue
		}
		if !sub.Submerged() {
			// 上浮 / 过渡：清空记忆，回到水面可见规则。
			sub.ClearDetection()
			continue
		}
		// 舰船声呐与水听器
		m.combatBuckets.eachShip(
			sub.CurPos.RX, sub.CurPos.RY, sonarQueryRadius(),
			func(detector *objUnit.BattleShip) bool {
				if detector.BelongPlayer == sub.BelongPlayer || detector.CurHP <= 0 {
					return false
				}
				radius := detector.SonarRange()
				if radius > 0 && sub.CurPos.Distance(detector.CurPos) <= radius {
					sub.MarkDetected(detector.BelongPlayer, now+memory, sub.CurPos)
				}
				return false
			},
		)
		// 飞机目视：只算仍在空中、能提供视野的机型
		m.combatBuckets.eachPlane(
			sub.CurPos.RX, sub.CurPos.RY, objUnit.PlaneSonarRange,
			func(plane *objUnit.Plane) bool {
				if plane.BelongPlayer == sub.BelongPlayer || plane.CurHP <= 0 || !plane.ProvidesSight() {
					return false
				}
				if sub.CurPos.Distance(plane.CurPos) <= objUnit.PlaneSonarRange {
					sub.MarkDetected(plane.BelongPlayer, now+memory, sub.CurPos)
				}
				return false
			},
		)
	}
}
