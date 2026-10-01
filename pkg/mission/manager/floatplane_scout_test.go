package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// 美国战列舰 / 巡洋舰要能弹射水上侦察机，这是「从战舰起飞飞机侦察」的功能保障。
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

// 小黄鸭、水滴这类特殊船也必须照雾，否则开迷雾时它们自己那一圈永远是黑的。
func TestSpecialShipsProvideSight(t *testing.T) {
	if got := objUnit.ResolveShipSight(objUnit.ShipTypeDefault, 0, 1); got != objUnit.SightRangeSpecial {
		t.Fatalf("特殊船视距 = %v, want %v", got, objUnit.SightRangeSpecial)
	}
}

// 三种美军水上侦察机都要在册并且是 scout 型，否则舰船编组会引用到打不出搜索的机型。
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

// 美国主力舰的水侦配备表：按状态年份对机型，改配置时这条会挡住漏改。
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
		"new_york": 1, "nevada": 1, "maryland": 1, "west_virginia": 1, "fargo": 1,
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
	// 亚特兰大级设计即无弹射器；阿肯色 1944 状态是 AG-17 防空训练舰；
	// 夏洛特是 CL-154 防空巡洋舰方案（同源的伍斯特级最终也没装）。
	for _, name := range []string{"atlanta", "arkansas", "charlotte"} {
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
