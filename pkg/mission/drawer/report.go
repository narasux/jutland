package drawer

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
	layoutUtils "github.com/narasux/jutland/pkg/utils/layout"
)

// 战报面板配色：深色底 + 金色描边。
var (
	battleReportPanelFill   = color.RGBA{R: 8, G: 15, B: 20, A: 242}
	battleReportPanelBorder = color.RGBA{R: 201, G: 174, B: 110, A: 232}
	battleReportTitleClr    = color.RGBA{R: 240, G: 232, B: 218, A: 255}
	battleReportSubtitleClr = color.RGBA{R: 175, G: 165, B: 150, A: 255}
	battleReportLabelClr    = color.RGBA{R: 214, G: 201, B: 178, A: 255}
	battleReportValueClr    = color.RGBA{R: 236, G: 244, B: 246, A: 255}
	battleReportLostClr     = color.RGBA{R: 236, G: 150, B: 130, A: 255}
	battleReportKillClr     = color.RGBA{R: 240, G: 88, B: 78, A: 255}
	battleReportOwnClr      = color.RGBA{R: 150, G: 220, B: 170, A: 255}
	battleReportNoteClr     = color.RGBA{R: 232, G: 190, B: 110, A: 255}
	battleReportDividerClr  = color.RGBA{R: 214, G: 201, B: 178, A: 80}
	battleReportCardFill    = color.RGBA{R: 16, G: 27, B: 33, A: 235}
	battleReportCardBorder  = color.RGBA{R: 96, G: 116, B: 120, A: 200}
	battleReportButtonFill  = color.RGBA{R: 26, G: 46, B: 54, A: 238}
	battleReportButtonBrd   = color.RGBA{R: 117, G: 170, B: 180, A: 255}
	battleReportHoverFill   = color.RGBA{R: 42, G: 70, B: 80, A: 250}
	battleReportHoverBrd    = color.RGBA{R: 232, G: 224, B: 198, A: 255}
	battleReportTabFill     = color.RGBA{R: 18, G: 28, B: 34, A: 210}
	battleReportTabActive   = color.RGBA{R: 68, G: 116, B: 138, A: 200}
	battleReportTabBorder   = color.RGBA{R: 96, G: 116, B: 120, A: 190}
	battleReportOverlay     = color.RGBA{R: 3, G: 8, B: 12, A: 200}
)

// reportTextAlign 战报文本的水平对齐方式。
type reportTextAlign int

const (
	reportAlignLeft reportTextAlign = iota
	reportAlignCenter
	reportAlignRight
)

