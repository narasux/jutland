package drawer

import (
	"image/color"
	"math"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/i18n"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
	layoutUtils "github.com/narasux/jutland/pkg/utils/layout"
)

// battleReportEventLog 返回详情页当前筛选结果；击杀筛选结果按日志版本缓存。
func (d *Drawer) battleReportEventLog(ms *state.MissionState) []state.BattleLogEntry {
	filter := ms.UI.BattleReportEventFilter
	if filter == state.BattleReportEventFilterAll {
		return ms.Stats.BattleLog
	}
	version := ms.Stats.BattleLogVersion
	cache := &d.eventLogCache
	if cache.ms == ms && cache.filter == filter && cache.version == version {
		return cache.log
	}
	cache.ms = ms
	cache.filter = filter
	cache.version = version
	cache.log = state.FilterBattleLog(ms.Stats.BattleLog, filter)
	return cache.log
}

// drawBattleReportEventLog 绘制可滚动的战况记录页。
// 记录按最新在前展示，scroll=0 时显示最后发生的战斗事件。
func (d *Drawer) drawBattleReportEventLog(
	screen *ebiten.Image,
	report state.BattleReportLayout,
	ms *state.MissionState,
	scale float64,
	bodyFont, monoFont *text.GoTextFaceSource,
) {
	panel := report.Content
	vector.FillRect(
		screen, float32(panel.X), float32(panel.Y), float32(panel.W), float32(panel.H),
		battleReportCardFill, false,
	)
	vector.StrokeRect(
		screen, float32(panel.X), float32(panel.Y), float32(panel.W), float32(panel.H),
		float32(max(1, scale)), battleReportCardBorder, false,
	)

	pad := 18 * scale
	titleRect := state.BattleReportRect{
		X: panel.X + pad, Y: panel.Y + 10*scale, W: panel.W - 2*pad, H: 30 * scale,
	}
	title := i18n.Text(i18n.MsgBattleReportEventTitle)
	d.drawReportText(
		screen, title, titleRect,
		fitReportFontSize(title, titleRect.W, 22*scale, bodyFont),
		bodyFont, battleReportTitleClr, reportAlignLeft,
	)

	log := d.battleReportEventLog(ms)
	countText := i18n.Format(i18n.MsgBattleReportEventCount, map[string]any{"Count": len(log)})
	countRect := state.BattleReportRect{
		X: panel.X + panel.W - pad - 220*scale, Y: titleRect.Y,
		W: 220 * scale, H: titleRect.H,
	}
	d.drawReportText(
		screen, countText, countRect,
		fitReportFontSize(countText, countRect.W, 14*scale, bodyFont),
		bodyFont, battleReportSubtitleClr, reportAlignRight,
	)
	d.drawBattleReportEventFilters(screen, report, ms.UI.BattleReportEventFilter, scale, bodyFont)

	dividerY := panel.Y + 46*scale
	vector.StrokeLine(
		screen,
		float32(panel.X+pad), float32(dividerY),
		float32(panel.X+panel.W-pad), float32(dividerY),
		float32(max(1, scale)), battleReportDividerClr, false,
	)

	rows := state.BattleReportEventRows
	rowH := 30 * scale
	listY := dividerY + 8*scale
	listH := float64(rows) * rowH
	scrollbar, hasScrollbar := state.BattleReportEventScrollbar(
		panel, len(log), ms.UI.BattleReportEventScroll, scale,
	)
	scroll := 0
	if hasScrollbar {
		scroll = min(max(0, ms.UI.BattleReportEventScroll), scrollbar.MaxScroll)
	}

	// 有滚动条时在右侧预留槽位，行文字和分隔线都不进入该槽位。
	contentLeft := panel.X + pad
	contentRight := panel.X + panel.W - pad
	if hasScrollbar {
		contentRight = scrollbar.TrackX - 4*scale
	}
	lineX := contentLeft + 14*scale
	lineW := contentRight - lineX

	if len(log) == 0 {
		empty := i18n.Text(i18n.MsgBattleReportEventEmpty)
		emptyRect := state.BattleReportRect{
			X: contentLeft, Y: listY, W: contentRight - contentLeft, H: listH,
		}
		d.drawReportText(
			screen, empty, emptyRect,
			fitReportFontSize(empty, emptyRect.W, 16*scale, bodyFont),
			bodyFont, battleReportSubtitleClr, reportAlignCenter,
		)
	} else {
		for row := 0; row < rows; row++ {
			index := len(log) - 1 - (scroll + row)
			if index < 0 {
				break
			}
			entry := log[index]
			y := listY + float64(row)*rowH
			dotX := panel.X + pad + 4*scale
			dotY := y + rowH/2
			dotClr := battleReportSubtitleClr
			switch entry.ActorPlayer {
			case ms.Player.CurPlayer:
				dotClr = battleReportOwnClr
			case ms.Player.CurEnemy:
				dotClr = battleReportLostClr
			}
			vector.FillCircle(
				screen, float32(dotX), float32(dotY), float32(max(2*scale, 2.5)), dotClr, false,
			)

			segments := battleReportEventSegments(entry)
			line := battleReportSegmentsText(segments)
			lineRect := state.BattleReportRect{
				X: lineX, Y: y, W: lineW, H: rowH,
			}
			d.drawBattleReportEventSegments(
				screen, segments, lineRect,
				fitReportFontSize(line, lineRect.W, 16*scale, bodyFont),
				bodyFont,
			)
			if row < rows-1 {
				separatorY := y + rowH
				vector.StrokeLine(
					screen,
					float32(contentLeft), float32(separatorY),
					float32(contentRight), float32(separatorY),
					float32(max(1, scale)), battleReportDividerClr, false,
				)
			}
		}
	}

	// 滚动条：仅在记录数超过一屏时出现。
	if hasScrollbar {
		vector.FillRect(
			screen, float32(scrollbar.TrackX), float32(scrollbar.TrackY),
			float32(scrollbar.TrackW), float32(scrollbar.TrackH),
			color.RGBA{R: 50, G: 68, B: 76, A: 190}, false,
		)
		vector.FillRect(
			screen, float32(scrollbar.TrackX), float32(scrollbar.ThumbY),
			float32(scrollbar.TrackW), float32(scrollbar.ThumbH),
			battleReportButtonBrd, false,
		)
	}

	if hasScrollbar {
		hint := i18n.Text(i18n.MsgBattleReportEventScrollHint)
		hintRect := state.BattleReportRect{
			X: panel.X + pad, Y: panel.Y + panel.H - 22*scale,
			W: panel.W - 2*pad, H: 18 * scale,
		}
		d.drawReportText(
			screen, hint, hintRect,
			fitReportFontSize(hint, hintRect.W, 13*scale, bodyFont),
			bodyFont, battleReportSubtitleClr, reportAlignLeft,
		)
	}
}

