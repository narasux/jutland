package building

import (
	"testing"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// registerTestPlane 注册测试用飞机模板（GetPlaneTargetObjType / NewPlane 依赖模板存在）。
func registerTestPlane(t *testing.T, fundsCost, timeCost int64) {
	t.Helper()
	oldTemplate, hadTemplate := objUnit.PlaneMap["F"]
	objUnit.PlaneMap["F"] = &objUnit.Plane{
		Name: "F", FundsCost: fundsCost, TimeCost: timeCost,
		TotalHP: 100, CurHP: 100, MaxSpeed: 0.1,
	}
	t.Cleanup(func() {
		if hadTemplate {
			objUnit.PlaneMap["F"] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, "F")
		}
	})
}

// newTestAirfield 创建测试用机场：单一机型，容量 3。
func newTestAirfield(t *testing.T, maxCount int64) *Airfield {
	t.Helper()
	registerTestPlane(t, 10, 0)
	return NewAirfield(
		objPos.New(50, 50), 90, 6, 0.8,
		faction.HumanAlpha, 1,
		[]objUnit.PlaneGroup{{Name: "F", MaxCount: maxCount}},
	)
}

// TestAirfieldBaseGeometry 验证机场作为静态基地的几何契约。
// 场景：跑道朝向 90°、长 6 格、宽 0.8 格，AircraftBase 接口按 MapBlockSize=128 换算像素。
// 预期：Uid 自洽、速度恒 0、朝向 90°、长度 768、宽度 102.4；失败意味着共用基地几何或换算破裂。
// 覆盖：断言 BaseUid/BaseSpeed/BaseRotation 与 6*128、0.8*128 两处精确值。
func TestAirfieldBaseGeometry(t *testing.T) {
	af := newTestAirfield(t, 3)
	if af.BaseUid() != af.Uid || af.BaseSpeed() != 0 {
		t.Fatal("airfield base uid/speed contract broken")
	}
	if af.BaseRotation() != 90 {
		t.Fatal("airfield base rotation should equal runway rotation")
	}
	// 长度 = 跑道长度 * MapBlockSize，保证既有几何换算无需修改
	if af.BaseLength() != float64(6*constants.MapBlockSize) {
		t.Fatalf("base length = %f, want %f", af.BaseLength(), float64(6*constants.MapBlockSize))
	}
	if af.BaseWidth() != 0.8*float64(constants.MapBlockSize) {
		t.Fatalf("base width = %f, want %f", af.BaseWidth(), 0.8*float64(constants.MapBlockSize))
	}
}

// TestStockPlane 验证生产完成的飞机直接入库，不生成实体，满编后拒绝继续增加。
// 场景：机场自动生产完成一架飞机，调用方只把机型名交给 StockPlane；机库按 Group.MaxCount 封顶。
// 预期：CurCount 从 0 增到 2；满编后再入库不增加；未知机型名也不改变库存。
// 覆盖：构造 MaxCount=2 的单机型机场，连续 StockPlane 两次、第三次、未知名，逐次断言 CurCount。
func TestStockPlane(t *testing.T) {
	af := newTestAirfield(t, 2)
	if len(af.Aircraft.Groups) != 1 || af.Aircraft.Groups[0].CurCount != 0 {
		t.Fatal("new airfield should start with empty stock")
	}
	af.StockPlane("F")
	af.StockPlane("F")
	if af.Aircraft.Groups[0].CurCount != 2 {
		t.Fatalf("CurCount = %d, want 2", af.Aircraft.Groups[0].CurCount)
	}
	// 满编后入库应被拒绝
	af.StockPlane("F")
	if af.Aircraft.Groups[0].CurCount != 2 {
		t.Fatal("full group should not accept more planes")
	}
	// 未知机型不入库
	af.StockPlane("unknown")
	if af.Aircraft.Groups[0].CurCount != 2 {
		t.Fatal("unknown plane name should not change stock")
	}
}

// TestAirfieldTakeOffSpawnsOnRunway 验证机场起飞与航母一致：在跑道起点生成实体并扣减库存。
// 场景：机场库存 1 架，警戒/玩家下令起飞；飞机应以滑跑阶段出现在跑道起点，而不是直接巡航。
// 预期：TakeOff 返回非 nil、IsOnGround 为 true、CurCount 变为 0；库存耗尽后返回 nil。
// 覆盖：StockPlane 后调用 Aircraft.TakeOff，断言阶段/库存，再调用一次断言拒绝。
func TestAirfieldTakeOffSpawnsOnRunway(t *testing.T) {
	af := newTestAirfield(t, 2)
	af.StockPlane("F")
	plane := af.Aircraft.TakeOff(af, af.Aircraft.Groups[0].TargetType)
	if plane == nil {
		t.Fatal("takeoff should spawn a plane with stock available")
	}
	if !plane.IsOnGround() {
		t.Fatal("spawned plane should be rolling on the runway (taking off)")
	}
	if af.Aircraft.Groups[0].CurCount != 0 {
		t.Fatalf("CurCount = %d, want 0 after takeoff", af.Aircraft.Groups[0].CurCount)
	}
	// 库存耗尽后无法再起飞
	if af.Aircraft.TakeOff(af, af.Aircraft.Groups[0].TargetType) != nil {
		t.Fatal("empty stock should refuse to take off")
	}
}