// DrawBattleReport 绘制战役总结面板。
// final 为 true 时是结算页（含返回 / 重试按钮），false 时为暂停实时统计。
func (d *Drawer) DrawBattleReport(
	screen *ebiten.Image,
	ms *state.MissionState,
	report state.BattleReportLayout,
	final bool,
) {
	scale := report.Scale
	if scale <= 0 {
		scale = 1
	}

	vector.FillRect(
		screen, 0, 0,
		float32(max(1, ms.View.Layout.Width)), float32(max(1, ms.View.Layout.Height)),
		battleReportOverlay, false,
	)

	panel := report.Panel
	vector.FillRect(
		screen, float32(panel.X), float32(panel.Y), float32(panel.W), float32(panel.H),
		battleReportPanelFill, false,
	)
	vector.StrokeRect(
		screen, float32(panel.X), float32(panel.Y), float32(panel.W), float32(panel.H),
		float32(max(1, 2*scale)), battleReportPanelBorder, false,
	)

	titleFont := font.LocalizedUI(font.Hang)
	bodyFont := font.LocalizedUI(font.Kai)
	monoFont := font.JetbrainsMono

	title := i18n.Text(i18n.MsgBattleReportTitle)
	d.drawReportText(
		screen, title, report.Title,
		fitReportFontSize(title, report.Title.W, 36*scale, titleFont),
		titleFont, battleReportTitleClr, reportAlignLeft,
	)

	// 结算状态做成右上角的「圆点 + 状态字 + 下划线」，不再使用和页签一样的按钮框。
	badgeLabel, badgeClr := battleReportBadge(final, ms.Stats.EndReason)
	badge := report.Badge
	badgeTextSize := 24 * scale
	maxBadgeTextW := badge.W - 30*scale
	badgeTextSize = fitReportFontSize(badgeLabel, maxBadgeTextW, badgeTextSize, titleFont)
	badgeTextW := layoutUtils.CalcTextWidth(badgeLabel, badgeTextSize, font.ForText(badgeLabel, titleFont))
	dotSize := max(5*scale, 4.0)
	gap := 8 * scale
	groupW := dotSize + gap + badgeTextW
	groupX := badge.X + badge.W - groupW
	dotCenterX := groupX + dotSize/2
	dotCenterY := badge.Y + badge.H/2
	vector.FillCircle(
		screen, float32(dotCenterX), float32(dotCenterY), float32(dotSize/2), badgeClr, false,
	)
	d.drawReportText(
		screen, badgeLabel,
		state.BattleReportRect{X: groupX + dotSize + gap, Y: badge.Y, W: badgeTextW, H: badge.H},
		badgeTextSize, titleFont, badgeClr, reportAlignLeft,
	)

	d.drawBattleReportTabs(screen, report, ms.UI.BattleReportPage, scale, bodyFont)

	// 副标题：任务名 · 阵营 · 作战时长。
	durationTick := ms.Core.SimTick
	if ms.Stats.Finalized {
		durationTick = ms.Stats.EndTick
	}
	missionName := ms.Core.MissionMD.DisplayName
	if missionName == "" {
		missionName = ms.Core.Mission
	}
	sideName := string(ms.Core.PlayerSide)
	if sideName == "" {
		sideName = "-"
	}
	subtitle := i18n.Format(i18n.MsgBattleReportSubtitle, map[string]any{
		"Mission":  missionName,
		"Side":     sideName,
		"Duration": state.BattleReportDurationText(durationTick),
	})
	d.drawReportText(
		screen, subtitle, report.Subtitle,
		fitReportFontSize(subtitle, report.Subtitle.W, 19*scale, bodyFont),
		bodyFont, battleReportSubtitleClr, reportAlignLeft,
	)

	dividerY := report.Subtitle.Y + report.Subtitle.H + 8*scale
	vector.StrokeLine(
		screen,
		float32(report.Title.X), float32(dividerY),
		float32(report.Title.X+report.Subtitle.W), float32(dividerY),
		float32(max(1, scale)), battleReportDividerClr, false,
	)

	own := ms.Stats.Side(ms.Player.CurPlayer)
	enemy := ms.Stats.Side(ms.Player.CurEnemy)
	switch ms.UI.BattleReportPage {
	case state.BattleReportPageShips:
		d.drawBattleReportShipDetails(
			screen,
			report,
			own,
			enemy,
			ms.UI.BattleReportShipScroll,
			scale,
			bodyFont,
			monoFont,
		)
	case state.BattleReportPagePlanes:
		d.drawBattleReportPlaneDetails(screen, report, own, enemy, scale, bodyFont, monoFont)
	case state.BattleReportPageEvents:
		d.drawBattleReportEventLog(screen, report, ms, scale, bodyFont, monoFont)
	default:
		d.drawBattleReportCards(screen, report, own, enemy, scale, bodyFont, monoFont)
		d.drawBattleReportOverviewSides(screen, report, ms, own, enemy, scale, bodyFont, monoFont)
		if own.PlanesLostWithShip > 0 {
			note := i18n.Format(i18n.MsgBattleReportWithShip, map[string]any{
				"Count": own.PlanesLostWithShip,
			})
			d.drawReportText(
				screen, note, report.Note,
				fitReportFontSize(note, report.Note.W, 16*scale, bodyFont),
				bodyFont, battleReportNoteClr, reportAlignLeft,
			)
		}
	}
	d.drawBattleReportFooter(screen, report, ms, own, scale, bodyFont)

	if final {
		d.drawReportButton(
			screen, report.BackButton,
			i18n.Text(i18n.MsgBattleReportBack), scale, bodyFont,
		)
		d.drawReportButton(
			screen, report.RetryButton,
			i18n.Text(i18n.MsgBattleReportRetry), scale, bodyFont,
		)
		return
	}
	hint := i18n.Text(i18n.MsgBattleReportLiveHint)
	d.drawReportText(
		screen, hint, report.Hint,
		fitReportFontSize(hint, report.Hint.W, 17*scale, bodyFont),
		bodyFont, battleReportSubtitleClr, reportAlignCenter,
	)
}

