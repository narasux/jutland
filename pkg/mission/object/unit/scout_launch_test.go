package unit

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
)

// TestTakeOffScoutDoublesRangeWithoutChangingTemplate 验证侦察机起飞时航程翻倍，且不修改全局模板。
// 场景：专职侦察机基础航程 10，搜索任务需要 20；若直接改模板 RemainRange，后续所有同型机都会继承翻倍。
// 预期：新飞机 RemainRange=20；PlaneMap 模板仍为 10；库存 2→1。
// 覆盖：注册模板，调用 TakeOffScout 后断言实例/模板/库存。
func TestTakeOffScoutDoublesRangeWithoutChangingTemplate(t *testing.T) {
	const name = "scout-range-test"
	template := &Plane{
		Name: name, Type: PlaneTypeScout,
		Range: 10, RemainRange: 10, SightRange: SightRangeScout,
	}
	PlaneMap[name] = template
	t.Cleanup(func() { delete(PlaneMap, name) })

	aircraft := &ShipAircraft{
		Groups: []PlaneGroup{{Name: name, CurCount: 2, MaxCount: 2}},
		deck:   &CarrierDeck{TakeoffPoints: []TakeoffPoint{{RunLength: 0.2}}},
	}
	ship := &BattleShip{
		Uid: "carrier", Length: 128, Width: 20, BelongPlayer: faction.HumanAlpha,
	}
	plane := aircraft.TakeOffScout(ship)
	if plane == nil {
		t.Fatal("expected a scout to launch")
	}
	if plane.RemainRange != 20 {
		t.Fatalf("remain range = %v, want 20", plane.RemainRange)
	}
	if template.RemainRange != 10 {
		t.Fatalf("template range = %v, want 10", template.RemainRange)
	}
	if aircraft.ScoutStock() != 1 {
		t.Fatalf("stock = %d, want 1", aircraft.ScoutStock())
	}
}

// registerSearchTestPlane 注册一张带航程的临时飞机模板，测试结束自动清掉。
func registerSearchTestPlane(t *testing.T, name string, planeType PlaneType, planeRange float64) {
	t.Helper()
	old, had := PlaneMap[name]
	PlaneMap[name] = &Plane{
		Name: name, Type: planeType,
		Range: planeRange, RemainRange: planeRange, SightRange: ResolvePlaneSight(planeType, 0),
	}
	t.Cleanup(func() {
		if had {
			PlaneMap[name] = old
			return
		}
		delete(PlaneMap, name)
	})
}

// TestSearchFallsBackToStrikePlane 验证没有侦察机的基地能拿对舰打击机顶上做搜索。
// 场景：航母只有 12 架鱼雷机（TargetType=TypeShip），没有专职侦察机；
// 若搜索只允许 scout 编组，美英航母会永远派不出搜索机。
// 预期：ScoutStock=0；SearchStock=12（侦察机不足 searchMinStock 时退回全部可用库存）；
// TakeOffSearch 返回 Type=PlaneTypeTorpedoBomber，RemainRange 保持模板 30（打击机顶班不翻倍）；
// 该组库存从 12 扣到 11。
// 覆盖：注册鱼雷机模板，构造 ShipAircraft 调用 ScoutStock/SearchStock/TakeOffSearch 并断言类型、航程、库存。
func TestSearchFallsBackToStrikePlane(t *testing.T) {
	const bomber = "bomber-search-test"
	registerSearchTestPlane(t, bomber, PlaneTypeTorpedoBomber, 30)

	aircraft := &ShipAircraft{
		Groups: []PlaneGroup{{
			Name: bomber, CurCount: 12, MaxCount: 12, TargetType: object.TypeShip,
		}},
		deck: &CarrierDeck{TakeoffPoints: []TakeoffPoint{{RunLength: 0.2}}},
	}
	ship := &BattleShip{
		Uid: "carrier", Length: 128, Width: 20, BelongPlayer: faction.HumanAlpha,
	}

	if got := aircraft.ScoutStock(); got != 0 {
		t.Fatalf("scout stock = %d, want 0", got)
	}
	if got := aircraft.SearchStock(); got != 12 {
		t.Fatalf("search stock = %d, want 12", got)
	}
	plane := aircraft.TakeOffSearch(ship)
	if plane == nil {
		t.Fatal("expected a strike plane to fly the search sortie")
	}
	if plane.Type != PlaneTypeTorpedoBomber {
		t.Fatalf("plane type = %s, want the torpedo bomber", plane.Type)
	}
	// 打击机顶班不再额外翻倍航程，油量本来就是它自己的。
	if plane.RemainRange != 30 {
		t.Fatalf("remain range = %v, want 30", plane.RemainRange)
	}
	if aircraft.Groups[0].CurCount != 11 {
		t.Fatalf("group count = %d, want 11", aircraft.Groups[0].CurCount)
	}
}