// TestAirfieldProduction 验证自动生产首帧开工、次帧完工，且 Update 不负责入库。
// 场景：容量 2 的 F 机型，资金 10、工时 0；连续两次 Update(100,nil) 推进单工位生产。
// 预期：首帧 completed 为空，次帧返回 ["F"]，库存仍 0；满编后不再产出。
// 覆盖：检查两次 Update 的完成列表与 CurCount，再入库 2 架验证容量拒绝生产。
func TestAirfieldProduction(t *testing.T) {
	af := newTestAirfield(t, 2)
	if completed := af.Update(100, nil); len(completed) != 0 {
		t.Fatal("production should not complete on the first tick")
	}
	completed := af.Update(100, nil)
	if len(completed) != 1 || completed[0] != "F" {
		t.Fatalf("completed = %v, want [F]", completed)
	}
	if af.Aircraft.Groups[0].CurCount != 0 {
		t.Fatal("Update should not stock planes itself")
	}
	// 满编后不再生产
	af.StockPlane("F")
	af.StockPlane("F")
	if completed := af.Update(100, nil); len(completed) != 0 {
		t.Fatal("full group should not produce")
	}
}

// TestAirfieldProductionPaused 验证资金不足只暂停计时、不丢已累计进度。
// 场景：F 资金 10、工时 0；先 Update(100) 开工，再 Update(0) 断资，最后 Update(100) 恢复。
// 预期：断资帧无完工，恢复帧返回 1 个 "F"；失败意味着暂停清空进度或断资仍产出。
// 覆盖：三次 Update 的 completed 长度依次为 0/0/1，覆盖 LastTickedAt 刷新与 ElapsedMs 累计。
func TestAirfieldProductionPaused(t *testing.T) {
	af := newTestAirfield(t, 2)

	// 开工一帧
	af.Update(100, nil)
	// 资金不足：暂停但不丢状态
	if completed := af.Update(0, nil); len(completed) != 0 {
		t.Fatal("production should pause without funds")
	}
	// 资金恢复后继续完成（累计时长模式）
	if completed := af.Update(100, nil); len(completed) != 1 {
		t.Fatal("production should resume after funds return")
	}
}

// registerTestBomber 注册第二机型测试模板（验证单工位生产切换）。
func registerTestBomber(t *testing.T, fundsCost, timeCost int64) {
	t.Helper()
	oldTemplate, hadTemplate := objUnit.PlaneMap["B"]
	objUnit.PlaneMap["B"] = &objUnit.Plane{
		Name: "B", FundsCost: fundsCost, TimeCost: timeCost,
		TotalHP: 100, CurHP: 100, MaxSpeed: 0.1,
	}
	t.Cleanup(func() {
		if hadTemplate {
			objUnit.PlaneMap["B"] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, "B")
		}
	})
}

