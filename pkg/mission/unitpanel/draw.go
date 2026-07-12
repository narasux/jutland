package unitpanel

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/object"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	shipImg "github.com/narasux/jutland/pkg/resources/images/ship"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
	textLayout "github.com/narasux/jutland/pkg/utils/layout"
)

var (
	panelBackground = color.RGBA{R: 8, G: 18, B: 23, A: 232}
	panelBorder     = color.RGBA{R: 85, G: 132, B: 143, A: 235}
	sectionFill     = color.RGBA{R: 12, G: 31, B: 38, A: 220}
	sectionBorder   = color.RGBA{R: 59, G: 97, B: 106, A: 220}
	buttonIdle      = color.RGBA{R: 21, G: 48, B: 56, A: 245}
	buttonHover     = color.RGBA{R: 31, G: 66, B: 75, A: 250}
	buttonPressed   = color.RGBA{R: 12, G: 34, B: 41, A: 250}
	handleFill      = color.RGBA{R: 18, G: 52, B: 62, A: 252}
	handleHover     = color.RGBA{R: 30, G: 76, B: 87, A: 255}
	handleBorder    = color.RGBA{R: 202, G: 181, B: 115, A: 255}
	handleShadow    = color.RGBA{R: 0, G: 0, B: 0, A: 120}
	disabledFill    = color.RGBA{R: 72, G: 35, B: 35, A: 235}
	mixedFill       = color.RGBA{R: 66, G: 59, B: 35, A: 235}
	progressTrack   = color.RGBA{R: 28, G: 52, B: 59, A: 245}
	progressFill    = color.RGBA{R: 196, G: 171, B: 101, A: 245}
)

// Draw 绘制底部把手，以及展开后的三栏单位面板。
func (p *Panel) Draw(screen *ebiten.Image, ms *state.MissionState, rightInset float64) {
	if ms.Core.MissionStatus != state.MissionRunning {
		return
	}
	p.rightInset = rightInset
	p.layout = calcLayout(ms.View.Layout, ms.UI.UnitPanelExpanded, rightInset)
	if ms.UI.UnitPanelExpanded {
		p.drawExpanded(screen, ms)
	}
	p.drawHandle(screen, ms)
}

func (p *Panel) drawExpanded(screen *ebiten.Image, ms *state.MissionState) {
	ui := p.layout
	vector.FillRect(screen, float32(ui.Panel.X), float32(ui.Panel.Y), float32(ui.Panel.W), float32(ui.Panel.H), panelBackground, false)
	vector.StrokeRect(screen, float32(ui.Panel.X), float32(ui.Panel.Y), float32(ui.Panel.W), float32(ui.Panel.H), 2, panelBorder, false)
	vector.StrokeLine(screen, float32(ui.Header.X), float32(ui.Header.Y+ui.Header.H), float32(ui.Header.X+ui.Header.W), float32(ui.Header.Y+ui.Header.H), 1, panelBorder, false)

	ships := selectedShips(ms)
	header := i18n.Text(i18n.MsgUnitPanelNoSelection)
	if len(ships) == 1 {
		header = objUnit.GetShipDisplayName(ships[0].Name)
	} else if len(ships) > 1 {
		header = i18n.Format(i18n.MsgUnitPanelSelectedCount, map[string]any{"Count": len(ships)})
	}
	p.drawText(screen, header, ui.Header.X+18, ui.Header.Y+8, 20, font.LocalizedUI(font.Hang), colorx.White)

	for _, area := range []rect{ui.Visual, ui.Info, ui.Systems} {
		vector.FillRect(screen, float32(area.X), float32(area.Y), float32(area.W), float32(area.H), sectionFill, false)
		vector.StrokeRect(screen, float32(area.X), float32(area.Y), float32(area.W), float32(area.H), 1, sectionBorder, false)
	}
	if len(ships) == 0 {
		p.drawCenteredText(screen, i18n.Text(i18n.MsgUnitPanelNoSelection), ui.Panel, 20, colorx.Silver)
		return
	}

	if len(ships) == 1 {
		p.drawShipVisual(screen, ships[0])
	} else {
		p.drawFocusList(screen, ms, ships)
	}
	p.drawBasicInfo(screen, ms, ships)
	p.drawSystems(screen, ms, ships)
}

