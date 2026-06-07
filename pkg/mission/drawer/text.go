package drawer

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/action"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// drawPauseOverlay 绘制暂停遮罩和操作面板
func (d *Drawer) drawPauseOverlay(screen *ebiten.Image, ms *state.MissionState) {
	if ms.Core.MissionStatus != state.MissionPaused {
		return
	}

	ui := state.CalcPauseUILayout(ms.View.Layout)
	vector.FillRect(
		screen, 0, 0, float32(ms.View.Layout.Width), float32(ms.View.Layout.Height),
		color.RGBA{R: 3, G: 10, B: 14, A: 150}, false,
	)
	vector.FillRect(
		screen, float32(ui.Panel.X), float32(ui.Panel.Y), float32(ui.Panel.W), float32(ui.Panel.H),
		color.RGBA{R: 10, G: 25, B: 31, A: 228}, false,
	)
	vector.StrokeRect(
		screen, float32(ui.Panel.X), float32(ui.Panel.Y), float32(ui.Panel.W), float32(ui.Panel.H),
		2, color.RGBA{R: 104, G: 132, B: 139, A: 230}, false,
	)

	title := "任务暂停"
	primaryText, dangerText := "继续", "放弃"
	if ms.Core.ConfirmQuitMission {
		title = "确认放弃任务？"
		primaryText, dangerText = "返回", "确认"
	}

	d.drawCenteredPauseText(screen, title, ui.Panel.X+ui.Panel.W/2, ui.Panel.Y+46, 30, colorx.White)
	d.drawPauseButton(
		screen, ui.PrimaryButton, primaryText,
		color.RGBA{R: 33, G: 63, B: 69, A: 235},
		color.RGBA{R: 117, G: 170, B: 180, A: 255},
	)
	d.drawPauseButton(
		screen, ui.DangerButton, dangerText,
		color.RGBA{R: 77, G: 37, B: 33, A: 235},
		color.RGBA{R: 214, G: 118, B: 91, A: 255},
	)
}

// drawPauseButton 绘制暂停面板按钮，并根据鼠标悬停状态调整边框
func (d *Drawer) drawPauseButton(
	screen *ebiten.Image,
	rect state.PauseUIRect,
	label string,
	fill color.RGBA,
	border color.RGBA,
) {
	sx, sy := ebiten.CursorPosition()
	hovered := rect.Contains(sx, sy)
	if hovered {
		fill.A = 255
		border = color.RGBA{R: 232, G: 224, B: 198, A: 255}
	}

	vector.FillRect(screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H), fill, false)
	vector.StrokeRect(screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H), 2, border, false)
	d.drawCenteredPauseText(screen, label, rect.X+rect.W/2, rect.Y+8, 18, colorx.White)
}

// drawCenteredPauseText 使用实际字体测量宽度，避免中文标题居中偏移
func (d *Drawer) drawCenteredPauseText(
	screen *ebiten.Image,
	textStr string,
	centerX, y, fontSize float64,
	textColor color.Color,
) {
	textFace := text.GoTextFace{Source: font.Kai, Size: fontSize}
	textW, _ := text.Measure(textStr, &textFace, 0)
	d.drawText(screen, textStr, centerX-textW/2, y, fontSize, font.Kai, textColor)
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

// drawDebugPrint DEBUG: 绘制调试信息
func (d *Drawer) drawDebugPrint(screen *ebiten.Image, ms *state.MissionState) {
	// 对光标指到的对象进行信息展示
	if ms.UI.DebugFlags.ShowCursorPosObjInfo {
		pos := action.DetectCursorPosOnMap(ms)
		var allBattleUnits []objUnit.BattleUnit
		for _, ship := range ms.Arena.Ships {
			allBattleUnits = append(allBattleUnits, ship)
		}
		for _, plane := range ms.Arena.Planes {
			allBattleUnits = append(allBattleUnits, plane)
		}
		// 统计出在当前光标位置的战舰/战机
		var curCursorPosObjs []objUnit.BattleUnit
		for _, ut := range allBattleUnits {
			movementState := ut.MovementState()
			geometricSize := ut.GeometricSize()
			if geometry.IsPointInRotatedRectangle(
				pos.RX, pos.RY,
				movementState.CurPos.RX, movementState.CurPos.RY,
				geometricSize.Length/constants.MapBlockSize,
				geometricSize.Width/constants.MapBlockSize,
				movementState.CurRotation,
			) {
				curCursorPosObjs = append(curCursorPosObjs, ut)
				break
			}
		}
		// 逐行展示
		for idx, bu := range curCursorPosObjs {
			ebitenutil.DebugPrintAt(screen, bu.Detail(), 0, (idx+1)*20)
		}
	}
}
