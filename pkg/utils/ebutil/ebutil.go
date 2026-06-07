package ebutil

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

// TrimLineSegment 按起点和终点留白裁剪线段
func TrimLineSegment(
	startX, startY, endX, endY float64,
	startGap, endGap float64,
) (float64, float64, float64, float64, bool) {
	dx := endX - startX
	dy := endY - startY
	totalDist := math.Sqrt(dx*dx + dy*dy)
	if totalDist <= startGap+endGap {
		return 0, 0, 0, 0, false
	}

	ux, uy := dx/totalDist, dy/totalDist
	return startX + ux*startGap,
		startY + uy*startGap,
		endX - ux*endGap,
		endY - uy*endGap,
		true
}

// DrawCrossMarker 绘制以 x, y 为中心的十字标记
func DrawCrossMarker(screen *ebiten.Image, x, y, crossLen, strokeWidth float64, clr color.Color) {
	vector.StrokeLine(
		screen,
		float32(x-crossLen),
		float32(y),
		float32(x+crossLen),
		float32(y),
		float32(strokeWidth),
		clr,
		false,
	)
	vector.StrokeLine(
		screen,
		float32(x),
		float32(y-crossLen),
		float32(x),
		float32(y+crossLen),
		float32(strokeWidth),
		clr,
		false,
	)
}

// DrawFlagMarker 绘制以 x, y 为旗杆底部的旗帜标记
func DrawFlagMarker(screen *ebiten.Image, x, y, poleHeight float64, clr color.Color) {
	poleX := float32(x)
	poleTopY := float32(y - poleHeight)
	poleBottomY := float32(y)
	flagW := float32(poleHeight * 0.75)
	flagH := float32(poleHeight * 0.5)
	outline := color.RGBA{R: 2, G: 8, B: 10, A: 190}

	vector.StrokeLine(screen, poleX, poleTopY, poleX, poleBottomY, 7, outline, false)
	drawFlagFace(screen, poleX-2, poleTopY-2, flagW+4, flagH+4, outline)

	vector.StrokeLine(screen, poleX, poleTopY, poleX, poleBottomY, 3, clr, false)
	drawFlagFace(screen, poleX, poleTopY, flagW, flagH, clr)
}

func drawFlagFace(screen *ebiten.Image, x, y, width, height float32, clr color.Color) {
	var path vector.Path
	path.MoveTo(x, y)
	path.LineTo(x+width, y+height*0.45)
	path.LineTo(x, y+height)
	path.Close()

	opts := &vector.DrawPathOptions{}
	opts.ColorScale.ScaleWithColor(clr)
	vector.FillPath(screen, &path, nil, opts)
}
