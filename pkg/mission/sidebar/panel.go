// Package sidebar 实现任务运行页右上角的停靠式下拉战术面板（参考红警 3 的小地图位置）。
// 面板贴着屏幕顶缘与右缘（各留 12px 细缝保证镜头边缘滚动可用），从上方下拉展开，
// 收起后只留底缘把手；由页签条划分为「地图 + 战舰信息」与「设置」两个页签，共用同一宽度；
// 资金与己方舰数画在战舰名称同一行；底部单位信息在页签 1 内纵向堆叠并支持滚动。
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
	// handleW / handleH 是面板底缘「展开/收起」把手的尺寸（胶囊形）。
	handleW = 132.0
	handleH = 24.0
	// handleGap 是面板底缘与把手之间的间距。
	handleGap = 4.0
	// handleChevronSize 是把手 V 形箭头的开口半宽。
	handleChevronSize = 6.0
	// panelMarginTop / panelMarginRight 是面板与屏幕顶缘、右缘的细缝宽度。
	// 缝隙里的战场区域不消耗鼠标，保证镜头的顶部/右侧/右上角边缘滚动始终可用。
	panelMarginTop   = 12.0
	panelMarginRight = 12.0
	// panelHeightRatio 是展开面板高度占屏幕高度的比例（参考红警 3：面板停靠在屏幕右上角）。
	panelHeightRatio = 0.86
	// tabBarHeight 是「地图 / 设置」页签条的屏幕像素高度。
	tabBarHeight = 36.0
	// scrollbarW 是内容滚动条的宽度。
	scrollbarW = 6.0
	// mapMaxHeightRatio 是小地图区域占屏幕高度的比例上限，防止竖长地图挤压下方内容。
	mapMaxHeightRatio = 0.45
)

// panelBgColor 等由共享 theme 包统一提供。
var (
	panelBgColor   = theme.PanelBackground
	panelLineColor = theme.PanelBorder
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
	// TabBattle 展示小地图 + 战舰信息（可滚动）。
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
	Viewport rect
}

// Panel 是任务运行中的右上角停靠式下拉战术面板。
type Panel struct {
	abbrMap *ebiten.Image
	// mapAspect 是地图实际宽高比（宽/高），用于让小地图保持地图原始比例。
	mapAspect float64
	layout    sidebarLayout
	tab       Tab
	scrollY   float64
	units     *unitpanel.Panel
}

// New 创建任务侧栏。
func New(mission string) *Panel {
	missionMD := md.Get(mission)
	misLayout := layout.NewScreenLayout()
	// 与全屏缩略地图一致，按地图实际宽高比合成，避免非正方形地图被拉伸变形
	abbrMap := abbrMapImg.NewComposite(
		missionMD.MapCfg.Source,
		missionMD.MapCfg.Width,
		missionMD.MapCfg.Height,
		misLayout.Height,
	)

	return &Panel{
		abbrMap:   abbrMap,
		mapAspect: float64(missionMD.MapCfg.Width) / float64(missionMD.MapCfg.Height),
		units:     unitpanel.New(),
	}
}

