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

// CalcTextWidth 计算文本宽度
func CalcTextWidth(text string, fontSize float64) float64 {
	// 字体原因，宽度大致是 0.4 的文字高度
	return fontSize * float64(len(text)) / 5 * 2
}

// SetOptsCenterRotation 中心旋转（需要在最后的 GeoM.Translate 前执行，即先旋转，再位移）
func SetOptsCenterRotation(opts *ebiten.DrawImageOptions, img *ebiten.Image, rotation float64) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	opts.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	opts.GeoM.Rotate(rotation * math.Pi / 180)
	opts.GeoM.Translate(float64(w)/2, float64(h)/2)
}

func angleFromNorthClockwise(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	rad := math.Atan2(dx, dy)    // 计算向量的角度（逆时针方向从 x 轴到向量）
	deg := rad * (180 / math.Pi) // 转换为角度
	if deg < 0 {
		deg = 360 + deg // 如果角度为负，调整为 0 到 360 之间
	}
	// 将角度从 x 轴正向的顺时针转换至 y 轴正向的顺时针
	return math.Mod(450-deg, 360)
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
	rotation := angleFromNorthClockwise(float64(x1), float64(y1), float64(x2), float64(y2)) + 90
	log.Println("x1", x1, "y1", y1, "x2", x2, "y2", y2)
	log.Println("w", w, "h", h, "length", length, "rotation", rotation)
	DebugPrint(screen, fmt.Sprintf("\n\nlength: %f, rotation: %f", length, rotation))

	img := ebiten.NewImage(thickness, int(length))
	img.Fill(lineColor)

	opts := &ebiten.DrawImageOptions{}
	SetOptsCenterRotation(opts, img, rotation)
	opts.GeoM.Translate(float64(x1+x2)/2, float64(y1+y2)/2)
	screen.DrawImage(img, opts)
}
