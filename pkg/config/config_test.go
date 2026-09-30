package config

import (
	"testing"

	"github.com/yosuke-furukawa/json5/encoding/json5"
)

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

func TestMissingAIStrategiesStayEnabled(t *testing.T) {
	settings := *NewDefaultGameSettings()
	if err := json5.Unmarshal([]byte(`{"language":"en"}`), &settings); err != nil {
		t.Fatalf("decode defaults: %v", err)
	}
	if settings.AI != DefaultAIStrategies() {
		t.Fatalf("missing strategies = %+v", settings.AI)
	}

	settings = *NewDefaultGameSettings()
	if err := json5.Unmarshal([]byte(`{"aiStrategies":{"attack":false,"scout":false}}`), &settings); err != nil {
		t.Fatalf("decode partial: %v", err)
	}
	if settings.AI.Attack || settings.AI.Scout || !settings.AI.Defend || !settings.AI.Counter || !settings.AI.Raid {
		t.Fatalf("partial strategies = %+v", settings.AI)
	}
}
