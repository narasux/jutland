package cheat

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/state"
)

func TestShowHitBoxes(t *testing.T) {
	cheat := &ShowHitBoxes{}
	if !cheat.Match("SHOW hitboxes") {
		t.Fatal("ShowHitBoxes should ignore case and whitespace")
	}

	misState := &state.MissionState{}
	if got := cheat.Exec(misState); got != "Toggled hit boxes: on" {
		t.Fatalf("unexpected enable output: %q", got)
	}
	if !misState.UI.DebugFlags.ShowHitBoxes {
		t.Fatal("ShowHitBoxes should enable the hit-box overlay")
	}

	if got := cheat.Exec(misState); got != "Toggled hit boxes: off" {
		t.Fatalf("unexpected disable output: %q", got)
	}
	if misState.UI.DebugFlags.ShowHitBoxes {
		t.Fatal("ShowHitBoxes should disable the hit-box overlay")
	}
}

func TestDebugAllEnablesHitBoxes(t *testing.T) {
	misState := &state.MissionState{}
	(&DebugAll{}).Exec(misState)

	if !misState.UI.DebugFlags.ShowHitBoxes {
		t.Fatal("DebugAll should enable the hit-box overlay")
	}
}

func TestShowAirfieldRunway(t *testing.T) {
	cheat := &ShowAirfieldRunway{}
	if !cheat.Match("show AIRFIELD runway") {
		t.Fatal("ShowAirfieldRunway should ignore case")
	}

	misState := &state.MissionState{}
	if got := cheat.Exec(misState); got != "Toggled show airfield runway: on" {
		t.Fatalf("unexpected enable output: %q", got)
	}
	if !misState.UI.DebugFlags.ShowAirfieldRunway {
		t.Fatal("ShowAirfieldRunway should enable the runway overlay")
	}

	if got := cheat.Exec(misState); got != "Toggled show airfield runway: off" {
		t.Fatalf("unexpected disable output: %q", got)
	}
	if misState.UI.DebugFlags.ShowAirfieldRunway {
		t.Fatal("ShowAirfieldRunway should disable the runway overlay")
	}
}

func TestDebugAllEnablesAirfieldRunway(t *testing.T) {
	misState := &state.MissionState{}
	(&DebugAll{}).Exec(misState)

	if !misState.UI.DebugFlags.ShowAirfieldRunway {
		t.Fatal("DebugAll should enable the airfield runway overlay")
	}
}
