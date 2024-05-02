package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/narasux/jutland/pkg/resources/colorx"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

// Drawer 图像绘制工具
type Drawer struct{}

// NewDrawer ...
func NewDrawer() *Drawer {
	return &Drawer{}
}

// 绘制背景
func (d *Drawer) drawBackground(screen *ebiten.Image, bg *ebiten.Image) {
	w, h := bg.Bounds().Dx(), bg.Bounds().Dy()

	scaleX := float64(screen.Bounds().Dx()) / float64(w)
	scaleY := float64(screen.Bounds().Dy()) / float64(h)

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(scaleX, scaleY)
	screen.DrawImage(bg, opts)
}

// 绘制游戏标题
func (d *Drawer) drawGameTitle(screen *ebiten.Image, textStr string) {
	fontSize := float64(128)
	// TODO 优化居中效果
	posX := (float64(screen.Bounds().Dx()) - ebutil.CalcTextWidth(textStr, fontSize)) / 5 * 3
	posY := float64(screen.Bounds().Dy() / 5 * 4)
	d.drawText(screen, textStr, posX, posY, fontSize, font.Hang, colorx.White)
}

// 绘制游戏菜单
func (d *Drawer) drawGameMenu(screen *ebiten.Image, states *menuButtonStates) {
	for _, b := range []*menuButton{
		states.MissionSelect,
		states.ShipCollection,
		states.GameSetting,
		states.ExitGame,
	} {
		d.drawText(screen, b.Text, b.PosX, b.PosY, b.FontSize, b.Font, b.Color)
	}
}

// 绘制文本
func (d *Drawer) drawText(
	screen *ebiten.Image,
	textStr string,
	posX, posY, textSize float64,
	textFont *text.GoTextFaceSource,
	color color.Color,
) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(posX, posY)
	op.ColorScale.ScaleWithColor(color)
	textFace := text.GoTextFace{
		Source: textFont,
		Size:   textSize,
	}
	text.Draw(screen, textStr, &textFace, op)
}

// 默认绘制配置
func (d *Drawer) genDefaultDrawImageOptions() *ebiten.DrawImageOptions {
	return &ebiten.DrawImageOptions{
		// 线性过滤：通过计算周围像素的加权平均值来进行插值，可使得边缘 & 色彩转换更加自然
		Filter: ebiten.FilterLinear,
	}
}
