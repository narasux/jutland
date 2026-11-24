package drawer

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

const (
	aiTakeoverBadgeX = 12.0
	aiTakeoverBadgeY = 12.0
)

// drawAITakeoverBadge 在运行态左上角提示玩家当前由 AI 托管。
// 暂停面板、地图模式和终端模式不画，避免和整屏 UI 抢视觉焦点。
func (d *Drawer) drawAITakeoverBadge(screen *ebiten.Image, ms *state.MissionState) {
	if ms.Core.MissionStatus != state.MissionRunning || !ms.Core.PlayerAIControl {
		return
	}
	badgeText := i18n.Text(i18n.MsgMissionAITakeoverActive)
	textFont := font.LocalizedUI(font.Kai)
	fontSize := 18.0
	face := text.GoTextFace{Source: textFont, Size: fontSize}
	textW, textH := text.Measure(badgeText, &face, 0)

	padX, padY := 10.0, 6.0
	x, y := aiTakeoverBadgeX, aiTakeoverBadgeY
	w, h := textW+2*padX, textH+2*padY
	vector.FillRect(
		screen, float32(x), float32(y), float32(w), float32(h),
		color.RGBA{R: 3, G: 12, B: 20, A: 190}, false,
	)
	vector.StrokeRect(
		screen, float32(x), float32(y), float32(w), float32(h),
		1, color.RGBA{R: 104, G: 132, B: 139, A: 230}, false,
	)
	d.drawText(screen, badgeText, x+padX, y+padY, fontSize, textFont, colorx.SkyBlue)
}
