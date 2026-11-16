package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestUSCapitalShipsCarryFloatplaneScouts 验证美国主力舰能弹射水上侦察机，保障「从战舰起飞飞机侦察」。
// 场景：开启迷雾设置后加载 PearlHarbor1941（SideP1），遍历 HumanAlpha 且 HasPlane 的舰船，要求 SearchStock≥searchMinStock=2。
// 预期：每艘可派机舰船的 TakeOffSearch 返回非 nil，机型 Type=PlaneTypeScout，SightRange≥SightRangeScout=36；checked>0。
// 覆盖：ShipAircraft.SearchStock/TakeOffSearch 的保留甲板库存规则（起飞后还要留一架）与侦察机视距。
func TestUSCapitalShipsCarryFloatplaneScouts(t *testing.T) {
	useFogSettings(t)
	m := New("PearlHarbor1941", faction.SideP1)

	checked := 0
	for _, ship := range m.state.Arena.Ships {
		if ship.BelongPlayer != faction.HumanAlpha || !ship.Aircraft.HasPlane {
			continue
		}
		checked++
		if ship.Aircraft.SearchStock() < 2 {
			t.Errorf("%s 的搜索库存只有 %d，派不出侦察机", ship.Name, ship.Aircraft.SearchStock())
			continue
		}
		plane := ship.Aircraft.TakeOffSearch(ship)
		if plane == nil {
			t.Errorf("%s 起飞搜索机失败", ship.Name)
			continue
		}
		if plane.Type != objUnit.PlaneTypeScout {
			t.Errorf("%s 派出的 %s 不是侦察机，type=%s", ship.Name, plane.Name, plane.Type)
		}
		if plane.SightRange < objUnit.SightRangeScout {
			t.Errorf("%s 的 %s 视距只有 %.0f", ship.Name, plane.Name, plane.SightRange)
		}
	}
	if checked == 0 {
		t.Fatal("珍珠港关卡里没有可派侦察机的美国主力舰")
	}
}

// TestAmidshipsCatapultsLaunchAbeam 验证两烟囱之间的弹射器向舷外弹射，而不是朝舰尾穿过烟囱。
// 场景：manchester、bolzano、muzio_attendolo 的弹射器都夹在前后烟囱之间，180° 滑跑会撞上后烟囱。
// 预期：三个甲板模板各恰有 1 个起飞点，LaunchAngle 为 90°（右舷正横）；失败意味着又回到朝舰尾的配置。
// 覆盖：从 initialize 后的 DeckMap 读取三个模板，逐个断言模板存在、起飞点数量与 90° 航向。
func TestAmidshipsCatapultsLaunchAbeam(t *testing.T) {
	for _, name := range []string{
		"UKCatapult_manchester",
		"ITCatapult_bolzano",
		"ITCatapult_muzio_attendolo",
	} {
		deck, ok := objUnit.DeckMap[name]
		require.Truef(t, ok, "缺少 %s 甲板模板", name)
		require.Lenf(t, deck.TakeoffPoints, 1, name)
		require.Equalf(t, float64(90), deck.TakeoffPoints[0].LaunchAngle, name)
	}
}

// TestUSFloatplaneScoutsAreConfigured 验证三型美军水上侦察机都在册且为 scout 型，避免舰船编组引用到打不出搜索的机型。
// 场景：遍历 OS2U、SOC、SC-1，从 objUnit.PlaneMap 取模板。
// 预期：三者都存在，Type 均为 PlaneTypeScout，SightRange 均等于 SightRangeScout=36。
// 覆盖：planes.json5 初始化后的机型类型与视距解析（ResolvePlaneSight）。
func TestUSFloatplaneScoutsAreConfigured(t *testing.T) {
	for _, name := range []string{"OS2U", "SOC", "SC-1"} {
		plane, ok := objUnit.PlaneMap[name]
		if !ok {
			t.Errorf("缺少飞机 %s", name)
			continue
		}
		if plane.Type != objUnit.PlaneTypeScout {
			t.Errorf("%s 的类型是 %s，应为 scout", name, plane.Type)
		}
		if plane.SightRange != objUnit.SightRangeScout {
			t.Errorf("%s 的视距 %v, want %v", name, plane.SightRange, objUnit.SightRangeScout)
		}
	}
}

