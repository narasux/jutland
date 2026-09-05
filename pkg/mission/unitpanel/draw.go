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
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	shipImg "github.com/narasux/jutland/pkg/resources/images/ship"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
	textLayout "github.com/narasux/jutland/pkg/utils/layout"
	"github.com/narasux/jutland/pkg/utils/theme"
)

var (
	sectionFill   = theme.CardBackground
	sectionBorder = theme.CardBorder
	buttonIdle    = theme.ButtonIdle
	buttonHover   = theme.ButtonHover
	progressTrack = theme.ProgressTrack
	progressFill  = theme.ProgressFill
	// toggle 状态按钮采用低对比淡色样式，弱化其存在感。
	toggleFill      = color.RGBA{R: 12, G: 31, B: 38, A: 70}
	toggleBorder    = color.RGBA{R: 90, G: 138, B: 148, A: 90}
	toggleHoverFill = color.RGBA{R: 31, G: 66, B: 75, A: 130}
	togglePressFill = color.RGBA{R: 12, G: 34, B: 41, A: 150}
)

// navChevronSize 是导航按钮 V 形箭头的开口半宽。
const navChevronSize = 5.0

// Draw 将单位信息内容绘制到滚动视口 region 内，并按 scrollY 上移。
// 内容先绘制到离屏缓冲再整体贴回屏幕，从而把滚动溢出的部分裁剪在视口内。
func (p *Panel) Draw(screen *ebiten.Image, ms *state.MissionState, region Rect, scrollY float64) {
	if ms.Core.MissionStatus != state.MissionRunning {
		return
	}
	if region.W <= 0 || region.H <= 0 {
		return
	}
	p.layout = p.calcLayout(ms, region, scrollY)
	p.viewport = region
	p.ensureBuffer(region)
	p.drawSections(p.renderBuf, ms)
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	opts.GeoM.Translate(region.X, region.Y)
	screen.DrawImage(p.renderBuf, opts)
}

func (p *Panel) ensureBuffer(region Rect) {
	w, h := int(math.Ceil(region.W)), int(math.Ceil(region.H))
	if p.renderBuf == nil || p.renderBuf.Bounds().Dx() != w || p.renderBuf.Bounds().Dy() != h {
		p.renderBuf = ebiten.NewImage(w, h)
	}
	p.renderBuf.Clear()
}

// drawSections 按纵向分区绘制单位信息内容（到 offscreen 缓冲）。
func (p *Panel) drawSections(screen *ebiten.Image, ms *state.MissionState) {
	// 选中机场时展示机场信息面板，与战舰面板互斥
	if af := selectedAirfield(ms); af != nil {
		p.drawAirfieldSections(screen, ms, af)
		return
	}
	ships := selectedShips(ms)
	title := i18n.Text(i18n.MsgUnitPanelNoSelection)
	if len(ships) == 1 {
		title = objUnit.GetShipDisplayName(ships[0].Name)
	} else if focus := focusedShip(ms); focus != nil {
		title = objUnit.GetShipDisplayName(focus.Name)
	}
	p.drawHeader(screen, ms, title)
	if len(ships) == 0 {
		return
	}

	ui := p.layout
	for _, area := range []Rect{ui.Visual, ui.Info, ui.Systems} {
		if area.W == 0 || area.H == 0 {
			continue
		}
		theme.FillRoundedRect(
			screen,
			area.X,
			area.Y,
			area.W,
			area.H,
			theme.CornerRadius,
			sectionFill,
			sectionBorder,
			theme.CardBorderWidth,
		)
	}

	if len(ships) == 1 {
		p.drawShipVisual(screen, ships[0])
	} else {
		p.drawFocusList(screen, ms, ships)
	}
	p.drawBasicInfo(screen, ms, ships)
	p.drawSystems(screen, ms, ships)
}

