// Package sidebar 实现任务运行页右侧 RTS 风格侧栏。
package sidebar

import (
	"image/color"
	"math"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/i18n"
	md "github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	abbrMapImg "github.com/narasux/jutland/pkg/resources/images/abbrmap"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
	"github.com/narasux/jutland/pkg/utils/layout"
)

const (
	handleW = 42
	handleH = 72
)

var (
	panelBgColor   = color.RGBA{R: 8, G: 18, B: 23, A: 222}
	panelLineColor = color.RGBA{R: 85, G: 132, B: 143, A: 235}
	cardBgColor    = color.RGBA{R: 12, G: 31, B: 38, A: 214}
)

type rect struct {
	X, Y, W, H float64
}

func (r rect) contains(x, y int) bool {
	fx, fy := float64(x), float64(y)
	return fx >= r.X && fx <= r.X+r.W && fy >= r.Y && fy <= r.Y+r.H
}

type uiSnapshot struct {
	width             int
	height            int
	expanded          bool
	forceDisplayState bool
	displayDamage     bool
}

// Panel 是任务运行中的右侧战术侧栏。
type Panel struct {
	ui      *ebitenui.UI
	abbrMap *ebiten.Image
	layout  sidebarLayout
	snap    uiSnapshot
}

type sidebarLayout struct {
	Screen layout.ScreenLayout
	Panel  rect
	Handle rect
	Map    rect
}

// New 创建任务侧栏
func New(mission string, ui *ebitenui.UI) *Panel {
	missionMD := md.Get(mission)
	misLayout := layout.NewScreenLayout()
	abbrMap := ebiten.NewImage(misLayout.Height, misLayout.Height)

	bg := abbrMapImg.Background
	w, h := bg.Bounds().Dx(), bg.Bounds().Dy()
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	opts.GeoM.Scale(float64(misLayout.Height)/float64(w), float64(misLayout.Height)/float64(h))
	abbrMap.DrawImage(abbrMapImg.Background, opts)
	abbrMap.DrawImage(abbrMapImg.Get(missionMD.MapCfg.Source), opts)

	return &Panel{ui: ui, abbrMap: abbrMap}
}

// Update 更新侧栏控件状态，并处理小地图点击
func (p *Panel) Update(ms *state.MissionState) {
	if ms.Core.MissionStatus != state.MissionRunning {
		ms.UI.SidebarConsumesCursor = false
		return
	}

	p.layout = calcLayout(ms.View.Layout, ms.UI.SidebarExpanded)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		sx, sy := ebiten.CursorPosition()
		if p.layout.Handle.contains(sx, sy) {
			ms.UI.SidebarExpanded = !ms.UI.SidebarExpanded
			ms.UI.SidebarConsumesCursor = true
			p.ensureUI(ms)
			return
		}
	}

	p.ensureUI(ms)
	p.ui.Update()
	p.layout = calcLayout(ms.View.Layout, ms.UI.SidebarExpanded)
	ms.UI.SidebarConsumesCursor = p.ConsumesCursor(ms)

	if ms.UI.SidebarExpanded && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		sx, sy := ebiten.CursorPosition()
		if p.layout.Map.contains(sx, sy) {
			p.centerCameraAtMinimap(ms, sx, sy)
		}
	}
}

// Draw 绘制任务侧栏
func (p *Panel) Draw(screen *ebiten.Image, ms *state.MissionState) {
	if ms.Core.MissionStatus != state.MissionRunning {
		return
	}

	p.ensureUI(ms)
	p.layout = calcLayout(ms.View.Layout, ms.UI.SidebarExpanded)
	if ms.UI.SidebarExpanded {
		p.drawPanel(screen, ms)
	}
	p.ui.Draw(screen)
	p.drawHandleFrame(screen, ms)
	p.drawHandleArrow(screen, ms)
}

