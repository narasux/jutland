package drawer

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/action"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/colorx"
	"github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

// 绘制箭头（当鼠标悬浮触发镜头移动时）
func (d *Drawer) drawArrowOnMapWhenHover(screen *ebiten.Image, ms *state.MissionState) {
	posX, posY, rotation, padding := 0, 0, 0.0, 10
	imgW, imgH := texture.ArrowWhiteImg.Bounds().Dx(), texture.ArrowWhiteImg.Bounds().Dy()
	switch action.DetectCursorHoverOnGameMap(ms.Layout) {
	case action.HoverScreenLeft:
		posX, posY, rotation = padding, ms.Layout.Camera.Height/2, 180
	case action.HoverScreenRight:
		posX, posY, rotation = ms.Layout.Camera.Width-imgW-padding, ms.Layout.Camera.Height/2, 0
	case action.HoverScreenTop:
		posX, posY, rotation = ms.Layout.Camera.Width/2, padding, 270
	case action.HoverScreenBottom:
		posX, posY, rotation = ms.Layout.Camera.Width/2, ms.Layout.Camera.Height-imgH-padding, 90
	case action.HoverScreenTopLeft:
		posX, posY, rotation = padding, padding, 225
	case action.HoverScreenTopRight:
		posX, posY, rotation = ms.Layout.Camera.Width-imgW-padding, padding, 315
	case action.HoverScreenBottomLeft:
		posX, posY, rotation = padding, ms.Layout.Camera.Height-imgH-padding, 135
	case action.HoverScreenBottomRight:
		posX, posY, rotation = ms.Layout.Camera.Width-imgW-padding, ms.Layout.Camera.Height-imgH-padding, 45
	default:
		return
	}

	opts := d.genDefaultDrawImageOptions()
	ebutil.SetOptsCenterRotation(opts, texture.ArrowWhiteImg, rotation)
	opts.GeoM.Translate(float64(posX), float64(posY))
	screen.DrawImage(texture.ArrowWhiteImg, opts)
}

// 绘制选择框
func (d *Drawer) drawSelectedArea(screen *ebiten.Image, ms *state.MissionState) {
	area := action.DetectCursorSelectArea(ms)
	if area == nil || !area.Selecting {
		return
	}
	x1, y1 := min(area.StartX, area.CurX), min(area.StartY, area.CurY)
	x2, y2 := max(area.StartX, area.CurX), max(area.StartY, area.CurY)

	ebutil.DrawVerticalLineOnScreen(screen, x1, y1, y2, 2, colorx.White)
	ebutil.DrawVerticalLineOnScreen(screen, x2, y1, y2, 2, colorx.White)
	ebutil.DrawHorizontalLineOnScreen(screen, x1, x2, y1, 2, colorx.White)
	ebutil.DrawHorizontalLineOnScreen(screen, x1, x2, y2, 2, colorx.White)
}
