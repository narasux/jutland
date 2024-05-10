package ebutil

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/narasux/jutland/pkg/envs"
)

// DebugPrint 调试输出二次封装，支持全局禁用
func DebugPrint(screen *ebiten.Image, str string) {
	if envs.Debug {
		ebitenutil.DebugPrint(screen, str)
	}
}

// DebugPrintAt 调试输出二次封装，支持全局禁用
func DebugPrintAt(screen *ebiten.Image, str string, x, y int) {
	if envs.Debug {
		ebitenutil.DebugPrintAt(screen, str, x, y)
	}
}

// CalcTextWidth 计算文本宽度
func CalcTextWidth(text string, fontSize float64) float64 {
	// 字体原因，宽度大致是 0.35 的文字高度
	return fontSize * float64(len(text)) / 20 * 7
}

// SetOptsCenterRotation 中心旋转（需要在最后的 GeoM.Translate 前执行，即先旋转，再位移）
func SetOptsCenterRotation(opts *ebiten.DrawImageOptions, img *ebiten.Image, rotation float64) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	opts.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	opts.GeoM.Rotate(rotation * math.Pi / 180)
	opts.GeoM.Translate(float64(w)/2, float64(h)/2)
}

func angleFromNorthClockwise(x1, y1, x2, y2 float64) float64 {
	// 将角度从 x 轴正向的顺时针转换至 y 轴正向的顺时针
	return math.Mod(math.Atan2(y2-y1, x2-x1)*180/math.Pi+90, 360)
}

// DrawLineOnScreen 在屏幕上绘制从 (x1, y1) 到 (x2, y2) 的直线
// FIXME 这个函数没有达到预期
func DrawLineOnScreen(screen *ebiten.Image, x1, y1, x2, y2, thickness int, lineColor color.Color) {
	if x1 == x2 && y1 == y2 {
		return
	}
	w := x2 - x1
	h := y2 - y1
	length := math.Sqrt(float64(w*w + h*h))
	rotation := angleFromNorthClockwise(float64(x1), float64(y1), float64(x2), float64(y2))
	log.Println("x1", x1, "y1", y1, "x2", x2, "y2", y2)
	log.Println("w", w, "h", h, "length", length, "rotation", rotation)
	DebugPrint(screen, fmt.Sprintf("\n\nlength: %f, rotation: %f", length, rotation))

	img := ebiten.NewImage(thickness, int(length))
	img.Fill(lineColor)

	opts := &ebiten.DrawImageOptions{}
	SetOptsCenterRotation(opts, img, rotation)
	opts.GeoM.Translate(float64(x1), float64(y1))
	screen.DrawImage(img, opts)
}

// DrawVerticalLineOnScreen 在屏幕上绘制从 (x, y1) 到 (x, y2) 的垂直直线
// TODO DrawLineOnScreen 实现后迁移
func DrawVerticalLineOnScreen(screen *ebiten.Image, x, y1, y2, thickness int, lineColor color.Color) {
	if y1 == y2 {
		return
	}

	img := ebiten.NewImage(thickness, y2-y1)
	img.Fill(lineColor)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(x), float64(y1))
	screen.DrawImage(img, opts)
}

// DrawHorizontalLineOnScreen 在屏幕上绘制从 (x1, y) 到 (x2, y) 的水平直线
// TODO DrawLineOnScreen 实现后迁移
func DrawHorizontalLineOnScreen(screen *ebiten.Image, x1, x2, y, thickness int, lineColor color.Color) {
	if x1 == x2 {
		return
	}

	img := ebiten.NewImage(x2-x1, thickness)
	img.Fill(lineColor)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(x1), float64(y))
	screen.DrawImage(img, opts)
}
