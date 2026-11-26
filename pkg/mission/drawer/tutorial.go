package drawer

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/i18n"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/tutorial"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/layout"
)

const (
	tutorialBodySize   = 16.0
	tutorialBodyLine   = 20.0
	tutorialKeycapH    = 28.0
	tutorialKeycapFont = 14.0
)

// DrawTutorial 绘制当前教学步骤卡；普通任务不会调用到这里。
func (d *Drawer) DrawTutorial(screen *ebiten.Image, ms *state.MissionState, view tutorial.View) {
	if !view.Visible {
		return
	}
	if ms.Core.MissionStatus == state.MissionInTerminal {
		return
	}
	if ms.Core.MissionStatus == state.MissionPaused && ms.UI.PauseStats {
		return
	}

	ui := tutorial.CalcLayout(float64(ms.View.Layout.Width), float64(ms.View.Layout.Height))
	card := ui.Card
	vector.FillRect(
		screen,
		float32(card.X), float32(card.Y), float32(card.W), float32(card.H),
		color.RGBA{R: 8, G: 20, B: 26, A: 224}, false,
	)
	vector.StrokeRect(
		screen,
		float32(card.X), float32(card.Y), float32(card.W), float32(card.H),
		2, color.RGBA{R: 116, G: 168, B: 178, A: 236}, false,
	)

	titleFont := font.LocalizedUI(font.Hang)
	bodyFont := font.LocalizedUI(font.Kai)

	if !view.Finished {
		progress := i18n.Format(i18n.MsgTutorialProgress, map[string]any{
			"Current": view.StepIndex,
			"Total":   view.StepCount,
		})
		d.drawText(
			screen, progress, ui.Header.X, ui.Header.Y,
			14, font.ForText(progress, font.LocalizedUI(font.Kai)),
			color.RGBA{R: 162, G: 184, B: 188, A: 255},
		)
	}

	title := view.Title
	titleFont = font.ForText(title, titleFont)
	d.drawText(screen, title, ui.Title.X, ui.Title.Y, 22, titleFont, color.RGBA{R: 240, G: 232, B: 218, A: 255})

	bodySize := tutorialBodySize
	bodyLine := tutorialBodyLine
	bodyFont = font.ForText(view.Body, bodyFont)
	lines := layout.WrapText(view.Body, ui.Body.W, bodySize, bodyFont)
	if len(lines) > 3 {
		bodySize = 14
		bodyLine = 18
		lines = layout.WrapText(view.Body, ui.Body.W, bodySize, bodyFont)
	}
	for idx, line := range lines {
		if idx >= 3 {
			break
		}
		d.drawText(
			screen, line, ui.Body.X, ui.Body.Y+float64(idx)*bodyLine,
			bodySize, font.ForText(line, bodyFont),
			color.RGBA{R: 214, G: 224, B: 220, A: 255},
		)
	}

	if view.StuckHint != "" {
		stuckFont := font.ForText(view.StuckHint, bodyFont)
		d.drawText(
			screen, view.StuckHint, ui.Body.X, ui.KeyRow.Y-16, 12, stuckFont,
			color.RGBA{R: 214, G: 178, B: 120, A: 255},
		)
	}
	d.drawTutorialHint(screen, ui.KeyRow, view.Hint)

	if view.Finished {
		d.drawTutorialButton(screen, ui.Next, i18n.Text(i18n.MsgTutorialBackToTutorials), true)
		d.drawTutorialButton(screen, ui.Skip, i18n.Text(i18n.MsgTutorialRestart), false)
		d.drawTutorialButton(screen, ui.Exit, i18n.Text(i18n.MsgTutorialExit), false)
		return
	}
	if view.AllowSkip {
		d.drawTutorialButton(screen, ui.Skip, i18n.Text(i18n.MsgTutorialSkip), false)
	}
	if view.Mode == tutorial.ModeInfo || view.Mode == tutorial.ModeScripted || view.Mode == tutorial.ModeFree {
		d.drawTutorialButton(screen, ui.Next, i18n.Text(i18n.MsgTutorialNext), true)
	}
	d.drawTutorialButton(screen, ui.Exit, i18n.Text(i18n.MsgTutorialExit), false)
}

// drawTutorialTarget 在目标点画脉冲圈，避免玩家在开阔海面上找不到该点哪里。
func (d *Drawer) drawTutorialTarget(screen *ebiten.Image, ms *state.MissionState, target objPos.MapPos) {
	x, y := ms.CameraPosToScreen(target)
	if x < 0 || y < 0 || x > float64(screen.Bounds().Dx()) || y > float64(screen.Bounds().Dy()) {
		return
	}
	pulse := 15 + 4*math.Sin(float64(ms.Core.SimTick)*0.15)
	clr := color.RGBA{R: 230, G: 196, B: 118, A: 235}
	vector.StrokeCircle(screen, float32(x), float32(y), float32(pulse), 2, clr, true)
	vector.StrokeCircle(
		screen, float32(x), float32(y), float32(pulse+9), 1,
		color.RGBA{R: 230, G: 196, B: 118, A: 140}, true,
	)
	vector.StrokeLine(screen, float32(x-6), float32(y), float32(x+6), float32(y), 1, clr, true)
	vector.StrokeLine(screen, float32(x), float32(y-6), float32(x), float32(y+6), 1, clr, true)
}