// drawBattleReportEventFilters 绘制「全部 / 击杀」筛选按钮。
func (d *Drawer) drawBattleReportEventFilters(
	screen *ebiten.Image,
	report state.BattleReportLayout,
	active state.BattleReportEventFilter,
	scale float64,
	bodyFont *text.GoTextFaceSource,
) {
	labels := [2]string{
		i18n.Text(i18n.MsgBattleReportEventFilterAll),
		i18n.Text(i18n.MsgBattleReportEventFilterKills),
	}
	sx, sy := ebiten.CursorPosition()
	for idx, rect := range report.EventFilters {
		if rect.Empty() {
			continue
		}
		fill, border := battleReportTabFill, battleReportTabBorder
		textClr := battleReportLabelClr
		if state.BattleReportEventFilter(idx) == active {
			fill, border, textClr = battleReportTabActive, battleReportButtonBrd, colorx.White
		} else if rect.Contains(sx, sy) {
			fill, border = battleReportHoverFill, battleReportHoverBrd
		}
		vector.FillRect(
			screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H), fill, false,
		)
		vector.StrokeRect(
			screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
			float32(max(1, scale)), border, false,
		)
		label := labels[idx]
		d.drawReportText(
			screen, label, rect,
			fitReportFontSize(label, rect.W-10*scale, 13*scale, bodyFont),
			bodyFont, textClr, reportAlignCenter,
		)
	}
}

// battleReportEventSegment 是详情行的一段文字，动作词可以单独着色。
type battleReportEventSegment struct {
	Text  string
	Color color.Color
}

// 动作占位符只在拼行时使用，不会出现在最终文本中。
const battleReportActionMarker = "\x00"

// 伤害数值占位符，用于把暴击伤害从整行里拆出来单独着色。
const battleReportDamageMarker = "\x01"

// battleReportDamageColor 返回伤害数值的暴击配色，与浮动伤害数字保持一致。
func battleReportDamageColor(critical objBullet.CriticalType) color.Color {
	switch critical {
	case objBullet.CriticalTypeThreeTimes:
		return colorx.Yellow
	case objBullet.CriticalTypeTenTimes:
		return colorx.Red
	default:
		return battleReportValueClr
	}
}

// appendBattleReportDamageSegments 把带伤害标记的文本拆成「普通 / 伤害」两种颜色。
func appendBattleReportDamageSegments(
	segments []battleReportEventSegment, text string, damageColor color.Color,
) []battleReportEventSegment {
	for idx, part := range strings.Split(text, battleReportDamageMarker) {
		if part == "" {
			continue
		}
		clr := color.Color(battleReportValueClr)
		if idx%2 == 1 {
			clr = damageColor
		}
		segments = append(segments, battleReportEventSegment{Text: part, Color: clr})
	}
	return segments
}