// ConsumesCursor 判断当前鼠标位置是否应由侧栏消费
func (p *Panel) ConsumesCursor(ms *state.MissionState) bool {
	if ms.Core.MissionStatus != state.MissionRunning {
		return false
	}

	ui := calcLayout(ms.View.Layout, ms.UI.SidebarExpanded)
	sx, sy := ebiten.CursorPosition()
	if ui.Handle.contains(sx, sy) {
		return true
	}
	return ms.UI.SidebarExpanded && ui.Panel.contains(sx, sy)
}

func (p *Panel) ensureUI(ms *state.MissionState) {
	snap := uiSnapshot{
		width:             ms.View.Layout.Width,
		height:            ms.View.Layout.Height,
		expanded:          ms.UI.SidebarExpanded,
		forceDisplayState: ms.UI.GameOpts.ForceDisplayState,
		displayDamage:     ms.UI.GameOpts.DisplayDamageNumber,
	}
	if p.ui != nil && p.snap == snap {
		return
	}
	p.snap = snap
	p.layout = calcLayout(ms.View.Layout, ms.UI.SidebarExpanded)
	p.buildUI(ms)
}

func (p *Panel) buildUI(ms *state.MissionState) {
	transparentButton := &widget.ButtonImage{
		Idle:    image.NewNineSliceColor(color.RGBA{}),
		Hover:   image.NewNineSliceColor(color.RGBA{}),
		Pressed: image.NewNineSliceColor(color.RGBA{}),
	}
	checkboxImage := &widget.ButtonImage{
		Idle:    image.NewNineSliceColor(color.RGBA{}),
		Hover:   image.NewNineSliceColor(color.RGBA{}),
		Pressed: image.NewNineSliceColor(color.RGBA{}),
	}

	root := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	toggle := widget.NewButton(
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(handleW, handleH),
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionEnd,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				Padding: &widget.Insets{
					Right: int(p.layout.Screen.Width) - int(p.layout.Handle.X) - handleW,
				},
			}),
		),
		widget.ButtonOpts.Image(transparentButton),
	)
	root.AddChild(toggle)

	if ms.UI.SidebarExpanded {
		p.addCheckboxButton(root, checkboxImage, 0, func() {
			ms.UI.GameOpts.ForceDisplayState = !ms.UI.GameOpts.ForceDisplayState
		})
		p.addCheckboxButton(root, checkboxImage, 1, func() {
			ms.UI.GameOpts.DisplayDamageNumber = !ms.UI.GameOpts.DisplayDamageNumber
		})
	}

	p.ui.Container = root
}

func (p *Panel) addCheckboxButton(
	root *widget.Container,
	buttonImage *widget.ButtonImage,
	index int,
	click func(),
) {
	buttonTop := int(p.settingRowsTop()) + index*44
	btn := widget.NewButton(
		widget.ButtonOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(int(p.layout.Panel.W)-32, 36),
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionEnd,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				Padding:            &widget.Insets{Right: 16, Top: buttonTop},
			}),
		),
		widget.ButtonOpts.Image(buttonImage),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			click()
		}),
	)
	root.AddChild(btn)
}

func (p *Panel) drawPanel(screen *ebiten.Image, ms *state.MissionState) {
	ui := p.layout
	vector.FillRect(
		screen,
		float32(ui.Panel.X),
		float32(ui.Panel.Y),
		float32(ui.Panel.W),
		float32(ui.Panel.H),
		panelBgColor,
		false,
	)
	vector.StrokeRect(
		screen,
		float32(ui.Panel.X),
		float32(ui.Panel.Y),
		float32(ui.Panel.W),
		float32(ui.Panel.H),
		2,
		panelLineColor,
		false,
	)

	p.drawMinimap(screen, ms)
	p.drawBattleInfo(screen, ms)
	p.drawSettings(screen, ms)
}

