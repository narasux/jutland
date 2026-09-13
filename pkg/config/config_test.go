package config

import "testing"

func TestNormalizeSpeedMultiplierKeepsThreePresets(t *testing.T) {
	for _, tc := range []struct {
		value float64
		want  float64
	}{
		{value: 0.1, want: SpeedSlowMultiplier},
		{value: SpeedSlowMultiplier, want: SpeedSlowMultiplier},
		{value: SpeedStandardMultiplier, want: SpeedStandardMultiplier},
		{value: SpeedFastMultiplier, want: SpeedFastMultiplier},
		{value: 4.0, want: SpeedFastMultiplier},
	} {
		if got := normalizeSpeedMultiplier(tc.value); got != tc.want {
			t.Fatalf("normalizeSpeedMultiplier(%v) = %v, want %v", tc.value, got, tc.want)
		}
	}
}
