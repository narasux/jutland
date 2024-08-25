package drawer

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/state"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
)

// 绘制终端
func (d *Drawer) drawTerminal(screen *ebiten.Image, ms *state.MissionState) {
	if ms.MissionStatus != state.MissionInTerminal {
		return
	}
	terminalImg := bgImg.MissionTerminal
	terminalWidth, terminalHeight := terminalImg.Bounds().Dx(), terminalImg.Bounds().Dy()

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(
		float64(ms.Layout.Width)/float64(terminalWidth),
		float64(ms.Layout.Height)/float64(terminalHeight),
	)
	screen.DrawImage(terminalImg, opts)

	xOffset, yOffset := float64(150), float64(170)
	for _, line := range ms.Terminal.Buffer {
		d.drawText(
			screen, line.String(),
			xOffset, yOffset,
			ms.Terminal.FontSize,
			ms.Terminal.Font,
			ms.Terminal.Color,
		)
		yOffset += ms.Terminal.FontSize + ms.Terminal.LineSpacing
	}
	d.drawText(
		screen,
		ms.Terminal.CurInputString(),
		xOffset, yOffset,
		ms.Terminal.FontSize,
		ms.Terminal.Font,
		ms.Terminal.Color,
	)
}
