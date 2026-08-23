// Package sidebar 实现任务运行页右侧的合并 RTS 风格侧栏。
// 面板由顶部页签条划分为「地图 + 战舰信息」与「设置」两个页签，共用同一宽度；
// 底部单位信息内容在页签 1 内纵向堆叠并支持滚动。
package sidebar

import (
	"image/color"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/i18n"
	md "github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/unitpanel"
	"github.com/narasux/jutland/pkg/resources/font"
	abbrMapImg "github.com/narasux/jutland/pkg/resources/images/abbrmap"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
	"github.com/narasux/jutland/pkg/utils/layout"
	"github.com/narasux/jutland/pkg/utils/theme"
)

const (
	handleW = 36
	handleH = 60
	// arrowHandleSize 是把手箭头的等边三角形边长。
	arrowHandleSize = 12.0
	// tabBarHeight 是「地图 / 设置」页签条的屏幕像素高度。
	tabBarHeight = 36.0
	// scrollbarW 是内容滚动条的宽度。
	scrollbarW = 6.0
)

// panelBgColor 等由共享 theme 包统一提供。
var (
	panelBgColor   = theme.PanelBackground
	panelLineColor = theme.PanelBorder
	cardBgColor    = theme.CardBackground
	scrollbarTrack = color.RGBA{R: 28, G: 52, B: 59, A: 200}
	scrollbarThumb = color.RGBA{R: 120, G: 160, B: 170, A: 230}
)

type rect struct {
	X, Y, W, H float64
}

func (r rect) contains(x, y int) bool {
	fx, fy := float64(x), float64(y)
	return fx >= r.X && fx <= r.X+r.W && fy >= r.Y && fy <= r.Y+r.H
}

// unitRect 把侧栏自身的矩形转换为单位信息面板使用的公共矩形。
func unitRect(r rect) unitpanel.Rect {
	return unitpanel.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}
}

// Tab 表示侧栏当前展示的页签。
type Tab int

const (
	// TabBattle 展示小地图 + 战场信息 + 战舰信息（可滚动）。
	TabBattle Tab = iota
	// TabSettings 展示两个游戏内显示选项。
	TabSettings
)

type sidebarLayout struct {
	Screen   layout.ScreenLayout
	Panel    rect
	Handle   rect
	TabBar   rect
	Map      rect
	Battle   rect
	Viewport rect
}

// Panel 是任务运行中的合并右侧战术侧栏。
type Panel struct {
	abbrMap *ebiten.Image
	layout  sidebarLayout
	tab     Tab
	scrollY float64
	units   *unitpanel.Panel
}

// New 创建任务侧栏。
func New(mission string) *Panel {
	missionMD := md.Get(mission)
	misLayout := layout.NewScreenLayout()
	abbrMap := ebiten.NewImage(misLayout.Height, misLayout.Height)

	bg := abbrMapImg.Background
	w, h := bg.Bounds().Dx(), bg.Bounds().Dy()
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	opts.GeoM.Scale(float64(misLayout.Height)/float64(w), float64(misLayout.Height)/float64(h))
	abbrMap.DrawImage(abbrMapImg.Background, opts)
	abbrMap.DrawImage(abbrMapImg.Get(missionMD.MapCfg.Source), opts)

	return &Panel{abbrMap: abbrMap, units: unitpanel.New()}
}

// Update 更新侧栏控件状态，处理把手、页签、滚动与内容点击，并返回单位信息面板产生的操作。
func (p *Panel) Update(ms *state.MissionState) []unitpanel.Action {
	if ms.Core.MissionStatus != state.MissionRunning {
		return nil
	}
	p.layout = calcLayout(ms.View.Layout, ms.UI.SidebarExpanded, p.tab)
	sx, sy := ebiten.CursorPosition()
	leftPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)

	if leftPressed && p.layout.Handle.contains(sx, sy) {
		ms.UI.SidebarExpanded = !ms.UI.SidebarExpanded
		return nil
	}
	if !ms.UI.SidebarExpanded {
		return nil
	}

	if leftPressed {
		if newTab, ok := p.tabAt(sx, sy); ok {
			if newTab != p.tab {
				p.tab = newTab
				p.scrollY = 0
			}
			return nil
		}
	}

	switch p.tab {
	case TabBattle:
		return p.updateBattle(ms, sx, sy, leftPressed)
	default:
		p.updateSettings(ms, sx, sy, leftPressed)
		return nil
	}
}