// drawHandle 绘制始终可见的底部展开/收起把手。
func (p *Panel) drawHandle(screen *ebiten.Image, ms *state.MissionState) {
	ui := p.layout.Handle
	screenX, screenY := ebiten.CursorPosition()
	fill := handleFill
	if ui.contains(screenX, screenY) {
		fill = handleHover
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			fill = buttonPressed
		}
	}
	shadow := ui
	shadow.Y += 3
	p.drawRoundedRect(screen, shadow, 8, handleShadow, handleShadow)
	p.drawRoundedRectWithStroke(screen, ui, 8, fill, handleBorder, 2)

	label := i18n.Text(i18n.MsgUnitPanelOpen)
	if ms.UI.UnitPanelExpanded {
		label = i18n.Text(i18n.MsgUnitPanelClose)
	}
	fontSize := 17.0
	textSource := font.LocalizedUI(font.Kai)
	label = p.fitText(label, ui.W-52, fontSize, textSource)
	const arrowW = 12.0
	labelW, labelH := text.Measure(label, &text.GoTextFace{Source: textSource, Size: fontSize}, 0)
	contentW := arrowW + 10 + labelW
	startX := ui.X + (ui.W-contentW)/2
	textY := ui.Y + (ui.H-labelH)/2
	p.drawHandleTriangle(screen, startX, ui.Y+ui.H/2, arrowW, ms.UI.UnitPanelExpanded)
	p.drawText(screen, label, startX+arrowW+10, textY, fontSize, textSource, colorx.White)
}

// drawHandleTriangle 使用矢量三角形保证展开、收起箭头不受字体缺字影响。
func (p *Panel) drawHandleTriangle(screen *ebiten.Image, x, centerY, size float64, expanded bool) {
	var path vector.Path
	if expanded {
		path.MoveTo(float32(x), float32(centerY-size/3))
		path.LineTo(float32(x+size), float32(centerY-size/3))
		path.LineTo(float32(x+size/2), float32(centerY+size/2))
	} else {
		path.MoveTo(float32(x+size/2), float32(centerY-size/2))
		path.LineTo(float32(x+size), float32(centerY+size/3))
		path.LineTo(float32(x), float32(centerY+size/3))
	}
	path.Close()
	opts := &vector.DrawPathOptions{AntiAlias: true}
	opts.ColorScale.ScaleWithColor(colorx.Gold)
	vector.FillPath(screen, &path, &vector.FillOptions{}, opts)
}

// drawShipVisual 复用增援点界面的朝向与缩放规则绘制侧视图和俯视图。
func (p *Panel) drawShipVisual(screen *ebiten.Image, ship *objUnit.BattleShip) {
	area := p.layout.Visual
	zoom := 2
	if area.W > 300 {
		zoom = 4
	}
	side := shipImg.GetSide(ship.Name, zoom)
	top := shipImg.GetTop(ship.Name, zoom)
	if side == nil || top == nil {
		return
	}

	contentW := area.W * 0.82
	contentH := area.H * 0.82
	sideScale := min(1, min(contentW/float64(side.Bounds().Dx()), contentH*0.34/float64(side.Bounds().Dy())))
	// 俯视图顺时针旋转 90 度后，显示宽高与源图互换。
	topW, topH := float64(top.Bounds().Dy()), float64(top.Bounds().Dx())
	topScale := min(1, min(contentW/topW, contentH*0.48/topH))
	centerX := area.X + area.W/2

	sideDrawW, sideDrawH := float64(side.Bounds().Dx())*sideScale, float64(side.Bounds().Dy())*sideScale
	sideOpts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	sideOpts.GeoM.Scale(sideScale, sideScale)
	sideOpts.GeoM.Translate(centerX-sideDrawW/2, area.Y+area.H*0.31-sideDrawH/2)
	screen.DrawImage(side, sideOpts)

	topDrawW, topDrawH := topW*topScale, topH*topScale
	topOpts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	ebutil.SetOptsCenterRotation(topOpts, top, 90)
	topOpts.GeoM.Scale(topScale, topScale)
	topOpts.GeoM.Translate(centerX-topDrawW/2, area.Y+area.H*0.70-topDrawH/2)
	topOpts.GeoM.Translate((topDrawW-topDrawH)/2, (topDrawH-topDrawW)/2)
	screen.DrawImage(top, topOpts)
}

