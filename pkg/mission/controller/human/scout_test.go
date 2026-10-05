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

// newScoutTestCatapultShip 造一艘带若干起飞点（弹射器）的航空战舰。
func newScoutTestCatapultShip(uid string, points int, groups []objUnit.PlaneGroup) *objUnit.BattleShip {
	ship := &objUnit.BattleShip{
		Uid: uid, Length: 100, Width: 16, CurPos: objPos.NewR(4, 4),
		BelongPlayer: faction.HumanAlpha,
		Aircraft:     objUnit.ShipAircraft{Groups: groups, HasPlane: len(groups) > 0},
	}
	takeoffPoints := make([]objUnit.TakeoffPoint, points)
	for idx := range takeoffPoints {
		takeoffPoints[idx] = objUnit.TakeoffPoint{RunLength: 0.2}
	}
	ship.Aircraft.ResolveDeck(&objUnit.CarrierDeck{TakeoffPoints: takeoffPoints})
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

// 有多个起飞点（弹射器）的航空战舰可以同时放出多架手动侦察机；
// 名额占满后再次下单只改离点击点最近那架的目标点。
func TestScoutOrderLaunchesUpToTakeoffPoints(t *testing.T) {
	const scout = "scout-multi-order-test"
	registerScoutTestPlane(t, scout, objUnit.PlaneTypeScout, object.TypeNone)

	cruiser := newScoutTestCatapultShip("cruiser", 2, []objUnit.PlaneGroup{
		{Name: scout, CurCount: 4, MaxCount: 4, TargetType: object.TypeNone},
	})
	ms := newScoutTestState(cruiser)

	if orders, _ := scoutAt(ms, objPos.New(20, 20)); len(orders) != 1 {
		t.Fatalf("orders = %d, want a first takeoff", len(orders))
	}
	if orders, _ := scoutAt(ms, objPos.New(30, 30)); len(orders) != 1 {
		t.Fatalf("orders = %d, want a second takeoff", len(orders))
	}
	if len(ms.Arena.Planes) != 2 {
		t.Fatalf("planes = %d, want two manual scouts for two catapults", len(ms.Arena.Planes))
	}

	// 两个起飞点都在飞：这一拍只改最近那架的目标点，不再起飞第三架。
	orders, scouted := scoutAt(ms, objPos.New(40, 40))
	if !scouted || len(orders) != 1 {
		t.Fatalf("scouted = %v, orders = %d, want a single retarget order", scouted, len(orders))
	}
	if len(ms.Arena.Planes) != 2 {
		t.Fatalf("planes = %d, want no third takeoff", len(ms.Arena.Planes))
	}
	for _, order := range orders {
		if order.Uid() == "" || order.Executed() {
			t.Fatalf("order = %v, want a retarget order for a flying scout", order)
		}
	}
}

// manualScoutsOf 数某艘舰现在有几架手动侦察机在天上。
func manualScoutsOf(ms *state.MissionState, shipUid string) int {
	count := 0
	for _, plane := range ms.Arena.Planes {
		if plane.BelongShip == shipUid && plane.ScoutManual {
			count++
		}
	}
	return count
}

// 选中一个编队连续点击时应该一艘接一艘轮着派：还没派过的基地优先，
// 而不是每次都用离点击点最近的那艘（派满名额后它就只能改点）。
func TestScoutOrderRotatesAcrossSelectedBases(t *testing.T) {
	const scout = "scout-rotation-order-test"
	registerScoutTestPlane(t, scout, objUnit.PlaneTypeScout, object.TypeNone)

	near := newScoutTestCarrier("carrier-near", []objUnit.PlaneGroup{
		{Name: scout, CurCount: 2, MaxCount: 2, TargetType: object.TypeNone},
	})
	far := newScoutTestCarrier("carrier-far", []objUnit.PlaneGroup{
		{Name: scout, CurCount: 2, MaxCount: 2, TargetType: object.TypeNone},
	})
	far.CurPos = objPos.NewR(20, 4)
	ms := newScoutTestState(near, far)

	// 第一次：两艘都没派过，点击靠近谁就先派谁。
	if orders, _ := scoutAt(ms, objPos.New(4, 4)); len(orders) != 1 {
		t.Fatalf("orders = %d, want a first takeoff", len(orders))
	}
	if got := manualScoutsOf(ms, near.Uid); got != 1 {
		t.Fatalf("near carrier scouts = %d, want 1", got)
	}

	// 第二次：仍然点在同一位置，但近的那艘名额已满，改派还没派过的那艘。
	if orders, _ := scoutAt(ms, objPos.New(4, 4)); len(orders) != 1 {
		t.Fatalf("orders = %d, want a second takeoff", len(orders))
	}
	if got := manualScoutsOf(ms, far.Uid); got != 1 {
		t.Fatalf("far carrier scouts = %d, want 1 (never launched first)", got)
	}
	if len(ms.Arena.Planes) != 2 {
		t.Fatalf("planes = %d, want two manual scouts", len(ms.Arena.Planes))
	}

	// 第三次：两艘都满了，才对离点击点最近的那架改点。
	orders, scouted := scoutAt(ms, objPos.New(4, 4))
	if !scouted || len(orders) != 1 {
		t.Fatalf("scouted = %v, orders = %d, want a single retarget order", scouted, len(orders))
	}
	if len(ms.Arena.Planes) != 2 {
		t.Fatalf("planes = %d, want no third takeoff", len(ms.Arena.Planes))
	}
}

// 「还没派过」优先于「还有名额」：刚派过一架的战列舰不该抢在没派过的航母前面。
func TestScoutOrderPrefersNeverLaunchedBase(t *testing.T) {
	const scout = "scout-priority-order-test"
	registerScoutTestPlane(t, scout, objUnit.PlaneTypeScout, object.TypeNone)

	carrier := newScoutTestCarrier("carrier", []objUnit.PlaneGroup{
		{Name: scout, CurCount: 2, MaxCount: 2, TargetType: object.TypeNone},
	})
	carrier.CurPos = objPos.NewR(30, 30)
	battleship := newScoutTestCatapultShip("battleship", 2, []objUnit.PlaneGroup{
		{Name: scout, CurCount: 3, MaxCount: 3, TargetType: object.TypeNone},
	})
	battleship.CurPos = objPos.NewR(4, 4)
	ms := newScoutTestState(carrier, battleship)
	// 战列舰已经放出一架，还剩一个弹射器空位。
	ms.Arena.PutPlane(&objUnit.Plane{
		Uid: "battleship-scout", Type: objUnit.PlaneTypeScout, CurHP: 10, RemainRange: 30,
		BelongShip: battleship.Uid, ScoutManual: true, BelongPlayer: faction.HumanAlpha,
	})

	// 点击贴近战列舰：它有剩余名额，但航母还没派过，应该先派航母。
	if orders, _ := scoutAt(ms, objPos.New(4, 4)); len(orders) != 1 {
		t.Fatalf("orders = %d, want a takeoff from the never-launched carrier", len(orders))
	}
	if got := manualScoutsOf(ms, carrier.Uid); got != 1 {
		t.Fatalf("carrier scouts = %d, want 1", got)
	}
	if got := manualScoutsOf(ms, battleship.Uid); got != 1 {
		t.Fatalf("battleship scouts = %d, want it left untouched", got)
	}
}

// 排在前面的基地这一拍什么都做不了（弹射器冷却、又没有在飞的侦察机）时，
// 指令要顺延给下一个候选，不能白白吞掉这次点击。
func TestScoutOrderFallsThroughToNextBase(t *testing.T) {
	const scout = "scout-fallthrough-order-test"
	registerScoutTestPlane(t, scout, objUnit.PlaneTypeScout, object.TypeNone)

	cooling := newScoutTestCarrier("cooling", []objUnit.PlaneGroup{
		{Name: scout, CurCount: 2, MaxCount: 2, TargetType: object.TypeNone},
	})
	// 起飞点刚起飞过，还在冷却；这一拍派不出第二架。
	cooling.Aircraft.TakeOffTime = 60
	if plane := cooling.Aircraft.TakeOffScout(cooling); plane == nil {
		t.Fatal("准备阶段应该能起飞一架侦察机")
	}
	fresh := newScoutTestCarrier("fresh", []objUnit.PlaneGroup{
		{Name: scout, CurCount: 2, MaxCount: 2, TargetType: object.TypeNone},
	})
	fresh.CurPos = objPos.NewR(20, 4)
	ms := newScoutTestState(cooling, fresh)

	// 点击贴着还在冷却的那艘：它优先，但派不出也改不了点，指令顺延给另一艘。
	if orders, _ := scoutAt(ms, objPos.New(4, 4)); len(orders) != 1 {
		t.Fatalf("orders = %d, want a takeoff from the other base", len(orders))
	}
	if got := manualScoutsOf(ms, fresh.Uid); got != 1 {
		t.Fatalf("fresh base scouts = %d, want 1", got)
	}
	if got := manualScoutsOf(ms, cooling.Uid); got != 0 {
		t.Fatalf("cooling base scouts = %d, want 0", got)
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