func (p *Panel) updateBattle(ms *state.MissionState, sx, sy int, leftPressed bool) []unitpanel.Action {
	if leftPressed && p.layout.Map.contains(sx, sy) {
		p.centerCameraAtMinimap(ms, sx, sy)
	}
	if _, wheelY := ebiten.Wheel(); wheelY != 0 && p.layout.Viewport.contains(sx, sy) {
		p.scrollY = p.clampScroll(ms, p.scrollY-float64(wheelY)*48)
	} else {
		p.scrollY = p.clampScroll(ms, p.scrollY)
	}
	return p.units.Update(ms, unitRect(p.layout.Viewport), p.scrollY)
}

// updateSettings 处理设置页签内两个复选框的点击。
func (p *Panel) updateSettings(ms *state.MissionState, sx, sy int, leftPressed bool) {
	if !leftPressed {
		return
	}
	if p.settingsRowRect(0).contains(sx, sy) {
		ms.UI.GameOpts.ForceDisplayState = !ms.UI.GameOpts.ForceDisplayState
	} else if p.settingsRowRect(1).contains(sx, sy) {
		ms.UI.GameOpts.DisplayDamageNumber = !ms.UI.GameOpts.DisplayDamageNumber
	}
}

// clampScroll 将滚动偏移限制在内容可视范围内。
func (p *Panel) clampScroll(ms *state.MissionState, scroll float64) float64 {
	contentH := p.units.MeasureContent(ms, unitRect(p.layout.Viewport))
	return math.Max(0, math.Min(scroll, math.Max(0, contentH-p.layout.Viewport.H)))
}

// Draw 绘制任务侧栏。
func (p *Panel) Draw(screen *ebiten.Image, ms *state.MissionState) {
	if ms.Core.MissionStatus != state.MissionRunning {
		return
	}
	p.layout = calcLayout(ms.View.Layout, ms.UI.SidebarExpanded, p.tab)
	if ms.UI.SidebarExpanded {
		p.drawPanel(screen, ms)
	}
	p.drawHandleFrame(screen, ms)
	p.drawHandleArrow(screen, ms)
}

// ConsumesCursor 判断当前鼠标位置是否应由侧栏消费。
func (p *Panel) ConsumesCursor(ms *state.MissionState) bool {
	sx, sy := ebiten.CursorPosition()
	return p.consumesCursorAt(ms, sx, sy)
}

func (p *Panel) consumesCursorAt(ms *state.MissionState, sx, sy int) bool {
	if ms.Core.MissionStatus != state.MissionRunning {
		return false
	}
	ui := calcLayout(ms.View.Layout, ms.UI.SidebarExpanded, p.tab)
	if ui.Handle.contains(sx, sy) {
		return true
	}
	return ms.UI.SidebarExpanded && ui.Panel.contains(sx, sy)
}

// OccupiedWidth 返回展开侧栏遮挡战场的屏幕像素宽度。
func (p *Panel) OccupiedWidth(ms *state.MissionState) float64 {
	if !ms.UI.SidebarExpanded || ms.Core.MissionStatus != state.MissionRunning {
		return 0
	}
	return calcLayout(ms.View.Layout, true, p.tab).Panel.W
}

func (p *Panel) drawPanel(screen *ebiten.Image, ms *state.MissionState) {
	ui := p.layout
	theme.FillRoundedRect(
		screen,
		ui.Panel.X, ui.Panel.Y, ui.Panel.W, ui.Panel.H,
		theme.CornerRadius,
		panelBgColor, panelLineColor, theme.PanelBorderWidth,
	)
	p.drawTabBar(screen)
	switch p.tab {
	case TabBattle:
		p.drawMinimap(screen, ms)
		p.drawBattleInfo(screen, ms)
		p.units.Draw(screen, ms, unitRect(ui.Viewport), p.scrollY)
		p.drawScrollbar(screen, ms)
	default:
		p.drawSettings(screen, ms)
	}
}

