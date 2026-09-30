package settings

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/resources/font"
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

func TestClosedControlsShareTheWidestLabel(t *testing.T) {
	face := font.LanguageSelectorFace(22)
	got := closedControlWidth(face)
	want := int(text.Advance("标准", face)+0.5) + controlPadX
	if got != want {
		t.Fatalf("closed width = %d, want %d", got, want)
	}
}

func TestStrategySummaryListsEnabledStrategies(t *testing.T) {
	ui := &UI{localAI: config.AIStrategies{Attack: true, Raid: true}}
	if got := ui.strategySummary(); got != "2/5" {
		t.Fatalf("summary = %q", got)
	}
	ui.localAI = config.AIStrategies{}
	if got := ui.strategySummary(); got != "0/5" {
		t.Fatalf("empty summary = %q", got)
	}
}
