package mark

import (
	"image/color"
	"testing"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

func TestDamageFlagTextKeepsSmallDamageVisible(t *testing.T) {
	if got := DamageFlagText(12.9); got != "12" {
		t.Fatalf("damage text = %q, want 12", got)
	}
	if got := DamageFlagText(0.4); got != "0.40" {
		t.Fatalf("damage text = %q, want 0.40", got)
	}
	mark := NewText(objPos.New(1, 2), DamageFlagText(12.9), 16, color.White, 20)
	if mark.Text != "12" || mark.Img != nil {
		t.Fatalf("mark = %+v, want integer text and no cached image", mark)
	}
}