func (p *Panel) drawFocusList(screen *ebiten.Image, ms *state.MissionState, ships []*objUnit.BattleShip) {
	p.drawText(screen, i18n.Format(i18n.MsgUnitPanelSelectedCount, map[string]any{"Count": len(ships)}), p.layout.Visual.X+10, p.layout.Visual.Y+7, 15, font.LocalizedUI(font.Kai), colorx.Silver)
	p.drawActionButton(screen, p.previousFocusRect(), "‹", toggleAllowed)
	p.drawActionButton(screen, p.nextFocusRect(), "›", toggleAllowed)
	for _, entry := range p.visibleFocusShips(ships, ms.Interaction.FocusedShipUid) {
		selected := entry.Ship.Uid == ms.Interaction.FocusedShipUid
		fill := color.RGBA{A: 0}
		border := sectionBorder
		if selected {
			fill = color.RGBA{R: 58, G: 69, B: 55, A: 225}
			border = colorx.Gold
		}
		if entry.Rect.containsCursor() {
			fill = buttonHover
		}
		p.drawRoundedRect(screen, entry.Rect, 4, fill, border)
		name := p.fitText(objUnit.GetShipDisplayName(entry.Ship.Name), entry.Rect.W-68, 14, font.LocalizedUI(font.Kai))
		p.drawText(screen, name, entry.Rect.X+8, entry.Rect.Y+5, 14, font.LocalizedUI(font.Kai), colorx.White)
		hp := fmt.Sprintf("%.0f%%", entry.Ship.CurHP/entry.Ship.TotalHP*100)
		p.drawText(screen, hp, entry.Rect.X+entry.Rect.W-52, entry.Rect.Y+5, 13, font.JetbrainsMono, colorx.Silver)
	}
}

func (r rect) containsCursor() bool {
	x, y := ebiten.CursorPosition()
	return r.contains(x, y)
}

// drawBasicInfo 绘制单舰基础信息或多选舰队汇总。
func (p *Panel) drawBasicInfo(screen *ebiten.Image, ms *state.MissionState, ships []*objUnit.BattleShip) {
	area := p.layout.Info
	p.drawText(screen, i18n.Text(i18n.MsgUnitPanelBasicInfo), area.X+10, area.Y+5, 16, font.LocalizedUI(font.Hang), colorx.White)
	if len(ships) > 1 {
		p.drawFleetInfo(screen, ships)
		return
	}
	ship := ships[0]
	target := ms.Arena.Ships[ship.AttackTarget]
	targetName, distance := i18n.Text(i18n.MsgUnitPanelNoTarget), i18n.Text(i18n.MsgUnitPanelNoTarget)
	if target != nil {
		targetName = objUnit.GetShipDisplayName(target.Name)
		distance = fmt.Sprintf("%.1f", ship.CurPos.Distance(target.CurPos))
	}
	group := i18n.Text(i18n.MsgUnitPanelNoTarget)
	if ship.GroupID != object.GroupIDNone {
		group = fmt.Sprintf("%d", ship.GroupID)
	}
	status := i18n.Text(i18n.MsgUnitPanelIdle)
	if ship.AttackTarget != "" {
		status = i18n.Text(i18n.MsgUnitPanelEngaging)
	} else if ship.CurSpeed > 0 {
		status = i18n.Text(i18n.MsgUnitPanelMoving)
	}
	items := []struct{ label, value string }{
		{i18n.Text(i18n.MsgUnitPanelName), objUnit.GetShipDisplayName(ship.Name)},
		{i18n.Text(i18n.MsgUnitPanelType), ship.Type.ToDisplay()},
		{"HP", fmt.Sprintf("%.0f / %.0f (%.0f%%)", ship.CurHP, ship.TotalHP, ship.CurHP/ship.TotalHP*100)},
		{i18n.Text(i18n.MsgUnitPanelSpeed), fmt.Sprintf("%.1f / %.1f kn", ship.CurSpeed*600, ship.MaxSpeed*600)},
		{i18n.Text(i18n.MsgUnitPanelHeading), fmt.Sprintf("%03.0f°", math.Mod(ship.CurRotation+360, 360))},
		{i18n.Text(i18n.MsgUnitPanelTarget), targetName},
		{i18n.Text(i18n.MsgUnitPanelDistance), distance},
		{i18n.Text(i18n.MsgUnitPanelStatus), status},
		{i18n.Text(i18n.MsgUnitPanelGroup), group},
	}
	if ship.Aircraft.HasPlane {
		aircraft := ship.Aircraft.Status(ship.Uid, ms.Arena.Planes).Total
		items = append(items, struct{ label, value string }{
			i18n.Text(i18n.MsgUnitPanelAircraft),
			fmt.Sprintf("%d / %d", aircraft.Alive(), aircraft.Initial()),
		})
	}
	p.drawInfoItems(screen, items, target != nil)
}

