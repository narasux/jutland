package warfog

import (
	"math"

	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// VisibilityCalculator 视野计算器
type VisibilityCalculator struct{}

// NewVisibilityCalculator 创建视野计算器
func NewVisibilityCalculator() *VisibilityCalculator {
	return &VisibilityCalculator{}
}

// UpdateVisibility 更新视野（基于所有友方单位）
func (vc *VisibilityCalculator) UpdateVisibility(
	fog *FogOfWar,
	ships map[string]*objUnit.BattleShip,
	player faction.Player,
) {
	if !fog.IsDirty() {
		return
	}

	// 清空当前视野
	vc.clearVisibility(fog)

	// 合并所有友方单位的视野
	for _, ship := range ships {
		if ship.BelongPlayer == player && ship.CurHP > 0 {
			vc.addUnitVisibility(fog, ship)
		}
	}

	// 清除脏标记
	fog.dirty = false
}

// clearVisibility 清空当前视野
func (vc *VisibilityCalculator) clearVisibility(fog *FogOfWar) {
	for x := 0; x < fog.MapWidth; x++ {
		for y := 0; y < fog.MapHeight; y++ {
			fog.VisibleGrid[x][y] = false
		}
	}
}

// addUnitVisibility 添加单个单位的视野
func (vc *VisibilityCalculator) addUnitVisibility(fog *FogOfWar, ship *objUnit.BattleShip) {
	center := ship.CurPos
	sightRange := ship.GetSightRange()

	// 计算视野半径（向上取整）
	radius := int(math.Ceil(sightRange))
	cx, cy := int(center.RX), int(center.RY)

	// 遍历圆形区域内的所有格子
	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			x, y := cx+dx, cy+dy

			// 边界检查
			if !fog.isValidCell(x, y) {
				continue
			}

			// 使用距离平方判断（性能优化）
			distSq := float64(dx*dx + dy*dy)
			if distSq <= sightRange*sightRange {
				fog.VisibleGrid[x][y] = true
				fog.ExploredGrid[x][y] = true
			}
		}
	}
}