// drawHeader 在舰名同一行右侧绘制当前资金与己方舰数，不展示敌方数量。
func (p *Panel) drawHeader(screen *ebiten.Image, ms *state.MissionState, title string) {
	area := p.layout.Header
	funds := i18n.Format(i18n.MsgSidebarFunds, map[string]any{"Funds": ms.Player.CurFunds})
	fleet := i18n.Format(i18n.MsgSidebarAllyFleet, map[string]any{
		"Count": ms.Fleet(ms.Player.CurPlayer).Total,
	})
	stats := funds + "  " + fleet
	statsSize := float64(theme.SizeCaption)
	statsFace := theme.Body()
	statsW := textLayout.CalcTextWidth(stats, statsSize, statsFace)
	statsX := area.X + area.W - statsW
	p.drawText(screen, stats, statsX, area.Y+8, statsSize, statsFace, colorx.Silver)

	nameSize := float64(theme.SizeBody)
	maxNameW := statsX - area.X - 12
	if maxNameW < 36 {
		maxNameW = 36
	}
	p.drawText(
		screen,
		p.fitText(title, maxNameW, nameSize, theme.Title()),
		area.X+4,
		area.Y+4,
		nameSize,
		theme.Title(),
		colorx.White,
	)
}

// drawShipVisual 复用增援点界面的朝向与缩放规则绘制侧视图和俯视图。
// 侧视图与俯视图共用同一像素比例，保证舰船在两视图中的比例与大小一致。
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

	sideW, sideH := float64(side.Bounds().Dx()), float64(side.Bounds().Dy())
	// 俯视图顺时针旋转 90 度后，显示宽高与源图互换。
	topW, topH := float64(top.Bounds().Dy()), float64(top.Bounds().Dx())

	contentW := area.W * 0.78
	contentH := area.H * 0.72
	// 与增援点预览一致：侧视图与俯视图取同一个缩放，避免两视图比例不一致。
	scale := min(1, contentW/sideW, contentH*0.35/sideH, contentW/topW, contentH*0.48/topH)
	sideDrawW, sideDrawH := sideW*scale, sideH*scale
	topDrawW, topDrawH := topW*scale, topH*scale
	centerX := area.X + area.W/2
	sideY := area.Y + area.H*0.34
	topY := area.Y + area.H*0.68

	sideOpts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	sideOpts.GeoM.Scale(scale, scale)
	sideOpts.GeoM.Translate(centerX-sideDrawW/2, sideY-sideDrawH/2)
	screen.DrawImage(side, sideOpts)

	topOpts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	ebutil.SetOptsCenterRotation(topOpts, top, 90)
	topOpts.GeoM.Scale(scale, scale)
	topOpts.GeoM.Translate(centerX-topDrawW/2, topY-topDrawH/2)
	topOpts.GeoM.Translate((topDrawW-topDrawH)/2, (topDrawH-topDrawW)/2)
	screen.DrawImage(top, topOpts)
}

func (p *Panel) drawFocusList(screen *ebiten.Image, ms *state.MissionState, ships []*objUnit.BattleShip) {
	p.drawText(
		screen,
		i18n.Format(i18n.MsgUnitPanelSelectedCount, map[string]any{"Count": len(ships)}),
		p.layout.Visual.X+10,
		p.layout.Visual.Y+7,
		theme.SizeBody,
		theme.Body(),
		colorx.Silver,
	)
	p.drawNavButton(screen, p.previousFocusRect(), theme.TriangleLeft)
	p.drawNavButton(screen, p.nextFocusRect(), theme.TriangleRight)
	for _, entry := range p.visibleFocusShips(ships, ms.Interaction.FocusedShipUid) {
		selected := entry.Ship.Uid == ms.Interaction.FocusedShipUid
		fill := color.RGBA{A: 0}
		border := sectionBorder
		if selected {
			fill = color.RGBA{R: 58, G: 69, B: 55, A: 225}
			border = colorx.Gold
		}
		if p.cursorAt(entry.Rect) {
			fill = buttonHover
		}
		p.drawRoundedRect(screen, entry.Rect, theme.CornerRadius, fill, border)
		name := p.fitText(objUnit.GetShipDisplayName(entry.Ship.Name), entry.Rect.W-68, theme.SizeBody-1, theme.Body())
		p.drawText(screen, name, entry.Rect.X+8, entry.Rect.Y+5, theme.SizeBody-1, theme.Body(), colorx.White)
		hp := fmt.Sprintf("%.0f%%", entry.Ship.CurHP/entry.Ship.TotalHP*100)
		p.drawText(
			screen,
			hp,
			entry.Rect.X+entry.Rect.W-52,
			entry.Rect.Y+5,
			theme.SizeCaption,
			theme.Numeric(),
			colorx.Silver,
		)
	}
}

