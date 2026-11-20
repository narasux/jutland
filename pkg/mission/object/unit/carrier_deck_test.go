package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// registerTestPlane 注册临时飞机模板并注册清理函数。
func registerTestPlane(t *testing.T, name string, planeType PlaneType) {
	t.Helper()
	oldTemplate, hadTemplate := PlaneMap[name]
	PlaneMap[name] = &Plane{Name: name, Type: planeType, MaxSpeed: 0.12}
	t.Cleanup(func() {
		if hadTemplate {
			PlaneMap[name] = oldTemplate
		} else {
			delete(PlaneMap, name)
		}
	})
}

// TestLandingConfigForSlotAlternatesSeaSides 验证水上飞机着舰位按槽位交替分配到左右舷，甲板模式不受影响。
// 场景：水上飞机母舰/无甲板舰用 LandingModeSea 回收飞机，多机同时着舰需要左右分舷避免叠在一起。
// 预期：slot0 在左舷、slot1 在右舷，横向幅度为配置值；配置幅度过小时回落到默认 1.1；Deck 模式两个槽位相同。
// 覆盖：分别用 Sea/窄 Sea/Deck 三套甲板调用 landingConfigForSlot(0/1)，断言 Lateral 符号与绝对值。
func TestLandingConfigForSlotAlternatesSeaSides(t *testing.T) {
	sa := &ShipAircraft{}
	sa.ResolveDeck(&CarrierDeck{
		Name: "sea-sides",
		Landing: LandingConfig{
			Mode:           LandingModeSea,
			Forward:        0.5,
			Lateral:        1.1,
			ApproachLength: 3,
		},
	})

	left := sa.landingConfigForSlot(0)
	right := sa.landingConfigForSlot(1)
	if left.Lateral >= 0 || right.Lateral <= 0 {
		t.Fatalf("sea slots should alternate sides: slot0=%v slot1=%v", left.Lateral, right.Lateral)
	}
	requireClose(t, math.Abs(left.Lateral), 1.1)
	requireClose(t, math.Abs(right.Lateral), 1.1)

	// 配置幅度不足舷外时，回落到默认 1.1 舰宽
	sa.ResolveDeck(&CarrierDeck{
		Name:    "sea-narrow",
		Landing: LandingConfig{Mode: LandingModeSea, Forward: 0.5, Lateral: 0.2},
	})
	narrow := sa.landingConfigForSlot(0)
	requireClose(t, narrow.Lateral, -1.1)

	// 甲板回收不受槽位影响
	sa.ResolveDeck(&CarrierDeck{
		Name:    "deck",
		Landing: LandingConfig{Mode: LandingModeDeck, Forward: 0.75, Lateral: 0.15},
	})
	deckLeft := sa.landingConfigForSlot(0)
	deckRight := sa.landingConfigForSlot(1)
	requireClose(t, deckLeft.Lateral, 0.15)
	requireClose(t, deckRight.Lateral, 0.15)
}

// TestSeaLandingTouchdownUsesOppositeSides 验证水上着舰的实际溅落点落在舰尾左右两舷，且前后位置一致。
// 场景：绘制与回收逻辑都调用 takeoffLandingPos 计算触点；若两个槽位都落在同侧，多机会重叠。
// 预期：左槽 lateral<0、右槽>0，两侧绝对值相等，forward 都为 0（舰尾回收点）。
// 覆盖：解析 Sea 甲板后对 slot0/1 调用 takeoffLandingPos，断言符号/对称性/纵向位置。
func TestSeaLandingTouchdownUsesOppositeSides(t *testing.T) {
	ship := &BattleShip{
		Length:      256,
		Width:       64,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		Aircraft:    ShipAircraft{},
	}
	sa := &ShipAircraft{}
	sa.ResolveDeck(&CarrierDeck{
		Name: "seaplane-tender",
		Landing: LandingConfig{
			Mode:           LandingModeSea,
			Forward:        0.5,
			Lateral:        1.1,
			ApproachLength: 3,
		},
	})
	ship.Aircraft = *sa

	left := takeoffLandingPos(ship, ship.Aircraft.landingConfigForSlot(0))
	right := takeoffLandingPos(ship, ship.Aircraft.landingConfigForSlot(1))
	if left.lateral >= 0 || right.lateral <= 0 {
		t.Fatalf("sea touchdown should use both sides: left=%v right=%v", left.lateral, right.lateral)
	}
	requireClose(t, left.forward, 0)
	requireClose(t, right.forward, 0)
	requireClose(t, math.Abs(left.lateral), math.Abs(right.lateral))
}

