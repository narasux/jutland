package unit

import "testing"

// TestAircraftStatusClassifiesFlightPhasesAndLosses 验证机库状态按飞行阶段聚合，并把差额计为损失。
// 场景：航母机库有 10 架战斗机库存 4、6 架轰炸机库存 2；场上有起飞/巡航/待场/着舰/其他基地的飞机。
// 预期：战斗机 4 待命/2 作战/2 返航/2 损失；轰炸机 2/1/0/3；总计存活 11、初始 16。
// 覆盖：构造 6 架不同阶段飞机，调用 Status 后逐组断言。
func TestAircraftStatusClassifiesFlightPhasesAndLosses(t *testing.T) {
	aircraft := ShipAircraft{Groups: []PlaneGroup{
		{Name: "fighter", MaxCount: 10, CurCount: 4},
		{Name: "bomber", MaxCount: 6, CurCount: 2},
	}}
	planes := map[string]*Plane{
		"f1": {Name: "fighter", BelongShip: "carrier", Flight: FlightState{Phase: PlaneFlightPhaseTakingOff}},
		"f2": {Name: "fighter", BelongShip: "carrier", Flight: FlightState{Phase: PlaneFlightPhaseCruising}},
		"f3": {Name: "fighter", BelongShip: "carrier", Flight: FlightState{Phase: PlaneFlightPhaseLandingStaging}},
		"f4": {Name: "fighter", BelongShip: "carrier", Flight: FlightState{Phase: PlaneFlightPhaseLandingDeck}},
		"b1": {Name: "bomber", BelongShip: "carrier", Flight: FlightState{Phase: ""}},
		"x1": {Name: "fighter", BelongShip: "other", Flight: FlightState{Phase: PlaneFlightPhaseCruising}},
	}

	status := aircraft.Status("carrier", planes)
	fighter := status.Groups[0]
	if fighter.Standby != 4 || fighter.InCombat != 2 || fighter.Returning != 2 || fighter.Lost != 2 {
		t.Fatalf("fighter status = %+v", fighter)
	}
	bomber := status.Groups[1]
	if bomber.Standby != 2 || bomber.InCombat != 1 || bomber.Returning != 0 || bomber.Lost != 3 {
		t.Fatalf("bomber status = %+v", bomber)
	}
	if status.Total.Alive() != 11 || status.Total.Initial() != 16 {
		t.Fatalf("total status = %+v", status.Total)
	}
}

// TestDisabledAircraftCannotTakeOff 验证 Disable 开关会阻止整组飞机起飞。
// 场景：玩家在单位面板禁用航母舰载机，或机场停用警戒；此时即使库存大于 0 也不能起飞。
// 预期：TakeOff 返回 nil。
// 覆盖：构造 Disable=true、库存 1 的机库，调用 TakeOff 并断言 nil。
func TestDisabledAircraftCannotTakeOff(t *testing.T) {
	aircraft := ShipAircraft{Disable: true, Groups: []PlaneGroup{{Name: "fighter", MaxCount: 1, CurCount: 1}}}
	if plane := aircraft.TakeOff(&BattleShip{}, 0); plane != nil {
		t.Fatal("disabled aircraft group took off")
	}
}

// TestStandbyCount 验证甲板备机总数只累加各组库存 CurCount，不含已出击与返航中的飞机。
// 场景：机库三组飞机——战斗机 4/10、轰炸机 0/6、鱼雷机 3/8（CurCount 只记待命库存）；
// 出击中和返航中的飞机已经离开库存，不应重复计入备机；空机库也必须返回 0 而不是越界。
// 预期：StandbyCount() = 4 + 0 + 3 = 7；零值 ShipAircraft 返回 0。
// 覆盖：三组库存调用 StandbyCount 断言 7，再用 ShipAircraft{} 断言 0。
func TestStandbyCount(t *testing.T) {
	aircraft := ShipAircraft{Groups: []PlaneGroup{
		{Name: "fighter", MaxCount: 10, CurCount: 4},
		{Name: "bomber", MaxCount: 6, CurCount: 0},
		{Name: "torpedo", MaxCount: 8, CurCount: 3},
	}}

	if got := aircraft.StandbyCount(); got != 7 {
		t.Fatalf("StandbyCount() = %d, want 7", got)
	}
	empty := ShipAircraft{}
	if got := empty.StandbyCount(); got != 0 {
		t.Fatalf("empty StandbyCount() = %d, want 0", got)
	}
}