// infoItem 是信息列中的单行：label 为字段名，value 为字段值。
// numeric 表示值应靠右对齐并使用等宽字体。
type infoItem struct {
	label   string
	value   string
	numeric bool
}

// infoGroup 是信息列中的分组节：包含节标题与其所属的字段行。
type infoGroup struct {
	title string
	items []infoItem
}

// drawBasicInfo 绘制单舰基础信息或多选舰队汇总。
func (p *Panel) drawBasicInfo(screen *ebiten.Image, ms *state.MissionState, ships []*objUnit.BattleShip) {
	area := p.layout.Info
	if area.W == 0 || area.H == 0 {
		return
	}
	p.drawText(
		screen,
		i18n.Text(i18n.MsgUnitPanelBasicInfo),
		area.X+10,
		area.Y+5,
		theme.SizeSection,
		theme.Title(),
		colorx.White,
	)

	groups := p.buildInfoGroups(ms, ships)
	if len(groups) == 0 {
		return
	}
	info := p.infoGroupLayout(area, groups)
	labelW := math.Min(70.0, area.W*0.34)

	y := info.contentTop
	for _, group := range groups {
		if group.title != "" {
			p.drawText(screen, group.title, area.X+10, y, theme.SizeCaption, theme.Title(), colorx.Gold)
			y += info.lineH
		}
		for _, item := range group.items {
			p.drawInfoRow(screen, y, info.lineH, labelW, item)
			y += info.lineH
		}
	}

	if len(ships) == 1 && p.hasTarget {
		p.drawNavButton(screen, p.targetButtonRect(), theme.TriangleRight)
	}
}

// infoGroupLayout 计算分组节的统一行高与内容起始 Y，供绘制与命中区共用同一几何。
// 纵向布局不再压缩行高，使用固定 infoLineH，溢出交由滚动条处理。
func (p *Panel) infoGroupLayout(area Rect, groups []infoGroup) infoLayout {
	return infoLayout{contentTop: area.Y + infoTitleH, lineH: infoLineH}
}

type infoLayout struct {
	contentTop float64
	lineH      float64
}

// targetButtonRect 返回单舰有攻击目标时"→"居中按钮的位置，与信息分组行对齐。
func (p *Panel) targetButtonRect() Rect {
	return p.targetRect
}

// computeTargetRect 定位"→"居中按钮，使其落在「目标」字段行的右侧。
func (p *Panel) computeTargetRect(ms *state.MissionState, ship *objUnit.BattleShip) Rect {
	groups := p.shipGroups(ms, ship)
	info := p.infoGroupLayout(p.layout.Info, groups)
	y := info.contentTop
	targetLabel := i18n.Text(i18n.MsgUnitPanelTarget)
	for _, group := range groups {
		if group.title != "" {
			y += info.lineH
		}
		for _, item := range group.items {
			if item.label == targetLabel {
				return Rect{
					X: p.layout.Info.X + p.layout.Info.W - 40,
					Y: y + (info.lineH-20)/2,
					W: 26,
					H: 20,
				}
			}
			y += info.lineH
		}
	}
	return Rect{}
}

// buildInfoGroups 根据单选/多选组装分组节。
func (p *Panel) buildInfoGroups(ms *state.MissionState, ships []*objUnit.BattleShip) []infoGroup {
	if len(ships) > 1 {
		return []infoGroup{p.fleetGroup(ships)}
	}
	return p.shipGroups(ms, ships[0])
}

