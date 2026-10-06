package instruction

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// airfieldTestPlaneStats 是测试机型的运行期数值（与 planes.json5 的口径一致）。
type airfieldTestPlaneStats struct {
	maxSpeed     float64
	acceleration float64
	rotateSpeed  float64
}

func newAirfieldLandingTestState(
	t *testing.T, planeName string, maxCount int64,
) (*state.MissionState, *objBuilding.Airfield, *objUnit.Plane) {
	t.Helper()
	return newAirfieldLandingTestStateWithStats(t, planeName, maxCount, airfieldTestPlaneStats{
		maxSpeed: 0.12, acceleration: 0.01, rotateSpeed: 12,
	})
}

func newAirfieldLandingTestStateWithStats(
	t *testing.T, planeName string, maxCount int64, stats airfieldTestPlaneStats,
) (*state.MissionState, *objBuilding.Airfield, *objUnit.Plane) {
	t.Helper()
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	oldTemplate, hadTemplate := objUnit.PlaneMap[planeName]
	objUnit.PlaneMap[planeName] = &objUnit.Plane{
		Name:         planeName,
		TotalHP:      100,
		CurHP:        100,
		MaxSpeed:     stats.maxSpeed,
		Acceleration: stats.acceleration,
		RotateSpeed:  stats.rotateSpeed,
		RemainRange:  100,
	}
	t.Cleanup(func() {
		if hadTemplate {
			objUnit.PlaneMap[planeName] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, planeName)
		}
	})

	// 静态基地：跑道沿正北（rotation 0），长 6 格，位于地图中部
	airfield := objBuilding.NewAirfield(
		objPos.NewR(50, 50), 0, 6, 0.8,
		faction.HumanAlpha, 1,
		[]objUnit.PlaneGroup{{Name: planeName, MaxCount: maxCount}},
	)
	plane := objUnit.NewPlane(planeName, objPos.NewR(50, 54), 0, airfield.Uid, faction.HumanAlpha)

	return &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{
				MapCfg: &mapcfg.MapCfg{Width: 100, Height: 100},
			},
		},
		Arena: state.MissionArenaState{
			Airfields: map[string]*objBuilding.Airfield{airfield.Uid: airfield},
			Planes:    map[string]*objUnit.Plane{plane.Uid: plane},
		},
	}, airfield, plane
}

func TestPlaneReturnRecoversAtAirfield(t *testing.T) {
	ms, airfield, plane := newAirfieldLandingTestState(t, "test-airfield-plane", 2)
	returnInstr := NewPlaneReturn(plane.Uid)
	exec := func() {
		t.Helper()
		if err := returnInstr.Exec(ms); err != nil {
			t.Fatal(err)
		}
	}

	// 巡航返航：进入着陆等待段
	exec()
	if plane.FlightPhase != objUnit.PlaneFlightPhaseLandingStaging {
		t.Fatalf("phase = %s, want %s", plane.FlightPhase, objUnit.PlaneFlightPhaseLandingStaging)
	}
	if airfield.Aircraft.Groups[0].CurCount != 0 {
		t.Fatal("CurCount should not change while waiting")
	}

	// 推进等待段至进入圆弧（机场等待门在 5.5 个跑道长之后，路程较长）
	for frame := 0; frame < 1500 && plane.FlightPhase == objUnit.PlaneFlightPhaseLandingStaging; frame++ {
		exec()
	}
	if plane.FlightPhase != objUnit.PlaneFlightPhaseLandingApproach {
		t.Fatalf("phase = %s, want %s (pos=%s rotation=%.2f)",
			plane.FlightPhase, objUnit.PlaneFlightPhaseLandingApproach,
			plane.CurPos.String(), plane.CurRotation)
	}
	// 推进最终进近段
	for frame := 0; frame < 600 && plane.FlightPhase == objUnit.PlaneFlightPhaseLandingApproach; frame++ {
		exec()
	}
	if plane.FlightPhase != objUnit.PlaneFlightPhaseLandingDeck {
		t.Fatalf("phase = %s, want %s", plane.FlightPhase, objUnit.PlaneFlightPhaseLandingDeck)
	}
	// 推进着舰滑跑（刹车段 = 3 个跑道长，路程较长）
	for frame := 0; frame < 2000 && !returnInstr.Executed(); frame++ {
		exec()
	}
	if !returnInstr.Executed() {
		t.Fatal("return instruction did not finish")
	}

	// 机场回收（与航母一致）：着舰后直接入库，活动实体移除、库存 +1
	if _, ok := ms.Arena.Planes[plane.Uid]; ok {
		t.Fatal("recovered plane entity should be removed from the arena")
	}
	if airfield.Aircraft.Groups[0].CurCount != 1 {
		t.Fatalf("CurCount = %d, want 1 after airfield recovery", airfield.Aircraft.Groups[0].CurCount)
	}
}

// 宽转弯的重型轰炸机（RotateSpeed 只有 1~2°/帧）也必须能在陆地机场完成返航回收：
// 入口容差按转弯能力放宽之前，它们会永远在待场与入口之间反复重引导。
func TestPlaneReturnRecoversAtAirfieldForWideTurningHeavyBomber(t *testing.T) {
	// 数值取自 B-17G / He177A 一档的重型轰炸机。
	stats := airfieldTestPlaneStats{maxSpeed: 0.0428, acceleration: 0.015, rotateSpeed: 1.5}
	ms, airfield, plane := newAirfieldLandingTestStateWithStats(t, "test-heavy-bomber", 2, stats)
	returnInstr := NewPlaneReturn(plane.Uid)

	// 从跑道外侧进场，给足着舰流程需要的帧数（重机待场与进近都更慢）。
	for frame := 0; frame < 12000 && !returnInstr.Executed(); frame++ {
		if err := returnInstr.Exec(ms); err != nil {
			t.Fatal(err)
		}
	}
	if !returnInstr.Executed() {
		t.Fatalf("重型轰炸机未能完成着舰，phase = %s pos = %s", plane.FlightPhase, plane.CurPos.String())
	}
	if airfield.Aircraft.Groups[0].CurCount != 1 {
		t.Fatalf("CurCount = %d, want 1 after airfield recovery", airfield.Aircraft.Groups[0].CurCount)
	}
}
