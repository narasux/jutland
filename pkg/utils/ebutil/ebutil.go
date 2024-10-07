package ebutil

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// NewImageWithColor 新建填充颜色的图片
func NewImageWithColor(width, height int, color color.Color) *ebiten.Image {
	img := ebiten.NewImage(width, height)
	img.Fill(color)
	return img
}

// 中心旋转（需要在最后的 GeoM.Translate 前执行，即先旋转，再位移）
func SetOptsCenterRotation(opts *ebiten.DrawImageOptions, img *ebiten.Image, rotation float64) {
	if rotation == 0 {
		return
	}

	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	opts.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	opts.GeoM.Rotate(rotation * math.Pi / 180)
	opts.GeoM.Translate(float64(w)/2, float64(h)/2)
}