func (p *Panel) drawFleetInfo(screen *ebiten.Image, ships []*objUnit.BattleShip) {
	var hpPercent, speed float64
	for _, ship := range ships {
		hpPercent += ship.CurHP / ship.TotalHP * 100
		speed += ship.CurSpeed * 600
	}
	focusName := i18n.Text(i18n.MsgUnitPanelNoTarget)
	if ship := focusedShipFromSlice(ships, p.lastFocus); ship != nil {
		focusName = objUnit.GetShipDisplayName(ship.Name)
	}
	items := []struct{ label, value string }{
		{i18n.Text(i18n.MsgUnitPanelName), focusName},
		{i18n.Text(i18n.MsgUnitPanelTotal), fmt.Sprintf("%d", len(ships))},
		{i18n.Text(i18n.MsgUnitPanelAverageHP), fmt.Sprintf("%.0f%%", hpPercent/float64(len(ships)))},
		{i18n.Text(i18n.MsgUnitPanelAverageSpeed), fmt.Sprintf("%.1f kn", speed/float64(len(ships)))},
	}
	p.drawInfoItems(screen, items, false)
}

func focusedShipFromSlice(ships []*objUnit.BattleShip, uid string) *objUnit.BattleShip {
	for _, ship := range ships {
		if ship.Uid == uid {
			return ship
		}
	}
	return nil
}

func (p *Panel) drawInfoItems(screen *ebiten.Image, items []struct{ label, value string }, targetButton bool) {
	rowH := p.infoRowHeight()
	fontSize := min(15, max(12, rowH-2))
	labelW := min(72, p.layout.Info.W*0.27)
	for index, item := range items {
		y := p.layout.Info.Y + 26 + float64(index)*rowH
		p.drawText(screen, item.label, p.layout.Info.X+10, y, fontSize, font.LocalizedUI(font.Kai), colorx.Gold)
		maxW := p.layout.Info.W - labelW - 18
		if targetButton && index == 5 {
			maxW -= 30
		}
		value := p.fitText(item.value, maxW, fontSize, font.LocalizedUI(font.Kai))
		p.drawText(screen, value, p.layout.Info.X+labelW, y, fontSize, font.LocalizedUI(font.Kai), colorx.White)
	}
	if targetButton {
		p.drawActionButton(screen, p.targetButtonRect(), "→", toggleAllowed)
	}
}

func (p *Panel) drawSystems(screen *ebiten.Image, ms *state.MissionState, ships []*objUnit.BattleShip) {
	weaponTab, aircraftTab := p.tabRects()
	p.drawTab(screen, weaponTab, i18n.Text(i18n.MsgUnitPanelWeaponTab), p.tab == TabWeapons)
	p.drawTab(screen, aircraftTab, i18n.Text(i18n.MsgUnitPanelAircraftTab), p.tab == TabAircraft)
	if p.tab == TabAircraft {
		p.drawAircraft(screen, ms, ships)
		return
	}
	p.drawWeapons(screen, ms, ships)
}

