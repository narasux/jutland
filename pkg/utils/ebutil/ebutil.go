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

// CalcTextWidth 计算文本宽度
func CalcTextWidth(text string, fontSize float64) float64 {
	// 字体原因，宽度大致是 0.4 的文字高度
	return fontSize * float64(len(text)) / 5 * 2
}

// UpdateOptsForCenterRotate 中心旋转（需要在最后的 GeoM.Translate 前执行，即先旋转，再位移）
func UpdateOptsForCenterRotate(opts *ebiten.DrawImageOptions, img *ebiten.Image, rotate int) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	opts.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	opts.GeoM.Rotate(float64(rotate) * math.Pi / 180)
	opts.GeoM.Translate(float64(w)/2, float64(h)/2)
}