// drawBattleReportTabs 绘制总览 / 舰艇明细 / 飞机明细页签。
func (d *Drawer) drawBattleReportTabs(
	screen *ebiten.Image,
	report state.BattleReportLayout,
	page state.BattleReportPage,
	scale float64,
	bodyFont *text.GoTextFaceSource,
) {
	labels := [4]string{
		i18n.Text(i18n.MsgBattleReportPageOverview),
		i18n.Text(i18n.MsgBattleReportPageShips),
		i18n.Text(i18n.MsgBattleReportPagePlanes),
		i18n.Text(i18n.MsgBattleReportPageEvents),
	}
	sx, sy := ebiten.CursorPosition()
	for idx, tab := range report.Tabs {
		if tab.Empty() {
			continue
		}
		active := page == state.BattleReportPage(idx)
		hovered := tab.Contains(sx, sy)
		fill, border := battleReportTabFill, battleReportTabBorder
		textClr := battleReportSubtitleClr
		if active {
			fill, border, textClr = battleReportTabActive, battleReportButtonBrd, colorx.White
		} else if hovered {
			border, textClr = battleReportHoverBrd, battleReportLabelClr
		}
		vector.FillRect(
			screen, float32(tab.X), float32(tab.Y), float32(tab.W), float32(tab.H),
			fill, false,
		)
		vector.StrokeRect(
			screen, float32(tab.X), float32(tab.Y), float32(tab.W), float32(tab.H),
			float32(max(1, scale)), border, false,
		)
		label := labels[idx]
		d.drawReportText(
			screen, label, tab,
			fitReportFontSize(label, tab.W-10*scale, 15*scale, bodyFont),
			bodyFont, textClr, reportAlignCenter,
		)
	}
}

// drawBattleReportCards 绘制四张核心卡。
func (d *Drawer) drawBattleReportCards(
	screen *ebiten.Image,
	report state.BattleReportLayout,
	own, enemy state.BattleSideStats,
	scale float64,
	bodyFont, monoFont *text.GoTextFaceSource,
) {
	type cardData struct {
		title string
		value string
		sub   string
	}
	cards := [4]cardData{
		{
			title: i18n.Text(i18n.MsgBattleReportBuiltShipsCard),
			value: groupThousands(int64(own.ShipsBuilt)),
			sub: i18n.Format(i18n.MsgBattleReportBuiltShipsSub, map[string]any{
				"Count": own.ShipsInitial,
			}),
		},
		{
			title: i18n.Text(i18n.MsgBattleReportShipLossesCard),
			value: groupThousands(int64(own.ShipsLost)),
			sub: i18n.Format(i18n.MsgBattleReportShipLossesSub, map[string]any{
				"Tonnage": groupThousands(int64(math.Round(own.SunkTonnage))),
			}),
		},
		{
			title: i18n.Text(i18n.MsgBattleReportEnemySunkCard),
			value: groupThousands(int64(enemy.ShipsLost)),
			sub: i18n.Format(i18n.MsgBattleReportEnemySunkSub, map[string]any{
				"Tonnage": groupThousands(int64(math.Round(enemy.SunkTonnage))),
			}),
		},
		{
			title: i18n.Text(i18n.MsgBattleReportPlaneLossesCard),
			value: groupThousands(int64(own.PlanesLost)),
			sub: i18n.Format(i18n.MsgBattleReportPlaneLossesSub, map[string]any{
				"Count": enemy.PlanesLost,
			}),
		},
	}

	for idx, card := range report.Cards {
		data := cards[idx]
		vector.FillRect(
			screen, float32(card.X), float32(card.Y), float32(card.W), float32(card.H),
			battleReportCardFill, false,
		)
		vector.StrokeRect(
			screen, float32(card.X), float32(card.Y), float32(card.W), float32(card.H),
			float32(max(1, scale)), battleReportCardBorder, false,
		)

		titleRect := state.BattleReportRect{
			X: card.X, Y: card.Y + 10*scale, W: card.W, H: 22 * scale,
		}
		valueRect := state.BattleReportRect{
			X: card.X, Y: card.Y + 34*scale, W: card.W, H: 42 * scale,
		}
		subRect := state.BattleReportRect{
			X: card.X, Y: card.Y + 78*scale, W: card.W, H: 20 * scale,
		}
		d.drawReportText(
			screen, data.title, titleRect,
			fitReportFontSize(data.title, card.W-20*scale, 17*scale, bodyFont),
			bodyFont, battleReportSubtitleClr, reportAlignCenter,
		)
		d.drawReportText(
			screen, data.value, valueRect,
			fitReportFontSize(data.value, card.W-20*scale, 42*scale, monoFont),
			monoFont, battleReportTitleClr, reportAlignCenter,
		)
		d.drawReportText(
			screen, data.sub, subRect,
			fitReportFontSize(data.sub, card.W-16*scale, 16*scale, bodyFont),
			bodyFont, battleReportLabelClr, reportAlignCenter,
		)
	}
}