// tabAt 返回光标下方对应的页签；不在页签条内时返回 false。
func (p *Panel) tabAt(sx, sy int) (Tab, bool) {
	if !p.layout.TabBar.contains(sx, sy) {
		return 0, false
	}
	half := p.layout.TabBar.W / 2
	if float64(sx) < p.layout.TabBar.X+half {
		return TabBattle, true
	}
	return TabSettings, true
}

// tabRect 返回页签条中第 index 个页签的几何区域。
func (p *Panel) tabRect(index int) rect {
	ui := p.layout
	w := ui.TabBar.W / 2
	return rect{X: ui.TabBar.X + float64(index)*w, Y: ui.TabBar.Y, W: w, H: ui.TabBar.H}
}

func (p *Panel) drawTabBar(screen *ebiten.Image) {
	ui := p.layout
	separatorY := ui.TabBar.Y + ui.TabBar.H
	vector.StrokeLine(screen, float32(ui.Panel.X), float32(separatorY), float32(ui.Panel.X+ui.Panel.W), float32(separatorY), 1, panelLineColor, false)

	labels := []string{tabLabel(TabBattle), tabLabel(TabSettings)}
	for index, label := range labels {
		area := p.tabRect(index)
		active := Tab(index) == p.tab
		fill, border, textColor := theme.ButtonIdle, theme.CardBorder, colorx.Silver
		if active {
			fill, border, textColor = color.RGBA{R: 58, G: 69, B: 55, A: 235}, colorx.Gold, colorx.White
		}
		theme.FillRoundedRect(screen, area.X+4, area.Y+6, area.W-8, area.H-12, theme.CornerRadius, fill, border, theme.CardBorderWidth)
		p.drawCenteredText(screen, label, area, theme.SizeBody, textColor)
	}
}

func (p *Panel) drawSettings(screen *ebiten.Image, ms *state.MissionState) {
	p.drawCheckboxRow(screen, 0, i18n.Text(i18n.MsgSidebarShowState), ms.UI.GameOpts.ForceDisplayState)
	p.drawCheckboxRow(screen, 1, i18n.Text(i18n.MsgSidebarDamageNumbers), ms.UI.GameOpts.DisplayDamageNumber)
}

// settingsRowTop 是设置页签内容起始 Y。
func (p *Panel) settingsRowTop() float64 {
	return p.layout.TabBar.Y + p.layout.TabBar.H + 24
}

// settingsRowRect 返回第 index 个设置行的点击区域。
func (p *Panel) settingsRowRect(index int) rect {
	x := p.layout.Panel.X + 20
	y := p.settingsRowTop() + float64(index)*44
	return rect{X: x, Y: y, W: p.layout.Panel.W - 40, H: 40}
}

func (p *Panel) drawCheckboxRow(screen *ebiten.Image, index int, label string, checked bool) {
	row := p.settingsRowRect(index)
	boxSize := float32(18)
	contentY := row.Y + 11
	boxX, boxY := float32(row.X), float32(contentY)
	textColor := lo.Ternary(checked, colorx.Gold, colorx.Silver)
	borderColor := lo.Ternary(checked, colorx.Gold, panelLineColor)

	vector.StrokeRect(screen, boxX, boxY, boxSize, boxSize, 2, borderColor, false)
	if checked {
		vector.StrokeLine(screen, boxX+4, boxY+9, boxX+8, boxY+14, 3, colorx.Gold, false)
		vector.StrokeLine(screen, boxX+8, boxY+14, boxX+15, boxY+4, 3, colorx.Gold, false)
	}
	p.drawText(screen, label, row.X+34, contentY, 18, font.LocalizedUI(font.Kai), textColor)
}