// TestUSFloatplaneAssignments 验证美国主力舰水侦配备表：按状态年份对机型，并逐舰核对弹射器甲板。
// 场景：按 want 表检查 15 艘舰的 Groups[0].Name 与 Aircraft.Deck；再核对 USCatapult_<舰名> 模板、起飞点数量与单台朝向左舷 -60；最后检查 atlanta、charlotte 不应有水侦。
// 预期：1930 年代 new_mexico/new_york/st_louis 用 SOC，1940-1944 arizona/baltimore/alaska/west_virginia/arkansas 用 OS2U，1945 起 lowa/montana/brooklyn/oregon_city 及战后 puerto_rico/des_moines/fargo 用 SC-1；new_york/west_virginia/fargo/arkansas 为 1 个起飞点，其余为 2 个；单台弹射器 LaunchAngle=-60；atlanta/charlotte 无编组。
// 覆盖：ShipMap 的 Aircraft.Groups/Deck 配置、DeckMap 模板存在性与弹射点数量/朝向一致性。
func TestUSFloatplaneAssignments(t *testing.T) {
	want := map[string]string{
		// 1930 年代状态：还没有翠鸟，战列舰与巡洋舰都用海鸥
		"new_mexico": "SOC",
		"new_york":   "SOC",
		"st_louis":   "SOC",
		// 1940-1944：翠鸟
		"arizona":       "OS2U",
		"baltimore":     "OS2U",
		"alaska":        "OS2U",
		"west_virginia": "OS2U",
		"arkansas":      "OS2U",
		// 1945 起：海鹰
		"lowa":        "SC-1",
		"montana":     "SC-1",
		"brooklyn":    "SC-1",
		"oregon_city": "SC-1",
		// 战后三艘同样带弹射器：阿拉斯加级、得梅因级、法戈级
		"puerto_rico": "SC-1",
		"des_moines":  "SC-1",
		"fargo":       "SC-1",
	}
	for name, planeName := range want {
		ship, ok := objUnit.ShipMap[name]
		if !ok {
			t.Errorf("缺少舰船 %s", name)
			continue
		}
		got := ""
		if len(ship.Aircraft.Groups) > 0 {
			got = ship.Aircraft.Groups[0].Name
		}
		if got != planeName {
			t.Errorf("%s 的水侦是 %s, want %s", name, got, planeName)
		}
		if ship.Aircraft.Deck == "" {
			t.Errorf("%s 没有配弹射器甲板", name)
		}
	}

	// 弹射器位置是按舰船俯视图逐舰标注出来的，每艘一个 USCatapult_<舰名> 模板；
	// 台数与朝向也要对得上：单台统一向左舷（launchAngle -60）。
	singlePort := map[string]int{
		"new_york": 1, "nevada": 1, "maryland": 1, "west_virginia": 1, "fargo": 1, "arkansas": 1,
	}
	for name := range want {
		ship := objUnit.ShipMap[name]
		wantDeck := "USCatapult_" + name
		if ship.Aircraft.Deck != wantDeck {
			t.Errorf("%s 的甲板是 %s, want %s", name, ship.Aircraft.Deck, wantDeck)
			continue
		}
		deck, ok := objUnit.DeckMap[wantDeck]
		if !ok {
			t.Errorf("缺少甲板模板 %s", wantDeck)
			continue
		}
		wantPoints := 2
		if n, isSingle := singlePort[name]; isSingle {
			wantPoints = n
		}
		if len(deck.TakeoffPoints) != wantPoints {
			t.Errorf("%s 有 %d 个弹射点, want %d", name, len(deck.TakeoffPoints), wantPoints)
		}
		if _, isSingle := singlePort[name]; isSingle {
			for _, point := range deck.TakeoffPoints {
				if point.LaunchAngle != -60 {
					t.Errorf("%s 单台弹射器应该向左舷（-60），实际 %v", name, point.LaunchAngle)
				}
			}
		}
	}

	// 这几艘根本没有航空设施，不该配水侦。
	// 亚特兰大级设计即无弹射器；夏洛特是 CL-154 防空巡洋舰方案（同源的伍斯特级最终也没装）。
	for _, name := range []string{"atlanta", "charlotte"} {
		ship, ok := objUnit.ShipMap[name]
		if !ok {
			t.Errorf("缺少舰船 %s", name)
			continue
		}
		if len(ship.Aircraft.Groups) > 0 {
			t.Errorf("%s 不该有水侦，却配了 %s", name, ship.Aircraft.Groups[0].Name)
		}
	}
}
