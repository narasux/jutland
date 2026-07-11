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
		CurPos:      objPos.NewR(10, 10),
		CurRotation: 0,
	}

	requireClose(t, carrierTakeoffStartPos(ship).RX, 10)
	requireClose(t, carrierTakeoffStartPos(ship).RY, 10)
	requireClose(t, carrierTakeoffEndPos(ship).RX, 10)
	requireClose(t, carrierTakeoffEndPos(ship).RY, 6)
	requireClose(t, carrierLandingFinalStartPos(ship).RX, 10)
	requireClose(t, carrierLandingFinalStartPos(ship).RY, 13)
	requireClose(t, carrierLandingDeckEndPos(ship).RX, 10)
	requireClose(t, carrierLandingDeckEndPos(ship).RY, 10)

	ship.CurRotation = 90
	requireClose(t, carrierTakeoffStartPos(ship).RX, 10)
	requireClose(t, carrierTakeoffStartPos(ship).RY, 10)
}

func TestTakeoffUsesSmoothMonotonicAcceleration(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      247,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	plane := &Plane{
		MaxSpeed:     533.0 / 5400,
		Acceleration: 30.0 / 600,
		CurHP:        100,
		RemainRange:  100,
	}
	plane.StartTakeoff(ship)
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
	if frames < 50 || frames > 90 {
		t.Fatalf("takeoff duration = %d frames, want 50..90", frames)
	}
	if firstSpeedStep > plane.MaxSpeed*0.01 {
		t.Fatalf("first takeoff speed step = %v, want <= %v", firstSpeedStep, plane.MaxSpeed*0.01)
	}
	requireClose(t, plane.CurSpeed, plane.MaxSpeed)
	requireClose(t, plane.VisualScaleMultiplier(), 1)
}