// drawBattleReportOverviewSides 绘制双方分开的汇总栏。
func (d *Drawer) drawBattleReportOverviewSides(
	screen *ebiten.Image,
	report state.BattleReportLayout,
	ms *state.MissionState,
	own, enemy state.BattleSideStats,
	scale float64,
	bodyFont, monoFont *text.GoTextFaceSource,
) {
	d.drawBattleReportSidePanel(
		screen, report.Sides[0], i18n.Text(i18n.MsgBattleReportPlayer), own, true,
		ms, scale, bodyFont, monoFont,
	)
	d.drawBattleReportSidePanel(
		screen, report.Sides[1], i18n.Text(i18n.MsgBattleReportEnemy), enemy, false,
		ms, scale, bodyFont, monoFont,
	)
}

// drawBattleReportSidePanel 绘制单个阵营的汇总表。
func (d *Drawer) drawBattleReportSidePanel(
	screen *ebiten.Image,
	rect state.BattleReportRect,
	title string,
	side state.BattleSideStats,
	ownSide bool,
	ms *state.MissionState,
	scale float64,
	bodyFont, monoFont *text.GoTextFaceSource,
) {
	vector.FillRect(
		screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
		battleReportCardFill, false,
	)
	vector.StrokeRect(
		screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
		float32(max(1, scale)), battleReportCardBorder, false,
	)

	pad := 18 * scale
	contentX := rect.X + pad
	innerW := rect.W - 2*pad
	titleRect := state.BattleReportRect{
		X: contentX, Y: rect.Y + 10*scale, W: innerW, H: 28 * scale,
	}
	titleClr := battleReportLostClr
	if ownSide {
		titleClr = battleReportOwnClr
	}
	d.drawReportText(
		screen, title, titleRect,
		fitReportFontSize(title, innerW, 22*scale, bodyFont),
		bodyFont, titleClr, reportAlignLeft,
	)

	headerY := rect.Y + 44*scale
	headerH := 24 * scale
	labelW := innerW * 0.30
	numW := (innerW - labelW) / 4
	headers := [4]string{
		i18n.Text(i18n.MsgBattleReportInitial),
		i18n.Text(i18n.MsgBattleReportBuilt),
		i18n.Text(i18n.MsgBattleReportLost),
		i18n.Text(i18n.MsgBattleReportRemaining),
	}
	for idx, header := range headers {
		cell := state.BattleReportRect{
			X: contentX + labelW + float64(idx)*numW, Y: headerY, W: numW, H: headerH,
		}
		d.drawReportText(
			screen, header, cell,
			fitReportFontSize(header, numW-6*scale, 14*scale, bodyFont),
			bodyFont, battleReportSubtitleClr, reportAlignRight,
		)
	}
	headerLineY := headerY + headerH
	vector.StrokeLine(
		screen,
		float32(contentX), float32(headerLineY),
		float32(contentX+innerW), float32(headerLineY),
		float32(max(1, scale)), battleReportDividerClr, false,
	)

	rowH := 36 * scale
	rowsY := headerLineY + 4*scale
	rowRect := func(row int, x, w float64) state.BattleReportRect {
		return state.BattleReportRect{X: x, Y: rowsY + float64(row)*rowH, W: w, H: rowH}
	}
	drawShipRow := func(row int, label string, values [4]int) {
		d.drawReportText(
			screen, label, rowRect(row, contentX, labelW),
			fitReportFontSize(label, labelW-6*scale, 17*scale, bodyFont),
			bodyFont, battleReportLabelClr, reportAlignLeft,
		)
		for col, value := range values {
			clr := battleReportValueClr
			if col == 2 && value > 0 {
				clr = battleReportLostClr
			}
			text := groupThousands(int64(value))
			cell := rowRect(row, contentX+labelW+float64(col)*numW, numW)
			d.drawReportText(
				screen, text, cell,
				fitReportFontSize(text, numW-6*scale, 17*scale, monoFont),
				monoFont, clr, reportAlignRight,
			)
		}
	}
	drawShipRow(0, i18n.Text(i18n.MsgBattleReportShips), [4]int{
		side.ShipsInitial, side.ShipsBuilt, side.ShipsLost, side.RemainingShips(),
	})
	drawShipRow(1, i18n.Text(i18n.MsgBattleReportAircraft), [4]int{
		side.PlanesInitial, side.PlanesBuilt, side.PlanesLost, side.RemainingPlanes(),
	})

	mergedRows := []struct {
		label string
		value string
		lost  bool
	}{
		{
			label: i18n.Text(i18n.MsgBattleReportInitialTonnage),
			value: groupThousands(int64(math.Round(side.InitialTonnage))),
		},
		{
			label: i18n.Text(i18n.MsgBattleReportTonnage),
			value: groupThousands(int64(math.Round(side.SunkTonnage))),
			lost:  true,
		},
		{
			label: i18n.Text(i18n.MsgBattleReportActiveTonnage),
			value: groupThousands(int64(math.Round(side.SurvivingTonnage()))),
		},
		{
			label: i18n.Text(i18n.MsgBattleReportDamage),
			value: groupThousands(int64(math.Round(side.DamageDealt))),
		},
	}
	for idx, row := range mergedRows {
		rowIndex := idx + 2
		d.drawReportText(
			screen, row.label, rowRect(rowIndex, contentX, labelW),
			fitReportFontSize(row.label, labelW-6*scale, 17*scale, bodyFont),
			bodyFont, battleReportLabelClr, reportAlignLeft,
		)
		clr := battleReportValueClr
		if row.lost {
			clr = battleReportLostClr
		}
		cell := rowRect(rowIndex, contentX+labelW, innerW-labelW)
		d.drawReportText(
			screen, row.value, cell,
			fitReportFontSize(row.value, innerW-labelW, 17*scale, monoFont),
			monoFont, clr, reportAlignRight,
		)
	}

	// 资金单独一行：玩家侧展示初始 → 结余（收入 / 支出），敌方显示无数据。
	fundsRow := 2 + len(mergedRows)
	d.drawReportText(
		screen, i18n.Text(i18n.MsgBattleReportFunds), rowRect(fundsRow, contentX, labelW),
		fitReportFontSize(i18n.Text(i18n.MsgBattleReportFunds), labelW-6*scale, 17*scale, bodyFont),
		bodyFont, battleReportLabelClr, reportAlignLeft,
	)
	fundsText := i18n.Text(i18n.MsgBattleReportNoData)
	if ownSide {
		fundsEnd := ms.Stats.FundsStart + ms.Stats.FundsEarned - ms.Stats.FundsSpent
		fundsText = fmt.Sprintf(
			"%s → %s (+%s / -%s)",
			groupThousands(ms.Stats.FundsStart),
			groupThousands(fundsEnd),
			groupThousands(ms.Stats.FundsEarned),
			groupThousands(ms.Stats.FundsSpent),
		)
	}
	fundsCell := rowRect(fundsRow, contentX+labelW, innerW-labelW)
	d.drawReportText(
		screen, fundsText, fundsCell,
		fitReportFontSize(fundsText, innerW-labelW, 16*scale, monoFont),
		monoFont, battleReportValueClr, reportAlignRight,
	)
}