// shipGroups 把单舰信息压成单页紧凑列表。
func (p *Panel) shipGroups(ms *state.MissionState, ship *objUnit.BattleShip) []infoGroup {
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

	items := []infoItem{
		{label: i18n.Text(i18n.MsgUnitPanelName), value: objUnit.GetShipDisplayName(ship.Name)},
		{label: i18n.Text(i18n.MsgUnitPanelType), value: ship.Type.ToDisplay()},
		{label: "HP", value: fmt.Sprintf("%.0f%%", ship.CurHP/ship.TotalHP*100), numeric: true},
		{
			label:   i18n.Text(i18n.MsgUnitPanelSpeed),
			value:   fmt.Sprintf("%.1f / %.1f", ship.CurSpeed*600, ship.MaxSpeed*600),
			numeric: true,
		},
		{
			label:   i18n.Text(i18n.MsgUnitPanelHeading),
			value:   fmt.Sprintf("%03.0f°", math.Mod(ship.CurRotation+360, 360)),
			numeric: true,
		},
		{label: i18n.Text(i18n.MsgUnitPanelStatus), value: status},
		{label: i18n.Text(i18n.MsgUnitPanelGroup), value: group, numeric: true},
	}
	if ship.Aircraft.HasPlane {
		aircraft := ship.Aircraft.Status(ship.Uid, ms.Arena.Planes).Total
		items = append(items, infoItem{
			label:   i18n.Text(i18n.MsgUnitPanelAircraft),
			value:   fmt.Sprintf("%d / %d", aircraft.Alive(), aircraft.Initial()),
			numeric: true,
		})
	}
	items = append(
		items,
		infoItem{label: i18n.Text(i18n.MsgUnitPanelTarget), value: targetName},
		infoItem{label: i18n.Text(i18n.MsgUnitPanelDistance), value: distance, numeric: true},
	)

	return []infoGroup{{title: "", items: items}}
}

// fleetGroup 组装多选舰队的汇总卡。
func (p *Panel) fleetGroup(ships []*objUnit.BattleShip) infoGroup {
	var hpPercent, speed float64
	for _, ship := range ships {
		hpPercent += ship.CurHP / ship.TotalHP * 100
		speed += ship.CurSpeed * 600
	}
	focusName := i18n.Text(i18n.MsgUnitPanelNoTarget)
	if ship := focusedShipFromSlice(ships, p.lastFocus); ship != nil {
		focusName = objUnit.GetShipDisplayName(ship.Name)
	}
	return infoGroup{
		title: i18n.Text(i18n.MsgUnitPanelGroupFleet),
		items: []infoItem{
			{label: i18n.Text(i18n.MsgUnitPanelName), value: focusName},
			{label: i18n.Text(i18n.MsgUnitPanelTotal), value: fmt.Sprintf("%d", len(ships)), numeric: true},
			{
				label:   i18n.Text(i18n.MsgUnitPanelAverageHP),
				value:   fmt.Sprintf("%.0f%%", hpPercent/float64(len(ships))),
				numeric: true,
			},
			{
				label:   i18n.Text(i18n.MsgUnitPanelAverageSpeed),
				value:   fmt.Sprintf("%.1f kn", speed/float64(len(ships))),
				numeric: true,
			},
		},
	}
}

// drawInfoRow 绘制单个 label/value 行；numeric 时值靠右对齐。
func (p *Panel) drawInfoRow(screen *ebiten.Image, y, lineH, labelW float64, item infoItem) {
	fontSize := math.Min(theme.SizeBody, math.Max(theme.SizeCaption, lineH-2))
	area := p.layout.Info
	p.drawText(screen, item.label, area.X+10, y, fontSize, theme.Body(), colorx.Gold)

	valueFace := theme.Body()
	if item.numeric {
		valueFace = theme.Numeric()
	}
	valueX := area.X + 10 + labelW
	maxW := area.W - labelW - 20
	value := p.fitText(item.value, maxW, fontSize, valueFace)
	if item.numeric {
		vw := textLayout.CalcTextWidth(value, fontSize, valueFace)
		valueX = area.X + area.W - 12 - vw
	}
	p.drawText(screen, value, valueX, y, fontSize, valueFace, colorx.White)
}

