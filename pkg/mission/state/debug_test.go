package state

import "testing"

func TestDebugFlagsIsActiveWithHitBoxes(t *testing.T) {
	if !(DebugFlags{ShowHitBoxes: true}).IsActive() {
		t.Fatal("ShowHitBoxes should activate debug mode")
	}
	if (DebugFlags{}).IsActive() {
		t.Fatal("zero-value DebugFlags should be inactive")
	}
}