// TestSearchPrefersDedicatedScout 验证有侦察机时优先派侦察机，并保留搜索航程翻倍。
// 场景：航母同时带 12 架鱼雷机与 4 架侦察机（模板航程 10），侦察机库存 ≥ searchMinStock=2；
// 此时 SearchStock 只应统计侦察机，起飞也必须选中 scout 编组。
// 预期：SearchStock=4；TakeOffSearch 返回 Type=PlaneTypeScout，RemainRange=10×2=20。
// 覆盖：注册侦察机与鱼雷机两种模板，构造双组机库，断言 SearchStock 与起飞机型/航程。
func TestSearchPrefersDedicatedScout(t *testing.T) {
	const (
		scout  = "scout-prefer-test"
		bomber = "bomber-prefer-test"
	)
	registerSearchTestPlane(t, scout, PlaneTypeScout, 10)
	registerSearchTestPlane(t, bomber, PlaneTypeTorpedoBomber, 30)

	aircraft := &ShipAircraft{
		Groups: []PlaneGroup{
			{Name: bomber, CurCount: 12, MaxCount: 12, TargetType: object.TypeShip},
			{Name: scout, CurCount: 4, MaxCount: 4},
		},
		deck: &CarrierDeck{TakeoffPoints: []TakeoffPoint{{RunLength: 0.2}}},
	}

	if got := aircraft.SearchStock(); got != 4 {
		t.Fatalf("search stock = %d, want the 4 scouts", got)
	}
	plane := aircraft.TakeOffSearch(&BattleShip{
		Uid: "carrier", Length: 128, Width: 20, BelongPlayer: faction.HumanAlpha,
	})
	if plane == nil || plane.Type != PlaneTypeScout {
		t.Fatalf("plane = %v, want a dedicated scout", plane)
	}
	if plane.RemainRange != 20 {
		t.Fatalf("remain range = %v, want 20", plane.RemainRange)
	}
}

// TestSearchNeedsTwoPlanesOnDeck 验证自动搜索的出击门槛：只剩一架侦察机时留在甲板上。
// 场景：航母只有 1 架侦察机；自动搜索若派走最后一架，甲板就再没有可用侦察力量。
// 预期：SearchStock=1，低于 searchMinStock=2，CanSearch 返回 false。
// 覆盖：构造单架侦察机机库，比较 SearchStock 与 searchMinStock。
func TestSearchNeedsTwoPlanesOnDeck(t *testing.T) {
	const scout = "scout-lonely-test"
	registerSearchTestPlane(t, scout, PlaneTypeScout, 10)

	aircraft := &ShipAircraft{
		Groups: []PlaneGroup{{Name: scout, CurCount: 1, MaxCount: 4}},
		deck:   &CarrierDeck{TakeoffPoints: []TakeoffPoint{{RunLength: 0.2}}},
	}
	if got := aircraft.SearchStock(); got != 1 {
		t.Fatalf("search stock = %d, want 1", got)
	}
	if aircraft.SearchStock() >= searchMinStock {
		t.Fatal("只剩一架时应低于搜索出击门槛，把最后一架留在甲板上")
	}
}