func focusedShipFromSlice(ships []*objUnit.BattleShip, uid string) *objUnit.BattleShip {
	for _, ship := range ships {
		if ship.Uid == uid {
			return ship
		}
	}
	return nil
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

func (p *Panel) drawTab(screen *ebiten.Image, area Rect, label string, active bool) {
	fill, border, textColor := buttonIdle, sectionBorder, colorx.Silver
	if active {
		fill, border, textColor = color.RGBA{R: 58, G: 69, B: 55, A: 235}, colorx.Gold, colorx.White
	} else if p.cursorAt(area) {
		fill = buttonHover
	}
	p.drawRoundedRect(screen, area, theme.CornerRadius, fill, border)
	p.drawCenteredText(screen, label, area, theme.SizeCaption, textColor)
}

func (p *Panel) drawWeapons(screen *ebiten.Image, ms *state.MissionState, ships []*objUnit.BattleShip) {
	rows := weaponRows(ms, nowMillis())
	if len(rows) == 0 {
		p.drawCenteredText(
			screen,
			i18n.Text(i18n.MsgUnitPanelNoTarget),
			p.layout.Systems,
			theme.SizeSection,
			colorx.Silver,
		)
		return
	}
	p.drawText(
		screen,
		i18n.Text(i18n.MsgUnitPanelAllWeapons),
		p.layout.Systems.X+4,
		p.layout.Systems.Y+38,
		float64(theme.SizeBody),
		theme.Body(),
		colorx.White,
	)
	p.drawToggleDot(screen, p.allWeaponsToggleRect(), allWeaponsToggle(ships))
	for index, row := range rows {
		p.drawWeaponRow(screen, row, index, len(rows))
	}
}

// drawWeaponRow 采用两行式布局并弱化开关：左侧小圆点表示启用状态（可点击），
// 第一行是武器名与就绪数，第二行是装填进度条与剩余时间。
func (p *Panel) drawWeaponRow(screen *ebiten.Image, row weaponRow, index, rowCount int) {
	step := p.weaponRowStep(rowCount)
	y := p.layout.Systems.Y + 66 + float64(index)*step
	area := p.layout.Systems
	fontSize := float64(theme.SizeBody)

	p.drawToggleDot(screen, p.weaponToggleRect(index, rowCount), row.Toggle)

	// 第一行：武器名（左）+ 就绪数（右对齐）。
	p.drawText(screen, p.weaponLabel(row.Type), area.X+44, y+5, fontSize, theme.Body(), colorx.White)
	ready := fmt.Sprintf("%d/%d", row.Ready, row.Equipped)
	readyFont := font.ForText(ready, theme.Numeric())
	readyX := area.X + area.W - 8 - textLayout.CalcTextWidth(ready, fontSize, readyFont)
	p.drawText(screen, ready, readyX, y+5, fontSize, readyFont, colorx.Silver)

	// 第二行：进度条（左）+ 剩余时间（右对齐）。
	barY := y + 27
	status := i18n.Text(i18n.MsgUnitPanelReady)
	if row.RemainingMillis > 0 {
		status = i18n.Format(
			i18n.MsgUnitPanelSeconds,
			map[string]any{"Seconds": fmt.Sprintf("%.1f", float64(row.RemainingMillis)/1e3)},
		)
	}
	statusFont := font.ForText(status, theme.Numeric())
	statusW := textLayout.CalcTextWidth(status, fontSize, statusFont)
	statusX := area.X + area.W - 8 - statusW
	barX := area.X + 44
	barW := statusX - barX - 8
	if barW < 20 {
		barW = 20
	}
	p.drawProgress(screen, Rect{X: barX, Y: barY, W: barW, H: 8}, row.Progress)
	p.drawText(screen, status, statusX, barY-2, fontSize, statusFont, colorx.Silver)
}

// drawToggleDot 绘制一个小巧的状态圆点用作开关：启用(绿)/禁用(红圈空芯)/混合(金)。
// 光标悬停到圆点区域时给出圆形光晕，提示可点击。
func (p *Panel) drawToggleDot(screen *ebiten.Image, area Rect, state toggleState) {
	cx, cy := area.X+area.W/2, area.Y+area.H/2
	if p.cursorAt(area) {
		vector.FillCircle(screen, float32(cx), float32(cy), 12, buttonHover, false)
	}
	vector.FillCircle(screen, float32(cx), float32(cy), 9, toggleDotBorder(state), false)
	vector.FillCircle(screen, float32(cx), float32(cy), 6, toggleDotFill(state), false)
}

func toggleDotBorder(state toggleState) color.Color {
	switch state {
	case toggleDisabled:
		return color.RGBA{R: 196, G: 96, B: 96, A: 255}
	case toggleMixed:
		return colorx.Gold
	default:
		return colorx.Green
	}
}

func toggleDotFill(state toggleState) color.Color {
	switch state {
	case toggleDisabled:
		return color.RGBA{R: 16, G: 30, B: 38, A: 255}
	case toggleMixed:
		return color.RGBA{R: 196, G: 171, B: 101, A: 255}
	default:
		return colorx.Green
	}
}

// drawNavButton 绘制小巧的上一艘/下一艘导航按钮，使用矢量 V 形箭头，避免字体缺失字形导致空白。
func (p *Panel) drawNavButton(screen *ebiten.Image, area Rect, dir theme.TriangleDir) {
	fill, border := toggleFill, toggleBorder
	if p.cursorAt(area) {
		fill = toggleHoverFill
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			fill = togglePressFill
		}
	}
	p.drawRoundedRect(screen, area, theme.CornerRadius, fill, border)
	theme.DrawChevron(screen, area.X+area.W/2, area.Y+area.H/2, navChevronSize, dir, theme.HandleArrow)
}