func (p *Panel) drawTab(screen *ebiten.Image, area rect, label string, active bool) {
	fill, border, textColor := buttonIdle, sectionBorder, colorx.Silver
	if active {
		fill, border, textColor = color.RGBA{R: 58, G: 69, B: 55, A: 235}, colorx.Gold, colorx.White
	} else if area.containsCursor() {
		fill = buttonHover
	}
	p.drawRoundedRect(screen, area, 5, fill, border)
	p.drawCenteredText(screen, label, area, 15, textColor)
}

func (p *Panel) drawWeapons(screen *ebiten.Image, ms *state.MissionState, ships []*objUnit.BattleShip) {
	rows := weaponRows(ms, nowMillis())
	if len(rows) == 0 {
		p.drawCenteredText(screen, i18n.Text(i18n.MsgUnitPanelNoTarget), p.layout.Systems, 18, colorx.Silver)
		return
	}
	p.drawText(screen, i18n.Text(i18n.MsgUnitPanelAllWeapons), p.layout.Systems.X+4, p.layout.Systems.Y+45, 15, font.LocalizedUI(font.Kai), colorx.White)
	p.drawActionButton(screen, p.allWeaponsToggleRect(), p.toggleLabel(allWeaponsToggle(ships)), allWeaponsToggle(ships))
	for index, row := range rows {
		p.drawWeaponRow(screen, row, index, len(rows))
	}
}

// drawWeaponRow 将就绪数、进度、剩余时间和开关放在同一行。
func (p *Panel) drawWeaponRow(screen *ebiten.Image, row weaponRow, index, rowCount int) {
	step := p.weaponRowStep(rowCount)
	y := p.layout.Systems.Y + 68 + float64(index)*step
	fontSize := min(14, max(11, step-8))
	nameW := min(72, p.layout.Systems.W*0.17)
	readyW := min(90, p.layout.Systems.W*0.22)
	timeW := min(62, p.layout.Systems.W*0.15)
	buttonW := 82.0
	progressX := p.layout.Systems.X + nameW + readyW
	progressW := max(34, p.layout.Systems.W-nameW-readyW-timeW-buttonW-12)

	p.drawText(screen, p.weaponLabel(row.Type), p.layout.Systems.X+4, y+4, fontSize, font.LocalizedUI(font.Kai), colorx.White)
	ready := i18n.Format(i18n.MsgUnitPanelReadyCount, map[string]any{"Ready": row.Ready, "Total": row.Equipped})
	p.drawText(screen, ready, p.layout.Systems.X+nameW, y+4, fontSize, font.LocalizedUI(font.Kai), colorx.Silver)
	p.drawProgress(screen, rect{X: progressX, Y: y + 5, W: progressW, H: max(8, min(12, step-10))}, row.Progress)
	timeLabel := i18n.Text(i18n.MsgUnitPanelReady)
	if row.RemainingMillis > 0 {
		timeLabel = i18n.Format(i18n.MsgUnitPanelSeconds, map[string]any{"Seconds": fmt.Sprintf("%.1f", float64(row.RemainingMillis)/1e3)})
	}
	timeLabel = p.fitText(timeLabel, timeW-4, fontSize, font.LocalizedUI(font.Kai))
	p.drawText(screen, timeLabel, progressX+progressW+6, y+4, fontSize, font.LocalizedUI(font.Kai), colorx.Silver)
	p.drawActionButton(screen, p.weaponToggleRect(index, rowCount), p.toggleLabel(row.Toggle), row.Toggle)
}