// TestManualScoutFallsBackToStrikePlane 验证玩家手动侦察没有侦察机时派一架打击机顶上。
// 场景：航母只有 12 架鱼雷机（TargetType=TypeShip）；玩家明确下达侦察命令，
// 基地不能以“没有 scout”为由拒绝派机。
// 预期：CanManualScout 返回 true；TakeOffManualScout 返回 Type=PlaneTypeTorpedoBomber。
// 覆盖：注册鱼雷机模板，构造 ShipAircraft 依次调用 CanManualScout 与 TakeOffManualScout 断言机型。
func TestManualScoutFallsBackToStrikePlane(t *testing.T) {
	const bomber = "bomber-manual-test"
	registerSearchTestPlane(t, bomber, PlaneTypeTorpedoBomber, 30)

	aircraft := &ShipAircraft{
		Groups: []PlaneGroup{{
			Name: bomber, CurCount: 12, MaxCount: 12, TargetType: object.TypeShip,
		}},
		deck: &CarrierDeck{TakeoffPoints: []TakeoffPoint{{RunLength: 0.2}}},
	}
	if !aircraft.CanManualScout() {
		t.Fatal("有打击机库存时手动侦察应该可以派机")
	}
	plane := aircraft.TakeOffManualScout(&BattleShip{
		Uid: "carrier", Length: 128, Width: 20, BelongPlayer: faction.HumanAlpha,
	})
	if plane == nil || plane.Type != PlaneTypeTorpedoBomber {
		t.Fatalf("plane = %v, want the torpedo bomber", plane)
	}
}

// TestManualScoutLaunchesTheLastScout 验证手动侦察不受“甲板留一架”限制，可派最后一架侦察机。
// 场景：航母只剩 1 架侦察机（模板航程 10）；自动搜索因库存 1 < searchMinStock=2 留在甲板，
// 但玩家手动下令时不应再保留甲板库存。
// 预期：CanSearch 返回 false；CanManualScout 返回 true；
// TakeOffManualScout 返回 Type=PlaneTypeScout，RemainRange=10×2=20。
// 覆盖：单架侦察机机库依次调用 CanSearch/CanManualScout/TakeOffManualScout，断言可用性与机型/航程。
func TestManualScoutLaunchesTheLastScout(t *testing.T) {
	const scout = "scout-manual-test"
	registerSearchTestPlane(t, scout, PlaneTypeScout, 10)

	aircraft := &ShipAircraft{
		Groups: []PlaneGroup{{Name: scout, CurCount: 1, MaxCount: 4, TargetType: object.TypeNone}},
		deck:   &CarrierDeck{TakeoffPoints: []TakeoffPoint{{RunLength: 0.2}}},
	}
	if aircraft.CanSearch() {
		t.Fatal("自动搜索这时应该留在甲板上")
	}
	if !aircraft.CanManualScout() {
		t.Fatal("玩家手动侦察应该允许派最后一架侦察机")
	}
	plane := aircraft.TakeOffManualScout(&BattleShip{
		Uid: "carrier", Length: 128, Width: 20, BelongPlayer: faction.HumanAlpha,
	})
	if plane == nil || plane.Type != PlaneTypeScout {
		t.Fatalf("plane = %v, want a dedicated scout", plane)
	}
	if plane.RemainRange != 20 {
		t.Fatalf("remain range = %v, want 20", plane.RemainRange)
	}
}

// TestManualScoutRejectsFighterOnlyCarrier 验证只有战斗机的航母派不出侦察机：制空机不顶侦察。
// 场景：航母带 12 架战斗机（TargetType=TypePlane），没有 scout、也没有对舰打击机；
// 若把任意库存都当成可派，玩家会把制空机浪费在侦察任务上。
// 预期：CanManualScout 返回 false；TakeOffManualScout 返回 nil。
// 覆盖：注册战斗机模板，构造纯战斗机机库，调用 CanManualScout 与 TakeOffManualScout 断言拒绝。
func TestManualScoutRejectsFighterOnlyCarrier(t *testing.T) {
	const fighter = "fighter-manual-test"
	registerSearchTestPlane(t, fighter, PlaneTypeFighter, 10)

	aircraft := &ShipAircraft{
		Groups: []PlaneGroup{{
			Name: fighter, CurCount: 12, MaxCount: 12, TargetType: object.TypePlane,
		}},
		deck: &CarrierDeck{TakeoffPoints: []TakeoffPoint{{RunLength: 0.2}}},
	}
	if aircraft.CanManualScout() {
		t.Fatal("只有战斗机时不该认为可以派侦察机")
	}
	if plane := aircraft.TakeOffManualScout(&BattleShip{
		Uid: "carrier", Length: 128, Width: 20, BelongPlayer: faction.HumanAlpha,
	}); plane != nil {
		t.Fatalf("plane = %v, want nil", plane)
	}
}
