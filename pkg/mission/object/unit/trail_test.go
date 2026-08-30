package unit

import (
	"testing"

	"github.com/narasux/jutland/pkg/common/constants"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

func TestSwordfishTrailsFollowTailAnimationWithoutPerTickStacking(t *testing.T) {
	ship := &BattleShip{
		TypeAbbr: "Swordfish",
		Length:   180,
		Width:    76,
		MaxSpeed: 1,
		CurSpeed: 1,
		CurPos:   objPos.NewR(10, 10),
		Animation: ShipAnimation{
			TopFrames:  []string{"01", "02", "03", "04", "05", "04", "03", "02"},
			FrameTicks: 6,
		},
		LastTrailAnimationStep: -1,
	}

	left := ship.GenTrails()
	if len(left) != 2 {
		t.Fatalf("first animation step trails=%d, want 2", len(left))
	}
	if left[0].Shape != textureImg.TrailShapeCircle || left[1].Shape != textureImg.TrailShapeCircle {
		t.Fatalf("unexpected swordfish trail shapes: %v, %v", left[0].Shape, left[1].Shape)
	}
	if left[0].CurSize >= 10 || left[1].CurSize >= 10 {
		t.Fatalf("water disturbance is too large: %v/%v", left[0].CurSize, left[1].CurSize)
	}
	if left[0].Pos.RX >= ship.CurPos.RX || left[0].Pos.RY <= ship.CurPos.RY {
		t.Fatalf("left-tail wake position=%s, want behind and left of ship", left[0].Pos.String())
	}
	if trails := ship.GenTrails(); trails != nil {
		t.Fatalf("same animation step generated %d duplicate trails", len(trails))
	}

	ship.AnimationAge = 24
	right := ship.GenTrails()
	if len(right) != 2 {
		t.Fatalf("right animation step trails=%d, want 2", len(right))
	}
	if right[0].Pos.RX <= ship.CurPos.RX || right[0].Pos.RY <= ship.CurPos.RY {
		t.Fatalf("right-tail wake position=%s, want behind and right of ship", right[0].Pos.String())
	}
	if left[0].DiffusionRate <= 0 || right[0].DiffusionRate <= 0 {
		t.Fatalf("water disturbance must expand before fading")
	}
}

func TestSurfaceShipWakeMatchesOriginalWhitePair(t *testing.T) {
	ship := &BattleShip{
		Length:   128,
		Width:    20,
		MaxSpeed: 0.1,
		CurSpeed: 0.05,
		CurPos:   objPos.NewR(10, 10),
	}

	trails := ship.GenTrails()
	if len(trails) != 2 {
		t.Fatalf("default hull trails=%d, want 2", len(trails))
	}
	if trails[0].Color != nil || trails[1].Color != nil {
		t.Fatalf("original foam is default white, got %v %v", trails[0].Color, trails[1].Color)
	}
	if trails[0].CurSize != ship.Width*0.6 || trails[1].CurSize != ship.Width {
		t.Fatalf("unexpected wake size: %v %v", trails[0].CurSize, trails[1].CurSize)
	}
	if trails[0].Pos.RY >= ship.CurPos.RY {
		t.Fatalf("bow wake %s should be forward of ship", trails[0].Pos.String())
	}
	if trails[1].Pos.RY <= ship.CurPos.RY {
		t.Fatalf("stern wake %s should be aft of ship", trails[1].Pos.String())
	}
	if trails[0].CurLife != ship.Length/8+555*ship.CurSpeed {
		t.Fatalf("default bow life=%v, want full ship length", trails[0].CurLife)
	}
}

func TestConfiguredWakeHullsEmitPerHull(t *testing.T) {
	ship := &BattleShip{
		Length:   493,
		Width:    54,
		MaxSpeed: 0.1,
		CurSpeed: 0.05,
		CurPos:   objPos.NewR(20, 20),
		WakeHulls: []WakeHull{
			{Lateral: 0, Width: 54, Front: 0.25, Back: -0.45},
			{Lateral: 93.5, Width: 28, Front: 0.08, Back: -0.46},
			{Lateral: -93.5, Width: 28, Front: 0.08, Back: -0.46},
		},
	}
	trails := ship.GenTrails()
	if len(trails) != 6 {
		t.Fatalf("trimaran trails=%d, want 6", len(trails))
	}

	var xs []float64
	for i := 0; i < len(trails); i += 2 {
		xs = append(xs, trails[i].Pos.RX)
	}
	if xs[1] <= xs[0] || xs[2] >= xs[0] {
		t.Fatalf("hull wake X=%v, want center then starboard then port", xs)
	}

	mainStern := trails[1]
	sideStern := trails[3]
	fullShipBowLife := ship.Length/8 + 555*ship.CurSpeed
	sideWakeLength := ship.Length * (0.08 - (-0.46))
	if mainStern.Pos.RY < ship.CurPos.RY+ship.Length/constants.MapBlockSize*0.40 {
		t.Fatalf("main stern wake %s is still under the flight deck", mainStern.Pos.String())
	}
	if sideStern.CurSize != 28 {
		t.Fatalf("side stern size=%v, want widened 28", sideStern.CurSize)
	}
	if sideStern.CurLife >= fullShipBowLife || sideStern.CurLife != sideWakeLength/9+380*ship.CurSpeed {
		t.Fatalf("side life=%v, want hull-span life shorter than full ship %v", sideStern.CurLife, fullShipBowLife)
	}
}

func TestStoppedSwordfishResetsTrailAnimationStep(t *testing.T) {
	ship := &BattleShip{
		CurSpeed: 0,
		Animation: ShipAnimation{
			TopFrames:  []string{"01"},
			FrameTicks: 6,
		},
		AnimationAge:           12,
		LastTrailAnimationStep: 2,
	}
	ship.AdvanceAnimation(1)
	if ship.AnimationAge != 0 || ship.LastTrailAnimationStep != -1 {
		t.Fatalf("stopped animation state=%v/%d, want 0/-1", ship.AnimationAge, ship.LastTrailAnimationStep)
	}
}
