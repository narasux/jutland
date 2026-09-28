package texture

import (
	"image/color"
	"testing"
)

func TestTrailImagesAreSharedTemplates(t *testing.T) {
	circle := TrailImage(TrailShapeCircle)
	rect := TrailImage(TrailShapeRect)
	if circle == nil || rect == nil {
		t.Fatal("missing trail template")
	}
	if TrailImage(TrailShapeCircle) != circle || TrailImage(TrailShapeRect) != rect {
		t.Fatal("trail templates are not shared")
	}
	if circle.Bounds().Dx() != TrailCircleDiameter || circle.Bounds().Dy() != TrailCircleDiameter {
		t.Fatalf("circle size = %v, want %d", circle.Bounds().Size(), TrailCircleDiameter)
	}
	if rect.Bounds().Dx() != 1 || rect.Bounds().Dy() != 30 {
		t.Fatalf("rect size = %v, want 1x30", rect.Bounds().Size())
	}

	top := trailRectPixels.NRGBAAt(0, 0)
	if top.A != 0 {
		t.Fatalf("rect top alpha = %d, want 0", top.A)
	}
	bottom := trailRectPixels.NRGBAAt(0, 29)
	if bottom != (color.NRGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("rect bottom = %+v, want opaque white", bottom)
	}
}

func TestTrailAlphaByteMatchesUint8Life(t *testing.T) {
	if TrailAlphaByte(100.9) != 100 {
		t.Fatalf("alpha byte = %d, want 100", TrailAlphaByte(100.9))
	}
	overflow := 300.0
	if TrailAlphaByte(overflow) != uint8(overflow) {
		t.Fatalf("alpha byte = %d, want %d", TrailAlphaByte(overflow), uint8(overflow))
	}
	red, green, blue, alpha := TrailColorScale(nil, 100.9)
	if alpha != float32(100)/255 {
		t.Fatalf("color scale alpha = %v, want %v", alpha, float32(100)/255)
	}
	if red != alpha || green != alpha || blue != alpha {
		t.Fatalf("premultiplied white = (%v,%v,%v), want alpha %v", red, green, blue, alpha)
	}
}