// TestDeckValidateNormalizesFields 验证 CarrierDeck.Validate 把配置中的越界值夹到合法范围并补默认值。
// 场景：手写 carrier_decks.json5 时 Forward 可能 >1、Lateral 超舷、RunLength 为 0/负、Landing.Mode 拼错。
// 预期：Forward 夹到 [0,1]、Lateral 夹到 [-0.5,0.5]；RunLength/TakeOffTime/ApproachLength 补默认或下限；
// 未知 Mode 回退 Deck，Sea 模式保留。
// 覆盖：构造含多组越界字段的甲板，Validate 后逐项断言；再测 Sea 短进近段。
func TestDeckValidateNormalizesFields(t *testing.T) {
	deck := &CarrierDeck{
		Name: "normalize-test",
		TakeoffPoints: []TakeoffPoint{
			{Forward: 1.5, Lateral: -2, RunLength: 0, TakeOffTime: -3},
			{Forward: -1, Lateral: 0.8, RunLength: 0.2},
		},
		Landing: LandingConfig{Mode: "bogus", Forward: 2, ApproachLength: -1},
	}
	deck.Validate()

	p0, p1 := deck.TakeoffPoints[0], deck.TakeoffPoints[1]
	requireClose(t, p0.Forward, 1)
	requireClose(t, p0.Lateral, -0.5)
	requireClose(t, p0.RunLength, defaultTakeoffRunLength)
	requireClose(t, p0.TakeOffTime, 0)
	requireClose(t, p1.Forward, 0)
	requireClose(t, p1.Lateral, 0.5)
	requireClose(t, p1.RunLength, 0.2)

	requireClose(t, deck.Landing.Forward, 1)
	requireClose(t, deck.Landing.ApproachLength, defaultApproachLength)
	if deck.Landing.Mode != LandingModeDeck {
		t.Fatalf("landing mode = %s, want %s", deck.Landing.Mode, LandingModeDeck)
	}

	// 过短的进近段抬升到下限，sea 模式保留
	seaDeck := &CarrierDeck{
		Name:    "sea-test",
		Landing: LandingConfig{Mode: LandingModeSea, Forward: 0.5, ApproachLength: 0.5},
	}
	seaDeck.Validate()
	if seaDeck.Landing.Mode != LandingModeSea {
		t.Fatalf("sea landing mode was overwritten: %s", seaDeck.Landing.Mode)
	}
	requireClose(t, seaDeck.Landing.ApproachLength, minApproachLength)
}

