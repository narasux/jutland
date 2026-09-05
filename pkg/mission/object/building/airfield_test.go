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
		faction.HumanAlpha, 14, 1,
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
	if completed := af.Update(100); len(completed) != 0 {
		t.Fatal("production should not complete on the first tick")
	}
	completed := af.Update(100)
	if len(completed) != 1 || completed[0] != "F" {
		t.Fatalf("completed = %v, want [F]", completed)
	}
	if af.Aircraft.Groups[0].CurCount != 0 {
		t.Fatal("Update should not stock planes itself")
	}
	// 满编后不再生产
	af.StockPlane("F")
	af.StockPlane("F")
	if completed := af.Update(100); len(completed) != 0 {
		t.Fatal("full group should not produce")
	}
}

func TestAirfieldProductionPaused(t *testing.T) {
	af := newTestAirfield(t, 2)

	// 开工一帧
	af.Update(100)
	// 资金不足：暂停但不丢状态
	if completed := af.Update(0); len(completed) != 0 {
		t.Fatal("production should pause without funds")
	}
	// 资金恢复后继续完成（累计时长模式）
	if completed := af.Update(100); len(completed) != 1 {
		t.Fatal("production should resume after funds return")
	}
}
