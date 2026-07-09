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
	requireClose(t, carrierLandingApproachPos(ship).RX, 10)
	requireClose(t, carrierLandingApproachPos(ship).RY, 14)
	requireClose(t, carrierLandingDeckStartPos(ship).RX, 10)
	requireClose(t, carrierLandingDeckStartPos(ship).RY, 10.9)
	requireClose(t, carrierLandingDeckEndPos(ship).RX, 10)
	requireClose(t, carrierLandingDeckEndPos(ship).RY, 10)

	ship.CurRotation = 90
	requireClose(t, carrierTakeoffStartPos(ship).RX, 10)
	requireClose(t, carrierTakeoffStartPos(ship).RY, 10)
}

func TestTakeoffPhaseTransitionsToCruise(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	defer func() { config.G = oldSettings }()

	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	plane := &Plane{
		MaxSpeed:     0.4,
		Acceleration: 0.2,
		CurHP:        100,
		RemainRange:  100,
	}
	plane.StartTakeoff(ship)

	if plane.FlightPhase != PlaneFlightPhaseTakingOff {
		t.Fatalf("phase = %s, want %s", plane.FlightPhase, PlaneFlightPhaseTakingOff)
	}
	requireClose(t, plane.VisualScaleMultiplier(), planeLowAltitudeVisualScale)
	requireClose(t, plane.CurSpeed, 0.1)

	cfg := &mapcfg.MapCfg{Width: 100, Height: 100}
	plane.UpdateTakeoff(cfg)
	if got, want := plane.CurSpeed, 0.132; math.Abs(got-want) > 1e-9 {
		t.Fatalf("takeoff speed after first update = %v, want %v", got, want)
	}
	for i := 0; i < 10 && plane.FlightPhase != PlaneFlightPhaseCruising; i++ {
		plane.UpdateTakeoff(cfg)
	}

	if plane.FlightPhase != PlaneFlightPhaseCruising {
		t.Fatalf("phase = %s, want %s", plane.FlightPhase, PlaneFlightPhaseCruising)
	}
	requireClose(t, plane.VisualScaleMultiplier(), 1)
}

func TestLandingPhaseDecelerates(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	defer func() { config.G = oldSettings }()

	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	plane := &Plane{
		MaxSpeed:     0.4,
		Acceleration: 0.2,
		CurHP:        100,
		CurPos:       objPos.NewR(50, 54),
		CurRotation:  0,
		CurSpeed:     0.4,
		RemainRange:  100,
	}

	plane.StartLandingApproach(ship)
	plane.UpdateLandingApproach(&mapcfg.MapCfg{Width: 100, Height: 100}, ship)
	if got, want := plane.CurSpeed, 0.368; math.Abs(got-want) > 1e-9 {
		t.Fatalf("landing approach speed after first update = %v, want %v", got, want)
	}
}

func TestLandingVisualScaleTransitionsSmoothly(t *testing.T) {
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	plane := &Plane{
		CurPos:               objPos.NewR(50, 54),
		FlightPhase:          PlaneFlightPhaseCruising,
		FlightVisualScaleEnd: 1,
	}

	plane.StartLandingApproach(ship)
	requireClose(t, plane.VisualScaleMultiplier(), 1)

	plane.CurPos = objPos.NewR(50, 53)
	requireClose(t, plane.VisualScaleMultiplier(), 0.8)

	plane.CurPos = carrierLandingApproachPos(ship)
	plane.StartLandingDeck(ship)
	requireClose(t, plane.FlightVisualScaleStart, landingDeckStartVisualScale)
	requireClose(t, plane.VisualScaleMultiplier(), landingDeckStartVisualScale)

	plane.CurPos = carrierLandingDeckEndPos(ship)
	requireClose(t, plane.VisualScaleMultiplier(), planeLowAltitudeVisualScale)
}
