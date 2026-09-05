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
	end := takeoffRunPos(ship, point, 256/128*0.6)
	requireClose(t, end.RX, 10)
	requireClose(t, end.RY, 10-(0.5-0.4+0.6)*256/128)

	// 带横向偏移与弹射偏转角的起飞点（伊势式两舷弹射器）
	angled := TakeoffPoint{Forward: 0.68, Lateral: -0.5, RunLength: 0.4, LaunchAngle: -90}
	angledStart := takeoffStartPos(ship, angled)
	requireClose(t, angledStart.RX, 10+(-0.5)*64/128)
	requireClose(t, angledStart.RY, 10-(0.5-0.68)*256/128)
	angledEnd := takeoffRunPos(ship, angled, 256/128*0.4)
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
		plane.UpdateTakeoff(cfg, ship)
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
	// 慢加速起步（initial 0.04、120 帧 S 曲线）下完整起飞约 110~140 帧
	if frames < 100 || frames > 150 {
		t.Fatalf("takeoff duration = %d frames, want 100..150", frames)
	}
	if firstSpeedStep > plane.MaxSpeed*0.01 {
		t.Fatalf("first takeoff speed step = %v, want <= %v", firstSpeedStep, plane.MaxSpeed*0.01)
	}
	requireClose(t, plane.CurSpeed, plane.MaxSpeed)
	requireClose(t, plane.VisualScaleMultiplier(), 1)
}

func TestTakeoffRunStaysOnDeckWhileCarrierTurns(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      256,
		Width:       64,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		CurSpeed:    0.05,
	}
	plane := &Plane{
		MaxSpeed:     0.12,
		Acceleration: 0.01,
		RotateSpeed:  12,
		CurHP:        100,
		RemainRange:  100,
	}
	point := TakeoffPoint{Forward: 0.4, Lateral: 0, RunLength: 0.6, LaunchAngle: 15}
	plane.StartTakeoff(ship, point)

	// 航母满舵转向期间完成起飞：滑跑段位置与机头必须锁定甲板弹射线
	previousRotation := plane.CurRotation
	previousSpeed := plane.CurSpeed
	speedAtLastRunFrame := 0.0
	reachedClimb := false
	for range 200 {
		if plane.FlightPhase != PlaneFlightPhaseTakingOff {
			break
		}
		moveTestCarrier(ship, 0.8)
		plane.UpdateTakeoff(nil, ship)
		if plane.takeoffDistance < plane.takeoffRunLength {
			requireClose(
				t,
				plane.CurRotation,
				normalizeAngle(ship.CurRotation+point.LaunchAngle),
			)
			expected := takeoffRunPos(ship, point, plane.takeoffDistance)
			if plane.CurPos.Distance(expected) > 1e-6 {
				t.Fatalf(
					"takeoff run left the deck line: %s, want near %s",
					plane.CurPos.String(),
					expected.String(),
				)
			}
			speedAtLastRunFrame = plane.CurSpeed
		} else if !reachedClimb {
			reachedClimb = true
			// 离舰合成速度：与滑跑段末帧的世界位移连续，不允许速度突跳
			if delta := math.Abs(plane.CurSpeed - speedAtLastRunFrame); delta > 0.03 {
				t.Fatalf(
					"liftoff speed jumped: run frame %v -> climb frame %v",
					speedAtLastRunFrame,
					plane.CurSpeed,
				)
			}
			if step := angleDifference(plane.CurRotation, previousRotation); step > plane.RotateSpeed+1e-9 {
				t.Fatalf(
					"heading snapped at liftoff: step = %.2f exceeds rotate speed %.2f",
					step,
					plane.RotateSpeed,
				)
			}
		}
		if step := angleDifference(plane.CurRotation, previousRotation); step > plane.RotateSpeed+1e-9 {
			t.Fatalf(
				"takeoff heading step = %.2f exceeds rotate speed %.2f",
				step,
				plane.RotateSpeed,
			)
		}
		if plane.CurSpeed+1e-6 < previousSpeed-0.03 {
			t.Fatalf("takeoff speed dropped: %v -> %v", previousSpeed, plane.CurSpeed)
		}
		previousRotation = plane.CurRotation
		previousSpeed = plane.CurSpeed
	}

	if !reachedClimb {
		t.Fatal("takeoff never left the run segment")
	}
	if plane.FlightPhase != PlaneFlightPhaseCruising {
		t.Fatalf("flight phase = %s, want %s", plane.FlightPhase, PlaneFlightPhaseCruising)
	}
}
