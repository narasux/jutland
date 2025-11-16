package utils

import "github.com/hajimehoshi/ebiten/v2"

// GenZoomImages 生成缩放后的图片（批量处理）
func GenZoomImages[T int | string](source map[T]*ebiten.Image, arcZoom float64) map[T]*ebiten.Image {
	target := make(map[T]*ebiten.Image, len(source))

	for name, img := range source {
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		opts.GeoM.Scale(1/arcZoom, 1/arcZoom)

		width := int(float64(img.Bounds().Dx())/arcZoom) + 1
		height := int(float64(img.Bounds().Dy())/arcZoom) + 1
		zoomImg := ebiten.NewImage(width, height)
		zoomImg.DrawImage(img, opts)
		target[name] = zoomImg
	}

	return target
}
