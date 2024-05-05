package drawer

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/narasux/jutland/pkg/mission/action"
	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/colorx"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/resources/images/mapblock"
	"github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

// Drawer 任务运行中图像绘制器
type Drawer struct{}

// NewDrawer ...
func NewDrawer() *Drawer {
	return &Drawer{}
}

// Draw 绘制任务关卡图像
func (d *Drawer) Draw(screen *ebiten.Image, misState *state.MissionState) {
	// 相机视野
	d.drawCameraView(screen, misState)
	// 地图元素
	d.drawBuildings(screen, misState)
	d.drawBattleShips(screen, misState)
	d.drawShotBullets(screen, misState)
	// 控制台
	d.drawConsole(screen, misState)
	// 用户行为
	d.drawArrowOnMapWhenHover(screen, misState)
	// FIXME 绘制选择框有 bug，需要先注释掉
	// d.drawSelectArea(screen, misState)
	d.drawTips(screen, misState)
}

// 绘制相机视野
func (d *Drawer) drawCameraView(screen *ebiten.Image, ms *state.MissionState) {
	for x := 0; x < ms.Camera.Width; x++ {
		for y := 0; y < ms.Camera.Height; y++ {
			opts := d.genDefaultDrawImageOptions()
			opts.GeoM.Translate(float64(x)*mapblock.BlockSize, float64(y)*mapblock.BlockSize)

			mapX, mapY := ms.Camera.Pos.MX+x, ms.Camera.Pos.MY+y
			char := ms.MissionMD.MapCfg.Map.Get(mapX, mapY)
			screen.DrawImage(mapblock.GetByCharAndPos(char, mapX, mapY), opts)
			// FIXME 去除该 debug
			//ebitenutil.DebugPrintAt(
			//	screen, fmt.Sprintf("\n(%d,%d)", mapX, mapY), x*mapblock.BlockSize, y*mapblock.BlockSize,
			//)
		}
	}
}

// 绘制控制台
func (d *Drawer) drawConsole(screen *ebiten.Image, ms *state.MissionState) {
	// 纯银色控制台
	consoleBGImg := ebiten.NewImage(ms.Layout.Console.Width, ms.Layout.Console.Height)
	consoleBGImg.Fill(colorx.DarkSilver)
	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(
		float64(ms.Layout.Console.TopLeft.SX),
		float64(ms.Layout.Console.TopLeft.SY),
	)
	screen.DrawImage(consoleBGImg, opts)

	// 小地图
	opts = d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(
		float64(ms.Layout.Console.SmallMap.TopLeft.SX),
		float64(ms.Layout.Console.SmallMap.TopLeft.SY),
	)
	// FIXME 目前先丢一个黑图片进去，将来应该是考虑地图，战舰，视野等等，渲染一个实时图片
	smallMapImg := ebiten.NewImage(ms.Layout.Console.SmallMap.Width, ms.Layout.Console.SmallMap.Height)
	smallMapImg.Fill(colorx.Black)
	screen.DrawImage(smallMapImg, opts)

	// 菜单
	opts = d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(
		float64(ms.Layout.Console.Menu.TopLeft.SX),
		float64(ms.Layout.Console.Menu.TopLeft.SY),
	)
	// FIXME 目前先丢一个银色图片进去，将来再考虑按钮之类的
	menuImg := ebiten.NewImage(ms.Layout.Console.Menu.Width, ms.Layout.Console.Menu.Height)
	menuImg.Fill(colorx.Silver)
	screen.DrawImage(menuImg, opts)
}

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
func (d *Drawer) drawSelectArea(screen *ebiten.Image, ms *state.MissionState) {
	area := action.DetectCursorSelectArea(ms)
	if !area.Selecting {
		return
	}
	x1, y1 := min(area.StartX, area.CurX), min(area.StartY, area.CurY)
	x2, y2 := max(area.StartX, area.CurX), max(area.StartY, area.CurY)

	ebutil.DrawLineOnScreen(screen, x1, y1, x1, y2, 2, colorx.White)
	ebutil.DrawLineOnScreen(screen, x1, y1, x2, y1, 2, colorx.White)
	ebutil.DrawLineOnScreen(screen, x1, y2, x2, y2, 2, colorx.White)
	ebutil.DrawLineOnScreen(screen, x2, y1, x2, y2, 2, colorx.White)
}

// 绘制建筑物
func (d *Drawer) drawBuildings(screen *ebiten.Image, ms *state.MissionState) {
}

// 绘制战舰
func (d *Drawer) drawBattleShips(screen *ebiten.Image, ms *state.MissionState) {
	for _, ship := range ms.Ships {
		ebitenutil.DebugPrint(screen,
			fmt.Sprintf("\n\nship.MX: %d, ship.MY: %d, ship.RX: %f, ship.RY: %f\nspeed: %f, rotation: %f",
				ship.CurPos.MX, ship.CurPos.MY,
				ship.CurPos.RX, ship.CurPos.RY,
				ship.CurSpeed, ship.CurRotation,
			))
		// 只有在屏幕中的才渲染
		if ship.CurPos.MX < ms.Camera.Pos.MX ||
			ship.CurPos.MX > ms.Camera.Pos.MX+ms.Camera.Width ||
			ship.CurPos.MY < ms.Camera.Pos.MY ||
			ship.CurPos.MY > ms.Camera.Pos.MY+ms.Camera.Height {
			continue
		}

		shipImg := obj.GetShipImg(ship.Name)
		opts := d.genDefaultDrawImageOptions()
		ebutil.SetOptsCenterRotation(opts, shipImg, ship.CurRotation)
		opts.GeoM.Translate(
			(ship.CurPos.RX-float64(ms.Camera.Pos.MX))*mapblock.BlockSize,
			(ship.CurPos.RY-float64(ms.Camera.Pos.MY))*mapblock.BlockSize,
		)
		screen.DrawImage(shipImg, opts)

		// TODO 绘制尾流（速度不同，尺寸不同，尾流不同？）
	}
}

// 绘制已发射的弹丸
func (d *Drawer) drawShotBullets(screen *ebiten.Image, ms *state.MissionState) {
}

// 绘制提示语
func (d *Drawer) drawTips(screen *ebiten.Image, ms *state.MissionState) {
	if ms.MissionStatus == state.MissionPaused {
		textStr, fontSize := "按下 Q 退出，按下 Enter 继续", float64(64)
		posX := (float64(ms.Layout.Camera.Width) - ebutil.CalcTextWidth(textStr, fontSize)) / 2
		posY := float64(ms.Layout.Height) / 2
		d.drawText(screen, textStr, posX, posY, fontSize, font.Hang, colorx.White)
	}
}

// 绘制文本
func (d *Drawer) drawText(
	screen *ebiten.Image,
	textStr string,
	posX, posY, fontSize float64,
	textFont *text.GoTextFaceSource,
	textColor color.Color,
) {
	opts := &text.DrawOptions{}
	opts.GeoM.Translate(posX, posY)
	opts.ColorScale.ScaleWithColor(textColor)
	textFace := text.GoTextFace{
		Source: textFont,
		Size:   fontSize,
	}
	text.Draw(screen, textStr, &textFace, opts)
}

func (d *Drawer) genDefaultDrawImageOptions() *ebiten.DrawImageOptions {
	return &ebiten.DrawImageOptions{
		// 线性过滤：通过计算周围像素的加权平均值来进行插值，可使得边缘 & 色彩转换更加自然
		Filter: ebiten.FilterLinear,
	}
}
