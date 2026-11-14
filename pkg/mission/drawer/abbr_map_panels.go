package drawer

import (
	"image/color"
	"math"
	"sort"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/layout"
)

// 全屏地图左右态势栏的几何与字号。面板宽度随地图两侧留白收缩，
// 窄屏降级为只显示标题与总数，避免再出现剪影越界的问题。
const (
	abbrPanelMargin     = 16.0
	abbrPanelMinWidth   = 80.0
	abbrPanelMaxWidth   = 340.0
	abbrPanelCompactW   = 150.0
	abbrPanelTop        = 48.0
	abbrPanelBottom     = 48.0
	abbrPanelTitleSize  = 28.0
	abbrPanelHeadSize   = 20.0
	abbrPanelBodySize   = 19.0
	abbrPanelFooterSize = 15.0
)

var (
	// 己方 / 敌方态势栏的强调色，与地图上的绿 / 红标记保持一致。
	abbrAllyAccent  = color.RGBA{R: 126, G: 198, B: 150, A: 255}
	abbrEnemyAccent = color.RGBA{R: 214, G: 106, B: 96, A: 255}
	// 分隔线用低饱和金色，和木纹背景、金色标题同一暖色系。
	abbrPanelDivider = color.RGBA{R: 196, G: 171, B: 101, A: 120}
)

// abbrPanelRect 是态势栏的屏幕矩形。
type abbrPanelRect struct {
	X, Y, W, H float64
}

// abbrSidePanels 是左右两块态势栏的布局结果。
type abbrSidePanels struct {
	Left, Right abbrPanelRect
	Compact     bool
}

// calcAbbrSidePanels 根据屏幕与缩略地图宽度计算左右态势栏矩形。
// 地图居中绘制，两侧留白相等；态势栏在各自留白内水平居中，
// 留白不足时返回零值，调用方直接跳过绘制。
func calcAbbrSidePanels(screenW, screenH, abbrMapW int) abbrSidePanels {
	colW := (float64(screenW) - float64(abbrMapW)) / 2
	panelW := math.Min(colW-2*abbrPanelMargin, abbrPanelMaxWidth)
	if panelW < abbrPanelMinWidth {
		return abbrSidePanels{}
	}
	h := float64(screenH) - abbrPanelTop - abbrPanelBottom
	inset := (colW - panelW) / 2
	return abbrSidePanels{
		Left: abbrPanelRect{
			X: inset, Y: abbrPanelTop, W: panelW, H: h,
		},
		Right: abbrPanelRect{
			X: float64(screenW) - inset - panelW, Y: abbrPanelTop, W: panelW, H: h,
		},
		Compact: panelW < abbrPanelCompactW,
	}
}

// fleetTypeRow 是按舰种汇总后的一行态势。
type fleetTypeRow struct {
	shipType objUnit.ShipType
	count    int
}

// fleetTypeRows 把按舰级聚合的编成再按舰种汇总，便于窄栏展示。
func fleetTypeRows(fleet state.Fleet) []fleetTypeRow {
	counts := make(map[objUnit.ShipType]int, len(fleet.Classes))
	for _, cls := range fleet.Classes {
		counts[cls.Kind.Type] += cls.Total
	}
	rows := make([]fleetTypeRow, 0, len(counts))
	for shipType, count := range counts {
		rows = append(rows, fleetTypeRow{shipType: shipType, count: count})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return shipTypeOrder(rows[i].shipType) < shipTypeOrder(rows[j].shipType)
	})
	return rows
}

// shipTypeOrder 是态势栏的舰种排序权重：主力舰在前，辅助与运输在后。
func shipTypeOrder(shipType objUnit.ShipType) int {
	switch shipType {
	case objUnit.ShipTypeAircraftCarrier:
		return 0
	case objUnit.ShipTypeBattleShip:
		return 1
	case objUnit.ShipTypeCruiser:
		return 2
	case objUnit.ShipTypeDestroyer:
		return 3
	case objUnit.ShipTypeFrigate:
		return 4
	case objUnit.ShipTypeTorpedoBoat:
		return 5
	case objUnit.ShipTypeSubmarine:
		return 6
	case objUnit.ShipTypeRepair:
		return 7
	case objUnit.ShipTypeHospital:
		return 8
	case objUnit.ShipTypeCargo:
		return 9
	default:
		return 10
	}
}