func (p *Panel) drawHandleFrame(screen *ebiten.Image, ms *state.MissionState) {
	ui := calcLayout(ms.View.Layout, ms.UI.SidebarExpanded, p.tab)
	sx, sy := ebiten.CursorPosition()
	fill := theme.HandleFill
	if ui.Handle.contains(sx, sy) {
		fill = theme.HandleHover
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			fill = theme.ButtonPressed
		}
	}
	handleW := float64(ui.Handle.W)
	if ms.UI.SidebarExpanded {
		handleW += 2
	}
	theme.FillRoundedRect(
		screen,
		ui.Handle.X, ui.Handle.Y, handleW, float64(ui.Handle.H),
		theme.HandleCornerRadius,
		fill, theme.HandleBorder, theme.CardBorderWidth,
	)
}

func (p *Panel) drawHandleArrow(screen *ebiten.Image, ms *state.MissionState) {
	ui := calcLayout(ms.View.Layout, ms.UI.SidebarExpanded, p.tab)
	cx, cy := ui.Handle.X+ui.Handle.W/2, ui.Handle.Y+ui.Handle.H/2
	dir := theme.TriangleLeft
	if ms.UI.SidebarExpanded {
		dir = theme.TriangleRight
	}
	theme.DrawTriangle(screen, cx, cy, arrowHandleSize, dir, theme.HandleArrow)
}

func (p *Panel) drawMinimap(screen *ebiten.Image, ms *state.MissionState) {
	ui := p.layout
	theme.FillRoundedRect(
		screen,
		ui.Map.X-2, ui.Map.Y-2, ui.Map.W+4, ui.Map.H+4,
		theme.CornerRadius,
		color.RGBA{R: 2, G: 8, B: 11, A: 245},
		panelLineColor, theme.CardBorderWidth,
	)

	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	opts.GeoM.Scale(ui.Map.W/float64(p.abbrMap.Bounds().Dx()), ui.Map.H/float64(p.abbrMap.Bounds().Dy()))
	opts.GeoM.Translate(ui.Map.X, ui.Map.Y)
	screen.DrawImage(p.abbrMap, opts)

	vector.StrokeRect(
		screen,
		float32(ui.Map.X),
		float32(ui.Map.Y),
		float32(ui.Map.W),
		float32(ui.Map.H),
		2,
		panelLineColor,
		false,
	)
	p.drawMinimapCamera(screen, ms)
	p.drawMinimapBuildings(screen, ms)
	p.drawMinimapShips(screen, ms)
	p.drawMinimapPlanes(screen, ms)
}

func (p *Panel) drawMinimapCamera(screen *ebiten.Image, ms *state.MissionState) {
	ui := p.layout
	mapW, mapH := float64(ms.Core.MissionMD.MapCfg.Width), float64(ms.Core.MissionMD.MapCfg.Height)
	x1 := ui.Map.X + ms.View.Camera.Pos.RX/mapW*ui.Map.W
	y1 := ui.Map.Y + ms.View.Camera.Pos.RY/mapH*ui.Map.H
	x2 := ui.Map.X + (ms.View.Camera.Pos.RX+float64(ms.View.Camera.Width))/mapW*ui.Map.W
	y2 := ui.Map.Y + (ms.View.Camera.Pos.RY+float64(ms.View.Camera.Height))/mapH*ui.Map.H
	vector.StrokeRect(screen, float32(x1), float32(y1), float32(x2-x1), float32(y2-y1), 2, colorx.White, false)
}

func (p *Panel) drawMinimapBuildings(screen *ebiten.Image, ms *state.MissionState) {
	for _, rp := range ms.Arena.ReinforcePoints {
		clr := lo.Ternary(rp.BelongPlayer == ms.Player.CurPlayer, colorx.Green, colorx.Red)
		x, y := p.mapToSidebar(ms, rp.Pos.RX, rp.Pos.RY)
		vector.FillCircle(screen, float32(x), float32(y), 3, clr, false)
	}
	for _, op := range ms.Arena.OilPlatforms {
		x, y := p.mapToSidebar(ms, op.Pos.RX, op.Pos.RY)
		vector.FillCircle(screen, float32(x), float32(y), 2.5, colorx.Gold, false)
	}
}

