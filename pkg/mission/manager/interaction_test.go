package manager

import "testing"

func TestConsumeZoomInputLimitsMouseWheelToOneStep(t *testing.T) {
	manager := &MissionManager{}

	if direction := manager.consumeZoomInput(12, 0); direction != 1 {
		t.Fatalf("first wheel direction = %d, want 1", direction)
	}
	for frame := 1; frame < wheelZoomCooldownTicks; frame++ {
		if direction := manager.consumeZoomInput(12, 0); direction != 0 {
			t.Fatalf("wheel direction during cooldown at frame %d = %d, want 0", frame, direction)
		}
	}
	if direction := manager.consumeZoomInput(12, 0); direction != 1 {
		t.Fatalf("wheel direction after cooldown = %d, want 1", direction)
	}
}

func TestConsumeZoomInputAccumulatesTrackpadMagnification(t *testing.T) {
	manager := &MissionManager{}

	if direction := manager.consumeZoomInput(0, 0.3); direction != 0 {
		t.Fatalf("first pinch direction = %d, want 0", direction)
	}
	if direction := manager.consumeZoomInput(0, 0.3); direction != 1 {
		t.Fatalf("accumulated pinch direction = %d, want 1", direction)
	}
}
