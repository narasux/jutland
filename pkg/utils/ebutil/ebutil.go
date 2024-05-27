package ebutil

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// NewImageWithColor 新建填充颜色的图片
func NewImageWithColor(width, height int, color color.Color) *ebiten.Image {
	img := ebiten.NewImage(width, height)
	img.Fill(color)
	return img
}
