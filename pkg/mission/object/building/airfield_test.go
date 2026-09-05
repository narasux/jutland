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

// TestStockPlane 生产完成的飞机直接入库：不生成实体，满编后不再增加。
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

// TestAirfieldTakeOffSpawnsOnRunway 与航母一致：起飞时在跑道起点刷新实体并扣减库存。
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

// TestAirfieldSingleStationProduction 单工位生产：默认制造可制造序列第一位，
// 同一时间只推进当前机型；切换机型后按新机型推进，其他机型保持不动。
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

// TestAirfieldDisabled 停用机场：只停警戒起飞，生产照常推进；恢复启用后警戒恢复。
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

// TestAirfieldProductionCapacity 生产只补充损失：待命 + 出击 < 上限才生产
// （起飞即计入容量，出击中不补；被击落恢复生产）。
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

// TestAirfieldProductionAutoSwitch 指定机型满编后自动顺延到下一个未满编
// 机型；全部满编则停止生产。
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