// battleReportEventSegments 把一条战况拆成「前缀 / 动作 / 后缀」，
// 击沉和击落动作单独用红色，其他部分保持正常文字颜色。
func battleReportEventSegments(entry state.BattleLogEntry) []battleReportEventSegment {
	actor := battleReportCombatantText(entry.ActorName, entry.ActorType, entry.ActorIsPlane)
	if entry.Action == state.BattleLogPlaneWithShip {
		return []battleReportEventSegment{{
			Text: i18n.Format(i18n.MsgBattleReportWithShipRow, map[string]any{
				"Time":  state.BattleReportDurationText(entry.Tick),
				"Actor": actor,
				"Count": groupThousands(int64(entry.Count)),
			}),
			Color: battleReportValueClr,
		}}
	}
	target := battleReportCombatantText(entry.TargetName, entry.TargetType, entry.TargetIsPlane)

	action := ""
	switch entry.Action {
	case state.BattleLogShipHit, state.BattleLogPlaneHit:
		action = i18n.Text(i18n.MsgBattleReportActionHit)
	case state.BattleLogShipSunk:
		if entry.ActorName == "" {
			action = i18n.Text(i18n.MsgBattleReportActionSunkUnknown)
		} else {
			action = i18n.Text(i18n.MsgBattleReportActionSunk)
		}
	case state.BattleLogPlaneShotDown:
		action = i18n.Text(i18n.MsgBattleReportActionShotDown)
	case state.BattleLogPlaneLost:
		action = i18n.Text(i18n.MsgBattleReportActionLost)
	case state.BattleLogPlaneWithShip:
		action = i18n.Text(i18n.MsgBattleReportActionWithShip)
	}

	weapon := battleReportWeaponText(entry)
	weaponPhrase := ""
	if weapon != "" {
		weaponPhrase = i18n.Format(i18n.MsgBattleReportEventUseWeapon, map[string]any{"Weapon": weapon})
	}

	damagePhrase := ""
	if entry.Hit.WeaponType != "" {
		damagePhrase = i18n.Format(i18n.MsgBattleReportEventDamage, map[string]any{
			"Damage": battleReportDamageMarker +
				battleReportDamageText(entry.Hit.Damage) +
				battleReportDamageMarker,
		})
	}
	if entry.TotalDamage > entry.Hit.Damage+0.5 {
		damagePhrase += i18n.Format(i18n.MsgBattleReportEventTotalDamage, map[string]any{
			"Damage": battleReportDamageText(entry.TotalDamage),
		})
	}

	suffix := ""
	if entry.Count > 1 {
		suffix += i18n.Format(i18n.MsgBattleReportEventRepeat, map[string]any{
			"Count": groupThousands(int64(entry.Count)),
		})
	}
	if entry.Tonnage > 0 {
		suffix += i18n.Format(i18n.MsgBattleReportEventTonnage, map[string]any{
			"Tonnage": groupThousands(int64(math.Round(entry.Tonnage))),
		})
	}

	row := i18n.Format(i18n.MsgBattleReportEventRow, map[string]any{
		"Time":   state.BattleReportDurationText(entry.Tick),
		"Actor":  actor,
		"Weapon": weaponPhrase,
		"Action": battleReportActionMarker,
		"Target": target,
		"Damage": damagePhrase,
		"Suffix": suffix,
	})
	damageColor := battleReportDamageColor(entry.Hit.Critical)
	prefix, rest, ok := strings.Cut(row, battleReportActionMarker)
	if !ok || action == "" {
		row = strings.ReplaceAll(row, battleReportActionMarker, action)
		return appendBattleReportDamageSegments(nil, row, damageColor)
	}

	actionColor := battleReportValueClr
	if (entry.Action == state.BattleLogShipSunk && entry.ActorName != "") ||
		entry.Action == state.BattleLogPlaneShotDown {
		actionColor = battleReportKillClr
	}
	segments := make([]battleReportEventSegment, 0, 5)
	segments = appendBattleReportDamageSegments(segments, prefix, damageColor)
	segments = append(segments, battleReportEventSegment{Text: action, Color: actionColor})
	segments = appendBattleReportDamageSegments(segments, rest, damageColor)
	return segments
}

// battleReportSegmentsText 拼接一行的完整文本，用于字号适配。
func battleReportSegmentsText(segments []battleReportEventSegment) string {
	var builder strings.Builder
	for _, segment := range segments {
		builder.WriteString(segment.Text)
	}
	return builder.String()
}

