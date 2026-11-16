package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// 本关用到的模板类型（不依赖 ShipMap，避免测试包再去加载整套资源配置）
var submarineTestShipTypes = map[string]string{
	"u_331":        "submarine",
	"i_168":        "submarine",
	"uss_nautilus": "submarine",
	"gurkha":       "destroyer",
	"porter":       "destroyer",
	"essex":        "aircraft_carrier",
}

// TestSubmarineTestMissionZones 验证 SubmarineTest 的 11 艘舰按四区配比布阵且互不压陆/串区。
// 场景：Get("SubmarineTest") 后以 RX<125、RY<=220 把 midway 切为 SW/NW/NE/SE 统计。
// 预期：SW 2 潜；NW 1 潜 2 驱；NE 1 潜 2 驱；SE 1 航母 1 驱 1 潜；舰均不在陆地。
// 覆盖：逐舰校验模板与陆地格，counts 精确相等，跨区任意两舰距 >20 格。
func TestSubmarineTestMissionZones(t *testing.T) {
	md := Get("SubmarineTest")
	require.Equal(t, "midway", md.MapCfg.Name)
	require.Len(t, md.InitShips, 11)

	zoneOf := func(pos objPos.MapPos) string {
		west := pos.RX < 125
		north := pos.RY <= 220
		switch {
		case west && north:
			return "NW"
		case west && !north:
			return "SW"
		case !west && north:
			return "NE"
		default:
			return "SE"
		}
	}

	counts := map[string]map[string]int{}
	positions := map[string][]objPos.MapPos{}
	for _, init := range md.InitShips {
		require.Falsef(
			t, md.MapCfg.Map.IsLand(init.Pos.MX, init.Pos.MY),
			"ship %s at %v is on land", init.ShipName, init.Pos,
		)
		shipType, ok := submarineTestShipTypes[init.ShipName]
		require.True(t, ok, "unexpected ship template %s", init.ShipName)
		zone := zoneOf(init.Pos)
		if counts[zone] == nil {
			counts[zone] = map[string]int{}
		}
		counts[zone][shipType]++
		positions[zone] = append(positions[zone], init.Pos)
	}

	require.Equal(t, map[string]int{"submarine": 2}, counts["SW"], "西南应是潜对潜")
	require.Equal(t, map[string]int{"submarine": 1, "destroyer": 2}, counts["NW"], "西北应是驱逐舰猎潜")
	require.Equal(t, map[string]int{"submarine": 1, "destroyer": 2}, counts["NE"], "东北应是敌舰猎潜")
	require.Equal(t, map[string]int{
		"aircraft_carrier": 1, "destroyer": 1, "submarine": 1,
	}, counts["SE"], "东南应是航空反潜")

	// 四个试验区必须彼此拉开距离，避免跨区交火
	zones := []string{"SW", "NW", "NE", "SE"}
	for i := 0; i < len(zones); i++ {
		for j := i + 1; j < len(zones); j++ {
			for _, left := range positions[zones[i]] {
				for _, right := range positions[zones[j]] {
					require.Greaterf(
						t, left.Distance(right), 20.0,
						"zone %s and %s are too close: %v vs %v", zones[i], zones[j], left, right,
					)
				}
			}
		}
	}
}
