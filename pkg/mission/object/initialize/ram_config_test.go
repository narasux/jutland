package initialize

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestSurfaceCombatShipsHaveExplicitRam 验证存量水面战斗舰在 ships.json5 中显式配置撞击数据。
// 场景：撞击系统上线后，战列 / 巡洋 / 驱逐 / 护卫舰曾依赖初始化阶段的按舰种推导，
// 配置里看不到 ram，既无法逐舰调参，也不进入静态战力。
// 预期：每艘上述舰种的 Ram.Damage > 0 且 Ram.Derived = false；航母 / 货轮 / 维修 /
// 医疗 / 潜艇 / 鱼雷艇不参与该约束。失败说明有新舰漏配，或又退回了隐式推导。
// 覆盖：遍历 AllShipNames 与 ShipMap，按舰种筛出目标舰并逐艘断言，样本量下限 100。
func TestSurfaceCombatShipsHaveExplicitRam(t *testing.T) {
	surfaceCombat := map[objUnit.ShipType]bool{
		objUnit.ShipTypeBattleShip: true,
		objUnit.ShipTypeCruiser:    true,
		objUnit.ShipTypeDestroyer:  true,
		objUnit.ShipTypeFrigate:    true,
	}
	checked := 0
	for _, name := range objUnit.AllShipNames {
		ship := objUnit.ShipMap[name]
		if ship == nil || !surfaceCombat[ship.Type] {
			continue
		}
		checked++
		if !ship.Ram.Enabled() || ship.Ram.Derived {
			t.Errorf("ship %q must have explicit ram config: %+v", name, ship.Ram)
		}
	}
	if checked < 100 {
		t.Fatalf("checked %d surface combat ships, want at least 100", checked)
	}
}