// drawBattleReportFooter 绘制交换比与保存率。
func (d *Drawer) drawBattleReportFooter(
	screen *ebiten.Image,
	report state.BattleReportLayout,
	ms *state.MissionState,
	own state.BattleSideStats,
	scale float64,
	bodyFont *text.GoTextFaceSource,
) {
	ratioText := i18n.Text(i18n.MsgBattleReportNoData)
	if ratio, ok := ms.Stats.ExchangeRatio(ms.Player.CurPlayer, ms.Player.CurEnemy); ok {
		if math.IsInf(ratio, 1) {
			ratioText = i18n.Text(i18n.MsgBattleReportNoLoss)
		} else {
			ratioText = fmt.Sprintf("%.1f:1", ratio)
		}
	}
	rateText := i18n.Text(i18n.MsgBattleReportNoData)
	if rate, ok := own.SurvivalRate(); ok {
		rateText = fmt.Sprintf("%d%%", int(math.Round(rate*100)))
	}
	exchange := i18n.Format(i18n.MsgBattleReportExchange, map[string]any{"Ratio": ratioText})
	survival := i18n.Format(i18n.MsgBattleReportSurvival, map[string]any{"Rate": rateText})
	footer := exchange + "   ·   " + survival

	// 交换比与保存率是次要信息：字号比正文小一档，颜色降到副标题级，避免抢视线。
	d.drawReportText(
		screen, footer, report.Footer,
		fitReportFontSize(footer, report.Footer.W, 16*scale, bodyFont),
		bodyFont, battleReportSubtitleClr, reportAlignLeft,
	)
}