func (p *Panel) drawMinimapShips(screen *ebiten.Image, ms *state.MissionState) {
	ships := lo.Values(ms.Arena.Ships)
	slices.SortFunc(ships, func(a, b *objUnit.BattleShip) int {
		return strings.Compare(a.Uid, b.Uid)
	})
	for _, ship := range ships {
		img := textureImg.GetAbbrShip(ship.TypeAbbr, ship.BelongPlayer != ms.Player.CurPlayer)
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		ebutil.SetOptsCenterRotation(opts, img, ship.CurRotation)
		opts.GeoM.Scale(0.7, 0.7)
		x, y := p.mapToSidebar(ms, ship.CurPos.RX, ship.CurPos.RY)
		w, h := float64(img.Bounds().Dx())*0.7, float64(img.Bounds().Dy())*0.7
		opts.GeoM.Translate(x-w/2, y-h/2)
		screen.DrawImage(img, opts)
	}
}

func (p *Panel) drawMinimapPlanes(screen *ebiten.Image, ms *state.MissionState) {
	planes := lo.Values(ms.Arena.Planes)
	slices.SortFunc(planes, func(a, b *objUnit.Plane) int {
		return strings.Compare(a.Uid, b.Uid)
	})
	for _, plane := range planes {
		img := textureImg.GetAbbrPlane(plane.BelongPlayer != ms.Player.CurPlayer)
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		ebutil.SetOptsCenterRotation(opts, img, plane.CurRotation)
		opts.GeoM.Scale(0.65, 0.65)
		x, y := p.mapToSidebar(ms, plane.CurPos.RX, plane.CurPos.RY)
		w, h := float64(img.Bounds().Dx())*0.65, float64(img.Bounds().Dy())*0.65
		opts.GeoM.Translate(x-w/2, y-h/2)
		screen.DrawImage(img, opts)
	}
}

func (p *Panel) drawBattleInfo(screen *ebiten.Image, ms *state.MissionState) {
	ui := p.layout
	p.drawCard(screen, ui.Battle.X, ui.Battle.Y, ui.Battle.W, ui.Battle.H)

	selfFleet := ms.Fleet(ms.Player.CurPlayer)
	enemyFleet := ms.Fleet(ms.Player.CurEnemy)
	bodyFont := font.LocalizedUI(font.Kai)
	p.drawText(
		screen,
		i18n.Format(i18n.MsgSidebarFunds, map[string]any{"Funds": ms.Player.CurFunds}),
		ui.Battle.X+12,
		ui.Battle.Y+14,
		18,
		bodyFont,
		colorx.White,
	)
	p.drawText(
		screen,
		i18n.Format(i18n.MsgSidebarAllyFleet, map[string]any{"Count": selfFleet.Total}),
		ui.Battle.X+12,
		ui.Battle.Y+40,
		16,
		bodyFont,
		colorx.Silver,
	)
	p.drawText(
		screen,
		i18n.Format(i18n.MsgSidebarEnemyFleet, map[string]any{"Count": enemyFleet.Total}),
		ui.Battle.X+12,
		ui.Battle.Y+64,
		16,
		bodyFont,
		colorx.Silver,
	)
}

func (p *Panel) drawScrollbar(screen *ebiten.Image, ms *state.MissionState) {
	ui := p.layout
	contentH := p.units.MeasureContent(ms, unitRect(ui.Viewport))
	maxScroll := contentH - ui.Viewport.H
	if maxScroll <= 0 {
		return
	}
	trackX := ui.Viewport.X + ui.Viewport.W - scrollbarW - 4
	theme.FillRoundedRect(screen, trackX, ui.Viewport.Y, scrollbarW, ui.Viewport.H, scrollbarW/2, scrollbarTrack, scrollbarTrack, 1)

	thumbH := ui.Viewport.H * ui.Viewport.H / contentH
	if thumbH < 24 {
		thumbH = 24
	}
	thumbY := ui.Viewport.Y + (ui.Viewport.H-thumbH)*(p.scrollY/maxScroll)
	theme.FillRoundedRect(screen, trackX, thumbY, scrollbarW, thumbH, scrollbarW/2, scrollbarThumb, scrollbarThumb, 1)
}

