package texture

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/utils/colorx"
)

const (
	// TrailCircleDiameter 是共用圆形尾迹模板的直径。绘制时再按 CurSize 缩放。
	TrailCircleDiameter = 32
	// 矩形模板宽 1、高 30，只涂下半段，使居中旋转后色块从弹丸向后伸出 15 倍宽度。
	trailRectWidth       = 1
	trailRectImageHeight = 30
)

var (
	trailCircle     *ebiten.Image
	trailRect       *ebiten.Image
	trailRectPixels *image.NRGBA
)

func init() {
	trailCircle = ebiten.NewImage(TrailCircleDiameter, TrailCircleDiameter)
	radius := float32(TrailCircleDiameter) / 2
	vector.DrawFilledCircle(trailCircle, radius, radius, radius, colorx.White, false)

	// 下半段纯白，上半段保持透明。用普通图片生成，避免 1 像素矩形被抗锯齿涂脏。
	trailRectPixels = image.NewNRGBA(image.Rect(0, 0, trailRectWidth, trailRectImageHeight))
	for y := trailRectImageHeight / 2; y < trailRectImageHeight; y++ {
		trailRectPixels.SetNRGBA(0, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	}
	trailRect = ebiten.NewImageFromImage(trailRectPixels)
}

type TrailShape int

const (
	// TrailShapeCircle 圆形尾流
	TrailShapeCircle TrailShape = iota
	// TrailShapeRect 矩形尾流（长度 = 宽度 15 倍，色块只占图片下半）
	TrailShapeRect
)

// TrailImage 返回该形状共用的白色模板。
func TrailImage(shape TrailShape) *ebiten.Image {
	switch shape {
	case TrailShapeCircle:
		return trailCircle
	case TrailShapeRect:
		return trailRect
	default:
		return nil
	}
}

// TrailDrawScale 把尾迹当前尺寸换成相对白色模板的缩放。
func TrailDrawScale(shape TrailShape, size float64) float64 {
	if shape == TrailShapeCircle {
		return size / float64(TrailCircleDiameter)
	}
	return size / float64(trailRectWidth)
}

// TrailAlphaByte 对齐原来烤进贴图的 uint8(life)，包括超过 255 时的截断。
func TrailAlphaByte(life float64) uint8 {
	return uint8(life)
}

// TrailColorScale 返回绘制时的预乘颜色。nil 颜色按白色。
// ColorScale 乘在已经预乘过的像素上，RGB 不随透明度一起缩小的话，尾迹会一直是实心白。
func TrailColorScale(clr color.Color, life float64) (red, green, blue, alpha float32) {
	if clr == nil {
		clr = colorx.White
	}
	r, g, b, _ := clr.RGBA()
	alpha = float32(TrailAlphaByte(life)) / 255
	return float32(r) / 65535 * alpha, float32(g) / 65535 * alpha, float32(b) / 65535 * alpha, alpha
}