// drawReportButton 绘制底部按钮，悬停时高亮边框。
func (d *Drawer) drawReportButton(
	screen *ebiten.Image,
	rect state.BattleReportRect,
	label string,
	scale float64,
	textFont *text.GoTextFaceSource,
) {
	if rect.Empty() {
		return
	}
	sx, sy := ebiten.CursorPosition()
	hovered := rect.Contains(sx, sy)
	fill, border := battleReportButtonFill, battleReportButtonBrd
	if hovered {
		fill, border = battleReportHoverFill, battleReportHoverBrd
	}
	vector.FillRect(
		screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H), fill, false,
	)
	vector.StrokeRect(
		screen, float32(rect.X), float32(rect.Y), float32(rect.W), float32(rect.H),
		float32(max(1, 2*scale)), border, false,
	)
	d.drawReportText(
		screen, label, rect,
		fitReportFontSize(label, rect.W-18*scale, 18*scale, textFont),
		textFont, colorx.White, reportAlignCenter,
	)
}

// drawReportText 在矩形内按指定对齐方式绘制单行文本并垂直居中。
func (d *Drawer) drawReportText(
	screen *ebiten.Image,
	value string,
	rect state.BattleReportRect,
	fontSize float64,
	textFont *text.GoTextFaceSource,
	textColor color.Color,
	align reportTextAlign,
) {
	if value == "" || rect.Empty() || fontSize <= 0 {
		return
	}
	measuredFont := font.ForText(value, textFont)
	width, height := text.Measure(value, &text.GoTextFace{Source: measuredFont, Size: fontSize}, 0)
	x := rect.X
	switch align {
	case reportAlignCenter:
		x = rect.X + (rect.W-width)/2
	case reportAlignRight:
		x = rect.X + rect.W - width
	}
	y := rect.Y + (rect.H-height)/2
	d.drawText(screen, value, x, y, fontSize, measuredFont, textColor)
}

// fitReportFontSize 把字号缩到文本能放进指定宽度。
func fitReportFontSize(value string, maxWidth, fontSize float64, textFont *text.GoTextFaceSource) float64 {
	if maxWidth <= 0 || fontSize <= 0 {
		return fontSize
	}
	width := layoutUtils.CalcTextWidth(value, fontSize, font.ForText(value, textFont))
	if width <= maxWidth || width <= 0 {
		return fontSize
	}
	return max(8, fontSize*maxWidth/width)
}

// battleReportBadge 返回结算徽标的文案与配色。
func battleReportBadge(final bool, reason state.BattleEndReason) (label string, accent color.RGBA) {
	if !final {
		return i18n.Text(i18n.MsgBattleReportInProgress), color.RGBA{R: 117, G: 200, B: 215, A: 255}
	}
	switch reason {
	case state.BattleEndVictory:
		return i18n.Text(i18n.MsgBattleReportVictory), color.RGBA{R: 130, G: 225, B: 160, A: 255}
	case state.BattleEndAborted:
		return i18n.Text(i18n.MsgBattleReportAborted), color.RGBA{R: 236, G: 198, B: 105, A: 255}
	default:
		return i18n.Text(i18n.MsgBattleReportDefeat), color.RGBA{R: 236, G: 135, B: 112, A: 255}
	}
}

// groupThousands 给整数加千位分隔符。
func groupThousands(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	digits := fmt.Sprintf("%d", value)
	first := len(digits) % 3
	if first == 0 {
		first = 3
	}
	var builder strings.Builder
	builder.Grow(len(digits) + len(digits)/3 + len(sign))
	builder.WriteString(sign)
	builder.WriteString(digits[:first])
	for idx := first; idx < len(digits); idx += 3 {
		builder.WriteByte(',')
		builder.WriteString(digits[idx : idx+3])
	}
	return builder.String()
}