// drawAircraft 绘制单一表头、各机型数据、合计行和起飞总开关。
func (p *Panel) drawAircraft(screen *ebiten.Image, ms *state.MissionState, ships []*objUnit.BattleShip) {
	rows, total := aircraftRows(ms)
	if len(rows) == 0 {
		p.drawCenteredText(screen, i18n.Text(i18n.MsgUnitPanelAircraftEmpty), p.layout.Systems, 17, colorx.Silver)
		return
	}
	p.drawText(screen, i18n.Text(i18n.MsgUnitPanelTakeoff), p.layout.Systems.X+4, p.layout.Systems.Y+45, 15, font.LocalizedUI(font.Kai), colorx.White)
	toggle := aircraftToggle(ships)
	p.drawActionButton(screen, p.aircraftToggleRect(), p.toggleLabel(toggle), toggle)

	tableY := p.layout.Systems.Y + 76
	rowH := min(26, max(20, (p.layout.Systems.H-80)/float64(len(rows)+2)))
	columns := []float64{0, 0.42, 0.565, 0.71, 0.855}
	headings := []string{
		i18n.Text(i18n.MsgUnitPanelPlaneType),
		i18n.Text(i18n.MsgUnitPanelStandby),
		i18n.Text(i18n.MsgUnitPanelInCombat),
		i18n.Text(i18n.MsgUnitPanelReturning),
		i18n.Text(i18n.MsgUnitPanelLost),
	}
	for index, heading := range headings {
		p.drawText(screen, heading, p.layout.Systems.X+columns[index]*p.layout.Systems.W+4, tableY, 13, font.LocalizedUI(font.Kai), colorx.Gold)
	}
	vector.StrokeLine(screen, float32(p.layout.Systems.X+4), float32(tableY+rowH-3), float32(p.layout.Systems.X+p.layout.Systems.W-4), float32(tableY+rowH-3), 1, sectionBorder, false)
	for index, row := range rows {
		p.drawAircraftRow(screen, row, tableY+rowH*float64(index+1), columns, rowH, false)
	}
	p.drawAircraftRow(screen, total, tableY+rowH*float64(len(rows)+1), columns, rowH, true)
}

func (p *Panel) drawAircraftRow(screen *ebiten.Image, row objUnit.AircraftGroupStatus, y float64, columns []float64, rowH float64, total bool) {
	name := objUnit.GetPlaneDisplayName(row.Name)
	textColor := colorx.White
	if total {
		name = i18n.Text(i18n.MsgUnitPanelTotal)
		textColor = colorx.Gold
		vector.StrokeLine(screen, float32(p.layout.Systems.X+4), float32(y-3), float32(p.layout.Systems.X+p.layout.Systems.W-4), float32(y-3), 1, sectionBorder, false)
	}
	values := []string{name, fmt.Sprintf("%d", row.Standby), fmt.Sprintf("%d", row.InCombat), fmt.Sprintf("%d", row.Returning), fmt.Sprintf("%d", row.Lost)}
	fontSize := min(14, max(11, rowH-8))
	for index, value := range values {
		maxW := p.layout.Systems.W * 0.14
		if index == 0 {
			maxW = p.layout.Systems.W * 0.40
		}
		value = p.fitText(value, maxW, fontSize, font.LocalizedUI(font.Kai))
		p.drawText(screen, value, p.layout.Systems.X+columns[index]*p.layout.Systems.W+4, y+2, fontSize, font.LocalizedUI(font.Kai), textColor)
	}
}

func (p *Panel) weaponLabel(weaponType objUnit.WeaponType) string {
	switch weaponType {
	case objUnit.WeaponTypeMainGun:
		return i18n.Text(i18n.MsgUnitPanelMainGun)
	case objUnit.WeaponTypeSecondaryGun:
		return i18n.Text(i18n.MsgUnitPanelSecondaryGun)
	case objUnit.WeaponTypeAntiAircraftGun:
		return i18n.Text(i18n.MsgUnitPanelAntiAircraftGun)
	case objUnit.WeaponTypeTorpedo:
		return i18n.Text(i18n.MsgUnitPanelTorpedo)
	case objUnit.WeaponTypeRocket:
		return i18n.Text(i18n.MsgUnitPanelRocket)
	default:
		return "-"
	}
}

func (p *Panel) toggleLabel(value toggleState) string {
	switch value {
	case toggleDisabled:
		return "○ " + i18n.Text(i18n.MsgUnitPanelDisabled)
	case toggleMixed:
		return "— " + i18n.Text(i18n.MsgUnitPanelMixed)
	default:
		return "● " + i18n.Text(i18n.MsgUnitPanelAllowed)
	}
}

