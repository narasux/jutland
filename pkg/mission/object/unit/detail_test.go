package unit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestDetailSpeedUsesSameSpeedMultiplier 锁定调试面板的速度口径。
//
// CurSpeed 是世界系速度（已乘全局速度倍率），比较基准必须同样乘倍率，否则在
// “快”（2.0）倍速下，每一架巡航中的飞机、每一艘船都会被显示成超过速度上限。
// 同时锁定飞行阶段字段：起降阶段（滑跑 / 待场 / 进近）的速度是世界系位移，
// 允许高于设计上限，调试行要能一眼看出超速读数来自哪个阶段。
func TestDetailSpeedUsesSameSpeedMultiplier(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })
	config.G = &config.GameSettings{SpeedMultiplier: 2}

	plane := &Plane{
		Name:        "A6M2",
		Uid:         "p",
		MaxSpeed:    533.0 / PlaneSpeedScale,
		CurSpeed:    533.0 / PlaneSpeedScale * 2,
		CurHP:       26,
		TotalHP:     26,
		FlightPhase: PlaneFlightPhaseCruising,
		CurPos:      objPos.NewR(10, 20),
	}
	got := plane.Detail()
	if want := fmt.Sprintf("Speed: %.2f/%.2f", plane.CurSpeed, plane.MaxSpeed*2); !strings.Contains(got, want) {
		t.Fatalf("巡航速度口径不一致: %s, 期望包含 %s", got, want)
	}
	if !strings.Contains(got, "Phase: cruising") {
		t.Fatalf("调试行缺少飞行阶段: %s", got)
	}

	// 待场进近的速度是世界系位移，允许高于设计上限；阶段字段必须能区分出来。
	plane.FlightPhase = PlaneFlightPhaseLandingStaging
	plane.CurSpeed = 0.09
	got = plane.Detail()
	if !strings.Contains(got, "Phase: landing_staging") {
		t.Fatalf("待场阶段未标注: %s", got)
	}
	if want := fmt.Sprintf("Speed: %.2f/%.2f", 0.09, plane.MaxSpeed*2); !strings.Contains(got, want) {
		t.Fatalf("待场速度口径不一致: %s, 期望包含 %s", got, want)
	}

	ship := &BattleShip{
		Name:     "akagi",
		Uid:      "s",
		MaxSpeed: 31.0 / ShipSpeedScale,
		CurSpeed: 31.0 / ShipSpeedScale * 2,
		CurHP:    100,
		TotalHP:  100,
		CurPos:   objPos.NewR(5, 6),
	}
	got = ship.Detail()
	if want := fmt.Sprintf("Speed: %.2f/%.2f", ship.CurSpeed, ship.MaxSpeed*2); !strings.Contains(got, want) {
		t.Fatalf("舰船速度口径不一致: %s, 期望包含 %s", got, want)
	}
}