// TestTakeOffUsesIndependentPointCooldowns 验证双弹射器各自独立冷却，同一时刻每个点只服务一架。
// 场景：航母两条弹射器同时放飞；若共用一个冷却计时，第二架会被第一架挡住，出击效率异常。
// 预期：同一 tick 两架从不同点起飞、库存 -2；第三架被两个冷却挡住；只释放点 0 后第四架从点 0 起飞，
// 随后再次被挡。
// 覆盖：构造双点 6s 冷却甲板，连续 TakeOff 并手动回拨单点计时，断言数量/位置/库存。
func TestTakeOffUsesIndependentPointCooldowns(t *testing.T) {
	useDefaultSettings(t)
	registerTestPlane(t, "test-deck-fighter", PlaneTypeFighter)

	// 双弹射器：同一舰船可并行起飞，但每个点同一时刻只服务一架
	deck := &CarrierDeck{
		Name: "cooldown-test",
		TakeoffPoints: []TakeoffPoint{
			{Forward: 0.4, Lateral: 0, RunLength: 0.5, TakeOffTime: 6},
			{Forward: 0.4, Lateral: -0.4, RunLength: 0.5, TakeOffTime: 6},
		},
		Landing: LandingConfig{Mode: LandingModeDeck, Forward: 0.5, ApproachLength: 3},
	}
	ship := &BattleShip{
		Uid:         "carrier-cooldown-test",
		Length:      256,
		Width:       64,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		Aircraft:    ShipAircraft{},
	}
	sa := &ShipAircraft{
		TakeOffTime: 6,
		Groups: []PlaneGroup{{
			Name:       "test-deck-fighter",
			MaxCount:   10,
			CurCount:   10,
			TargetType: object.TypePlane,
		}},
	}
	sa.ResolveDeck(deck)
	ship.Aircraft = *sa

	p1 := sa.TakeOff(ship, object.TypePlane)
	if p1 == nil {
		t.Fatal("first takeoff failed")
	}
	p2 := sa.TakeOff(ship, object.TypePlane)
	if p2 == nil {
		t.Fatal("second takeoff failed: parallel points should serve independently")
	}
	if p1.CurPos.Distance(p2.CurPos) < 0.01 {
		t.Fatalf("parallel takeoffs spawned at the same point: %s", p1.CurPos.String())
	}
	if sa.Groups[0].CurCount != 8 {
		t.Fatalf("group count = %d, want 8", sa.Groups[0].CurCount)
	}

	// 两个点都在冷却中：互斥性生效，无法连续起飞第三架
	if p3 := sa.TakeOff(ship, object.TypePlane); p3 != nil {
		t.Fatal("third takeoff should be blocked by per-point cooldowns")
	}

	// 点 0 冷却结束（点 1 仍在冷却）：只有点 0 可用
	sa.takeoffPointTimes[0] -= int64((6 + 1) * 1e3 / config.G.SpeedMultiplier)
	p4 := sa.TakeOff(ship, object.TypePlane)
	if p4 == nil {
		t.Fatal("takeoff should succeed after the first point's cooldown expires")
	}
	expectedStart := takeoffStartPos(ship, deck.TakeoffPoints[0])
	if p4.CurPos.Distance(expectedStart) > 0.01 {
		t.Fatalf("plane took off from wrong point: %s, want near %s",
			p4.CurPos.String(), expectedStart.String())
	}

	// 点 0 刚重新占用、点 1 仍在冷却：再次起飞被阻止
	if p5 := sa.TakeOff(ship, object.TypePlane); p5 != nil {
		t.Fatal("takeoff should be blocked while every point is cooling down")
	}
}