// abbrSituationPanel 是一侧态势栏的展示数据。
type abbrSituationPanel struct {
	title   string
	summary string
	rows    []fleetTypeRow
	footer  string
	accent  color.Color
}

// drawAbbrSituationPanels 绘制全屏地图左右两侧的舰队态势摘要。
func (d *Drawer) drawAbbrSituationPanels(screen *ebiten.Image, ms *state.MissionState) {
	panels := calcAbbrSidePanels(
		ms.View.Layout.Width, ms.View.Layout.Height, d.abbrMap.Bounds().Dx(),
	)
	if panels.Left.W <= 0 {
		return
	}

	ownFleet := ms.ContactFleet(ms.Player.CurPlayer)
	enemyFleet := ms.ContactFleet(ms.Player.CurEnemy)
	enemySummary := i18n.Format(i18n.MsgMapContacts, map[string]any{"Count": enemyFleet.Total})
	if enemyFleet.Total == 0 {
		enemySummary = i18n.Text(i18n.MsgMapNoContacts)
	}

	own := abbrSituationPanel{
		title:   i18n.Text(i18n.MsgMapOwnFleet),
		summary: abbrShipCount(ownFleet.Total),
		rows:    fleetTypeRows(ownFleet),
		footer:  abbrLossFooter(ms, ms.Player.CurPlayer),
		accent:  abbrAllyAccent,
	}
	enemy := abbrSituationPanel{
		title:   i18n.Text(i18n.MsgMapEnemyIntel),
		summary: enemySummary,
		rows:    fleetTypeRows(enemyFleet),
		footer:  abbrKillFooter(ms, ms.Player.CurEnemy),
		accent:  abbrEnemyAccent,
	}

	d.drawAbbrSituationPanel(screen, panels.Left, own, panels.Compact)
	d.drawAbbrSituationPanel(screen, panels.Right, enemy, panels.Compact)
}

// abbrLossFooter 返回我方损失摘要。
func abbrLossFooter(ms *state.MissionState, player faction.Player) string {
	lost, tonnage := 0, 0.0
	if stats := battleStatsOf(ms, player); stats != nil {
		lost, tonnage = stats.ShipsLost, stats.SunkTonnage
	}
	return i18n.Format(i18n.MsgMapLosses, map[string]any{
		"Ships":   abbrShipCount(lost),
		"Tonnage": abbrTonnageSuffix(tonnage),
	})
}

// abbrKillFooter 返回敌方损失（战果）摘要。
func abbrKillFooter(ms *state.MissionState, player faction.Player) string {
	lost, tonnage := 0, 0.0
	if stats := battleStatsOf(ms, player); stats != nil {
		lost, tonnage = stats.ShipsLost, stats.SunkTonnage
	}
	return i18n.Format(i18n.MsgMapKills, map[string]any{
		"Ships":   abbrShipCount(lost),
		"Tonnage": abbrTonnageSuffix(tonnage),
	})
}

func battleStatsOf(ms *state.MissionState, player faction.Player) *state.BattleSideStats {
	if ms.Stats.Sides == nil {
		return nil
	}
	return ms.Stats.Sides[player]
}

func abbrShipCount(count int) string {
	return i18n.Plural(i18n.MsgMapShipCount, count, map[string]any{"Count": count})
}

// abbrTonnageSuffix 返回「 · 1,234 t」形式的吨位后缀；没有沉没吨位时为空。
func abbrTonnageSuffix(tonnage float64) string {
	if tonnage <= 0 {
		return ""
	}
	return i18n.Format(i18n.MsgMapTonnageSuffix, map[string]any{
		"Tonnage": groupThousands(int64(math.Round(tonnage))),
	})
}

