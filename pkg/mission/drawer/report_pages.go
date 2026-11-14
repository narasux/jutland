package drawer

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/i18n"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

// battleReportDetailRow 明细页的一行（舰级或机型）。
type battleReportDetailRow struct {
	Name      string
	Initial   int
	Built     int
	Lost      int
	Remaining int
}

func (r battleReportDetailRow) total() int {
	return r.Initial + r.Built + r.Lost
}

// battleReportRowBadge 明细行名前的标识：舰艇用缩略图，飞机用机型缩写标签。
type battleReportRowBadge struct {
	icon  func(string) *ebiten.Image
	label func(string) string
	color color.Color
}

// buildBattleShipDetailRows 汇总单阵营舰艇明细，按吨位降序返回全部舰级。
func buildBattleShipDetailRows(stats state.BattleSideStats) []battleReportDetailRow {
	type shipRow struct {
		battleReportDetailRow
		tonnage float64
	}
	byName := detailRowsByName(
		stats.ShipsInitialByClass, stats.ShipsBuiltByClass, stats.ShipsLostByClass,
	)
	rows := make([]shipRow, 0, len(byName))
	for _, row := range byName {
		tonnage := 0.0
		if ship, ok := objUnit.ShipMap[row.Name]; ok {
			tonnage = ship.Tonnage
		}
		rows = append(rows, shipRow{battleReportDetailRow: *row, tonnage: tonnage})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].tonnage != rows[j].tonnage {
			return rows[i].tonnage > rows[j].tonnage
		}
		if rows[i].total() != rows[j].total() {
			return rows[i].total() > rows[j].total()
		}
		return rows[i].Name < rows[j].Name
	})

	detail := make([]battleReportDetailRow, 0, len(rows))
	for _, row := range rows {
		// 没有初始 / 建造 / 损失记录的空行不占表格行，也不应触发滚动条。
		if row.Initial+row.Built+row.Lost+row.Remaining == 0 {
			continue
		}
		detail = append(detail, row.battleReportDetailRow)
	}
	return detail
}