func (p *Panel) drawHandleFrame(screen *ebiten.Image, ms *state.MissionState) {
	ui := calcLayout(ms.View.Layout, ms.UI.SidebarExpanded)
	sx, sy := ebiten.CursorPosition()
	bgColor := color.RGBA{R: 21, G: 48, B: 56, A: 235}
	if ui.Handle.contains(sx, sy) {
		bgColor = color.RGBA{R: 27, G: 58, B: 66, A: 245}
	}
	handleW := float32(ui.Handle.W)
	if ms.UI.SidebarExpanded {
		handleW += 2
	}
	vector.FillRect(
		screen,
		float32(ui.Handle.X),
		float32(ui.Handle.Y),
		handleW,
		float32(ui.Handle.H),
		bgColor,
		false,
	)

	x, y := float32(ui.Handle.X), float32(ui.Handle.Y)
	w, h := float32(ui.Handle.W), float32(ui.Handle.H)
	edgeColor := color.RGBA{R: 128, G: 168, B: 174, A: 225}
	// 展开时右侧与面板连在一起，不画分割边。
	vector.StrokeLine(screen, x, y+1, x, y+h-1, 2, edgeColor, false)
	vector.StrokeLine(screen, x, y, x+w, y, 2, edgeColor, false)
	vector.StrokeLine(screen, x, y+h, x+w, y+h, 2, edgeColor, false)
	if !ms.UI.SidebarExpanded {
		vector.StrokeLine(screen, x+w, y+1, x+w, y+h-1, 2, edgeColor, false)
	}
}

func (p *Panel) drawHandleArrow(screen *ebiten.Image, ms *state.MissionState) {
	ui := calcLayout(ms.View.Layout, ms.UI.SidebarExpanded)
	cx, cy := float32(ui.Handle.X+ui.Handle.W/2), float32(ui.Handle.Y+ui.Handle.H/2)
	arrowW, arrowH := float32(14), float32(24)
	fillColor := color.RGBA{R: 232, G: 224, B: 198, A: 255}
	strokeColor := color.RGBA{R: 6, G: 16, B: 19, A: 230}

	var path vector.Path
	if ms.UI.SidebarExpanded {
		path.MoveTo(cx-arrowW/2, cy-arrowH/2)
		path.LineTo(cx+arrowW/2, cy)
		path.LineTo(cx-arrowW/2, cy+arrowH/2)
	} else {
		path.MoveTo(cx+arrowW/2, cy-arrowH/2)
		path.LineTo(cx-arrowW/2, cy)
		path.LineTo(cx+arrowW/2, cy+arrowH/2)
	}
	path.Close()
	strokeOpts := &vector.DrawPathOptions{AntiAlias: false}
	strokeOpts.ColorScale.ScaleWithColor(strokeColor)
	fillOpts := &vector.DrawPathOptions{AntiAlias: false}
	fillOpts.ColorScale.ScaleWithColor(fillColor)
	vector.FillPath(screen, &path, &vector.FillOptions{}, fillOpts)
	vector.StrokePath(screen, &path, &vector.StrokeOptions{Width: 3}, strokeOpts)
}

