package drawer

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/narasux/jutland/pkg/i18n"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/action"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// drawPauseOverlay 绘制暂停遮罩和操作面板
func (d *Drawer) drawPauseOverlay(screen *ebiten.Image, ms *state.MissionState) {
	if ms.Core.MissionStatus != state.MissionPaused {
		return
	}

	// debug 模式下不绘制遮罩和确认面板，只显示文字提示
	if ms.UI.DebugFlags.IsActive() {
		d.drawDebugPauseHint(screen, ms)
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

	title := i18n.Text(i18n.MsgMissionPaused)
	primaryText, dangerText := i18n.Text(i18n.MsgMissionResume), i18n.Text(i18n.MsgMissionAbandon)
	if ms.Core.ConfirmQuitMission {
		title = i18n.Text(i18n.MsgMissionConfirmAbandon)
		primaryText, dangerText = i18n.Text(i18n.MsgBack), i18n.Text(i18n.MsgConfirm)
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

// drawDebugPauseHint 在 debug 模式下绘制简洁的暂停提示文字（无遮罩）
func (d *Drawer) drawDebugPauseHint(screen *ebiten.Image, ms *state.MissionState) {
	hint := i18n.Text(i18n.MsgMissionDebugPauseHint)

	textFont := font.LocalizedUI(font.Kai)
	textFace := text.GoTextFace{Source: textFont, Size: 22}
	textW, textH := text.Measure(hint, &textFace, 0)

	// 左下角位置，带半透明暗底
	padX, padY := 12.0, 8.0
	bgX := 8.0
	bgY := float64(ms.View.Layout.Height) - 100.0
	bgW := textW + padX*2
	bgH := textH + padY*2
	vector.FillRect(
		screen,
		float32(bgX), float32(bgY),
		float32(bgW), float32(bgH),
		color.RGBA{R: 3, G: 12, B: 20, A: 190},
		false,
	)

	d.drawText(screen, hint, bgX+padX, bgY+padY, 22, textFont, colorx.White)
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
	textFont := font.LocalizedUI(font.Kai)
	textFace := text.GoTextFace{Source: textFont, Size: fontSize}
	textW, _ := text.Measure(textStr, &textFace, 0)
	d.drawText(screen, textStr, centerX-textW/2, y, fontSize, textFont, textColor)
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
	if !ms.UI.DebugFlags.ShowCursorPosObjInfo {
		return
	}
	lines := cursorDebugLines(ms, *action.DetectCursorPosOnMap(ms))
	if len(lines) == 0 {
		return
	}
	d.drawDebugLines(screen, lines)
}

// cursorDebugLines 汇总光标位置下所有重叠对象的调试文本。
// 命中对象按 Uid 排序：map 遍历顺序随机，不排序会让同一位置的多行信息逐帧跳动。
func cursorDebugLines(ms *state.MissionState, pos objPos.MapPos) []string {
	var hits []objUnit.BattleUnit
	for _, ship := range ms.Arena.Ships {
		if unitAtCursor(ship, pos) {
			hits = append(hits, ship)
		}
	}
	for _, plane := range ms.Arena.Planes {
		if unitAtCursor(plane, pos) {
			hits = append(hits, plane)
		}
	}
	if len(hits) == 0 {
		return nil
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].ID() < hits[j].ID() })

	var lines []string
	for _, ut := range hits {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		switch u := ut.(type) {
		case *objUnit.Plane:
			// 带上攻击目标与所属基地，逐枚释放器给出“为什么不投弹”的判定。
			lines = append(lines, u.DetailLines(objUnit.DetailContext{
				Enemy:     planeAttackTarget(ms, u),
				Terrain:   missionTerrain(ms),
				BaseLabel: planeBaseLabel(ms, u),
			})...)
		case *objUnit.BattleShip:
			lines = append(lines, u.DetailLines()...)
		default:
			lines = append(lines, ut.Detail())
		}
	}
	return lines
}

// unitAtCursor 判断单位是否覆盖光标所在的地图位置。
func unitAtCursor(ut objUnit.BattleUnit, pos objPos.MapPos) bool {
	movementState := ut.MovementState()
	geometricSize := ut.GeometricSize()
	return geometry.IsPointInRotatedRectangle(
		pos.RX, pos.RY,
		movementState.CurPos.RX, movementState.CurPos.RY,
		geometricSize.Length/constants.MapBlockSize,
		geometricSize.Width/constants.MapBlockSize,
		movementState.CurRotation,
	)
}

// planeAttackTarget 返回飞机当前攻击目标；目标不存在时返回 nil。
func planeAttackTarget(ms *state.MissionState, plane *objUnit.Plane) objUnit.Hurtable {
	if plane.CurAttackTarget == "" {
		return nil
	}
	if ship, ok := ms.Arena.Ships[plane.CurAttackTarget]; ok && ship != nil {
		return ship
	}
	if enemy, ok := ms.Arena.Planes[plane.CurAttackTarget]; ok && enemy != nil {
		return enemy
	}
	return nil
}

// missionTerrain 返回当前任务地图数据；缺失时返回 nil（跳过鱼雷陆地判定）。
func missionTerrain(ms *state.MissionState) *mapcfg.MapData {
	if ms.Core.MissionMD.MapCfg == nil {
		return nil
	}
	return &ms.Core.MissionMD.MapCfg.Map
}

// planeBaseLabel 把飞机所属基地（BelongShip 是随机 uuid）翻译成可读标识。
// 基地类型定义在 building 包，unit 不能反向依赖，所以在这里解析：
// 航母 / 其他载机舰船给出舰种 + 舰名，陆地机场给出跑道坐标，基地已不存在时返回空串。
func planeBaseLabel(ms *state.MissionState, plane *objUnit.Plane) string {
	base, ok := ms.FindAircraftBase(plane.BelongShip)
	if !ok {
		return ""
	}
	switch b := base.(type) {
	case *objUnit.BattleShip:
		kind := "ship"
		if b.Type == objUnit.ShipTypeAircraftCarrier {
			kind = "carrier"
		}
		return fmt.Sprintf("%s %s(%s)", kind, objUnit.GetShipDisplayName(b.Name), b.Uid)
	case *objBuilding.Airfield:
		return fmt.Sprintf("airfield@(%d,%d)(%s)", b.Pos.MX, b.Pos.MY, b.Uid)
	default:
		return base.BaseUid()
	}
}

// drawDebugLines 用本地化字体绘制调试文本块，并垫半透明底板保证在任何地图上都可读。
func (d *Drawer) drawDebugLines(screen *ebiten.Image, lines []string) {
	const (
		fontSize   = 14.0
		lineHeight = 17.0
		padX, padY = 8.0, 6.0
	)
	textFont := font.LocalizedUI(font.Kai)
	textFace := text.GoTextFace{Source: textFont, Size: fontSize}
	measure := func(s string) float64 {
		width, _ := text.Measure(s, &textFace, 0)
		return width
	}
	// 折行宽度同时受屏幕宽度约束，窄窗口下不会把文字画到屏幕外。
	maxWidth := min(debugWrapWidth, float64(screen.Bounds().Dx())-(debugOriginX+padX)*2)
	lines = wrapDebugLines(lines, maxWidth, measure)

	boxWidth := 0.0
	for _, line := range lines {
		boxWidth = max(boxWidth, measure(line))
	}
	vector.FillRect(
		screen, debugOriginX, debugOriginY,
		float32(boxWidth+padX*2), float32(float64(len(lines))*lineHeight+padY*2),
		color.RGBA{R: 3, G: 12, B: 20, A: 190}, false,
	)
	for idx, line := range lines {
		d.drawText(
			screen, line, debugOriginX+padX, debugOriginY+padY+float64(idx)*lineHeight,
			fontSize, textFont, colorx.White,
		)
	}
}

// debugWrapWidth 是调试文本块折行的目标宽度（像素）。
// 取值需容纳最长的释放器身份行（约 730px），否则会在行尾留下一个孤零零的单词。
const debugWrapWidth = 780.0

// debugOriginX / debugOriginY 是调试文本块左上角。
// pkg/game 每帧最后在 (0,0) 用 Ebiten 点阵字体（字格 6x16）画 "VER x FPS y"，
// 而且画在本面板之上，所以纵向必须整块让开那一行。
const (
	debugOriginX = 4.0
	debugOriginY = 20.0
)

// debugHangingIndent 是折行后续行相对首行的额外缩进，
// 用来区分“新的释放器”和“同一释放器的续行”。
const debugHangingIndent = "  "

// wrapDebugLines 按像素宽度给调试文本折行。
// 单词本身超宽（例如很长的释放器配置名）时按字符硬拆，保证输出不会溢出底板。
// measure 由调用方注入，便于脱离字体做单元测试。
func wrapDebugLines(lines []string, maxWidth float64, measure func(string) float64) []string {
	if maxWidth <= 0 {
		return lines
	}
	var wrapped []string
	for _, line := range lines {
		if line == "" || measure(line) <= maxWidth {
			wrapped = append(wrapped, line)
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
		prefix, current := indent, ""
		flush := func() {
			if current == "" {
				return
			}
			wrapped = append(wrapped, prefix+current)
			prefix, current = indent+debugHangingIndent, ""
		}
		for _, word := range strings.Fields(line) {
			switch {
			case current == "":
				current = word
			case measure(prefix+current+" "+word) <= maxWidth:
				current += " " + word
			default:
				flush()
				current = word
			}
			for current != "" && measure(prefix+current) > maxWidth {
				cut := debugCutIndex(prefix, current, maxWidth, measure)
				wrapped = append(wrapped, prefix+current[:cut])
				prefix, current = indent+debugHangingIndent, current[cut:]
			}
		}
		flush()
	}
	return wrapped
}

// debugCutIndex 返回按 rune 切分、且 prefix+token 不超过 maxWidth 的最大字节下标。
func debugCutIndex(prefix, token string, maxWidth float64, measure func(string) float64) int {
	runes := []rune(token)
	for i := len(runes) - 1; i >= 1; i-- {
		if measure(prefix+string(runes[:i])) <= maxWidth {
			return len(string(runes[:i]))
		}
	}
	// 连一个字符都放不下时至少前进一个字符，避免死循环。
	return len(string(runes[:1]))
}
