package mission

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	obj "github.com/narasux/jutland/pkg/object"
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

// Draw 绘制任务图像（仅相机页面）
func (d *Drawer) Draw(screen *ebiten.Image, state *MissionState) {
	d.drawMapAndConsole(screen, state)
	d.drawArrowOnMapWhenHover(screen, state)
	d.drawBuildings(screen, state)
	d.drawBattleShips(screen, state)
	d.drawShotBullets(screen, state)
	d.drawTips(screen, state)
}

// 绘制背景（地图资源 + 控制台）
func (d *Drawer) drawMapAndConsole(screen *ebiten.Image, state *MissionState) {
	for x := 0; x < state.Camera.Width; x++ {
		for y := 0; y < state.Camera.Height; y++ {
			opts := d.genDefaultDrawImageOptions()
			opts.GeoM.Translate(float64(x)*mapblock.BlockSize, float64(y)*mapblock.BlockSize)

			mapX, mapY := state.Camera.Pos.MX+x, state.Camera.Pos.MY+y
			char := state.MissionMD.MapCfg.Map.Get(mapX, mapY)
			screen.DrawImage(mapblock.GetByCharAndPos(char, mapX, mapY), opts)
		}
	}

	// 纯银色控制台
	consoleBGImg := ebiten.NewImage(state.Layout.Console.Width, state.Layout.Console.Height)
	consoleBGImg.Fill(colorx.DarkSilver)
	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(
		float64(state.Layout.Console.TopLeft.SX),
		float64(state.Layout.Console.TopLeft.SY),
	)
	screen.DrawImage(consoleBGImg, opts)

	// 小地图
	opts = d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(
		float64(state.Layout.Console.SmallMap.TopLeft.SX),
		float64(state.Layout.Console.SmallMap.TopLeft.SY),
	)
	// FIXME 目前先丢一个黑图片进去，将来应该是考虑地图，战舰，视野等等，渲染一个实时图片
	smallMapImg := ebiten.NewImage(state.Layout.Console.SmallMap.Width, state.Layout.Console.SmallMap.Height)
	smallMapImg.Fill(colorx.Black)
	screen.DrawImage(smallMapImg, opts)

	// 菜单
	opts = d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(
		float64(state.Layout.Console.Menu.TopLeft.SX),
		float64(state.Layout.Console.Menu.TopLeft.SY),
	)
	// FIXME 目前先丢一个银色图片进去，将来再考虑按钮之类的
	menuImg := ebiten.NewImage(state.Layout.Console.Menu.Width, state.Layout.Console.Menu.Height)
	menuImg.Fill(colorx.Silver)
	screen.DrawImage(menuImg, opts)
}

// 绘制箭头（当鼠标悬浮触发镜头移动时）
func (d *Drawer) drawArrowOnMapWhenHover(screen *ebiten.Image, state *MissionState) {
	posX, posY, rotate, padding := 0, 0, 0, 10
	imgW, imgH := texture.ArrowWhiteImg.Bounds().Dx(), texture.ArrowWhiteImg.Bounds().Dy()
	switch detectCursorHoverOnGameMap(state.Layout) {
	case HoverScreenLeft:
		posX, posY, rotate = padding, state.Layout.Camera.Height/2, 180
	case HoverScreenRight:
		posX, posY, rotate = state.Layout.Camera.Width-imgW-padding, state.Layout.Camera.Height/2, 0
	case HoverScreenTop:
		posX, posY, rotate = state.Layout.Camera.Width/2, padding, 270
	case HoverScreenBottom:
		posX, posY, rotate = state.Layout.Camera.Width/2, state.Layout.Camera.Height-imgH-padding, 90
	case HoverScreenTopLeft:
		posX, posY, rotate = padding, padding, 225
	case HoverScreenTopRight:
		posX, posY, rotate = state.Layout.Camera.Width-imgW-padding, padding, 315
	case HoverScreenBottomLeft:
		posX, posY, rotate = padding, state.Layout.Camera.Height-imgH-padding, 135
	case HoverScreenBottomRight:
		posX, posY, rotate = state.Layout.Camera.Width-imgW-padding, state.Layout.Camera.Height-imgH-padding, 45
	default:
		return
	}

	opts := d.genDefaultDrawImageOptions()
	ebutil.UpdateOptsForCenterRotate(opts, texture.ArrowWhiteImg, rotate)
	opts.GeoM.Translate(float64(posX), float64(posY))
	screen.DrawImage(texture.ArrowWhiteImg, opts)
}

// 绘制建筑物
func (d *Drawer) drawBuildings(screen *ebiten.Image, state *MissionState) {
}

// 绘制战舰
func (d *Drawer) drawBattleShips(screen *ebiten.Image, state *MissionState) {
	for _, ship := range state.Ships {
		// 只有在屏幕中的才渲染
		if ship.Pos.MX < state.Camera.Pos.MX ||
			ship.Pos.MX > state.Camera.Pos.MX+state.Camera.Width ||
			ship.Pos.MY < state.Camera.Pos.MY ||
			ship.Pos.MY > state.Camera.Pos.MY+state.Camera.Height {
			continue
		}
		shipImg := obj.GetShipImg(ship.Name)
		opts := d.genDefaultDrawImageOptions()
		ebutil.UpdateOptsForCenterRotate(opts, shipImg, ship.Rotate)
		opts.GeoM.Translate(
			float64((ship.Pos.MX-state.Camera.Pos.MX)*mapblock.BlockSize),
			float64((ship.Pos.MY-state.Camera.Pos.MY)*mapblock.BlockSize),
		)
		screen.DrawImage(shipImg, opts)
	}
}

// 绘制已发射的弹丸
func (d *Drawer) drawShotBullets(screen *ebiten.Image, state *MissionState) {
}

// 绘制提示语
func (d *Drawer) drawTips(screen *ebiten.Image, state *MissionState) {
	if state.MissionStatus == MissionPaused {
		textStr, fontSize := "按下 Q 退出，按下 Enter 继续", float64(64)
		posX := (float64(state.Layout.Camera.Width) - ebutil.CalcTextWidth(textStr, fontSize)) / 2
		posY := float64(state.Layout.Height) / 2
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