// drawAircraft 绘制单一表头、各机型数据、合计行和起飞总开关。
func (p *Panel) drawAircraft(screen *ebiten.Image, ms *state.MissionState, ships []*objUnit.BattleShip) {
	rows, total := aircraftRows(ms)
	if len(rows) == 0 {
		p.drawCenteredText(
			screen,
			i18n.Text(i18n.MsgUnitPanelAircraftEmpty),
			p.layout.Systems,
			theme.SizeSection,
			colorx.Silver,
		)
		return
	}
	p.drawText(
		screen,
		i18n.Text(i18n.MsgUnitPanelTakeoff),
		p.layout.Systems.X+4,
		p.layout.Systems.Y+38,
		float64(theme.SizeBody),
		theme.Body(),
		colorx.White,
	)
	toggle := aircraftToggle(ships)
	p.drawToggleDot(screen, p.aircraftToggleRect(), toggle)

	tableY := p.layout.Systems.Y + 72
	rowH := aircraftRowH
	columns := []float64{0, 0.5, 0.66, 0.81, 0.9}
	headings := []string{
		i18n.Text(i18n.MsgUnitPanelPlaneType),
		i18n.Text(i18n.MsgUnitPanelStandby),
		i18n.Text(i18n.MsgUnitPanelInCombat),
		i18n.Text(i18n.MsgUnitPanelReturning),
		i18n.Text(i18n.MsgUnitPanelLost),
	}
	for index, heading := range headings {
		p.drawText(
			screen,
			heading,
			p.layout.Systems.X+columns[index]*p.layout.Systems.W+4,
			tableY,
			theme.SizeCaption,
			theme.Body(),
			colorx.Gold,
		)
	}
	vector.StrokeLine(
		screen,
		float32(p.layout.Systems.X+4),
		float32(tableY+rowH-3),
		float32(p.layout.Systems.X+p.layout.Systems.W-4),
		float32(tableY+rowH-3),
		1,
		sectionBorder,
		false,
	)
	for index, row := range rows {
		p.drawAircraftRow(screen, row, tableY+rowH*float64(index+1), columns, rowH, false)
	}
	p.drawAircraftRow(screen, total, tableY+rowH*float64(len(rows)+1), columns, rowH, true)
}

