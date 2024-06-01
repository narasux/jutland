package drawer

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/layout"
)

// 绘制提示语
func (d *Drawer) drawTips(screen *ebiten.Image, ms *state.MissionState) {
	if ms.MissionStatus == state.MissionPaused {
		textStr, fontSize := "按下 Q 退出，按下 Enter 继续", float64(64)
		posX := (float64(ms.Layout.Width) - layout.CalcTextWidth(textStr, fontSize)) / 2
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
