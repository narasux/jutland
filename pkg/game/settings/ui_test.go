package settings

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
)

func TestSpeedOptionsOnlyExposeThreePresets(t *testing.T) {
	want := []float64{
		config.SpeedSlowMultiplier,
		config.SpeedStandardMultiplier,
		config.SpeedFastMultiplier,
	}
	if len(speedOptions) != len(want) {
		t.Fatalf("speed option count = %d, want %d", len(speedOptions), len(want))
	}
	for idx, value := range want {
		if speedOptions[idx].Value != value {
			t.Fatalf("speed option %d = %v, want %v", idx, speedOptions[idx].Value, value)
		}
	}
}