// TestTakeOffRoutesPlaneTypesToWhitelistedPoints 验证按机型白名单选择弹射位，
// 白名单不可用时回退到无白名单备用点。
// 场景：短跑道只供战斗机、长跑道供轰炸机/鱼雷机；配置 TakeoffPoint.PlaneTypes 后，机种必须走对应点。
// 预期：对舰先出俯冲轰炸机走长点，对空出战斗机走短点；长点冷却时鱼雷机回退到备用点；
// 长点恢复后仍优先白名单点。
// 覆盖：三点甲板 + 三种机型，逐步 TakeOff 并比较出生点与机名。
func TestTakeOffRoutesPlaneTypesToWhitelistedPoints(t *testing.T) {
	useDefaultSettings(t)
	registerTestPlane(t, "test-route-fighter", PlaneTypeFighter)
	registerTestPlane(t, "test-route-dive", PlaneTypeDiveBomber)
	registerTestPlane(t, "test-route-torpedo", PlaneTypeTorpedoBomber)

	deck := &CarrierDeck{
		Name: "route-test",
		TakeoffPoints: []TakeoffPoint{
			// 短点专供战斗机
			{
				Forward: 0.4, Lateral: 0, RunLength: 0.5, TakeOffTime: 60,
				PlaneTypes: []PlaneType{PlaneTypeFighter},
			},
			// 长点专供轰炸机与鱼雷机
			{
				Forward: 0.55, Lateral: 0.2, RunLength: 0.7, TakeOffTime: 60,
				PlaneTypes: []PlaneType{PlaneTypeDiveBomber, PlaneTypeTorpedoBomber},
			},
			// 无白名单的备用点：任意机型
			{Forward: 0.3, Lateral: -0.3, RunLength: 0.5, TakeOffTime: 60},
		},
		Landing: LandingConfig{Mode: LandingModeDeck, Forward: 0.5, ApproachLength: 3},
	}
	ship := &BattleShip{
		Uid:         "carrier-route-test",
		Length:      256,
		Width:       64,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		Aircraft:    ShipAircraft{},
	}
	sa := &ShipAircraft{
		TakeOffTime: 60,
		Groups: []PlaneGroup{
			{Name: "test-route-fighter", MaxCount: 4, CurCount: 4, TargetType: object.TypePlane},
			{Name: "test-route-dive", MaxCount: 4, CurCount: 4, TargetType: object.TypeShip},
			{Name: "test-route-torpedo", MaxCount: 4, CurCount: 4, TargetType: object.TypeShip},
		},
	}
	sa.ResolveDeck(deck)
	ship.Aircraft = *sa

	// 对舰打击：战斗机组不参与，轰炸机按顺序走长点而非备用点
	dive := sa.TakeOff(ship, object.TypeShip)
	if dive == nil || dive.Name != "test-route-dive" {
		t.Fatalf("dive bomber takeoff = %v, want test-route-dive", dive)
	}
	expectedStart := takeoffStartPos(ship, deck.TakeoffPoints[1])
	if dive.CurPos.Distance(expectedStart) > 0.01 {
		t.Fatalf("dive bomber took off from wrong point: %s", dive.CurPos.String())
	}

	// 空战：战斗机走短点
	fighter := sa.TakeOff(ship, object.TypePlane)
	if fighter == nil || fighter.Name != "test-route-fighter" {
		t.Fatalf("fighter takeoff = %v, want test-route-fighter", fighter)
	}
	expectedStart = takeoffStartPos(ship, deck.TakeoffPoints[0])
	if fighter.CurPos.Distance(expectedStart) > 0.01 {
		t.Fatalf("fighter took off from wrong point: %s", fighter.CurPos.String())
	}

	// 轰炸机组清空后：鱼雷机按白名单走长点（长点被首次起飞占用，先解除冷却）
	sa.Groups[1].CurCount = 0
	sa.takeoffPointTimes[1] -= int64((60 + 1) * 1e3 / config.G.SpeedMultiplier)
	torpedo := sa.TakeOff(ship, object.TypeShip)
	if torpedo == nil || torpedo.Name != "test-route-torpedo" {
		t.Fatalf("torpedo bomber takeoff = %v, want test-route-torpedo", torpedo)
	}
	expectedStart = takeoffStartPos(ship, deck.TakeoffPoints[1])
	if torpedo.CurPos.Distance(expectedStart) > 0.01 {
		t.Fatalf("torpedo bomber took off from wrong point: %s", torpedo.CurPos.String())
	}

	// 恢复轰炸机库存并让长点、备用点冷却结束
	sa.Groups[1].CurCount = 4
	sa.takeoffPointTimes[1] -= int64((60 + 1) * 1e3 / config.G.SpeedMultiplier)
	sa.takeoffPointTimes[2] -= int64((60 + 1) * 1e3 / config.G.SpeedMultiplier)
	dive2 := sa.TakeOff(ship, object.TypeShip)
	if dive2 == nil || dive2.Name != "test-route-dive" {
		t.Fatalf("second dive bomber takeoff = %v, want test-route-dive", dive2)
	}
	expectedStart = takeoffStartPos(ship, deck.TakeoffPoints[1])
	if dive2.CurPos.Distance(expectedStart) > 0.01 {
		t.Fatalf("second dive bomber should prefer the whitelisted long point: %s",
			dive2.CurPos.String())
	}

	// 长点冷却中：同目标类型的下一个机种轮转到无白名单的备用点
	torpedoFallback := sa.TakeOff(ship, object.TypeShip)
	if torpedoFallback == nil || torpedoFallback.Name != "test-route-torpedo" {
		t.Fatalf("fallback torpedo bomber takeoff = %v, want test-route-torpedo", torpedoFallback)
	}
	expectedStart = takeoffStartPos(ship, deck.TakeoffPoints[2])
	if torpedoFallback.CurPos.Distance(expectedStart) > 0.01 {
		t.Fatalf("fallback torpedo bomber should use the unwhitelisted point: %s",
			torpedoFallback.CurPos.String())
	}
}