// TestAirfieldSingleStationProduction 验证单工位生产：默认制造序列第一位，同一时间只推进当前机型。
// 场景：机库有 F/B 两型，CurProducing=F；玩家切换制造 B 后，F 进度应保留但不再推进。
// 预期：默认 CurProducing=F；F 先完工 1 架；切 B 后 B 完工 1 架而 F 仍为 1；未知机型回退到第 0 组。
// 覆盖：构造双机型机场，分两轮 Update+StockPlane，断言 CurProducing/两组 CurCount/ProducingGroupIdx。
func TestAirfieldSingleStationProduction(t *testing.T) {
	registerTestPlane(t, 10, 0)
	registerTestBomber(t, 10, 0)
	af := NewAirfield(
		objPos.New(50, 50), 90, 6, 0.8,
		faction.HumanAlpha, 1,
		[]objUnit.PlaneGroup{{Name: "F", MaxCount: 2}, {Name: "B", MaxCount: 2}},
	)
	// 默认制造第一位（F）
	if af.CurProducing != "F" {
		t.Fatalf("default producing = %q, want F", af.CurProducing)
	}
	// 开工 + 完成第 1 架 F（TimeCost 0，第二帧完工）
	if completed := af.Update(100, nil); len(completed) != 0 {
		t.Fatal("production should not complete on the first tick")
	}
	for _, name := range af.Update(100, nil) {
		af.StockPlane(name)
	}
	if af.Aircraft.Groups[0].CurCount != 1 {
		t.Fatalf("F stock = %d, want 1", af.Aircraft.Groups[0].CurCount)
	}
	// 单工位：B 不应同时开工
	if _, ok := af.Production["B"]; ok {
		t.Fatal("B should not start production while F is selected")
	}

	// 指定制造 B：切换后 B 开工并完成，F 不再推进
	af.CurProducing = "B"
	if completed := af.Update(100, nil); len(completed) != 0 {
		t.Fatal("production should not complete on the first tick after switching")
	}
	for _, name := range af.Update(100, nil) {
		af.StockPlane(name)
	}
	if af.Aircraft.Groups[1].CurCount != 1 {
		t.Fatalf("B stock = %d, want 1", af.Aircraft.Groups[1].CurCount)
	}
	if af.Aircraft.Groups[0].CurCount != 1 {
		t.Fatal("F should stay parked while B is being produced")
	}

	// 已不存在的机型回退到可制造序列第一位
	af.CurProducing = "unknown"
	if af.ProducingGroupIdx() != 0 {
		t.Fatal("unknown producing type should fall back to the first group")
	}
}

// TestAirfieldDisabled 验证停用机场只停警戒起飞，生产照常推进，恢复后警戒恢复。
// 场景：玩家停用机场避免自动出击，但仍希望它继续补充飞机；停用期间资金/生产不应被冻结。
// 预期：Disabled=true 时 CanAlertLaunch 为 false，但 Update 仍能完工；恢复 false 后 CanAlertLaunch 为 true。
// 覆盖：先测启用状态，再置 Disabled 跑两次 Update，最后恢复并断言。
func TestAirfieldDisabled(t *testing.T) {
	af := newTestAirfield(t, 2)
	if !af.CanAlertLaunch() {
		t.Fatal("enabled airfield with stock should alert launch")
	}

	af.Disabled = true
	if af.CanAlertLaunch() {
		t.Fatal("disabled airfield should not alert launch")
	}
	// 停用不影响生产
	if completed := af.Update(100, nil); len(completed) != 0 {
		t.Fatal("production should not complete on the first tick")
	}
	if completed := af.Update(100, nil); len(completed) != 1 {
		t.Fatal("production should continue while disabled")
	}
	af.StockPlane("F")
	if af.ProductionProgress("F") != 0 {
		t.Fatal("progress should restart after completion")
	}

	// 恢复启用：警戒恢复
	af.Disabled = false
	if !af.CanAlertLaunch() {
		t.Fatal("re-enabled airfield should alert launch again")
	}
}

// TestAirfieldProductionCapacity 验证生产只补充损失：待命 + 出击 < MaxCount 才生产。
// 场景：机库满编（2 架）不生产；起飞 1 架后待命 1 + 出击 1 = 2 仍不生产；被击落后待命 1 < 2 才恢复生产。
// 预期：满编/满容量时 Update 不完工；损失后第二次 Update 完工 1 架。
// 覆盖：手动改 CurCount 并用 flying map 表示出击数，分三段调用 Update 断言完工列表。
func TestAirfieldProductionCapacity(t *testing.T) {
	af := newTestAirfield(t, 2)

	// 满编（库存 2）：不生产
	af.StockPlane("F")
	af.StockPlane("F")
	if completed := af.Update(100, nil); len(completed) != 0 {
		t.Fatal("full group should not produce")
	}
	// 起飞一架（库存 1 + 出击 1 = 上限）：仍不生产，不能无限生产
	af.Aircraft.Groups[0].CurCount = 1
	if completed := af.Update(100, map[string]int64{"F": 1}); len(completed) != 0 {
		t.Fatal("stock + flying == max should not produce")
	}
	// 出击的被击落（待命 1 < 上限）：恢复生产补损失
	if completed := af.Update(100, nil); len(completed) != 0 {
		t.Fatal("production should not complete on the first tick")
	}
	if completed := af.Update(100, nil); len(completed) != 1 {
		t.Fatal("production should resume after losses")
	}
}

