package texture

import (
	"image/color"
	"testing"
)

// TestTrailImagesAreSharedTemplates 验证尾流用两张全局共享的白色模板，尺寸和像素分布固定。
// 场景：每发炮弹都取尾流贴图；每次新建图片会吃显存，模板尺寸变化也会让尾迹失真。
// 预期：两次 TrailImage 返回同一指针；圆形 32×32；矩形 1×30，顶部透明、底部不透明白；
// 失败表示绘制时会重复建图或尾迹方向/长度约定被改坏。
// 覆盖：比较两张模板的指针与 Bounds，再读 trailRectPixels 的 (0,0)/(0,29) 断言 alpha。
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

// TestTrailAlphaByteMatchesUint8Life 验证尾流透明度按 uint8(life) 截断，并把 alpha 预乘到 RGB。
// 场景：尾流寿命是浮点（如 100.9），绘制层用 0..1 的 ColorScale，超 255 时按 uint8 转换。
// 预期：TrailAlphaByte(100.9)=100，300.0 与 uint8(300.0) 一致；TrailColorScale(nil,100.9) 的
// alpha=100/255，红绿蓝都等于 alpha（白色预乘）；失败表示尾流偏亮/偏暗或颜色未随透明度衰减。
// 覆盖：断言 TrailAlphaByte(100.9)=100 与超界回绕，再检查 TrailColorScale 的四个分量。
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
