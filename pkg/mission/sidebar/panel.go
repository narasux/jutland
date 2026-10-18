// Package sidebar 实现任务运行页右上角的停靠式下拉战术面板（参考红警 3 的小地图位置）。
// 面板贴着屏幕顶缘与右缘（各留 12px 细缝保证镜头边缘滚动可用），从上方下拉展开，
// 收起后只留底缘把手；顶部页签条现在只有一个「地图 / 单位」标题：显示类设置已迁到
// 主菜单的游戏设置，侧栏不再提供第二个页签；
// 资金与己方舰数画在战舰名称同一行；单位信息在标题下方纵向堆叠并支持滚动。
package sidebar

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/drawer"
	md "github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/unitpanel"
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

// sidebarLayout 面板、把手、页签条与小地图的几何区域。
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
	fog     drawer.FogImageCache
	// mapAspect 是地图实际宽高比（宽/高），用于让小地图保持地图原始比例。
	mapAspect float64
	layout    sidebarLayout
	scrollY   float64
	units     *unitpanel.Panel
	layoutKey sidebarLayoutKey
	layoutOK  bool
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
	p.layout = p.ensureLayout(ms)
	sx, sy := ebiten.CursorPosition()
	leftPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)

	if leftPressed && p.layout.Handle.contains(sx, sy) {
		ms.UI.SidebarExpanded = !ms.UI.SidebarExpanded
		return nil
	}
	if !ms.UI.SidebarExpanded {
		return nil
	}

	// 页签条现在只剩「地图 / 单位」一个标题，点它只做消费，不再切页
	if leftPressed && p.layout.TabBar.contains(sx, sy) {
		return nil
	}
	return p.updateBattle(ms, sx, sy, leftPressed)
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
	p.layout = p.ensureLayout(ms)
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
	ui := p.ensureLayout(ms)
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
	p.drawMinimap(screen, ms)
	p.units.Draw(screen, ms, unitRect(ui.Viewport), p.scrollY)
	p.drawScrollbar(screen, ms)
}

// tabRect 返回页签条里标题的几何区域（页签条现在只有一个标题）。
func (p *Panel) tabRect() rect {
	return p.layout.TabBar
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

	// 页签条只剩一个标题：显示设置已迁到主菜单的游戏设置，不再有第二个页签
	area := p.tabRect()
	theme.FillRoundedRect(
		screen,
		area.X+4,
		area.Y+6,
		area.W-8,
		area.H-12,
		theme.CornerRadius,
		color.RGBA{R: 58, G: 69, B: 55, A: 235},
		colorx.Gold,
		theme.CardBorderWidth,
	)
	p.drawCenteredText(screen, i18n.Text(i18n.MsgSidebarTabBattle), area, theme.SizeBody, colorx.White)
}

func (p *Panel) drawHandleFrame(screen *ebiten.Image, ms *state.MissionState) {
	ui := p.ensureLayout(ms)
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
	ui := p.ensureLayout(ms)
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
	p.drawMinimapFog(screen, ms)

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

func (p *Panel) drawMinimapFog(screen *ebiten.Image, ms *state.MissionState) {
	if ms.Core.MissionMD.MapCfg == nil {
		return
	}
	ui := p.layout
	mapW := float64(ms.Core.MissionMD.MapCfg.Width)
	mapH := float64(ms.Core.MissionMD.MapCfg.Height)
	drawer.BlitFog(
		screen, ms, &p.fog,
		ui.Map.W/(mapW*4), ui.Map.H/(mapH*4),
		ui.Map.X, ui.Map.Y,
	)
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
	for _, ship := range ms.Arena.OrderedShips() {
		if ms.ConcealsEnemy(ship.BelongPlayer, ship.CurPos.MX, ship.CurPos.MY) {
			continue
		}
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
	for _, plane := range ms.Arena.OrderedPlanes() {
		if ms.ConcealsEnemy(plane.BelongPlayer, plane.CurPos.MX, plane.CurPos.MY) {
			continue
		}
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
	// 小地图点击是玩家手动定位，放弃进行中的相机定位，避免两者抢相机。
	ms.View.Camera.Target = nil
	ms.View.Camera.Pos = nextPos
}

func (p *Panel) mapToSidebar(ms *state.MissionState, rx, ry float64) (float64, float64) {
	ui := p.layout
	return ui.Map.X + rx/float64(ms.Core.MissionMD.MapCfg.Width)*ui.Map.W,
		ui.Map.Y + ry/float64(ms.Core.MissionMD.MapCfg.Height)*ui.Map.H
}

type sidebarLayoutKey struct {
	width, height int
	expanded      bool
	aspect        float64
}

func (p *Panel) ensureLayout(ms *state.MissionState) sidebarLayout {
	key := sidebarLayoutKey{
		width:    ms.View.Layout.Width,
		height:   ms.View.Layout.Height,
		expanded: ms.UI.SidebarExpanded,
		aspect:   p.mapAspect,
	}
	if p.layoutOK && p.layoutKey == key {
		return p.layout
	}
	p.layout = calcLayout(ms.View.Layout, ms.UI.SidebarExpanded, p.mapAspect)
	p.layoutKey = key
	p.layoutOK = true
	return p.layout
}

func calcLayout(screen layout.ScreenLayout, expanded bool, mapAspect float64) sidebarLayout {
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
