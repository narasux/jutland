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

// TestScoutOrderSendsBomberFromCarrierWithoutScout 验证无侦察机的航母仍用打击机执行侦察。
// 场景：HumanAlpha 航母只带 4 架 torpedo bomber（CurCount=MaxCount=4，
// TargetType=TypeShip），没有 PlaneTypeScout；对 (20,20) 调用 scoutAt。
// 预期：scouted=true 且只有 1 条指令，类型为 *instr.PlaneScout；新飞机
// Type=PlaneTypeTorpedoBomber、ScoutManual=true，UI.GameMarks[IDScout] 标出目标点。
// 失败会让 Shift+右键退化成航母自身移动。
// 覆盖：registerScoutTestPlane 注册 bomber-scout-order-test，newScoutTestCarrier /
// newScoutTestState 选中航母，断言指令类型、飞机字段与侦察标记。
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

// TestScoutOrderRetargetsFlyingScout 验证已有手动侦察在飞时只改点、不重复起飞。
// 场景：航母带 2 架 scout，场上已有 1 架 ScoutManual=true 的 flying-scout；
// 对 (20,20) 调用 scoutAt。
// 预期：scouted=true 且恰好 1 条指令，Uid 为 NamePlaneScout/flying-scout；
// Arena.Planes 仍只有 1 架，不能为同一次点击再起飞第二架。
// 覆盖：newScoutTestCarrier 注册 scout-scout-order-test，Arena.PutPlane 预置在飞
// 侦察机，断言指令 Uid 与飞机总数。
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

// TestScoutOrderLaunchesUpToTakeoffPoints 验证航空战舰按起飞点数量派多架手动侦察机。
// 场景：巡洋舰 cruiser 有 2 个起飞点（弹射器）和 4 架 scout；依次对 (20,20)、
// (30,30) 下单，再对 (40,40) 下第三次单。
// 预期：前两次各起飞 1 架，场上共 2 架（manualScoutLimit=TakeoffPointCount=2）；
// 第三次名额已满，只生成 1 条改点指令（Uid 非空且未 Executed），不再起飞第三架。
// 覆盖：newScoutTestCatapultShip 造双弹射器舰，连续调用 scoutAt，断言每次指令数、
// 飞机总数与改点指令状态。
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

// TestScoutOrderRotatesAcrossSelectedBases 验证编队连续点击时轮流派机。
// 场景：选中 near (4,4) 与 far (20,4) 两艘各带 2 架 scout 的航母；对 (4,4)
// 连点三次。
// 预期：第一次 near 起飞 1 架；第二次 near 名额已满，改为从未派过的 far 起飞 1 架；
// 第三次两艘都满，只对最近那架改点，场上始终 2 架。
// 覆盖：newScoutTestState 同时选中两舰，manualScoutsOf 统计各舰手动侦察机数，
// 每轮 scoutAt 断言指令数与飞机总数；排序依据是 scoutPriorityNeverLaunched <
// HasSlot < Retarget，同档取到点击点最近者。
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

// TestScoutOrderPrefersNeverLaunchedBase 验证「还没派过」优先于「还有名额」。
// 场景：航母 carrier 在 (30,30) 尚未派过；战列舰 battleship 在 (4,4) 有 2 个
// 起飞点、已放飞 1 架（ScoutManual=true）；点击 (4,4) 更靠近战列舰。
// 预期：指令从航母起飞（manualScoutsOf(carrier)=1），战列舰保持 1 架不动；
// 若按距离优先，近处还有空位的战列舰会一直抢在航母前面。
// 覆盖：newScoutTestCatapultShip 双弹射器 + 预置 battleship-scout，调用 scoutAt
// 后分别统计两舰手动侦察机数。
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

// TestScoutOrderFallsThroughToNextBase 验证前排基地无法响应时指令顺延。
// 场景：cooling 航母在点击点旁，但 TakeOffTime=60 且已调用 TakeOffScout 起飞
// 一架（弹射器冷却）；fresh 航母在 (20,4)，对 (4,4) 附近点击。
// 预期：本次仍产生 1 条指令，从 fresh 起飞 1 架（manualScoutsOf(fresh)=1），
// cooling 保持 0 架；不能白吞这次点击。
// 覆盖：newScoutTestCarrier 预置冷却状态，scoutAt 后按舰统计手动侦察机数。
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

// TestScoutOrderSkipsReturningScout 验证返航中的侦察机不占改点名额。
// 场景：航母带 2 架 scout，场上已有 1 架 ForceReturn=true、ScoutManual=true 的
// returning-scout；对 (20,20) 下单。
// 预期：产生 1 条新指令，且 Uid 不是 NamePlaneScout/returning-scout；返航机
// 不再参与改点，应派新飞机。
// 覆盖：Arena.PutPlane 预置返航机，scoutAt 后检查指令数量与 Uid；
// manualScouts 用 MustReturn() 排除返航机。
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

// TestScoutOrderConsumesClickOnFighterOnlyBase 验证纯战斗机基地也消费 Shift+右键。
// 场景：航母只带 4 架 fighter（TargetType=TypePlane，没有对地/侦察挂载），
// 对 (20,20) 调用 scoutAt。
// 预期：scouted=true 但 orders 为 0——点击不能漏给舰船移动；基地无法派侦察机
// 也算一次侦察指令已被处理。
// 覆盖：registerScoutTestPlane 注册 fighter-scout-order-test，newScoutTestCarrier
// 后断言 scouted 与指令数。
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

// TestScoutOrderLeavesPlainWarshipAlone 验证纯水面战舰不响应侦察点击。
// 场景：只选中一艘没有舰载机的驱逐舰 dd (4,4)，对 (20,20) 调用 scoutAt。
// 预期：scouted=false 且 orders 为 0，点击留给原有的舰船移动逻辑。
// 覆盖：newScoutTestState 只放驱逐舰，直接调用 scoutAt 断言两个返回值。
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
