package drawer

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/action"
	"github.com/narasux/jutland/pkg/mission/state"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

// 绘制箭头（当鼠标悬浮触发镜头移动时）
func (d *Drawer) drawArrowOnMapWhenHover(screen *ebiten.Image, ms *state.MissionState) {
	posX, posY, rotation, padding := 0, 0, 0.0, 10
	imgW, imgH := textureImg.ArrowWhite.Bounds().Dx(), textureImg.ArrowWhite.Bounds().Dy()
	switch action.DetectCursorHoverOnGameMap(ms.Layout) {
	case action.HoverScreenLeft:
		posX, posY, rotation = padding, ms.Layout.Height/2, 180
	case action.HoverScreenRight:
		posX, posY, rotation = ms.Layout.Width-imgW-padding, ms.Layout.Height/2, 0
	case action.HoverScreenTop:
		posX, posY, rotation = ms.Layout.Width/2, padding, 270
	case action.HoverScreenBottom:
		posX, posY, rotation = ms.Layout.Width/2, ms.Layout.Height-imgH-padding, 90
	case action.HoverScreenTopLeft:
		posX, posY, rotation = padding, padding, 225
	case action.HoverScreenTopRight:
		posX, posY, rotation = ms.Layout.Width-imgW-padding, padding, 315
	case action.HoverScreenBottomLeft:
		posX, posY, rotation = padding, ms.Layout.Height-imgH-padding, 135
	case action.HoverScreenBottomRight:
		posX, posY, rotation = ms.Layout.Width-imgW-padding, ms.Layout.Height-imgH-padding, 45
	default:
		return
	}

	opts := d.genDefaultDrawImageOptions()
	ebutil.SetOptsCenterRotation(opts, textureImg.ArrowWhite, rotation)
	opts.GeoM.Translate(float64(posX), float64(posY))
	screen.DrawImage(textureImg.ArrowWhite, opts)
}

// 绘制选择框
func (d *Drawer) drawSelectedArea(screen *ebiten.Image, ms *state.MissionState) {
	area := action.DetectCursorSelectArea(ms)
	if area == nil || !ms.IsAreaSelecting {
		return
	}
	x1, y1 := float32(min(area.StartX, area.CurX)), float32(min(area.StartY, area.CurY))
	x2, y2 := float32(max(area.StartX, area.CurX)), float32(max(area.StartY, area.CurY))

	vector.StrokeRect(screen, x1, y1, x2-x1, y2-y1, 2, colorx.White, false)
}

// 绘制标识
func (d *Drawer) drawMarks(screen *ebiten.Image, ms *state.MissionState) {
	for _, mark := range ms.GameMarks {
		opts := d.genDefaultDrawImageOptions()
		opts.GeoM.Translate(
			(mark.Pos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-float64(mark.Img.Bounds().Dx()/2),
			(mark.Pos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-float64(mark.Img.Bounds().Dy()/2),
		)
		screen.DrawImage(mark.Img, opts)
	}
}
