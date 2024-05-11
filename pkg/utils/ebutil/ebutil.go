package ebutil

import (
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
