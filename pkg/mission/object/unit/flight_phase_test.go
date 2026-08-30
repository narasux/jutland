package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

func requireClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func angleDifference(left, right float64) float64 {
	delta := math.Mod(left-right+540, 360) - 180
	return math.Abs(delta)
}

func useDefaultSettings(t *testing.T) {
	t.Helper()
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })
}

func TestCarrierFlightPhasePoints(t *testing.T) {
	ship := &BattleShip{
		Length:      256,
		Width:       64,
		CurPos:      objPos.NewR(10, 10),
		CurRotation: 0,
	}
	// 甲板中部直跑：起点位于舰艏后方 0.4 舰长，沿舰艏方向滑跑 0.6 舰长
	point := TakeoffPoint{Forward: 0.4, Lateral: 0, RunLength: 0.6}
	start := takeoffStartPos(ship, point)
	requireClose(t, start.RX, 10)
	requireClose(t, start.RY, 10-(0.5-0.4)*256/128)
	end := takeoffEndPos(start, ship.CurRotation, 256/128*0.6)
	requireClose(t, end.RX, 10)
	requireClose(t, end.RY, 10-(0.5-0.4+0.6)*256/128)

	// 带横向偏移与弹射偏转角的起飞点（伊势式两舷弹射器）
	angled := TakeoffPoint{Forward: 0.68, Lateral: -0.5, RunLength: 0.4, LaunchAngle: -90}
	angledStart := takeoffStartPos(ship, angled)
	requireClose(t, angledStart.RX, 10+(-0.5)*64/128)
	requireClose(t, angledStart.RY, 10-(0.5-0.68)*256/128)
	angledEnd := takeoffEndPos(angledStart, angled.LaunchAngle, 256/128*0.4)
	requireClose(t, angledEnd.RX, angledStart.RX-256/128*0.4)
	requireClose(t, angledEnd.RY, angledStart.RY)

	// 着舰点与最终进近段起点（默认配置：舰中回收、3 舰长直线进近）
	landing := LandingConfig{Forward: 0.5, Lateral: 0, ApproachAngle: 0, ApproachLength: 3}
	touchdown := takeoffLandingPos(ship, landing)
	requireClose(t, touchdown.forward, 0)
	requireClose(t, touchdown.lateral, 0)
	finalStart := landingFinalStartOffset(ship, landing)
	requireClose(t, finalStart.forward, -3)
	requireClose(t, finalStart.lateral, 0)

	ship.CurRotation = 90
	// 旋转 90 度后舰艏指向 +X：起点沿舰艏前移 (0.5-0.4) 舰长
	requireClose(t, takeoffStartPos(ship, point).RX, 10+(0.5-0.4)*256/128)
	requireClose(t, takeoffStartPos(ship, point).RY, 10)
}

func TestTakeoffUsesSmoothMonotonicAcceleration(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      247,
		Width:       42,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	plane := &Plane{
		MaxSpeed:     533.0 / 5400,
		Acceleration: 30.0 / 600,
		CurHP:        100,
		RemainRange:  100,
	}
	plane.StartTakeoff(ship, TakeoffPoint{Forward: 0.4, Lateral: 0, RunLength: 0.6})
	initialSpeed := plane.CurSpeed
	previousSpeed := initialSpeed
	previousScale := plane.VisualScaleMultiplier()
	firstSpeedStep := 0.0
	cfg := &mapcfg.MapCfg{Width: 100, Height: 100}
	frames := 0
	for frames < 180 && plane.FlightPhase != PlaneFlightPhaseCruising {
		plane.UpdateTakeoff(cfg)
		if frames == 0 {
			firstSpeedStep = plane.CurSpeed - initialSpeed
		}
		if plane.CurSpeed+1e-9 < previousSpeed {
			t.Fatalf("takeoff speed decreased: %v -> %v", previousSpeed, plane.CurSpeed)
		}
		scale := plane.VisualScaleMultiplier()
		if scale+1e-9 < previousScale {
			t.Fatalf("takeoff scale decreased: %v -> %v", previousScale, scale)
		}
		previousSpeed, previousScale = plane.CurSpeed, scale
		frames++
	}

	if plane.FlightPhase != PlaneFlightPhaseCruising {
		t.Fatalf("takeoff did not reach cruising phase")
	}
	if frames < 45 || frames > 90 {
		t.Fatalf("takeoff duration = %d frames, want 45..90", frames)
	}
	if firstSpeedStep > plane.MaxSpeed*0.01 {
		t.Fatalf("first takeoff speed step = %v, want <= %v", firstSpeedStep, plane.MaxSpeed*0.01)
	}
	requireClose(t, plane.CurSpeed, plane.MaxSpeed)
	requireClose(t, plane.VisualScaleMultiplier(), 1)
}
