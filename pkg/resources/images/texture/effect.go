package texture

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// EffectCircleDiameter 命中特效粒子的模板直径，绘制时按世界尺寸缩放。
const EffectCircleDiameter = 64

var softCircleImg *ebiten.Image

func init() {
	// 命中特效走"运行时粒子"而不是序列帧：模板就是一个中心不透明、边缘平滑衰减的白色柔圆。
	// 和尾流（白色圆 + ColorScale 缩放）是同一套画法，所以水花看起来和尾流是一路货。
	// 之前的序列帧方案在 45-90 像素下会被看成"某个形状"，而且每帧重新随机、跳帧明显。
	softCircleImg = ebiten.NewImage(EffectCircleDiameter, EffectCircleDiameter)
	pixels := image.NewNRGBA(image.Rect(0, 0, EffectCircleDiameter, EffectCircleDiameter))
	center := float64(EffectCircleDiameter-1) / 2
	for y := 0; y < EffectCircleDiameter; y++ {
		for x := 0; x < EffectCircleDiameter; x++ {
			dist := math.Hypot(float64(x)-center, float64(y)-center) / center
			alpha := EffectCircleAlpha(dist)
			// WritePixels 要求预乘 alpha：白色 + 预乘就是 (a, a, a, a)
			value := uint8(alpha * 255)
			pixels.SetNRGBA(x, y, color.NRGBA{R: value, G: value, B: value, A: value})
		}
	}
	softCircleImg.WritePixels(pixels.Pix)
}

// EffectCircle 返回命中特效粒子用的白色柔圆模板。
func EffectCircle() *ebiten.Image { return softCircleImg }

// EffectCircleAlpha 返回柔圆模板在相对半径 dist（0 中心、1 边缘）处的透明度。
// 生成模板与离屏预览共用这一条曲线，保证预览与实际绘制一致。
func EffectCircleAlpha(dist float64) float64 {
	if dist >= 1 || dist < 0 {
		return 0
	}
	return math.Pow(1-dist, 1.7)
}