func (d *Drawer) drawTutorialHint(
	screen *ebiten.Image,
	rect tutorial.Rect,
	hint tutorial.Hint,
) {
	x := rect.X
	y := rect.Y
	if len(hint.Actions) == 0 {
		return
	}

	hold := hint.Hold
	specs := make([]tutorial.ActionSpec, 0, len(hint.Actions))
	for _, action := range hint.Actions {
		spec, ok := tutorial.TokensFor(action)
		if !ok {
			continue
		}
		specs = append(specs, spec)
		if spec.Hold {
			hold = true
		}
	}
	if len(specs) == 0 {
		return
	}

	labelFont := font.LocalizedUI(font.Kai)
	if hold {
		holdText := i18n.Text(i18n.MsgTutorialHold)
		d.drawText(
			screen,
			holdText,
			x,
			y+7,
			14,
			font.ForText(holdText, labelFont),
			color.RGBA{R: 162, G: 184, B: 188, A: 255},
		)
		x += layout.CalcTextWidth(holdText, 14, font.ForText(holdText, labelFont)) + 8
	}

	for specIdx, spec := range specs {
		if specIdx > 0 {
			d.drawText(screen, "+", x, y+6, 16, font.JetbrainsMono, color.RGBA{R: 162, G: 184, B: 188, A: 255})
			x += 16
		}
		for tokenIdx, token := range spec.Tokens {
			label := tutorialTokenLabel(token)
			if label == "" {
				continue
			}
			tokenFont := font.ForText(label, font.JetbrainsMono)
			textW := layout.CalcTextWidth(label, tutorialKeycapFont, tokenFont)
			keyW := textW + 18
			vector.FillRect(
				screen, float32(x), float32(y), float32(keyW), tutorialKeycapH,
				color.RGBA{R: 30, G: 54, B: 60, A: 240}, false,
			)
			vector.StrokeRect(
				screen, float32(x), float32(y), float32(keyW), tutorialKeycapH,
				1, color.RGBA{R: 128, G: 178, B: 188, A: 240}, false,
			)
			d.drawText(
				screen, label, x+9, y+7, tutorialKeycapFont, tokenFont,
				color.RGBA{R: 238, G: 230, B: 210, A: 255},
			)
			x += keyW + 6
			if tokenIdx < len(spec.Tokens)-1 {
				d.drawText(screen, "+", x, y+6, 16, font.JetbrainsMono, color.RGBA{R: 162, G: 184, B: 188, A: 255})
				x += 16
			}
		}
	}
}

func tutorialTokenLabel(token tutorial.Token) string {
	switch token.Kind {
	case tutorial.TokenKey:
		return token.Label
	case tutorial.TokenMouseLeft:
		return i18n.Text(i18n.MsgTutorialMouseLeft)
	case tutorial.TokenMouseRight:
		return i18n.Text(i18n.MsgTutorialMouseRight)
	case tutorial.TokenMouseDrag:
		return i18n.Text(i18n.MsgTutorialMouseDrag)
	case tutorial.TokenMouseWheel:
		return i18n.Text(i18n.MsgTutorialMouseWheel)
	default:
		return ""
	}
}

func (d *Drawer) drawTutorialButton(screen *ebiten.Image, rect tutorial.Rect, label string, primary bool) {
	fill := color.RGBA{R: 30, G: 54, B: 60, A: 230}
	border := color.RGBA{R: 116, G: 156, B: 166, A: 220}
	textColor := color.RGBA{R: 220, G: 226, B: 220, A: 255}
	if primary {
		fill = color.RGBA{R: 40, G: 92, B: 108, A: 238}
		border = color.RGBA{R: 144, G: 206, B: 216, A: 245}
		textColor = color.RGBA{R: 245, G: 240, B: 224, A: 255}
	}
	vector.FillRect(screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H), fill, false)
	vector.StrokeRect(screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H), 1, border, false)
	textFont := font.ForText(label, font.LocalizedUI(font.Kai))
	size := 15.0
	textW := layout.CalcTextWidth(label, size, textFont)
	for size > 11 && textW > rect.W-12 {
		size--
		textW = layout.CalcTextWidth(label, size, textFont)
	}
	d.drawText(
		screen, label,
		rect.X+(rect.W-textW)/2,
		rect.Y+(rect.H-size)/2-2,
		size, textFont, textColor,
	)
}
