package unit

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
)

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

// 没有侦察机的基地应该能拿打击机顶上做搜索，否则美英航母永远派不出搜索机。
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

// 有侦察机时优先派侦察机，并保留翻倍航程。
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

// 只剩一架侦察机时不派：起飞后甲板要留一架。
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