// drawActionButton 绘制允许、禁用和混合三态圆角按钮。
func (p *Panel) drawActionButton(screen *ebiten.Image, area rect, label string, state toggleState) {
	fill, border := buttonIdle, panelBorder
	switch state {
	case toggleDisabled:
		fill = disabledFill
	case toggleMixed:
		fill = mixedFill
	}
	if area.containsCursor() {
		fill = buttonHover
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			fill = buttonPressed
		}
	}
	p.drawRoundedRect(screen, area, 5, fill, border)
	fontSize := min(14, max(11, area.H-11))
	label = p.fitText(label, area.W-8, fontSize, font.LocalizedUI(font.Kai))
	p.drawCenteredText(screen, label, area, fontSize, colorx.White)
}

// drawProgress 绘制下一次可发射的装填进度条。
func (p *Panel) drawProgress(screen *ebiten.Image, area rect, progress float64) {
	p.drawRoundedRect(screen, area, area.H/2, progressTrack, sectionBorder)
	progress = min(1, max(0, progress))
	if progress <= 0 {
		return
	}
	fillArea := area
	fillArea.W *= progress
	if fillArea.W < fillArea.H {
		fillArea.W = fillArea.H
	}
	p.drawRoundedRect(screen, fillArea, area.H/2, progressFill, progressFill)
}

// drawRoundedRect 使用矢量路径绘制主题化圆角矩形。
func (p *Panel) drawRoundedRect(screen *ebiten.Image, area rect, radius float64, fill, stroke color.Color) {
	p.drawRoundedRectWithStroke(screen, area, radius, fill, stroke, 1)
}

// drawRoundedRectWithStroke 使用指定描边宽度绘制主题化圆角矩形。
func (p *Panel) drawRoundedRectWithStroke(
	screen *ebiten.Image,
	area rect,
	radius float64,
	fill, stroke color.Color,
	strokeWidth float32,
) {
	radius = min(radius, min(area.W, area.H)/2)
	x, y, w, h, r := float32(area.X), float32(area.Y), float32(area.W), float32(area.H), float32(radius)
	var path vector.Path
	path.MoveTo(x+r, y)
	path.LineTo(x+w-r, y)
	path.QuadTo(x+w, y, x+w, y+r)
	path.LineTo(x+w, y+h-r)
	path.QuadTo(x+w, y+h, x+w-r, y+h)
	path.LineTo(x+r, y+h)
	path.QuadTo(x, y+h, x, y+h-r)
	path.LineTo(x, y+r)
	path.QuadTo(x, y, x+r, y)
	path.Close()
	fillOpts := &vector.DrawPathOptions{AntiAlias: true}
	fillOpts.ColorScale.ScaleWithColor(fill)
	vector.FillPath(screen, &path, &vector.FillOptions{}, fillOpts)
	strokeOpts := &vector.DrawPathOptions{AntiAlias: true}
	strokeOpts.ColorScale.ScaleWithColor(stroke)
	vector.StrokePath(screen, &path, &vector.StrokeOptions{Width: strokeWidth}, strokeOpts)
}

func (p *Panel) drawText(screen *ebiten.Image, value string, x, y, size float64, source *text.GoTextFaceSource, textColor color.Color) {
	opts := &text.DrawOptions{}
	opts.GeoM.Translate(x, y)
	opts.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, value, &text.GoTextFace{Source: source, Size: size}, opts)
}

func (p *Panel) drawCenteredText(screen *ebiten.Image, value string, area rect, size float64, textColor color.Color) {
	source := font.LocalizedUI(font.Kai)
	width, height := text.Measure(value, &text.GoTextFace{Source: source, Size: size}, 0)
	p.drawText(screen, value, area.X+(area.W-width)/2, area.Y+(area.H-height)/2, size, source, textColor)
}

func (p *Panel) fitText(value string, maxWidth, size float64, source *text.GoTextFaceSource) string {
	if textLayout.CalcTextWidth(value, size, source) <= maxWidth {
		return value
	}
	runes := []rune(strings.TrimSpace(value))
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		candidate := string(runes) + "…"
		if textLayout.CalcTextWidth(candidate, size, source) <= maxWidth {
			return candidate
		}
	}
	return "…"
}