// TestTakeOffWithinRangeSkipsShortRangeGroups 验证按目标距离起飞时跳过航程不足的机组。
// 场景：目标距航母 80 格，机库同时有航程 20 和 100 的俯冲轰炸机；短航程机起飞后无法抵达/返航。
// 预期：TakeOffWithinRange 只返回长航程机组。
// 覆盖：构造两组同类型不同 Range 的飞机，调用 TakeOffWithinRange(80) 断言机名。
func TestTakeOffWithinRangeSkipsShortRangeGroups(t *testing.T) {
	useDefaultSettings(t)
	const shortName = "test-range-short"
	const longName = "test-range-long"
	registerTestPlane(t, shortName, PlaneTypeDiveBomber)
	registerTestPlane(t, longName, PlaneTypeDiveBomber)
	PlaneMap[shortName].Range = 20
	PlaneMap[longName].Range = 100

	deck := &CarrierDeck{
		Name: "range-test",
		TakeoffPoints: []TakeoffPoint{
			{Forward: 0.5, Lateral: 0, RunLength: 0.5},
		},
		Landing: LandingConfig{Mode: LandingModeDeck, Forward: 0.5, ApproachLength: 3},
	}
	ship := &BattleShip{
		Uid:         "carrier-range-test",
		Length:      256,
		Width:       64,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	sa := &ShipAircraft{
		TakeOffTime: 60,
		Groups: []PlaneGroup{
			{Name: shortName, MaxCount: 1, CurCount: 1, TargetType: object.TypeShip},
			{Name: longName, MaxCount: 1, CurCount: 1, TargetType: object.TypeShip},
		},
	}
	sa.ResolveDeck(deck)
	ship.Aircraft = *sa

	plane := sa.TakeOffWithinRange(ship, object.TypeShip, 80)
	if plane == nil || plane.Name != longName {
		t.Fatalf("plane = %v, want long-range group for an 80-cell target", plane)
	}
}

// TestStartTakeoffAppliesLaunchAngle 验证弹射位 LaunchAngle 叠加到舰体航向，并从配置的舷侧位置出生。
// 场景：斜角甲板/侧舷弹射器让飞机斜着离舰；若忽略 LaunchAngle，飞机航向会与跑道方向不一致。
// 预期：CurRotation = normalize(舰体航向 + LaunchAngle)，CurPos 在 takeoffStartPos 附近，
// Flight.Phase 为 TakingOff。
// 覆盖：舰体 30° + LaunchAngle -90°，调用 StartTakeoff 后逐项断言。
func TestStartTakeoffAppliesLaunchAngle(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      256,
		Width:       64,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 30,
	}
	plane := &Plane{MaxSpeed: 0.12, CurHP: 100, RemainRange: 100}
	point := TakeoffPoint{Forward: 0.68, Lateral: -0.32, RunLength: 0.4, LaunchAngle: -90}
	plane.StartTakeoff(ship, point)

	// 起飞航向 = 舰体航向 + 弹射偏转角
	requireClose(t, plane.CurRotation, normalizeAngle(ship.CurRotation+point.LaunchAngle))
	// 起点位于配置的舷侧弹射位
	expectedStart := takeoffStartPos(ship, point)
	if plane.CurPos.Distance(expectedStart) > 0.01 {
		t.Fatalf("takeoff start = %s, want near %s",
			plane.CurPos.String(), expectedStart.String())
	}
	if plane.Flight.Phase != PlaneFlightPhaseTakingOff {
		t.Fatalf("flight phase = %s, want %s", plane.Flight.Phase, PlaneFlightPhaseTakingOff)
	}
}