func (p *Panel) drawAircraftRow(
	screen *ebiten.Image,
	row objUnit.AircraftGroupStatus,
	y float64,
	columns []float64,
	rowH float64,
	total bool,
) {
	name := objUnit.GetPlaneDisplayName(row.Name)
	textColor := colorx.White
	if total {
		name = i18n.Text(i18n.MsgUnitPanelTotal)
		textColor = colorx.Gold
		vector.StrokeLine(
			screen,
			float32(p.layout.Systems.X+4),
			float32(y-3),
			float32(p.layout.Systems.X+p.layout.Systems.W-4),
			float32(y-3),
			1,
			sectionBorder,
			false,
		)
	}
	values := []string{
		name,
		fmt.Sprintf("%d", row.Standby),
		fmt.Sprintf("%d", row.InCombat),
		fmt.Sprintf("%d", row.Returning),
		fmt.Sprintf("%d", row.Lost),
	}
	fontSize := math.Min(theme.SizeBody, math.Max(theme.SizeCaption, rowH-8))
	for index, value := range values {
		maxW := p.layout.Systems.W * 0.12
		if index == 0 {
			maxW = p.layout.Systems.W * 0.46
		}
		face := theme.Body()
		if index > 0 {
			face = theme.Numeric()
		}
		value = p.fitText(value, maxW, fontSize, face)
		p.drawText(
			screen,
			value,
			p.layout.Systems.X+columns[index]*p.layout.Systems.W+4,
			y+2,
			fontSize,
			face,
			textColor,
		)
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

// drawProgress 绘制下一次可发射的装填进度条。
func (p *Panel) drawProgress(screen *ebiten.Image, area Rect, progress float64) {
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

// drawRoundedRect 使用共享 theme 绘制主题化圆角矩形。
func (p *Panel) drawRoundedRect(screen *ebiten.Image, area Rect, radius float64, fill, stroke color.Color) {
	theme.FillRoundedRect(screen, area.X, area.Y, area.W, area.H, radius, fill, stroke, 1)
}

func (p *Panel) drawText(
	screen *ebiten.Image,
	value string,
	x, y, size float64,
	source *text.GoTextFaceSource,
	textColor color.Color,
) {
	opts := &text.DrawOptions{}
	opts.GeoM.Translate(x, y)
	opts.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, value, &text.GoTextFace{Source: source, Size: size}, opts)
}

func (p *Panel) drawCenteredText(screen *ebiten.Image, value string, area Rect, size float64, textColor color.Color) {
	source := theme.Body()
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

// drawAirfieldSections 绘制选中机场的面板内容：标题 + 信息卡（起飞间隔 + 驻场机队）。
func (p *Panel) drawAirfieldSections(screen *ebiten.Image, ms *state.MissionState, af *objBuilding.Airfield) {
	p.drawHeader(screen, ms, i18n.Text(i18n.MsgAirfieldPanelTitle))
	area := p.layout.Info
	if area.W != 0 && area.H != 0 {
		theme.FillRoundedRect(
			screen,
			area.X,
			area.Y,
			area.W,
			area.H,
			theme.CornerRadius,
			sectionFill,
			sectionBorder,
			theme.CardBorderWidth,
		)
	}
	p.drawAirfieldInfo(screen, ms, af)
}

// airfieldInfoItems 组装机场面板的基础信息行（起飞间隔）。
// 跑道位置/朝向/尺寸在地图选中态即可直观看到，不在面板重复展示。
func airfieldInfoItems(af *objBuilding.Airfield) []infoItem {
	return []infoItem{
		{
			label:   i18n.Text(i18n.MsgAirfieldTakeOffInterval),
			value:   fmt.Sprintf("%.1fs", af.Aircraft.TakeOffTime),
			numeric: true,
		},
	}
}

// baselineOffsetY 混排段基线对齐偏移：text.Draw 以行框顶部定位，不同字体的
// ascent 不同，同一 y 会让数字明显偏下；返回 target 字体相对 reference 字体
// 需要附加的 y 偏移（画在 referenceY + 偏移处，两者基线一致）。
func baselineOffsetY(reference, target *text.GoTextFaceSource, size float64) float64 {
	rm := (&text.GoTextFace{Source: reference, Size: size}).Metrics()
	tm := (&text.GoTextFace{Source: target, Size: size}).Metrics()
	return rm.HAscent - tm.HAscent
}

// drawAirfieldInfo 绘制机场信息卡：起飞间隔、机场开关（可点击）、当前生产
// 机型进度条、驻场机队机型行（可点击切换生产机型），行高统一 airfieldLineH。
func (p *Panel) drawAirfieldInfo(screen *ebiten.Image, ms *state.MissionState, af *objBuilding.Airfield) {
	area := p.layout.Info
	if area.W == 0 || area.H == 0 {
		return
	}
	labelW := math.Min(70.0, area.W*0.34)

	y := area.Y + airfieldInfoPad
	for _, item := range airfieldInfoItems(af) {
		p.drawInfoRow(screen, y, airfieldLineH, labelW, item)
		y += airfieldLineH
	}

	// 机场状态行：右侧状态圆点开关（绿=运行中，红=停用），停用时文字置灰
	p.drawText(
		screen, i18n.Text(i18n.MsgAirfieldStatusLabel),
		area.X+10, y, theme.SizeBody, theme.Body(), colorx.Gold,
	)
	statusClr := colorx.White
	statusText := i18n.MsgAirfieldOn
	if af.Disabled {
		statusClr, statusText = colorx.Silver, i18n.MsgAirfieldOff
	}
	p.drawText(
		screen, i18n.Text(statusText),
		area.X+10+labelW, y, theme.SizeBody, theme.Body(), statusClr,
	)
	if af.Disabled {
		p.drawToggleDot(screen, p.airfieldToggleRect(af), toggleDisabled)
	} else {
		p.drawToggleDot(screen, p.airfieldToggleRect(af), toggleAllowed)
	}
	y += airfieldLineH

	// 驻场机队节标题（金色，楷体）+ 机队表格；字号与上方信息行一致
	// （SizeBody），不再用小一号的说明档
	p.drawText(
		screen,
		i18n.Text(i18n.MsgAirfieldSquadTitle),
		area.X+10,
		y,
		theme.SizeBody,
		theme.Body(),
		colorx.Gold,
	)
	y += airfieldLineH

	// 机队表格：表头 + 各机型行，样式对齐舰载机页签的机型表
	// （金色表头 + 分隔线；机型名楷体、数值列等宽，列内基线对齐）；
	// 制造中列展示生产进度条（不显示百分比），资金不足时改显红色提示
	columns := []float64{0, 0.3, 0.48, 0.64, 0.78}
	headings := []string{
		i18n.Text(i18n.MsgUnitPanelPlaneType),
		i18n.Text(i18n.MsgUnitPanelStandby),
		i18n.Text(i18n.MsgAirfieldSortie),
		i18n.Text(i18n.MsgAirfieldLostCount),
		i18n.Text(i18n.MsgAirfieldProducingTag),
	}
	for index, heading := range headings {
		p.drawText(
			screen,
			heading,
			area.X+10+columns[index]*area.W,
			y,
			theme.SizeBody,
			theme.Body(),
			colorx.Gold,
		)
	}
	vector.StrokeLine(
		screen,
		float32(area.X+4),
		float32(y+airfieldLineH-3),
		float32(area.X+area.W-4),
		float32(y+airfieldLineH-3),
		1,
		sectionBorder,
		false,
	)
	y += airfieldLineH
	rows := airfieldSquadRows(ms, af)
	// 全部满编：无可生产机型，停止生产（在指定机型行提示容量已满）
	capacityFull := airfieldCapacityFull(rows)
	for index, row := range rows {
		clr := colorx.Silver
		if row.Producing {
			// 当前生产机型金色高亮
			clr = colorx.Gold
		} else if p.cursorAt(p.airfieldSquadRowRect(af, index)) {
			// 悬停反馈：可选生产机型提亮为白色
			clr = colorx.White
		}
		// 机型列（楷体，超长截断）
		p.drawText(
			screen,
			p.fitText(row.Name, columns[1]*area.W, theme.SizeBody, theme.Body()),
			area.X+10,
			y,
			theme.SizeBody,
			theme.Body(),
			clr,
		)
		// 数值列（等宽字体，按楷体基线对齐）
		values := []string{
			fmt.Sprintf("%d/%d", row.Stock, row.Max),
			fmt.Sprintf("%d", row.Flying),
			fmt.Sprintf("%d", row.Lost),
		}
		numericY := y + baselineOffsetY(theme.Body(), theme.Numeric(), theme.SizeBody)
		for vi, value := range values {
			p.drawText(
				screen,
				value,
				area.X+10+columns[vi+1]*area.W,
				numericY,
				theme.SizeBody,
				theme.Numeric(),
				clr,
			)
		}
		// 制造中列：生产机型显示进度条；资金不足时改显红色「资金不足」；
		// 全部满编时在指定机型行改显「容量已满」提示
		if row.Producing {
			if fundsLow(ms, af) {
				hint := i18n.Text(i18n.MsgFundsInsufficient)
				hintX := area.X + area.W - 12 - textLayout.CalcTextWidth(hint, theme.SizeBody, theme.Body())
				p.drawText(screen, hint, hintX, y, theme.SizeBody, theme.Body(), colorx.Red)
			} else {
				barX := area.X + columns[4]*area.W
				p.drawProgress(screen, Rect{
					X: barX, Y: y + (airfieldLineH-6)/2,
					W: area.W - 12 - (columns[4] * area.W), H: 6,
				}, float64(row.Progress)/100)
			}
		} else if capacityFull && index == af.ProducingGroupIdx() {
			hint := i18n.Text(i18n.MsgAirfieldCapacityFull)
			hintX := area.X + area.W - 12 - textLayout.CalcTextWidth(hint, theme.SizeBody, theme.Body())
			p.drawText(screen, hint, hintX, y, theme.SizeBody, theme.Body(), colorx.Silver)
		}
		y += airfieldLineH
	}
}