// drawAbbrSituationPanel 绘制单侧态势栏；窄屏只保留标题与摘要。
func (d *Drawer) drawAbbrSituationPanel(
	screen *ebiten.Image, r abbrPanelRect, p abbrSituationPanel, compact bool,
) {
	face := font.LocalizedUI(font.Hang)
	if compact {
		d.drawAbbrCompactPanel(screen, r, p, face)
		return
	}

	vector.DrawFilledRect(screen, float32(r.X), float32(r.Y+2), 4, 24, p.accent, false)
	d.drawText(screen, p.title, r.X+16, r.Y, abbrPanelTitleSize, face, colorx.Gold)
	d.drawText(screen, p.summary, r.X+16, r.Y+38, abbrPanelHeadSize, face, colorx.White)

	dividerY := r.Y + 72
	vector.StrokeLine(
		screen,
		float32(r.X), float32(dividerY),
		float32(r.X+r.W), float32(dividerY),
		1, abbrPanelDivider, false,
	)

	y := dividerY + 16
	for _, row := range p.rows {
		// 底部预留约两行，保证损失/战果摘要不会被舰种列表挤掉。
		if y+26 > r.Y+r.H-42 {
			break
		}
		d.drawAbbrFleetRow(screen, r, y, row, face)
		y += 29
	}

	y += 12
	for _, line := range layout.WrapText(p.footer, r.W, abbrPanelFooterSize, face) {
		if y+18 > r.Y+r.H {
			break
		}
		d.drawText(screen, line, r.X, y, abbrPanelFooterSize, face, colorx.Silver)
		y += 19
	}
}

// drawAbbrFleetRow 绘制一行「舰种 + 数量」，数量右对齐；长语种标签自动缩字号。
func (d *Drawer) drawAbbrFleetRow(
	screen *ebiten.Image, r abbrPanelRect, y float64, row fleetTypeRow, face *text.GoTextFaceSource,
) {
	label := row.shipType.ToDisplay()
	count := strconv.Itoa(row.count)
	countW, _ := text.Measure(count, &text.GoTextFace{Source: face, Size: abbrPanelBodySize}, 0)

	labelSize := abbrPanelBodySize
	maxLabelW := r.W - countW - 12
	for labelSize > 13 {
		labelW, _ := text.Measure(label, &text.GoTextFace{Source: face, Size: labelSize}, 0)
		if labelW <= maxLabelW {
			break
		}
		labelSize--
	}

	d.drawText(screen, label, r.X, y+(abbrPanelBodySize-labelSize), labelSize, face, colorx.White)
	d.drawText(screen, count, r.X+r.W-countW, y, abbrPanelBodySize, face, colorx.Silver)
}

// drawAbbrCompactPanel 在窄栏里只绘制标题与摘要，摘要过长时缩小字号。
func (d *Drawer) drawAbbrCompactPanel(
	screen *ebiten.Image, r abbrPanelRect, p abbrSituationPanel, face *text.GoTextFaceSource,
) {
	vector.StrokeLine(
		screen,
		float32(r.X), float32(r.Y),
		float32(r.X+r.W), float32(r.Y),
		2, p.accent, false,
	)

	titleSize := 20.0
	summarySize := 14.0
	titleW, _ := text.Measure(p.title, &text.GoTextFace{Source: face, Size: titleSize}, 0)
	summaryW, _ := text.Measure(p.summary, &text.GoTextFace{Source: face, Size: summarySize}, 0)
	for summarySize > 10 && summaryW > r.W {
		summarySize--
		summaryW, _ = text.Measure(p.summary, &text.GoTextFace{Source: face, Size: summarySize}, 0)
	}

	d.drawText(screen, p.title, r.X+(r.W-titleW)/2, r.Y+14, titleSize, face, colorx.Gold)
	d.drawText(screen, p.summary, r.X+(r.W-summaryW)/2, r.Y+42, summarySize, face, colorx.Silver)
}