// TestAirfieldProductionAutoSwitch 验证指定机型满编后自动顺延到下一个未满编机型，全部满编则停产。
// 场景：CurProducing=F 但 F 已满编、B 未满；机场应自动改产 B，而不是停在 F 上空转。
// 预期：ProducingTargetIdx 返回 1；Update 推进 B 并完工；两组都满编后 target 为 -1、Update 不完工。
// 覆盖：双机型各 MaxCount=2，先塞满 F，再 Update/StockPlane B，最后塞满 B 并断言。
func TestAirfieldProductionAutoSwitch(t *testing.T) {
	registerTestPlane(t, 10, 0)
	registerTestBomber(t, 10, 0)
	af := NewAirfield(
		objPos.New(50, 50), 90, 6, 0.8,
		faction.HumanAlpha, 1,
		[]objUnit.PlaneGroup{{Name: "F", MaxCount: 2}, {Name: "B", MaxCount: 2}},
	)

	// F 满编 + 指定 F：自动顺延到 B
	af.StockPlane("F")
	af.StockPlane("F")
	af.CurProducing = "F"
	if target := af.ProducingTargetIdx(nil); target != 1 {
		t.Fatalf("target idx = %d, want 1 (auto switch to B)", target)
	}
	if completed := af.Update(100, nil); len(completed) != 0 {
		t.Fatal("production should not complete on the first tick after switching")
	}
	for _, name := range af.Update(100, nil) {
		af.StockPlane(name)
	}
	if af.Aircraft.Groups[1].CurCount != 1 {
		t.Fatalf("B stock = %d, want 1", af.Aircraft.Groups[1].CurCount)
	}

	// 全部满编：停止生产
	af.StockPlane("B")
	if target := af.ProducingTargetIdx(nil); target != -1 {
		t.Fatalf("target idx = %d, want -1 (capacity full)", target)
	}
	if completed := af.Update(100, nil); len(completed) != 0 {
		t.Fatal("capacity full should stop production")
	}
}

// TestAirfieldSetTakeoffPoints 验证跑道起飞点数配置：缺省双点并行、1 单机串行、0 no-op。
// 场景：重轰炸机需要单架间隔起飞，轻型机保留双机并行；配置 0/负数表示沿用默认。
// 预期：默认双点同一 tick 可起飞 2 架；SetTakeoffPoints(1) 后同 tick 只起飞 1 架；
// SetTakeoffPoints(0) 不改动单点模式。
// 覆盖：构造同一机场三次，分别用默认/1/0 设置，连续 TakeOff 并断言成功数与库存。
func TestAirfieldSetTakeoffPoints(t *testing.T) {
	const planeName = "test-takeoff-bomber"
	oldTemplate, hadTemplate := objUnit.PlaneMap[planeName]
	objUnit.PlaneMap[planeName] = &objUnit.Plane{
		Name:   planeName,
		Type:   objUnit.PlaneTypeDiveBomber,
		Weapon: objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{}}},
	}
	t.Cleanup(func() {
		if hadTemplate {
			objUnit.PlaneMap[planeName] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, planeName)
		}
	})

	target := objUnit.GetPlaneTargetObjType(planeName)
	mk := func() *Airfield {
		return NewAirfield(
			objPos.New(50, 50), 90, 6, 0.8, faction.HumanAlpha, 10,
			[]objUnit.PlaneGroup{{Name: planeName, MaxCount: 4}},
		)
	}
	stock := func(af *Airfield) {
		for i := 0; i < 4; i++ {
			af.StockPlane(planeName)
		}
	}

	// 缺省双起飞点：同一 tick 可并行起飞两架
	af := mk()
	stock(af)
	if p1, p2 := af.Aircraft.TakeOff(af, target), af.Aircraft.TakeOff(af, target); p1 == nil || p2 == nil {
		t.Fatalf("default airfield should allow parallel double takeoff, p1=%v p2=%v", p1 != nil, p2 != nil)
	}

	// SetTakeoffPoints(1)：单机串行——同一 tick 第二架因起飞点冷却被拒
	af1 := mk()
	af1.SetTakeoffPoints(1)
	stock(af1)
	if q1, q2 := af1.Aircraft.TakeOff(af1, target), af1.Aircraft.TakeOff(af1, target); q1 == nil || q2 != nil {
		t.Fatalf("single-point airfield should take off one at a time, q1=%v q2=%v", q1 != nil, q2 != nil)
	}
	if got := af1.Aircraft.Groups[0].CurCount; got != 3 {
		t.Fatalf("after single launch stock = %d, want 3", got)
	}

	// SetTakeoffPoints(0) 是 no-op：保持既有（单点）起飞模式
	af2 := mk()
	af2.SetTakeoffPoints(1)
	af2.SetTakeoffPoints(0)
	stock(af2)
	if r1, r2 := af2.Aircraft.TakeOff(af2, target), af2.Aircraft.TakeOff(af2, target); r1 == nil || r2 != nil {
		t.Fatalf("SetTakeoffPoints(0) should keep single-point takeoff, r1=%v r2=%v", r1 != nil, r2 != nil)
	}
}
