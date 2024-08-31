package drawer

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/hacker"
	"github.com/narasux/jutland/pkg/mission/state"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
)

// 绘制终端
func (d *Drawer) drawTerminal(
	screen *ebiten.Image,
	ms *state.MissionState,
	terminal *hacker.Terminal,
) {
	if ms.MissionStatus != state.MissionInTerminal {
		return
	}
	terminalImg := bgImg.MissionTerminal
	terminalWidth := terminalImg.Bounds().Dx()
	terminalHeight := terminalImg.Bounds().Dy()

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(
		float64(ms.Layout.Width)/float64(terminalWidth),
		float64(ms.Layout.Height)/float64(terminalHeight),
	)
	screen.DrawImage(terminalImg, opts)

	xOffset, yOffset := float64(150), float64(170)
	for _, line := range terminal.Buffer {
		d.drawText(
			screen, line.String(),
			xOffset, yOffset,
			terminal.FontSize,
			terminal.Font,
			terminal.Color,
		)
		yOffset += terminal.FontSize + terminal.LineSpacing
	}
	d.drawText(
		screen,
		terminal.CurInputString(),
		xOffset, yOffset,
		terminal.FontSize,
		terminal.Font,
		terminal.Color,
	)
}