// buildBattlePlaneDetailRows 汇总单阵营飞机明细，按损失降序返回全部机型。
func buildBattlePlaneDetailRows(stats state.BattleSideStats) []battleReportDetailRow {
	byName := detailRowsByName(
		stats.PlanesInitialByType, stats.PlanesBuiltByType, stats.PlanesLostByType,
	)
	rows := make([]battleReportDetailRow, 0, len(byName))
	for _, row := range byName {
		// 没有初始 / 建造 / 损失记录的空行不占表格行。
		if row.Initial+row.Built+row.Lost+row.Remaining == 0 {
			continue
		}
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Lost != rows[j].Lost {
			return rows[i].Lost > rows[j].Lost
		}
		if rows[i].total() != rows[j].total() {
			return rows[i].total() > rows[j].total()
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

// detailRowsByName 合并初始 / 建造 / 损失三张表，并计算在役数量。
func detailRowsByName(
	initial, built, lost map[string]int,
) map[string]*battleReportDetailRow {
	byName := map[string]*battleReportDetailRow{}
	ensure := func(name string) *battleReportDetailRow {
		row := byName[name]
		if row == nil {
			row = &battleReportDetailRow{Name: name}
			byName[name] = row
		}
		return row
	}
	for name, count := range initial {
		ensure(name).Initial += count
	}
	for name, count := range built {
		ensure(name).Built += count
	}
	for name, count := range lost {
		ensure(name).Lost += count
	}
	for _, row := range byName {
		row.Remaining = max(0, row.Initial+row.Built-row.Lost)
	}
	return byName
}

// battleShipDetailTotals 返回舰艇明细的合计行。
func battleShipDetailTotals(stats state.BattleSideStats) battleReportDetailRow {
	return battleReportDetailRow{
		Initial:   stats.ShipsInitial,
		Built:     stats.ShipsBuilt,
		Lost:      stats.ShipsLost,
		Remaining: stats.RemainingShips(),
	}
}

// battlePlaneDetailTotals 返回飞机明细的合计行。
func battlePlaneDetailTotals(stats state.BattleSideStats) battleReportDetailRow {
	return battleReportDetailRow{
		Initial:   stats.PlanesInitial,
		Built:     stats.PlanesBuilt,
		Lost:      stats.PlanesLost,
		Remaining: stats.RemainingPlanes(),
	}
}

// drawBattleReportShipDetails 绘制双方舰艇明细页。
func (d *Drawer) drawBattleReportShipDetails(
	screen *ebiten.Image,
	report state.BattleReportLayout,
	own, enemy state.BattleSideStats,
	scroll [2]int,
	scale float64,
	bodyFont, monoFont *text.GoTextFaceSource,
) {
	d.drawBattleReportDetailPanel(
		screen, report.DetailSides[0],
		i18n.Text(i18n.MsgBattleReportOwnShips),
		i18n.Text(i18n.MsgBattleReportShipClass),
		i18n.Text(i18n.MsgBattleReportBuilt),
		buildBattleShipDetailRows(own),
		battleShipDetailTotals(own),
		objUnit.GetShipDisplayName,
		battleReportRowBadge{
			icon: func(name string) *ebiten.Image { return battleReportShipIcon(name, true) },
		},
		scroll[0],
		scale, bodyFont, monoFont,
	)
	d.drawBattleReportDetailPanel(
		screen, report.DetailSides[1],
		i18n.Text(i18n.MsgBattleReportEnemyShips),
		i18n.Text(i18n.MsgBattleReportShipClass),
		i18n.Text(i18n.MsgBattleReportBuilt),
		buildBattleShipDetailRows(enemy),
		battleShipDetailTotals(enemy),
		objUnit.GetShipDisplayName,
		battleReportRowBadge{
			icon: func(name string) *ebiten.Image { return battleReportShipIcon(name, false) },
		},
		scroll[1],
		scale, bodyFont, monoFont,
	)
}

// drawBattleReportPlaneDetails 绘制双方飞机明细页。
func (d *Drawer) drawBattleReportPlaneDetails(
	screen *ebiten.Image,
	report state.BattleReportLayout,
	own, enemy state.BattleSideStats,
	scroll [2]int,
	scale float64,
	bodyFont, monoFont *text.GoTextFaceSource,
) {
	d.drawBattleReportDetailPanel(
		screen, report.DetailSides[0],
		i18n.Text(i18n.MsgBattleReportOwnPlanes),
		i18n.Text(i18n.MsgBattleReportPlaneType),
		i18n.Text(i18n.MsgBattleReportReplenished),
		buildBattlePlaneDetailRows(own),
		battlePlaneDetailTotals(own),
		objUnit.GetPlaneDisplayName,
		battleReportRowBadge{
			label: battleReportPlaneTypeAbbr,
			color: battleReportOwnClr,
		},
		scroll[0],
		scale, bodyFont, monoFont,
	)
	d.drawBattleReportDetailPanel(
		screen, report.DetailSides[1],
		i18n.Text(i18n.MsgBattleReportEnemyPlanes),
		i18n.Text(i18n.MsgBattleReportPlaneType),
		i18n.Text(i18n.MsgBattleReportReplenished),
		buildBattlePlaneDetailRows(enemy),
		battlePlaneDetailTotals(enemy),
		objUnit.GetPlaneDisplayName,
		battleReportRowBadge{
			label: battleReportPlaneTypeAbbr,
			color: battleReportLostClr,
		},
		scroll[1],
		scale, bodyFont, monoFont,
	)
}

// battleReportPlaneTypeAbbr 返回机型缩写标签。
// 标签宽度固定为两个字符，超长缩写截断，其他机型用 O，保证各行名称对齐。
func battleReportPlaneTypeAbbr(name string) string {
	plane := objUnit.PlaneMap[name]
	if plane == nil {
		return ""
	}
	if plane.Type == objUnit.PlaneTypeOther {
		return "O"
	}
	abbr := []rune(plane.TypeAbbr)
	if len(abbr) > 2 {
		abbr = abbr[:2]
	}
	return string(abbr)
}

// battleReportShipIcon 获取舰级对应的缩略图标；配色与地图一致：我方绿色，敌方红色。
func battleReportShipIcon(name string, own bool) *ebiten.Image {
	abbr := "default"
	if ship := objUnit.ShipMap[name]; ship != nil && ship.TypeAbbr != "" {
		abbr = ship.TypeAbbr
	}
	return textureImg.GetAbbrShip(abbr, !own)
}

// battleReportDetailScrollStart 返回明细列表实际绘制的起始行。
// 没有滚动条时固定从第一行开始；有滚动条时按状态偏移夹紧，
// 保证绘制窗口与滚动条滑块位置一致。
func battleReportDetailScrollStart(
	scroll int, scrollbar state.BattleReportScrollbar, hasScrollbar bool,
) int {
	if !hasScrollbar {
		return 0
	}
	return min(max(0, scroll), scrollbar.MaxScroll)
}

// drawBattleReportTypeTag 绘制机型缩写标签，返回它占用的宽度（含与名称的间距）。
// 标签按两个字符的宽度固定，边框画成整数像素的实心矩形，避免缩放后四边粗细不均。
func (d *Drawer) drawBattleReportTypeTag(
	screen *ebiten.Image,
	label string,
	x, y, height, scale float64,
	clr color.Color,
	monoFont *text.GoTextFaceSource,
) float64 {
	if label == "" || height <= 0 || scale <= 0 {
		return 0
	}
	fontSize := 11 * scale
	// 等宽字体下两个字符就是固定标签宽度；单字符标签也占同样的宽度。
	textW, _ := text.Measure("MM", &text.GoTextFace{Source: monoFont, Size: fontSize}, 0)
	tagW := math.Round(max(height*1.15, textW+8*scale))
	tagH := math.Round(height)
	if tagW < 1 || tagH < 1 {
		return 0
	}
	x, y = math.Round(x), math.Round(y)
	border := float64(max(1, int(math.Round(scale))))
	vector.FillRect(
		screen, float32(x), float32(y), float32(tagW), float32(tagH),
		clr, false,
	)
	innerW, innerH := tagW-2*border, tagH-2*border
	if innerW > 0 && innerH > 0 {
		vector.FillRect(
			screen, float32(x+border), float32(y+border), float32(innerW), float32(innerH),
			battleReportTagFill, false,
		)
	}
	d.drawReportText(
		screen, label,
		state.BattleReportRect{X: x, Y: y, W: tagW, H: tagH},
		fontSize, monoFont, clr, reportAlignCenter,
	)
	return tagW + 7*scale
}

// drawBattleReportDetailPanel 绘制单个阵营的舰级 / 机型明细表。
// 舰艇 / 飞机列表超出可视行数时由 scroll 控制，滚动条绘制在右侧预留的独立槽位中，
// 表格分隔线只画到槽位之前，避免与滚动条重叠。
func (d *Drawer) drawBattleReportDetailPanel(
	screen *ebiten.Image,
	rect state.BattleReportRect,
	title, nameHeader, builtHeader string,
	rows []battleReportDetailRow,
	totals battleReportDetailRow,
	nameOf func(string) string,
	badge battleReportRowBadge,
	scroll int,
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
		X: contentX, Y: rect.Y + 10*scale, W: innerW, H: 30 * scale,
	}
	d.drawReportText(
		screen, title, titleRect,
		fitReportFontSize(title, innerW, 22*scale, bodyFont),
		bodyFont, battleReportTitleClr, reportAlignLeft,
	)

	rowH := state.BattleReportDetailRowHeight * scale
	totalH := 32 * scale
	headerY := rect.Y + 46*scale
	headerH := 26 * scale
	rowsY := headerY + headerH + 4*scale
	totalY := rect.Y + rect.H - pad - totalH
	listBottom := totalY - 6*scale
	visible := max(1, state.BattleReportDetailVisibleRows(rect.H, scale))
	scrollbar, hasScrollbar := state.BattleReportDetailScrollbar(rect, len(rows), scroll, scale)
	scroll = battleReportDetailScrollStart(scroll, scrollbar, hasScrollbar)

	// 右侧预留独立滚动条槽位，表格宽度与分隔线都到此为止。
	contentW := innerW
	if hasScrollbar {
		contentW -= 14 * scale
	}
	nameW := contentW * 0.34
	numW := (contentW - nameW) / 4
	headerSize := 14 * scale
	headers := [4]string{
		i18n.Text(i18n.MsgBattleReportInitial),
		builtHeader,
		i18n.Text(i18n.MsgBattleReportLost),
		i18n.Text(i18n.MsgBattleReportRemaining),
	}
	d.drawReportText(
		screen, nameHeader,
		state.BattleReportRect{X: contentX, Y: headerY, W: nameW, H: headerH},
		fitReportFontSize(nameHeader, nameW-6*scale, headerSize, bodyFont),
		bodyFont, battleReportSubtitleClr, reportAlignLeft,
	)
	for idx, header := range headers {
		cell := state.BattleReportRect{
			X: contentX + nameW + float64(idx)*numW, Y: headerY, W: numW, H: headerH,
		}
		d.drawReportText(
			screen, header, cell,
			fitReportFontSize(header, numW-6*scale, headerSize, bodyFont),
			bodyFont, battleReportSubtitleClr, reportAlignRight,
		)
	}
	headerLineY := headerY + headerH
	vector.StrokeLine(
		screen,
		float32(contentX), float32(headerLineY),
		float32(contentX+contentW), float32(headerLineY),
		float32(max(1, scale)), battleReportDividerClr, false,
	)

	if len(rows) == 0 {
		emptyRect := state.BattleReportRect{
			X: contentX, Y: rowsY, W: contentW, H: listBottom - rowsY,
		}
		empty := i18n.Text(i18n.MsgBattleReportNoRecords)
		d.drawReportText(
			screen, empty, emptyRect,
			fitReportFontSize(empty, contentW, 16*scale, bodyFont),
			bodyFont, battleReportSubtitleClr, reportAlignCenter,
		)
	} else {
		end := min(len(rows), scroll+visible)
		for idx := scroll; idx < end; idx++ {
			row := rows[idx]
			y := rowsY + float64(idx-scroll)*rowH
			name := nameOf(row.Name)
			rowNameX := contentX
			rowNameW := nameW
			if row.Name != "" {
				switch {
				case badge.icon != nil:
					icon := badge.icon(row.Name)
					iconSize := 18 * scale
					opts := d.genDefaultDrawImageOptions()
					// 缩略图素材朝上，战报里统一转 90 度让舰艏朝右。
					ebutil.SetOptsCenterRotation(opts, icon, 90)
					opts.GeoM.Scale(
						iconSize/float64(icon.Bounds().Dx()),
						iconSize/float64(icon.Bounds().Dy()),
					)
					opts.GeoM.Translate(contentX, y+(rowH-iconSize)/2)
					screen.DrawImage(icon, opts)
					rowNameX += iconSize + 7*scale
					rowNameW -= iconSize + 7*scale
				case badge.label != nil:
					tagH := 16 * scale
					leadingW := d.drawBattleReportTypeTag(
						screen, badge.label(row.Name),
						contentX, y+(rowH-tagH)/2, tagH, scale,
						badge.color, monoFont,
					)
					rowNameX += leadingW
					rowNameW -= leadingW
				}
			}
			d.drawReportText(
				screen, name,
				state.BattleReportRect{X: rowNameX, Y: y, W: max(0, rowNameW), H: rowH},
				fitReportFontSize(name, rowNameW-6*scale, 16*scale, bodyFont),
				bodyFont, battleReportValueClr, reportAlignLeft,
			)
			values := [4]int{row.Initial, row.Built, row.Lost, row.Remaining}
			for col, value := range values {
				clr := battleReportValueClr
				if col == 2 && value > 0 {
					clr = battleReportLostClr
				}
				text := groupThousands(int64(value))
				cell := state.BattleReportRect{
					X: contentX + nameW + float64(col)*numW, Y: y, W: numW, H: rowH,
				}
				d.drawReportText(
					screen, text, cell,
					fitReportFontSize(text, numW-6*scale, 16*scale, monoFont),
					monoFont, clr, reportAlignRight,
				)
			}
			if idx < len(rows)-1 {
				separatorY := y + rowH
				vector.StrokeLine(
					screen,
					float32(contentX), float32(separatorY),
					float32(contentX+contentW), float32(separatorY),
					float32(max(1, scale)), battleReportDividerClr, false,
				)
			}
		}
	}

	// 合计行固定在面板底部，滚动时始终可见。
	vector.StrokeLine(
		screen,
		float32(contentX), float32(totalY),
		float32(contentX+contentW), float32(totalY),
		float32(max(1, scale)), battleReportPanelBorder, false,
	)
	d.drawReportText(
		screen, i18n.Text(i18n.MsgBattleReportTotal),
		state.BattleReportRect{X: contentX, Y: totalY, W: nameW, H: totalH},
		fitReportFontSize(i18n.Text(i18n.MsgBattleReportTotal), nameW-6*scale, 16*scale, bodyFont),
		bodyFont, battleReportTitleClr, reportAlignLeft,
	)
	totalValues := [4]int{totals.Initial, totals.Built, totals.Lost, totals.Remaining}
	for col, value := range totalValues {
		clr := battleReportTitleClr
		if col == 2 && value > 0 {
			clr = battleReportLostClr
		}
		text := groupThousands(int64(value))
		cell := state.BattleReportRect{
			X: contentX + nameW + float64(col)*numW, Y: totalY, W: numW, H: totalH,
		}
		d.drawReportText(
			screen, text, cell,
			fitReportFontSize(text, numW-6*scale, 16*scale, monoFont),
			monoFont, clr, reportAlignRight,
		)
	}

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
}
