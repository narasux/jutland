package drawer

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/action"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/geometry"
	"github.com/narasux/jutland/pkg/utils/layout"
)

// 绘制提示语
func (d *Drawer) drawTips(screen *ebiten.Image, ms *state.MissionState) {
	if ms.MissionStatus == state.MissionPaused {
		textStr, fontSize := "按下 Q 退出，按下 Esc 继续", float64(64)
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

// drawDebugPrint DEBUG: 绘制调试信息
func (d *Drawer) drawDebugPrint(screen *ebiten.Image, ms *state.MissionState) {
	// 对光标指到的对象进行信息展示
	if ms.DebugFlags.ShowCursorPosObjInfo {
		pos := action.DetectCursorPosOnMap(ms)
		var allBattleUnits []objUnit.BattleUnit
		for _, ship := range ms.Ships {
			allBattleUnits = append(allBattleUnits, ship)
		}
		for _, plane := range ms.Planes {
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
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf(bu.Detail()), 0, (idx+1)*20)
		}
	}
}