func (p *Panel) drawCard(screen *ebiten.Image, x, y, w, h float64) {
	theme.FillRoundedRect(
		screen,
		x, y, w, h,
		theme.CornerRadius,
		cardBgColor, theme.CardBorder, theme.CardBorderWidth,
	)
}

func (p *Panel) drawText(
	screen *ebiten.Image,
	textStr string,
	posX, posY, fontSize float64,
	textFont *text.GoTextFaceSource,
	textColor color.Color,
) {
	opts := &text.DrawOptions{}
	opts.GeoM.Translate(posX, posY)
	opts.ColorScale.ScaleWithColor(textColor)
	textFace := text.GoTextFace{Source: textFont, Size: fontSize}
	text.Draw(screen, textStr, &textFace, opts)
}

func (p *Panel) drawCenteredText(screen *ebiten.Image, value string, area rect, size float64, textColor color.Color) {
	source := theme.Body()
	width, height := text.Measure(value, &text.GoTextFace{Source: source, Size: size}, 0)
	p.drawText(screen, value, area.X+(area.W-width)/2, area.Y+(area.H-height)/2, size, source, textColor)
}

func (p *Panel) centerCameraAtMinimap(ms *state.MissionState, sx, sy int) {
	ui := p.layout
	mapW := float64(ms.Core.MissionMD.MapCfg.Width)
	mapH := float64(ms.Core.MissionMD.MapCfg.Height)
	rx := (float64(sx) - ui.Map.X) / ui.Map.W * mapW
	ry := (float64(sy) - ui.Map.Y) / ui.Map.H * mapH

	nextPos := objPos.NewR(rx-float64(ms.View.Camera.Width)/2, ry-float64(ms.View.Camera.Height)/2)
	nextPos.EnsureBorder(ms.CameraPosBorder())
	ms.View.Camera.Pos = nextPos
}

func (p *Panel) mapToSidebar(ms *state.MissionState, rx, ry float64) (float64, float64) {
	ui := p.layout
	return ui.Map.X + rx/float64(ms.Core.MissionMD.MapCfg.Width)*ui.Map.W,
		ui.Map.Y + ry/float64(ms.Core.MissionMD.MapCfg.Height)*ui.Map.H
}

func tabLabel(tab Tab) string {
	if tab == TabSettings {
		return i18n.Text(i18n.MsgSidebarTabSettings)
	}
	return i18n.Text(i18n.MsgSidebarTabBattle)
}

func calcLayout(screen layout.ScreenLayout, expanded bool, tab Tab) sidebarLayout {
	panelW := math.Max(260, math.Min(float64(screen.Width)*0.24, 360))
	panelX := float64(screen.Width)
	if expanded {
		panelX -= panelW
	}

	handleX := panelX - handleW
	if !expanded {
		handleX = float64(screen.Width - handleW)
	}
	mapSize := panelW - 32
	tabBar := rect{X: panelX, Y: 0, W: panelW, H: tabBarHeight}
	mapR := rect{X: panelX + 16, Y: tabBarHeight + 20, W: mapSize, H: mapSize}
	battle := rect{X: panelX + 16, Y: mapR.Y + mapSize + 16, W: panelW - 32, H: 104}
	viewportTop := battle.Y + battle.H + 16
	viewport := rect{X: panelX, Y: viewportTop, W: panelW, H: math.Max(0, float64(screen.Height)-viewportTop-12)}
	if !expanded {
		viewport = rect{}
	}
	return sidebarLayout{
		Screen:   screen,
		Panel:    rect{X: panelX, Y: 0, W: panelW, H: float64(screen.Height)},
		Handle:   rect{X: handleX, Y: (float64(screen.Height) - handleH) / 2, W: handleW, H: handleH},
		TabBar:   tabBar,
		Map:      mapR,
		Battle:   battle,
		Viewport: viewport,
	}
}
