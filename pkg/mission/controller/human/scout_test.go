package human

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/object"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// registerScoutTestPlane 注册一张临时飞机模板，测试结束自动清掉。
func registerScoutTestPlane(t *testing.T, name string, planeType objUnit.PlaneType, targetType object.Type) {
	t.Helper()
	old, had := objUnit.PlaneMap[name]
	objUnit.PlaneMap[name] = &objUnit.Plane{
		Name: name, Type: planeType, TotalHP: 20, CurHP: 20, RemainRange: 30,
		MaxSpeed: 1, RotateSpeed: 30,
		SightRange: objUnit.ResolvePlaneSight(planeType, 0),
	}
	t.Cleanup(func() {
		if had {
			objUnit.PlaneMap[name] = old
			return
		}
		delete(objUnit.PlaneMap, name)
	})
}

// newScoutTestCarrier 造一艘航母，groups 为它的舰载机分组。
func newScoutTestCarrier(uid string, groups []objUnit.PlaneGroup) *objUnit.BattleShip {
	ship := &objUnit.BattleShip{
		Uid: uid, Length: 128, Width: 20, CurPos: objPos.NewR(4, 4),
		BelongPlayer: faction.HumanAlpha,
		Aircraft:     objUnit.ShipAircraft{Groups: groups, HasPlane: len(groups) > 0},
	}
	ship.Aircraft.ResolveDeck(&objUnit.CarrierDeck{TakeoffPoints: []objUnit.TakeoffPoint{{RunLength: 0.2}}})
	return ship
}

func newScoutTestState(ships ...*objUnit.BattleShip) *state.MissionState {
	shipMap := make(map[string]*objUnit.BattleShip, len(ships))
	selected := make([]string, 0, len(ships))
	for _, ship := range ships {
		shipMap[ship.Uid] = ship
		selected = append(selected, ship.Uid)
	}
	return &state.MissionState{
		Core: state.MissionCoreState{SimTick: 1},
		Arena: state.MissionArenaState{
			Ships:  shipMap,
			Planes: map[string]*objUnit.Plane{},
		},
		Interaction: state.MissionInteractionState{SelectedShips: selected},
		UI:          state.MissionUIState{GameMarks: map[objMark.ID]*objMark.Mark{}},
	}
}

// 没有侦察机的航母，Shift + 右键也要派打击机去侦察，而不是让它自己开过去。
func TestScoutOrderSendsBomberFromCarrierWithoutScout(t *testing.T) {
	const bomber = "bomber-scout-order-test"
	registerScoutTestPlane(t, bomber, objUnit.PlaneTypeTorpedoBomber, object.TypeShip)

	carrier := newScoutTestCarrier("carrier", []objUnit.PlaneGroup{
		{Name: bomber, CurCount: 4, MaxCount: 4, TargetType: object.TypeShip},
	})
	ms := newScoutTestState(carrier)

	orders, scouted := scoutAt(ms, objPos.New(20, 20))
	if !scouted {
		t.Fatal("选中航母时 Shift + 右键必须作为侦察指令被消费，不能退化成舰船移动")
	}
	if len(orders) != 1 {
		t.Fatalf("orders = %d, want 1 scout order", len(orders))
	}
	var plane *objUnit.Plane
	for _, order := range orders {
		scout, ok := order.(*instr.PlaneScout)
		if !ok {
			t.Fatalf("order = %T, want a scout order", order)
		}
		plane = ms.Arena.Planes[scout.PlaneUid()]
	}
	if plane == nil || plane.Type != objUnit.PlaneTypeTorpedoBomber {
		t.Fatalf("plane = %v, want the torpedo bomber", plane)
	}
	if !plane.ScoutManual {
		t.Fatal("手动派出的侦察机应该标记为手动")
	}
	if _, ok := ms.UI.GameMarks[objMark.IDScout]; !ok {
		t.Fatal("侦察下单后应该标记侦察目标点")
	}
}

