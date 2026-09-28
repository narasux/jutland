package state

import (
	"testing"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

func TestCameraContainsMargin(t *testing.T) {
	cam := &Camera{Pos: objPos.New(10, 20), Width: 5, Height: 4}

	tests := []struct {
		name   string
		pos    objPos.MapPos
		margin int
		want   bool
	}{
		{name: "inside without margin", pos: objPos.New(12, 22), margin: 0, want: true},
		{name: "left edge", pos: objPos.New(10, 22), margin: 0, want: true},
		{name: "right edge", pos: objPos.New(15, 22), margin: 0, want: true},
		{name: "top edge", pos: objPos.New(12, 20), margin: 0, want: true},
		{name: "bottom edge", pos: objPos.New(12, 24), margin: 0, want: true},
		{name: "one cell left is outside", pos: objPos.New(9, 22), margin: 0, want: false},
		{name: "one cell left is inside the margin", pos: objPos.New(9, 22), margin: 1, want: true},
		{name: "one cell right is inside the margin", pos: objPos.New(16, 22), margin: 1, want: true},
		{name: "one cell above is inside the margin", pos: objPos.New(12, 19), margin: 1, want: true},
		{name: "one cell below is inside the margin", pos: objPos.New(12, 25), margin: 1, want: true},
		{name: "corner of the margin", pos: objPos.New(9, 19), margin: 1, want: true},
		{name: "two cells left is outside the margin", pos: objPos.New(8, 22), margin: 1, want: false},
		{name: "two cells right is outside the margin", pos: objPos.New(17, 22), margin: 1, want: false},
		{name: "two cells above is outside the margin", pos: objPos.New(12, 18), margin: 1, want: false},
		{name: "two cells below is outside the margin", pos: objPos.New(12, 26), margin: 1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cam.ContainsMargin(tt.pos, tt.margin); got != tt.want {
				t.Fatalf("ContainsMargin(%v, %d) = %v, want %v", tt.pos, tt.margin, got, tt.want)
			}
		})
	}
}
