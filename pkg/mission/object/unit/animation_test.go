package unit

import "testing"

func TestShipAnimationSequenceAndIdle(t *testing.T) {
	ship := &BattleShip{
		Name: "swordfish", CurSpeed: 1,
		Animation: ShipAnimation{
			TopFrames:    []string{"01", "02", "03", "04", "05", "04", "03", "02"},
			IdleTopFrame: "03", FrameTicks: 6,
		},
	}
	if got := ship.CurrentTopImageName(); got != "01" {
		t.Fatalf("initial frame=%s", got)
	}
	ship.AnimationAge = 24
	if got := ship.CurrentTopImageName(); got != "05" {
		t.Fatalf("middle frame=%s", got)
	}
	ship.CurSpeed = 0
	if got := ship.CurrentTopImageName(); got != "03" {
		t.Fatalf("idle frame=%s", got)
	}
}