// 已经有手动侦察在飞时再次下单只改点，不重复起飞。
func TestScoutOrderRetargetsFlyingScout(t *testing.T) {
	const scout = "scout-scout-order-test"
	registerScoutTestPlane(t, scout, objUnit.PlaneTypeScout, object.TypeNone)

	carrier := newScoutTestCarrier("carrier", []objUnit.PlaneGroup{
		{Name: scout, CurCount: 2, MaxCount: 2, TargetType: object.TypeNone},
	})
	ms := newScoutTestState(carrier)
	flying := &objUnit.Plane{
		Uid: "flying-scout", Type: objUnit.PlaneTypeScout, CurHP: 10, RemainRange: 30,
		BelongShip: carrier.Uid, ScoutManual: true, BelongPlayer: faction.HumanAlpha,
	}
	ms.Arena.PutPlane(flying)

	orders, scouted := scoutAt(ms, objPos.New(20, 20))
	if !scouted || len(orders) != 1 {
		t.Fatalf("scouted = %v, orders = %d, want a single retarget order", scouted, len(orders))
	}
	for _, order := range orders {
		if order.Uid() != instr.GenInstrUid(instr.NamePlaneScout, flying.Uid) {
			t.Fatalf("order uid = %s, want the flying scout", order.Uid())
		}
	}
	if len(ms.Arena.Planes) != 1 {
		t.Fatalf("planes = %d, want no second takeoff", len(ms.Arena.Planes))
	}
}

// 完成任务正在返航的侦察机不再占用改点名额，新下单要派新飞机。
func TestScoutOrderSkipsReturningScout(t *testing.T) {
	const scout = "scout-returning-order-test"
	registerScoutTestPlane(t, scout, objUnit.PlaneTypeScout, object.TypeNone)

	carrier := newScoutTestCarrier("carrier", []objUnit.PlaneGroup{
		{Name: scout, CurCount: 2, MaxCount: 2, TargetType: object.TypeNone},
	})
	ms := newScoutTestState(carrier)
	returning := &objUnit.Plane{
		Uid: "returning-scout", Type: objUnit.PlaneTypeScout, CurHP: 10, RemainRange: 30,
		BelongShip: carrier.Uid, ScoutManual: true, ForceReturn: true,
		BelongPlayer: faction.HumanAlpha,
	}
	ms.Arena.PutPlane(returning)

	orders, _ := scoutAt(ms, objPos.New(20, 20))
	if len(orders) != 1 {
		t.Fatalf("orders = %d, want a scout order for the new plane", len(orders))
	}
	for _, order := range orders {
		if order.Uid() == instr.GenInstrUid(instr.NamePlaneScout, returning.Uid) {
			t.Fatal("正在返航的侦察机不该被改点")
		}
	}
}

// 只有战斗机的基地派不出侦察机，但这次点击仍要被消费掉。
func TestScoutOrderConsumesClickOnFighterOnlyBase(t *testing.T) {
	const fighter = "fighter-scout-order-test"
	registerScoutTestPlane(t, fighter, objUnit.PlaneTypeFighter, object.TypePlane)

	carrier := newScoutTestCarrier("carrier", []objUnit.PlaneGroup{
		{Name: fighter, CurCount: 4, MaxCount: 4, TargetType: object.TypePlane},
	})
	ms := newScoutTestState(carrier)

	orders, scouted := scoutAt(ms, objPos.New(20, 20))
	if !scouted {
		t.Fatal("选中航母时即使派不出侦察机，也不能把点击让给舰船移动")
	}
	if len(orders) != 0 {
		t.Fatalf("orders = %d, want none", len(orders))
	}
}

// 选中的全是普通战舰时，Shift + 右键仍走原来的移动逻辑。
func TestScoutOrderLeavesPlainWarshipAlone(t *testing.T) {
	destroyer := &objUnit.BattleShip{
		Uid: "dd", Length: 60, Width: 8, CurPos: objPos.NewR(4, 4),
		BelongPlayer: faction.HumanAlpha,
	}
	ms := newScoutTestState(destroyer)

	orders, scouted := scoutAt(ms, objPos.New(20, 20))
	if scouted || len(orders) != 0 {
		t.Fatalf("scouted = %v, orders = %d, want the click left to ship movement", scouted, len(orders))
	}
}
