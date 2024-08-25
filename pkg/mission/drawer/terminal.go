package drawer

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/state"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
)

// 绘制终端
func (d *Drawer) drawTerminal(screen *ebiten.Image, ms *state.MissionState) {
	if !ms.GameOpts.DisplayTerminal {
		return
	}
	terminal := bgImg.MissionTerminal
	terminalWidth, terminalHeight := terminal.Bounds().Dx(), terminal.Bounds().Dy()

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(float64(ms.Layout.Width)/float64(terminalWidth), float64(ms.Layout.Height)/float64(terminalHeight))
	screen.DrawImage(terminal, opts)
}
