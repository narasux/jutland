package state

import (
	"testing"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestCameraContainsMargin 验证相机外扩判定：margin=0 取闭区间，margin=n 时四周各扩 n 格。
// 场景：单位与特效裁剪依赖外扩判断，差一格会让贴边单位闪烁消失；相机取 Pos(10,20)、Width=5、Height=4。
// 预期：内部点与四边在内；(9,22) margin=0 外、margin=1 内；margin=1 时 16/19/25 与角 (9,19) 在内。
// 覆盖：15 个用例覆盖内部、四边、margin 界内外的 8/17/18/26 与外扩角，逐项断言 ContainsMargin。
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