func (p *Panel) drawMinimap(screen *ebiten.Image, ms *state.MissionState) {
	ui := p.layout
	vector.FillRect(
		screen,
		float32(ui.Map.X-2),
		float32(ui.Map.Y-2),
		float32(ui.Map.W+4),
		float32(ui.Map.H+4),
		color.RGBA{R: 2, G: 8, B: 11, A: 245},
		false,
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
	for _, ship := range ms.Arena.Ships {
		img := textureImg.GetAbbrShip(ship.Tonnage, ship.BelongPlayer != ms.Player.CurPlayer)
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		ebutil.SetOptsCenterRotation(opts, img, ship.CurRotation)
		opts.GeoM.Scale(0.55, 0.55)
		x, y := p.mapToSidebar(ms, ship.CurPos.RX, ship.CurPos.RY)
		opts.GeoM.Translate(x, y)
		screen.DrawImage(img, opts)
	}
}

func (p *Panel) drawMinimapPlanes(screen *ebiten.Image, ms *state.MissionState) {
	for _, plane := range ms.Arena.Planes {
		img := textureImg.GetAbbrPlane(plane.BelongPlayer != ms.Player.CurPlayer)
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		ebutil.SetOptsCenterRotation(opts, img, plane.CurRotation)
		opts.GeoM.Scale(0.65, 0.65)
		x, y := p.mapToSidebar(ms, plane.CurPos.RX, plane.CurPos.RY)
		opts.GeoM.Translate(x, y)
		screen.DrawImage(img, opts)
	}
}

func (p *Panel) drawBattleInfo(screen *ebiten.Image, ms *state.MissionState) {
	ui := p.layout
	y := ui.Map.Y + ui.Map.H + 20
	p.drawCard(screen, ui.Panel.X+16, y, ui.Panel.W-32, 78)

	selfFleet := ms.Fleet(ms.Player.CurPlayer)
	enemyFleet := ms.Fleet(ms.Player.CurEnemy)
	bodyFont := font.LocalizedUI(font.Kai)
	p.drawText(
		screen,
		i18n.Format(i18n.MsgSidebarFunds, map[string]any{"Funds": ms.Player.CurFunds}),
		ui.Panel.X+28,
		y+14,
		18,
		bodyFont,
		colorx.White,
	)
	p.drawText(
		screen,
		i18n.Format(i18n.MsgSidebarFleets, map[string]any{"Ally": selfFleet.Total, "Enemy": enemyFleet.Total}),
		ui.Panel.X+28,
		y+40,
		16,
		bodyFont,
		colorx.Silver,
	)
}

func (p *Panel) drawSettings(screen *ebiten.Image, ms *state.MissionState) {
	p.drawCheckboxRow(screen, 0, i18n.Text(i18n.MsgSidebarShowState), ms.UI.GameOpts.ForceDisplayState)
	p.drawCheckboxRow(screen, 1, i18n.Text(i18n.MsgSidebarDamageNumbers), ms.UI.GameOpts.DisplayDamageNumber)
}

func (p *Panel) drawCheckboxRow(screen *ebiten.Image, index int, label string, checked bool) {
	x := p.layout.Panel.X + 28
	y := p.settingRowsTop() + float64(index*44)
	boxSize := float32(18)
	contentY := y + 8
	boxX, boxY := float32(x), float32(contentY)
	textColor := lo.Ternary(checked, colorx.Gold, colorx.Silver)
	borderColor := lo.Ternary(checked, colorx.Gold, panelLineColor)

	vector.StrokeRect(screen, boxX, boxY, boxSize, boxSize, 2, borderColor, false)
	if checked {
		vector.StrokeLine(screen, boxX+4, boxY+9, boxX+8, boxY+14, 3, colorx.Gold, false)
		vector.StrokeLine(screen, boxX+8, boxY+14, boxX+15, boxY+4, 3, colorx.Gold, false)
	}
	p.drawText(screen, label, x+34, contentY, 18, font.LocalizedUI(font.Kai), textColor)
}

func (p *Panel) drawCard(screen *ebiten.Image, x, y, w, h float64) {
	vector.FillRect(screen, float32(x), float32(y), float32(w), float32(h), cardBgColor, false)
	vector.StrokeRect(
		screen,
		float32(x),
		float32(y),
		float32(w),
		float32(h),
		1,
		color.RGBA{R: 59, G: 97, B: 106, A: 210},
		false,
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

func (p *Panel) settingRowsTop() float64 {
	return p.layout.Map.Y + p.layout.Map.H + 116
}

func calcLayout(screen layout.ScreenLayout, expanded bool) sidebarLayout {
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
	return sidebarLayout{
		Screen: screen,
		Panel:  rect{X: panelX, Y: 0, W: panelW, H: float64(screen.Height)},
		Handle: rect{X: handleX, Y: (float64(screen.Height) - handleH) / 2, W: handleW, H: handleH},
		Map:    rect{X: panelX + 16, Y: 28, W: mapSize, H: mapSize},
	}
}
