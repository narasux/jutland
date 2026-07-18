package unit

import (
	"testing"

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