// drawBattleReportEventSegments 按段绘制详情行，保持同一字号与基线。
func (d *Drawer) drawBattleReportEventSegments(
	screen *ebiten.Image,
	segments []battleReportEventSegment,
	rect state.BattleReportRect,
	fontSize float64,
	bodyFont *text.GoTextFaceSource,
) {
	fullText := battleReportSegmentsText(segments)
	if fullText == "" || rect.Empty() || fontSize <= 0 {
		return
	}
	measuredFont := font.ForText(fullText, bodyFont)
	_, height := text.Measure(fullText, &text.GoTextFace{Source: measuredFont, Size: fontSize}, 0)
	x := rect.X
	y := rect.Y + (rect.H-height)/2
	for _, segment := range segments {
		if segment.Text == "" {
			continue
		}
		d.drawText(screen, segment.Text, x, y, fontSize, measuredFont, segment.Color)
		x += layoutUtils.CalcTextWidth(segment.Text, fontSize, measuredFont)
	}
}

// battleReportCombatantText 生成「驱逐舰 雪风 / 战斗机 FM-2」这样的类型 + 名称。
func battleReportCombatantText(name, kind string, isPlane bool) string {
	if name == "" {
		return i18n.Text(i18n.MsgBattleReportNoData)
	}
	displayName := name
	typeName := ""
	if isPlane {
		displayName = objUnit.GetPlaneTitle(name)
		if kind == "" {
			if plane := objUnit.PlaneMap[name]; plane != nil {
				kind = string(plane.Type)
			}
		}
		if kind != "" {
			typeName = objUnit.PlaneType(kind).ToDisplay()
		}
	} else {
		displayName = objUnit.GetShipDisplayName(name)
		if kind == "" {
			if ship := objUnit.ShipMap[name]; ship != nil {
				kind = string(ship.Type)
			}
		}
		if kind != "" {
			typeName = objUnit.ShipType(kind).ToDisplay()
		}
	}
	if typeName == "" {
		return displayName
	}
	return typeName + " " + displayName
}

// battleReportDamageText 小伤害保留一位小数，避免 0.11 被显示成 0。
func battleReportDamageText(value float64) string {
	if value > 0 && value < 10 && value != math.Trunc(value) {
		precision := 1
		if value < 1 {
			precision = 2
		}
		return strconv.FormatFloat(value, 'f', precision, 64)
	}
	return groupThousands(int64(math.Round(value)))
}

// battleReportCaliberText 优先使用配置里的精确口径（例如 7.62），旧数据回退整数。
func battleReportCaliberText(entry state.BattleLogEntry) string {
	if entry.Hit.Caliber != "" {
		return entry.Hit.Caliber
	}
	if entry.Hit.Diameter > 0 {
		return strconv.Itoa(entry.Hit.Diameter)
	}
	return ""
}

// battleReportWeaponText 根据弹药类型和发射平台生成「203mm 舰炮 / 533mm 鱼雷」等武器描述。
func battleReportWeaponText(entry state.BattleLogEntry) string {
	if entry.Hit.WeaponType == "" {
		return ""
	}
	weaponType := entry.Hit.WeaponType
	caliber := battleReportCaliberText(entry)
	switch weaponType {
	case string(objBullet.TypeShell):
		switch {
		case entry.ActorIsPlane:
			if caliber != "" {
				msgID := i18n.MsgBattleReportWeaponAircraftGun
				if entry.Hit.Diameter < 20 {
					msgID = i18n.MsgBattleReportWeaponAircraftMG
				}
				return i18n.Format(msgID, map[string]any{
					"Diameter": caliber,
				})
			}
		case entry.TargetIsPlane:
			if caliber != "" {
				return i18n.Format(i18n.MsgBattleReportWeaponAAGun, map[string]any{
					"Diameter": caliber,
				})
			}
		default:
			if caliber != "" {
				return i18n.Format(i18n.MsgBattleReportWeaponShell, map[string]any{
					"Diameter": caliber,
				})
			}
		}
	case string(objBullet.TypeTorpedo):
		if caliber != "" {
			return i18n.Format(i18n.MsgBattleReportWeaponTorpedo, map[string]any{
				"Diameter": caliber,
			})
		}
	case string(objBullet.TypeBomb):
		if entry.Hit.Weight > 0 {
			return i18n.Format(i18n.MsgBattleReportWeaponBomb, map[string]any{
				"Weight": entry.Hit.Weight,
			})
		}
		return i18n.Text(i18n.MsgBattleReportWeaponBombGeneric)
	case string(objBullet.TypeRocket):
		return i18n.Text(i18n.MsgBattleReportWeaponRocket)
	case string(objBullet.TypeLaser):
		return i18n.Text(i18n.MsgBattleReportWeaponLaser)
	}
	return i18n.Text(i18n.MsgBattleReportWeaponUnknown)
}
