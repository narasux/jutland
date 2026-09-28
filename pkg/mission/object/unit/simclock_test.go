package unit

import (
	"testing"
	"time"

	"github.com/narasux/jutland/pkg/config"
)

func TestReloadTicksFollowsSpeedAndIgnoresWallClock(t *testing.T) {
	if config.G == nil {
		config.G = config.NewDefaultGameSettings()
	}
	prev := config.G.SpeedMultiplier
	t.Cleanup(func() { config.G.SpeedMultiplier = prev })

	config.G.SpeedMultiplier = 1
	if got := ReloadTicks(1); got != 60 {
		t.Fatalf("1x reload ticks = %d, want 60", got)
	}
	config.G.SpeedMultiplier = 2
	if got := ReloadTicks(1); got != 30 {
		t.Fatalf("2x reload ticks = %d, want 30", got)
	}

	config.G.SpeedMultiplier = 1
	SetSimTick(10)
	gun := &Gun{ReloadTime: 1, ReloadStartTick: 10}
	time.Sleep(20 * time.Millisecond)
	if gun.Reloaded() {
		t.Fatal("wall clock advanced but gun reloaded before 60 ticks")
	}
	SetSimTick(69)
	if gun.Reloaded() {
		t.Fatal("gun reloaded one tick early")
	}
	SetSimTick(70)
	if !gun.Reloaded() {
		t.Fatal("gun still reloading after 60 ticks")
	}
}