// TestNewShipKeepsTakeoffDeckAfterCopy 验证 NewShip 深拷贝模板后仍保留已解析的甲板指针，飞机可以正常起飞。
// 场景：initShipMap 先给模板舰 ResolveDeck，再 deepcopy 生成局内舰；deepcopy 不复制未导出字段。
// 预期：新舰 HasPlane 为 true、Aircraft.deck 非 nil、TakeOff 成功、库存 4→3。
// 覆盖：注册模板舰和 DeckMap，NewShip 后检查 deck/起飞/库存。
func TestNewShipKeepsTakeoffDeckAfterCopy(t *testing.T) {
	useDefaultSettings(t)
	registerTestPlane(t, "test-newship-fighter", PlaneTypeFighter)

	// 模拟 initShipMap：模板舰解析甲板模板
	template := &BattleShip{
		Uid:         "carrier-template-test",
		Name:        "test-newship-carrier",
		TypeAbbr:    "CVT",
		Length:      256,
		Width:       64,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		Aircraft: ShipAircraft{
			TakeOffTime: 1,
			Deck:        "newship-test-deck",
			HasPlane:    true,
			Groups: []PlaneGroup{{
				Name:       "test-newship-fighter",
				MaxCount:   4,
				CurCount:   4,
				TargetType: object.TypePlane,
			}},
		},
	}
	testDeck := &CarrierDeck{
		Name: "newship-test-deck",
		TakeoffPoints: []TakeoffPoint{
			{Forward: 0.4, Lateral: 0, RunLength: 0.5, TakeOffTime: 0},
		},
		Landing: LandingConfig{Mode: LandingModeDeck, Forward: 0.5, ApproachLength: 3},
	}
	template.Aircraft.ResolveDeck(testDeck)
	ShipMap[template.Name] = template
	t.Cleanup(func() { delete(ShipMap, template.Name) })
	// 模拟 initCarrierDeckMap 的全局模板注册
	DeckMap[testDeck.Name] = testDeck
	t.Cleanup(func() { delete(DeckMap, testDeck.Name) })

	// NewShip 通过 deepcopy 复制模板；未导出的 deck 指针若丢失，
	// 会导致局内 TakeOff 直接返回 nil，飞机无法起飞
	uidGen := NewShipUidGenerator(faction.HumanAlpha)
	ship := NewShip(uidGen, template.Name, objPos.NewR(60, 60), 0, faction.HumanAlpha)
	if !ship.Aircraft.HasPlane {
		t.Fatal("copied ship lost aircraft")
	}
	if ship.Aircraft.deck == nil {
		t.Fatal("copied ship lost the resolved deck: deepcopy drops unexported fields")
	}
	plane := ship.Aircraft.TakeOff(ship, object.TypePlane)
	if plane == nil {
		t.Fatal("plane cannot take off after NewShip copy")
	}
	if ship.Aircraft.Groups[0].CurCount != 3 {
		t.Fatalf("group count = %d, want 3", ship.Aircraft.Groups[0].CurCount)
	}
}