// Update 更新侧栏控件状态，处理把手、页签、滚动与内容点击，并返回单位信息面板产生的操作。
func (p *Panel) Update(ms *state.MissionState) []unitpanel.Action {
	if ms.Core.MissionStatus != state.MissionRunning {
		return nil
	}
	p.layout = calcLayout(ms.View.Layout, ms.UI.SidebarExpanded, p.tab, p.mapAspect)
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
	p.layout = calcLayout(ms.View.Layout, ms.UI.SidebarExpanded, p.tab, p.mapAspect)
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
	ui := calcLayout(ms.View.Layout, ms.UI.SidebarExpanded, p.tab, p.mapAspect)
	if ui.Handle.contains(sx, sy) {
		return true
	}
	return ms.UI.SidebarExpanded && ui.Panel.contains(sx, sy)
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

// OpenUnitsTab 把侧栏切到「地图 / 单位」页签（选中机场等展示型单位时由
// manager 调用，配合 ms.UI.SidebarExpanded 让信息卡直接可见）。
func (p *Panel) OpenUnitsTab() {
	if p.tab != TabBattle {
		p.tab = TabBattle
		p.scrollY = 0
	}
}

func (p *Panel) drawTabBar(screen *ebiten.Image) {
	ui := p.layout
	separatorY := ui.TabBar.Y + ui.TabBar.H
	vector.StrokeLine(
		screen,
		float32(ui.Panel.X),
		float32(separatorY),
		float32(ui.Panel.X+ui.Panel.W),
		float32(separatorY),
		1,
		panelLineColor,
		false,
	)

	labels := []string{tabLabel(TabBattle), tabLabel(TabSettings)}
	for index, label := range labels {
		area := p.tabRect(index)
		active := Tab(index) == p.tab
		fill, border, textColor := theme.ButtonIdle, theme.CardBorder, colorx.Silver
		if active {
			fill, border, textColor = color.RGBA{R: 58, G: 69, B: 55, A: 235}, colorx.Gold, colorx.White
		}
		theme.FillRoundedRect(
			screen,
			area.X+4,
			area.Y+6,
			area.W-8,
			area.H-12,
			theme.CornerRadius,
			fill,
			border,
			theme.CardBorderWidth,
		)
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
	ui := calcLayout(ms.View.Layout, ms.UI.SidebarExpanded, p.tab, p.mapAspect)
	sx, sy := ebiten.CursorPosition()
	fill := theme.HandleFill
	if ui.Handle.contains(sx, sy) {
		fill = theme.HandleHover
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			fill = theme.ButtonPressed
		}
	}
	// 圆角取半高，渲染成胶囊形把手
	theme.FillRoundedRect(
		screen,
		ui.Handle.X, ui.Handle.Y, ui.Handle.W, ui.Handle.H,
		ui.Handle.H/2,
		fill, theme.HandleBorder, theme.CardBorderWidth,
	)
}

// drawHandleArrow 在把手中心绘制 V 形折线箭头：收起时朝下提示「下拉展开」，展开时朝上提示「收起」。
func (p *Panel) drawHandleArrow(screen *ebiten.Image, ms *state.MissionState) {
	ui := calcLayout(ms.View.Layout, ms.UI.SidebarExpanded, p.tab, p.mapAspect)
	dir := theme.TriangleDown
	if ms.UI.SidebarExpanded {
		dir = theme.TriangleUp
	}
	theme.DrawChevron(
		screen,
		ui.Handle.X+ui.Handle.W/2,
		ui.Handle.Y+ui.Handle.H/2,
		handleChevronSize,
		dir,
		theme.HandleArrow,
	)
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
	// 陆地机场（按阵营着色的圆点，停用时置灰；当前选中的机场加白色外圈）
	for _, af := range ms.Arena.Airfields {
		clr := colorx.Gray
		if !af.Disabled {
			clr = lo.Ternary(af.BelongPlayer == ms.Player.CurPlayer, colorx.Green, colorx.Red)
		}
		x, y := p.mapToSidebar(ms, af.Pos.RX, af.Pos.RY)
		vector.FillCircle(screen, float32(x), float32(y), 3, clr, false)
		if af.Uid == ms.Interaction.SelectedAirfieldUid {
			vector.StrokeCircle(screen, float32(x), float32(y), 5, 1.5, colorx.White, false)
		}
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

func (p *Panel) drawScrollbar(screen *ebiten.Image, ms *state.MissionState) {
	ui := p.layout
	contentH := p.units.MeasureContent(ms, unitRect(ui.Viewport))
	maxScroll := contentH - ui.Viewport.H
	if maxScroll <= 0 {
		return
	}
	trackX := ui.Viewport.X + ui.Viewport.W - scrollbarW - 4
	theme.FillRoundedRect(
		screen,
		trackX,
		ui.Viewport.Y,
		scrollbarW,
		ui.Viewport.H,
		scrollbarW/2,
		scrollbarTrack,
		scrollbarTrack,
		1,
	)

	thumbH := ui.Viewport.H * ui.Viewport.H / contentH
	if thumbH < 24 {
		thumbH = 24
	}
	thumbY := ui.Viewport.Y + (ui.Viewport.H-thumbH)*(p.scrollY/maxScroll)
	theme.FillRoundedRect(screen, trackX, thumbY, scrollbarW, thumbH, scrollbarW/2, scrollbarThumb, scrollbarThumb, 1)
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

func calcLayout(screen layout.ScreenLayout, expanded bool, tab Tab, mapAspect float64) sidebarLayout {
	panelW := math.Max(260, math.Min(float64(screen.Width)*0.24, 360))
	// 面板停靠在屏幕右上角（参考红警 3）：贴着顶缘与右缘，只留细缝保证边缘滚动可用
	panelX := float64(screen.Width) - panelMarginRight - panelW
	panelY := panelMarginTop
	panelH := 0.0
	if expanded {
		panelH = float64(screen.Height) * panelHeightRatio
	}

	// 把手挂在面板底缘；收起时面板高度为 0，把手贴着顶部作为「下拉」入口
	handle := rect{
		X: panelX + (panelW-handleW)/2,
		Y: panelY + panelH + handleGap,
		W: handleW,
		H: handleH,
	}

	// 小地图保持地图实际宽高比；竖长地图（如珍珠港 128x192）超高时等比缩小并水平居中
	mapMaxW := panelW - 32
	mapW, mapH := mapMaxW, mapMaxW
	if mapAspect > 0 {
		mapH = mapW / mapAspect
		if maxH := float64(screen.Height) * mapMaxHeightRatio; mapH > maxH {
			mapH = maxH
			mapW = maxH * mapAspect
		}
	}
	tabBar := rect{X: panelX, Y: panelY, W: panelW, H: tabBarHeight}
	mapR := rect{X: panelX + 16 + (mapMaxW-mapW)/2, Y: panelY + tabBarHeight + 20, W: mapW, H: mapH}
	viewportTop := mapR.Y + mapH + 16
	viewport := rect{
		X: panelX,
		Y: viewportTop,
		W: panelW,
		H: math.Max(0, panelY+panelH-viewportTop-16),
	}
	if !expanded {
		viewport = rect{}
	}
	return sidebarLayout{
		Screen:   screen,
		Panel:    rect{X: panelX, Y: panelY, W: panelW, H: panelH},
		Handle:   handle,
		TabBar:   tabBar,
		Map:      mapR,
		Viewport: viewport,
	}
}
