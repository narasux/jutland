package game

import (
	"fmt"
	"image"
	"image/color"
	"math"

	uiImage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/pkg/browser"

	objRef "github.com/narasux/jutland/pkg/mission/object/reference"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/resources/font"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
	planeImg "github.com/narasux/jutland/pkg/resources/images/plane"
	shipImg "github.com/narasux/jutland/pkg/resources/images/ship"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/layout"
)

type shipClassFilter string

const (
	shipClassAll        shipClassFilter = "all"
	shipClassCarrier    shipClassFilter = "carrier"
	shipClassBattleship shipClassFilter = "battleship"
	shipClassCruiser    shipClassFilter = "cruiser"
	shipClassDestroyer  shipClassFilter = "destroyer"
	shipClassFrigate    shipClassFilter = "frigate"
	shipClassTorpedo    shipClassFilter = "torpedo_boat"
	shipClassAuxiliary  shipClassFilter = "auxiliary"
	shipClassSpecial    shipClassFilter = "special"
)

func (f shipClassFilter) display() string {
	switch f {
	case shipClassCarrier:
		return "航空母舰"
	case shipClassBattleship:
		return "战列舰"
	case shipClassCruiser:
		return "巡洋舰"
	case shipClassDestroyer:
		return "驱逐舰"
	case shipClassFrigate:
		return "护卫舰"
	case shipClassTorpedo:
		return "鱼雷艇"
	case shipClassAuxiliary:
		return "辅助舰"
	case shipClassSpecial:
		return "特殊"
	default:
		return "全部"
	}
}

var shipClassFilters = []shipClassFilter{
	shipClassAll, shipClassCarrier, shipClassBattleship, shipClassCruiser,
	shipClassDestroyer, shipClassFrigate, shipClassTorpedo, shipClassAuxiliary, shipClassSpecial,
}

type planeTypeFilter string

const (
	planeTypeAll     planeTypeFilter = "all"
	planeTypeFighter planeTypeFilter = "fighter"
	planeTypeDive    planeTypeFilter = "dive_bomber"
	planeTypeTorpedo planeTypeFilter = "torpedo_bomber"
)

func (f planeTypeFilter) display() string {
	switch f {
	case planeTypeFighter:
		return "战斗机"
	case planeTypeDive:
		return "俯冲轰炸机"
	case planeTypeTorpedo:
		return "鱼雷轰炸机"
	default:
		return "全部"
	}
}

var planeTypeFilters = []planeTypeFilter{planeTypeAll, planeTypeFighter, planeTypeDive, planeTypeTorpedo}

type collectionUILayout struct {
	Blueprint     image.Rectangle
	Toolbar       image.Rectangle
	PlaneToolbar  image.Rectangle
	ShipArchive   collectionCard
	ShipCombat    collectionCard
	ShipSource    collectionCard
	PlaneViewport image.Rectangle
	VSlider       image.Rectangle
	HSlider       image.Rectangle
}

type collectionMetrics struct {
	Scale       float64
	ToolbarFont float64
	CardTitle   float64
	ShipName    float64
	Body        float64
	History     float64
	RadarLabel  float64
	Tooltip     float64
	PlaneTitle  float64
	PlaneCardW  int
	PlaneCardH  int
}

func calculateCollectionMetrics(width, height int) collectionMetrics {
	scale := clampFloat(min(float64(width)/1600, float64(height)/900), 1, 1.30)
	return collectionMetrics{
		Scale: scale, ToolbarFont: 20 * scale, CardTitle: 22 * scale,
		ShipName: 30 * scale, Body: 18 * scale, History: 18 * scale,
		RadarLabel: 16 * scale, Tooltip: 17 * scale, PlaneTitle: 27 * scale,
		PlaneCardW: int(math.Round(340 * scale)), PlaneCardH: int(math.Round(850 * scale)),
	}
}

type collectionLink struct {
	Area clickableArea
	URL  string
}

type collectionNameOption struct {
	Name  string
	Label string
}

type collectionDropdown int

const (
	collectionDropdownNone collectionDropdown = iota
	collectionDropdownNation
	collectionDropdownType
	collectionDropdownName
)

// CollectionUI 管理图鉴筛选、滚动和绘制。
type CollectionUI struct {
	drawer *Drawer
	root   widget.Containerer

	width   int
	height  int
	layout  collectionUILayout
	metrics collectionMetrics

	category collectionCategory

	shipNation objUnit.Nation
	shipClass  shipClassFilter
	curShip    string

	planeNation objUnit.Nation
	planeType   planeTypeFilter

	planeScrollX float64
	planeScrollY float64
	hSlider      *widget.Slider
	vSlider      *widget.Slider

	shipScales  abilityScales
	planeScales abilityScales
	radarHits   []radarHitArea
	links       []collectionLink

	planeCanvas    *ebiten.Image
	comboButtons   []*widget.ListComboButton
	tabButtons     map[collectionCategory]*widget.Button
	nationCombo    *widget.ListComboButton
	typeCombo      *widget.ListComboButton
	nameCombo      *widget.ListComboButton
	previousButton *widget.Button
	nextButton     *widget.Button
	pendingRebuild bool
	openDropdown   collectionDropdown
	dropdownOffset int
}

func NewCollectionUI(drawer *Drawer) *CollectionUI {
	ui := &CollectionUI{
		drawer:       drawer,
		category:     collectionCategoryShip,
		shipNation:   objUnit.NationAll,
		shipClass:    shipClassAll,
		curShip:      "lowa",
		planeNation:  objUnit.NationAll,
		planeType:    planeTypeAll,
		shipScales:   calculateShipAbilityScales(),
		planeScales:  calculatePlaneAbilityScales(),
		planeScrollX: 0,
		planeScrollY: 0,
	}
	screen := layout.NewScreenLayout()
	ui.resize(screen.Width, screen.Height)
	return ui
}

func (c *CollectionUI) Update() {
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		c.toggleCategory()
	}

	if c.category == collectionCategoryShip {
		_, wheelY := ebiten.Wheel()
		cursor := image.Pt(ebiten.CursorPosition())
		wheelNavigates := !cursor.In(c.layout.Toolbar) && !c.comboExpanded()
		offset := shipNavigationOffset(
			inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft), inpututil.IsKeyJustPressed(ebiten.KeyArrowUp),
			inpututil.IsKeyJustPressed(ebiten.KeyArrowRight), inpututil.IsKeyJustPressed(ebiten.KeyArrowDown),
			wheelY, wheelNavigates,
		)
		if offset != 0 {
			c.moveShip(offset)
		}
	} else {
		c.updatePlaneWheel()
	}

	c.updateManualToolbarInput()
	// 工具栏下拉框由原生输入层处理，避免全屏模式下 EbitenUI 临时窗口丢失点击。
	for _, combo := range c.comboButtons {
		if combo.ContentVisible() {
			combo.SetContentVisible(false)
		}
	}

	c.applyPendingRebuild()

	if isMouseButtonLeftJustPressed() {
		for _, link := range c.links {
			if isHoverArea(link.Area) {
				_ = browser.OpenURL(link.URL)
				break
			}
		}
	}
}

func shipNavigationOffset(left, up, right, down bool, wheelY float64, wheelAllowed bool) int {
	if left || up || (wheelAllowed && wheelY > 0) {
		return -1
	}
	if right || down || (wheelAllowed && wheelY < 0) {
		return 1
	}
	return 0
}

func (c *CollectionUI) applyPendingRebuild() {
	if !c.pendingRebuild {
		return
	}
	c.pendingRebuild = false
	c.buildUI()
}

// Container 返回由游戏共享 EbitenUI 主实例承载的图鉴容器。
func (c *CollectionUI) Container() widget.Containerer { return c.root }

func (c *CollectionUI) comboExpanded() bool {
	return c.openDropdown != collectionDropdownNone
}

func (c *CollectionUI) updateManualToolbarInput() {
	point := image.Pt(ebiten.CursorPosition())
	if c.openDropdown != collectionDropdownNone {
		_, wheelY := ebiten.Wheel()
		if wheelY != 0 && point.In(c.dropdownListRect(c.openDropdown)) {
			if wheelY > 0 {
				c.dropdownOffset--
			} else {
				c.dropdownOffset++
			}
			c.clampDropdownOffset()
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		c.handleToolbarClick(point)
	}
}

func (c *CollectionUI) handleToolbarClick(point image.Point) {
	if c.openDropdown != collectionDropdownNone && point.In(c.dropdownListRect(c.openDropdown)) {
		row := (point.Y - c.dropdownListRect(c.openDropdown).Min.Y) / c.dropdownRowHeight()
		entries := c.dropdownEntries(c.openDropdown)
		index := c.dropdownOffset + row
		if index >= 0 && index < len(entries) {
			c.selectDropdownEntry(c.openDropdown, entries[index])
		}
		return
	}

	if button := c.tabButtons[collectionCategoryShip]; button != nil && point.In(button.GetWidget().Rect) {
		c.category = collectionCategoryShip
		c.openDropdown = collectionDropdownNone
		c.pendingRebuild = true
		return
	}
	if button := c.tabButtons[collectionCategoryPlane]; button != nil && point.In(button.GetWidget().Rect) {
		c.category = collectionCategoryPlane
		c.openDropdown = collectionDropdownNone
		c.pendingRebuild = true
		return
	}
	if c.nationCombo != nil && point.In(c.nationCombo.GetWidget().Rect) {
		c.toggleDropdown(collectionDropdownNation)
		return
	}
	if c.typeCombo != nil && point.In(c.typeCombo.GetWidget().Rect) {
		c.toggleDropdown(collectionDropdownType)
		return
	}
	if c.nameCombo != nil && point.In(c.nameCombo.GetWidget().Rect) {
		c.toggleDropdown(collectionDropdownName)
		return
	}
	if c.previousButton != nil && point.In(c.previousButton.GetWidget().Rect) {
		c.openDropdown = collectionDropdownNone
		c.moveShip(-1)
		return
	}
	if c.nextButton != nil && point.In(c.nextButton.GetWidget().Rect) {
		c.openDropdown = collectionDropdownNone
		c.moveShip(1)
		return
	}
	c.openDropdown = collectionDropdownNone
}

func (c *CollectionUI) toggleDropdown(dropdown collectionDropdown) {
	if c.openDropdown == dropdown {
		c.openDropdown = collectionDropdownNone
		return
	}
	c.openDropdown = dropdown
	c.dropdownOffset = 0
	c.clampDropdownOffset()
}

func (c *CollectionUI) dropdownButton(dropdown collectionDropdown) *widget.ListComboButton {
	switch dropdown {
	case collectionDropdownNation:
		return c.nationCombo
	case collectionDropdownType:
		return c.typeCombo
	case collectionDropdownName:
		return c.nameCombo
	default:
		return nil
	}
}

func (c *CollectionUI) dropdownEntries(dropdown collectionDropdown) []any {
	switch dropdown {
	case collectionDropdownNation:
		entries := make([]any, 0, len(objUnit.AvailableNations()))
		for _, nation := range objUnit.AvailableNations() {
			entries = append(entries, nation)
		}
		return entries
	case collectionDropdownType:
		if c.category == collectionCategoryPlane {
			entries := make([]any, 0, len(planeTypeFilters))
			for _, planeType := range planeTypeFilters {
				entries = append(entries, planeType)
			}
			return entries
		}
		entries := make([]any, 0, len(shipClassFilters))
		for _, class := range shipClassFilters {
			entries = append(entries, class)
		}
		return entries
	case collectionDropdownName:
		ships := c.filteredShips()
		entries := make([]any, 0, len(ships))
		for idx, ship := range ships {
			entries = append(entries, collectionNameOption{
				Name: ship.Name, Label: fmt.Sprintf("%s  %02d/%02d", ship.DisplayName, idx+1, len(ships)),
			})
		}
		return entries
	default:
		return nil
	}
}

func (c *CollectionUI) dropdownEntryLabel(entry any) string {
	switch value := entry.(type) {
	case objUnit.Nation:
		return value.ToDisplay()
	case shipClassFilter:
		return value.display()
	case planeTypeFilter:
		return value.display()
	case collectionNameOption:
		return value.Label
	default:
		return ""
	}
}

func (c *CollectionUI) selectDropdownEntry(dropdown collectionDropdown, entry any) {
	switch dropdown {
	case collectionDropdownNation:
		if c.category == collectionCategoryPlane {
			c.setPlaneNation(entry.(objUnit.Nation))
		} else {
			c.setShipNation(entry.(objUnit.Nation))
		}
	case collectionDropdownType:
		if c.category == collectionCategoryPlane {
			c.setPlaneType(entry.(planeTypeFilter))
		} else {
			c.setShipClass(entry.(shipClassFilter))
		}
	case collectionDropdownName:
		c.setCurrentShip(entry.(collectionNameOption).Name)
	}
	c.openDropdown = collectionDropdownNone
}

func (c *CollectionUI) dropdownRowHeight() int {
	return max(32, int(math.Round(36*c.metrics.Scale)))
}

func (c *CollectionUI) dropdownVisibleRows(dropdown collectionDropdown) int {
	return min(7, len(c.dropdownEntries(dropdown)))
}

func (c *CollectionUI) clampDropdownOffset() {
	maxOffset := max(0, len(c.dropdownEntries(c.openDropdown))-c.dropdownVisibleRows(c.openDropdown))
	c.dropdownOffset = max(0, min(c.dropdownOffset, maxOffset))
}

func (c *CollectionUI) dropdownListRect(dropdown collectionDropdown) image.Rectangle {
	button := c.dropdownButton(dropdown)
	if button == nil {
		return image.Rectangle{}
	}
	buttonRect := button.GetWidget().Rect
	width := buttonRect.Dx()
	if dropdown == collectionDropdownName {
		width = max(width, int(math.Round(280*c.metrics.Scale)))
	}
	height := c.dropdownVisibleRows(dropdown) * c.dropdownRowHeight()
	return image.Rect(buttonRect.Min.X, buttonRect.Max.Y+2, buttonRect.Min.X+width, buttonRect.Max.Y+2+height)
}

func (c *CollectionUI) DrawOverlay(screen *ebiten.Image) {
	if c.openDropdown == collectionDropdownNone {
		return
	}
	rect := c.dropdownListRect(c.openDropdown)
	vector.FillRect(screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()), color.RGBA{20, 19, 17, 252}, false)
	vector.StrokeRect(screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()), 1.5, colorx.Gold, false)
	entries := c.dropdownEntries(c.openDropdown)
	cursor := image.Pt(ebiten.CursorPosition())
	rowHeight := c.dropdownRowHeight()
	for row := 0; row < c.dropdownVisibleRows(c.openDropdown); row++ {
		index := c.dropdownOffset + row
		if index >= len(entries) {
			break
		}
		rowRect := image.Rect(rect.Min.X, rect.Min.Y+row*rowHeight, rect.Max.X, rect.Min.Y+(row+1)*rowHeight)
		if cursor.In(rowRect) {
			vector.FillRect(screen, float32(rowRect.Min.X), float32(rowRect.Min.Y), float32(rowRect.Dx()), float32(rowRect.Dy()), color.RGBA{80, 69, 52, 255}, false)
		}
		c.drawer.drawText(
			screen, c.dropdownEntryLabel(entries[index]), float64(rowRect.Min.X)+12*c.metrics.Scale,
			float64(rowRect.Min.Y)+6*c.metrics.Scale, c.metrics.ToolbarFont, font.Kai, colorx.White,
		)
	}
}

func (c *CollectionUI) Draw(screen *ebiten.Image) {
	if c.width != screen.Bounds().Dx() || c.height != screen.Bounds().Dy() {
		c.resize(screen.Bounds().Dx(), screen.Bounds().Dy())
	}
	c.radarHits = c.radarHits[:0]
	c.links = c.links[:0]

	if c.category == collectionCategoryShip {
		c.drawShip(screen)
	} else {
		c.drawPlanes(screen)
	}
	c.drawer.drawRadarTooltip(screen, hoveredRadarArea(c.radarHits), c.metrics.Tooltip)
}

func (c *CollectionUI) resize(width, height int) {
	c.width, c.height = width, height
	c.metrics = calculateCollectionMetrics(width, height)
	c.layout = calculateCollectionUILayout(width, height)
	c.planeCanvas = ebiten.NewImage(c.layout.PlaneViewport.Dx(), c.layout.PlaneViewport.Dy())
	c.buildUI()
}

func calculateCollectionUILayout(width, height int) collectionUILayout {
	metrics := calculateCollectionMetrics(width, height)
	marginX := max(34, int(float64(width)*0.06))
	blueprintY := 42
	blueprintH := int(float64(height) * 0.48)
	blueprint := image.Rect(marginX, blueprintY, width-marginX, blueprintY+blueprintH)
	toolbarH := int(math.Round(48 * metrics.Scale))
	toolbar := image.Rect(marginX, blueprint.Max.Y+12, width-marginX, blueprint.Max.Y+12+toolbarH)
	detailsY := toolbar.Max.Y + 12
	detailsH := max(150, height-detailsY-38)
	detailsW := blueprint.Dx()
	gap := 18
	archiveW := int(float64(detailsW) * 0.20)
	combatW := int(float64(detailsW) * 0.50)
	sourceW := detailsW - archiveW - combatW - 2*gap

	planeToolbar := image.Rect(marginX, 26, width-marginX, 26+toolbarH)
	vSlider := image.Rect(width-marginX-16, planeToolbar.Max.Y+14, width-marginX, height-54)
	hSlider := image.Rect(marginX, height-40, vSlider.Min.X-12, height-24)
	planeViewport := image.Rect(marginX, planeToolbar.Max.Y+14, vSlider.Min.X-12, hSlider.Min.Y-12)

	return collectionUILayout{
		Blueprint:    blueprint,
		Toolbar:      toolbar,
		PlaneToolbar: planeToolbar,
		ShipArchive: collectionCard{
			X: float64(marginX), Y: float64(detailsY), W: float64(archiveW), H: float64(detailsH),
		},
		ShipCombat: collectionCard{
			X: float64(marginX + archiveW + gap), Y: float64(detailsY), W: float64(combatW), H: float64(detailsH),
		},
		ShipSource: collectionCard{
			X: float64(marginX + archiveW + combatW + 2*gap), Y: float64(detailsY), W: float64(sourceW), H: float64(detailsH),
		},
		PlaneViewport: planeViewport,
		VSlider:       vSlider,
		HSlider:       hSlider,
	}
}

func (c *CollectionUI) buildUI() {
	c.comboButtons = c.comboButtons[:0]
	c.tabButtons = map[collectionCategory]*widget.Button{}
	c.nationCombo, c.typeCombo, c.nameCombo = nil, nil, nil
	c.previousButton, c.nextButton = nil, nil
	root := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	toolbarRect := c.layout.Toolbar
	if c.category == collectionCategoryPlane {
		toolbarRect = c.layout.PlaneToolbar
	}
	toolbar := c.buildToolbar(toolbarRect)
	root.AddChild(toolbar)
	toolbar.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding:            &widget.Insets{Left: toolbarRect.Min.X, Top: toolbarRect.Min.Y},
	}

	if c.category == collectionCategoryPlane {
		c.vSlider = c.newScrollSlider(widget.DirectionVertical, c.planeScrollY, func(value float64) {
			c.planeScrollY = value
		})
		c.vSlider.GetWidget().MinWidth = c.layout.VSlider.Dx()
		c.vSlider.GetWidget().MinHeight = c.layout.VSlider.Dy()
		c.vSlider.GetWidget().LayoutData = widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionStart,
			Padding:            &widget.Insets{Left: c.layout.VSlider.Min.X, Top: c.layout.VSlider.Min.Y},
		}
		root.AddChild(c.vSlider)

		c.hSlider = c.newScrollSlider(widget.DirectionHorizontal, c.planeScrollX, func(value float64) {
			c.planeScrollX = value
		})
		c.hSlider.GetWidget().MinWidth = c.layout.HSlider.Dx()
		c.hSlider.GetWidget().MinHeight = c.layout.HSlider.Dy()
		c.hSlider.GetWidget().LayoutData = widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionStart,
			VerticalPosition:   widget.AnchorLayoutPositionStart,
			Padding:            &widget.Insets{Left: c.layout.HSlider.Min.X, Top: c.layout.HSlider.Min.Y},
		}
		root.AddChild(c.hSlider)
	} else {
		c.hSlider, c.vSlider = nil, nil
	}

	c.root = root
}

func (c *CollectionUI) buildToolbar(rect image.Rectangle) *widget.Container {
	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(uiImage.NewNineSliceColor(color.RGBA{18, 18, 18, 215})),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(rect.Dx(), rect.Dy())),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(int(math.Round(9*c.metrics.Scale))),
			widget.RowLayoutOpts.Padding(&widget.Insets{
				Left: int(math.Round(12 * c.metrics.Scale)), Right: int(math.Round(12 * c.metrics.Scale)),
				Top: int(math.Round(7 * c.metrics.Scale)), Bottom: int(math.Round(7 * c.metrics.Scale)),
			}),
		)),
	)
	shipTab := c.newTabButton("舰船", collectionCategoryShip)
	planeTab := c.newTabButton("飞机", collectionCategoryPlane)
	c.tabButtons[collectionCategoryShip] = shipTab
	c.tabButtons[collectionCategoryPlane] = planeTab
	panel.AddChild(shipTab)
	panel.AddChild(planeTab)

	if c.category == collectionCategoryShip {
		c.nationCombo = c.newNationCombo("国籍", c.shipNation, func(nation objUnit.Nation) {
			c.setShipNation(nation)
		})
		c.typeCombo = c.newShipClassCombo()
		c.nameCombo = c.newShipNameCombo()
		c.previousButton = c.newButton("‹", 42, false, func() { c.moveShip(-1) })
		c.nextButton = c.newButton("›", 42, false, func() { c.moveShip(1) })
		panel.AddChild(c.nationCombo)
		panel.AddChild(c.typeCombo)
		panel.AddChild(c.nameCombo)
		panel.AddChild(c.previousButton)
		panel.AddChild(c.nextButton)
	} else {
		c.nationCombo = c.newNationCombo("国籍", c.planeNation, func(nation objUnit.Nation) {
			c.setPlaneNation(nation)
		})
		c.typeCombo = c.newPlaneTypeCombo()
		panel.AddChild(c.nationCombo)
		panel.AddChild(c.typeCombo)
		panel.AddChild(c.newLabel(fmt.Sprintf("共 %d 架飞机", len(c.filteredPlanes())), c.metrics.ToolbarFont))
	}
	return panel
}

func (c *CollectionUI) newTabButton(label string, category collectionCategory) *widget.Button {
	return c.newButton(label, 72, c.category == category, func() {
		if c.category == category {
			return
		}
		c.category = category
		c.pendingRebuild = true
	})
}

func (c *CollectionUI) newButton(label string, width int, selected bool, clicked func()) *widget.Button {
	face := collectionFace(c.metrics.ToolbarFont, font.Kai)
	idle := color.RGBA{30, 28, 25, 230}
	hover := color.RGBA{80, 69, 52, 240}
	textIdle := colorx.White
	if selected {
		idle = color.RGBA{190, 157, 92, 255}
		hover = idle
		textIdle = colorx.Black
	}
	scale := c.metrics.Scale
	return widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(
			int(math.Round(float64(width)*scale)), int(math.Round(34*scale)),
		)),
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle: uiImage.NewNineSliceColor(idle), Hover: uiImage.NewNineSliceColor(hover),
			Pressed: uiImage.NewNineSliceColor(color.RGBA{45, 40, 34, 255}),
		}),
		widget.ButtonOpts.Text(label, face, &widget.ButtonTextColor{
			Idle: textIdle, Hover: colorx.Gold, Pressed: colorx.White,
		}),
		widget.ButtonOpts.TextPadding(&widget.Insets{
			Left: int(math.Round(12 * scale)), Right: int(math.Round(12 * scale)),
			Top: int(math.Round(6 * scale)), Bottom: int(math.Round(6 * scale)),
		}),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { clicked() }),
	)
}

func (c *CollectionUI) newLabel(label string, size float64) *widget.Label {
	return widget.NewLabel(widget.LabelOpts.Text(
		label, collectionFace(size, font.Kai),
		&widget.LabelColor{Idle: color.RGBA{214, 201, 178, 255}, Disabled: color.RGBA{214, 201, 178, 255}},
	))
}

func (c *CollectionUI) newNationCombo(
	prefix string, selected objUnit.Nation, changed func(objUnit.Nation),
) *widget.ListComboButton {
	entries := make([]any, 0, len(objUnit.AvailableNations()))
	for _, nation := range objUnit.AvailableNations() {
		entries = append(entries, nation)
	}
	return c.newCombo(
		entries, selected, 132,
		func(entry any) string { return prefix + "  " + entry.(objUnit.Nation).ToDisplay() + "⌄" },
		func(entry any) string { return entry.(objUnit.Nation).ToDisplay() },
		func(entry any) { changed(entry.(objUnit.Nation)) },
	)
}

func (c *CollectionUI) newShipClassCombo() *widget.ListComboButton {
	entries := make([]any, 0, len(shipClassFilters))
	for _, filter := range shipClassFilters {
		entries = append(entries, filter)
	}
	return c.newCombo(
		entries, c.shipClass, 148,
		func(entry any) string { return "舰种  " + entry.(shipClassFilter).display() + "⌄" },
		func(entry any) string { return entry.(shipClassFilter).display() },
		func(entry any) {
			c.setShipClass(entry.(shipClassFilter))
		},
	)
}

func (c *CollectionUI) newPlaneTypeCombo() *widget.ListComboButton {
	entries := make([]any, 0, len(planeTypeFilters))
	for _, filter := range planeTypeFilters {
		entries = append(entries, filter)
	}
	return c.newCombo(
		entries, c.planeType, 160,
		func(entry any) string { return "机种  " + entry.(planeTypeFilter).display() + "⌄" },
		func(entry any) string { return entry.(planeTypeFilter).display() },
		func(entry any) {
			c.setPlaneType(entry.(planeTypeFilter))
		},
	)
}

func (c *CollectionUI) newShipNameCombo() *widget.ListComboButton {
	ships := c.filteredShips()
	entries := make([]any, 0, len(ships))
	selected := collectionNameOption{}
	for idx, ship := range ships {
		option := collectionNameOption{
			Name: ship.Name, Label: fmt.Sprintf("%s  %02d/%02d", ship.DisplayName, idx+1, len(ships)),
		}
		entries = append(entries, option)
		if ship.Name == c.curShip {
			selected = option
		}
	}
	if len(entries) == 0 {
		entries = append(entries, collectionNameOption{Label: "无匹配舰船"})
		selected = entries[0].(collectionNameOption)
	}
	return c.newCombo(
		entries, selected, 210,
		func(entry any) string { return "舰名  " + entry.(collectionNameOption).Label + "⌄" },
		func(entry any) string { return entry.(collectionNameOption).Label },
		func(entry any) {
			option := entry.(collectionNameOption)
			if option.Name != "" {
				c.setCurrentShip(option.Name)
			}
		},
	)
}

func (c *CollectionUI) setShipNation(nation objUnit.Nation) {
	c.shipNation = nation
	c.ensureCurrentShip()
	c.pendingRebuild = true
}

func (c *CollectionUI) setShipClass(class shipClassFilter) {
	c.shipClass = class
	c.ensureCurrentShip()
	c.pendingRebuild = true
}

func (c *CollectionUI) setCurrentShip(name string) {
	c.curShip = name
	c.pendingRebuild = true
}

func (c *CollectionUI) setPlaneNation(nation objUnit.Nation) {
	c.planeNation = nation
	c.resetPlaneScroll()
	c.pendingRebuild = true
}

func (c *CollectionUI) setPlaneType(planeType planeTypeFilter) {
	c.planeType = planeType
	c.resetPlaneScroll()
	c.pendingRebuild = true
}

func (c *CollectionUI) newCombo(
	entries []any,
	selected any,
	width int,
	buttonLabel func(any) string,
	entryLabel func(any) string,
	selectedHandler func(any),
) *widget.ListComboButton {
	face := collectionFace(c.metrics.ToolbarFont, font.Kai)
	scale := c.metrics.Scale
	buttonImage := &widget.ButtonImage{
		Idle:    uiImage.NewNineSliceColor(color.RGBA{30, 28, 25, 235}),
		Hover:   uiImage.NewNineSliceColor(color.RGBA{80, 69, 52, 245}),
		Pressed: uiImage.NewNineSliceColor(color.RGBA{45, 40, 34, 255}),
	}
	textColor := &widget.ButtonTextColor{Idle: colorx.White, Hover: colorx.Gold, Pressed: colorx.White}
	sliderTrack := &widget.SliderTrackImage{
		Idle:  uiImage.NewNineSliceColor(color.RGBA{45, 40, 34, 255}),
		Hover: uiImage.NewNineSliceColor(color.RGBA{70, 61, 48, 255}),
	}
	sliderHandle := &widget.ButtonImage{
		Idle:    uiImage.NewNineSliceColor(color.RGBA{180, 158, 116, 255}),
		Hover:   uiImage.NewNineSliceColor(colorx.Gold),
		Pressed: uiImage.NewNineSliceColor(color.RGBA{145, 123, 83, 255}),
	}
	minHandleSize := int(math.Round(16 * scale))
	listImage := &widget.ScrollContainerImage{
		Idle: uiImage.NewNineSliceColor(color.RGBA{20, 19, 17, 250}),
		Mask: uiImage.NewNineSliceColor(color.White),
	}
	combo := widget.NewListComboButton(
		widget.ListComboButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(
			int(math.Round(float64(width)*scale)), int(math.Round(34*scale)),
		)),
		widget.ListComboButtonOpts.Entries(entries),
		widget.ListComboButtonOpts.InitialEntry(selected),
		widget.ListComboButtonOpts.ButtonParams(&widget.ButtonParams{
			Image: buttonImage,
			TextPadding: &widget.Insets{
				Left: int(math.Round(12 * scale)), Right: int(math.Round(12 * scale)),
				Top: int(math.Round(6 * scale)), Bottom: int(math.Round(6 * scale)),
			},
		}),
		widget.ListComboButtonOpts.Text(face, nil, textColor),
		widget.ListComboButtonOpts.ListParams(&widget.ListParams{
			ScrollContainerImage: listImage,
			Slider: &widget.SliderParams{
				TrackImage: sliderTrack, HandleImage: sliderHandle, MinHandleSize: &minHandleSize,
			},
			EntryFace: face,
			EntryColor: &widget.ListEntryColor{
				Unselected:                 colorx.White,
				Selected:                   colorx.Gold,
				DisabledUnselected:         colorx.Gray,
				DisabledSelected:           colorx.Gray,
				SelectingBackground:        color.RGBA{80, 69, 52, 255},
				SelectedBackground:         color.RGBA{45, 40, 34, 255},
				FocusedBackground:          color.RGBA{65, 55, 43, 255},
				SelectingFocusedBackground: color.RGBA{90, 76, 57, 255},
				SelectedFocusedBackground:  color.RGBA{65, 55, 43, 255},
				DisabledSelectedBackground: color.RGBA{30, 28, 25, 255},
			},
			EntryTextPadding: &widget.Insets{
				Left: int(math.Round(12 * scale)), Right: int(math.Round(12 * scale)),
				Top: int(math.Round(6 * scale)), Bottom: int(math.Round(6 * scale)),
			},
		}),
		widget.ListComboButtonOpts.EntryLabelFunc(buttonLabel, entryLabel),
		widget.ListComboButtonOpts.EntrySelectedHandler(func(args *widget.ListComboButtonEntrySelectedEventArgs) {
			selectedHandler(args.Entry)
		}),
		widget.ListComboButtonOpts.MaxContentHeight(int(math.Round(280*scale))),
	)
	c.comboButtons = append(c.comboButtons, combo)
	return combo
}

func (c *CollectionUI) newScrollSlider(
	direction widget.Direction, value float64, changed func(float64),
) *widget.Slider {
	return widget.NewSlider(
		widget.SliderOpts.Orientation(direction),
		widget.SliderOpts.MinMax(0, 1000),
		widget.SliderOpts.InitialCurrent(int(math.Round(value*1000))),
		widget.SliderOpts.Images(
			&widget.SliderTrackImage{
				Idle:  uiImage.NewNineSliceColor(color.RGBA{45, 40, 34, 220}),
				Hover: uiImage.NewNineSliceColor(color.RGBA{70, 61, 48, 240}),
			},
			&widget.ButtonImage{
				Idle:    uiImage.NewNineSliceColor(color.RGBA{180, 158, 116, 255}),
				Hover:   uiImage.NewNineSliceColor(colorx.Gold),
				Pressed: uiImage.NewNineSliceColor(color.RGBA{145, 123, 83, 255}),
			},
		),
		widget.SliderOpts.MinHandleSize(int(math.Round(36*c.metrics.Scale))),
		widget.SliderOpts.ChangedHandler(func(args *widget.SliderChangedEventArgs) {
			changed(float64(args.Current) / 1000)
		}),
	)
}

func collectionFace(size float64, source *text.GoTextFaceSource) *text.Face {
	face := text.Face(&text.GoTextFace{Source: source, Size: size})
	return &face
}

func (c *CollectionUI) toggleCategory() {
	if c.category == collectionCategoryShip {
		c.category = collectionCategoryPlane
	} else {
		c.category = collectionCategoryShip
	}
	c.pendingRebuild = true
}

func (c *CollectionUI) resetPlaneScroll() {
	c.planeScrollX, c.planeScrollY = 0, 0
}

func (c *CollectionUI) updatePlaneWheel() {
	if c.comboExpanded() {
		return
	}
	cursorX, cursorY := ebiten.CursorPosition()
	if !image.Pt(cursorX, cursorY).In(c.layout.PlaneViewport) {
		return
	}
	wheelX, wheelY := ebiten.Wheel()
	if wheelX == 0 && wheelY == 0 {
		return
	}
	if ebiten.IsKeyPressed(ebiten.KeyShift) || math.Abs(wheelX) > math.Abs(wheelY) {
		c.planeScrollX = clampFloat(c.planeScrollX-(wheelX+wheelY)*0.07, 0, 1)
		if c.hSlider != nil {
			c.hSlider.Current = int(math.Round(c.planeScrollX * 1000))
		}
		return
	}
	c.planeScrollY = clampFloat(c.planeScrollY-wheelY*0.09, 0, 1)
	if c.vSlider != nil {
		c.vSlider.Current = int(math.Round(c.planeScrollY * 1000))
	}
}

func clampFloat(value, minimum, maximum float64) float64 {
	return min(maximum, max(minimum, value))
}

func (c *CollectionUI) filteredShips() []*objUnit.BattleShip {
	ships := make([]*objUnit.BattleShip, 0, len(objUnit.GetAllShipNames()))
	for _, name := range objUnit.GetAllShipNames() {
		ship := objUnit.ShipMap[name]
		if ship == nil || (c.shipNation != objUnit.NationAll && ship.Nation != c.shipNation) ||
			!matchShipClass(ship.Type, c.shipClass) {
			continue
		}
		ships = append(ships, ship)
	}
	return ships
}

func matchShipClass(shipType objUnit.ShipType, filter shipClassFilter) bool {
	switch filter {
	case shipClassAll:
		return true
	case shipClassCarrier:
		return shipType == objUnit.ShipTypeAircraftCarrier
	case shipClassBattleship:
		return shipType == objUnit.ShipTypeBattleShip
	case shipClassCruiser:
		return shipType == objUnit.ShipTypeCruiser
	case shipClassDestroyer:
		return shipType == objUnit.ShipTypeDestroyer
	case shipClassFrigate:
		return shipType == objUnit.ShipTypeFrigate
	case shipClassTorpedo:
		return shipType == objUnit.ShipTypeTorpedoBoat
	case shipClassAuxiliary:
		return shipType == objUnit.ShipTypeCargo || shipType == objUnit.ShipTypeHospital
	case shipClassSpecial:
		return shipType == objUnit.ShipTypeDefault
	default:
		return false
	}
}

func (c *CollectionUI) filteredPlanes() []*objUnit.Plane {
	planes := make([]*objUnit.Plane, 0, len(objUnit.GetAllPlaneNames()))
	for _, name := range objUnit.GetAllPlaneNames() {
		plane := objUnit.PlaneMap[name]
		if plane == nil || (c.planeNation != objUnit.NationAll && plane.Nation != c.planeNation) ||
			!matchPlaneType(plane.Type, c.planeType) {
			continue
		}
		planes = append(planes, plane)
	}
	return planes
}

func matchPlaneType(planeType objUnit.PlaneType, filter planeTypeFilter) bool {
	return filter == planeTypeAll || string(planeType) == string(filter)
}

func (c *CollectionUI) ensureCurrentShip() {
	ships := c.filteredShips()
	for _, ship := range ships {
		if ship.Name == c.curShip {
			return
		}
	}
	if len(ships) > 0 {
		c.curShip = ships[0].Name
	} else {
		c.curShip = ""
	}
}

func (c *CollectionUI) moveShip(offset int) {
	ships := c.filteredShips()
	if len(ships) == 0 {
		return
	}
	idx := 0
	for current, ship := range ships {
		if ship.Name == c.curShip {
			idx = current
			break
		}
	}
	idx = (idx + offset + len(ships)) % len(ships)
	c.curShip = ships[idx].Name
	c.pendingRebuild = true
}

func (c *CollectionUI) drawShip(screen *ebiten.Image) {
	ship := objUnit.ShipMap[c.curShip]
	if ship == nil {
		c.drawer.drawText(screen, "当前筛选条件下没有舰船", 80, float64(c.layout.ShipArchive.Y+40), 24, font.Kai, colorx.White)
		return
	}
	c.drawShipBlueprint(screen, ship)
	c.drawShipArchive(screen, ship)
	c.drawShipCombat(screen, ship)
	c.drawShipSource(screen, ship)
}

func (c *CollectionUI) drawShipBlueprint(screen *ebiten.Image, ship *objUnit.BattleShip) {
	rect := c.layout.Blueprint
	bg := bgImg.MissionWindowParchment
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	opts.GeoM.Scale(float64(rect.Dx())/float64(bg.Bounds().Dx()), float64(rect.Dy())/float64(bg.Bounds().Dy()))
	opts.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	screen.DrawImage(bg, opts)
	vector.StrokeRect(screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()), 4, colorx.Silver, false)

	innerX, innerY := float64(rect.Min.X+24), float64(rect.Min.Y+18)
	innerW, innerH := float64(rect.Dx()-48), float64(rect.Dy()-36)
	halfHeight := (innerH - 12) / 2
	c.drawer.drawCollectionImageFit(screen, shipImg.GetSide(ship.Name, 4), innerX, innerY, innerW, halfHeight, 0, false)
	c.drawer.drawCollectionImageFit(screen, shipImg.GetTop(ship.Name, 4), innerX, innerY+halfHeight+12, innerW, halfHeight, 90, false)
}

func (c *CollectionUI) drawShipArchive(screen *ebiten.Image, ship *objUnit.BattleShip) {
	card := c.layout.ShipArchive
	scale := c.metrics.Scale
	c.drawer.drawCollectionCard(screen, card, "舰船档案", c.metrics)
	ref := objRef.GetReference(ship.Name)
	c.drawer.drawText(
		screen, ship.DisplayName, card.X+20*scale, card.Y+54*scale,
		c.metrics.ShipName, font.Hang, colorx.White,
	)
	items := []objRef.InfoItem{
		{Label: "国籍", Value: ship.Nation.ToDisplay()},
		{Label: "舰种", Value: ship.TypeAbbr + " / " + ship.Type.ToDisplay()},
		{Label: "年份", Value: fmt.Sprintf("%d", ship.Year)},
	}
	if ref != nil {
		items = append(items, ref.Specs...)
	} else {
		items = append(items,
			objRef.InfoItem{Label: "吨位", Value: fmt.Sprintf("%.0f", ship.Tonnage)},
			objRef.InfoItem{Label: "费用", Value: fmt.Sprintf("$%d / %ds", ship.FundsCost, ship.TimeCost)},
		)
	}
	c.drawCompactInfoItems(screen, deduplicateInfoItems(items), card, card.Y+96*scale)
}

func deduplicateInfoItems(items []objRef.InfoItem) []objRef.InfoItem {
	seen := map[string]bool{}
	result := make([]objRef.InfoItem, 0, len(items))
	for _, item := range items {
		if seen[item.Label] {
			continue
		}
		seen[item.Label] = true
		result = append(result, item)
	}
	return result
}

func (c *CollectionUI) drawShipCombat(screen *ebiten.Image, ship *objUnit.BattleShip) {
	card := c.layout.ShipCombat
	scale := c.metrics.Scale
	c.drawer.drawCollectionCard(screen, card, "武装与作战能力", c.metrics)
	c.drawer.drawText(
		screen, "综合战力", card.X+card.W-185*scale, card.Y+18*scale,
		c.metrics.Body, font.Kai, color.RGBA{214, 201, 178, 255},
	)
	c.drawer.drawText(
		screen, fmt.Sprintf("%d", ship.CombatPower.Total), card.X+card.W-78*scale, card.Y+15*scale,
		24*scale, font.Hang, colorx.White,
	)

	weaponW := card.W * 0.44
	items := shipArmamentItems(ship)
	weaponCard := collectionCard{X: card.X, Y: card.Y, W: weaponW, H: card.H}
	c.drawCompactInfoItems(screen, items, weaponCard, card.Y+58*scale)

	radius := min(card.H*0.28, (card.W-weaponW)*0.30)
	radius = max(48, radius)
	centerX := card.X + weaponW + (card.W-weaponW)/2
	centerY := card.Y + card.H/2 + 12
	hits := c.drawer.drawAbilityRadar(
		screen, centerX, centerY, radius,
		radarSubject{Name: ship.DisplayName, Power: ship.CombatPower}, c.shipScales, image.Point{},
		c.metrics.RadarLabel,
	)
	c.radarHits = append(c.radarHits, hits...)

	if ship.Type == objUnit.ShipTypeAircraftCarrier {
		c.drawer.drawText(
			screen, fmt.Sprintf("舰体 %d  ·  航空 %d", ship.CombatPower.Hull, ship.CombatPower.Aviation),
			card.X+weaponW+20*scale, card.Y+card.H-34*scale,
			c.metrics.Body, font.Kai, color.RGBA{175, 165, 150, 255},
		)
	}
}

func (c *CollectionUI) drawCompactInfoItems(
	screen *ebiten.Image, items []objRef.InfoItem, card collectionCard, startY float64,
) {
	fontSize := c.metrics.Body
	lineHeight := fontSize * 1.35
	scale := c.metrics.Scale
	maxRows := max(1, int((card.Y+card.H-startY-10*scale)/lineHeight))
	for idx, item := range items {
		if idx >= maxRows {
			if idx < len(items) {
				c.drawer.drawText(
					screen, "…", card.X+20*scale, startY+float64(idx-1)*lineHeight,
					fontSize, font.Kai, colorx.White,
				)
			}
			break
		}
		y := startY + float64(idx)*lineHeight
		c.drawer.drawText(
			screen, item.Label, card.X+20*scale, y,
			fontSize, font.Kai, color.RGBA{214, 201, 178, 255},
		)
		valueX := card.X + min(112*scale, max(76*scale, estimateCollectionTextWidth(item.Label, fontSize)+34*scale))
		value := item.Value
		maxWidth := card.X + card.W - 16*scale - valueX
		for estimateCollectionTextWidth(value, fontSize) > maxWidth && len([]rune(value)) > 1 {
			runes := []rune(value)
			value = string(runes[:len(runes)-1])
		}
		if value != item.Value && len([]rune(value)) > 1 {
			runes := []rune(value)
			value = string(runes[:len(runes)-1]) + "…"
		}
		c.drawer.drawText(screen, value, valueX, y, fontSize, font.Kai, colorx.White)
	}
}

func shipArmamentItems(ship *objUnit.BattleShip) []objRef.InfoItem {
	items := []objRef.InfoItem{}
	ref := objRef.GetReference(ship.Name)
	if ref != nil {
		items = append(items, ref.Armaments...)
	}
	for _, group := range ship.Aircraft.Groups {
		plane := objUnit.PlaneMap[group.Name]
		label := group.Name
		if plane != nil {
			label = plane.DisplayName
		}
		items = append(items, objRef.InfoItem{Label: "舰载机", Value: fmt.Sprintf("%s ×%d", label, group.MaxCount)})
	}
	if len(items) == 0 {
		items = append(items, objRef.InfoItem{Label: "武装", Value: "无"})
	}
	return items
}

func (c *CollectionUI) drawShipSource(screen *ebiten.Image, ship *objUnit.BattleShip) {
	card := c.layout.ShipSource
	scale := c.metrics.Scale
	c.drawer.drawCollectionCard(screen, card, "历史与来源", c.metrics)
	ref := objRef.GetReference(ship.Name)
	if ref == nil {
		c.drawer.drawText(
			screen, "暂无历史资料", card.X+20*scale, card.Y+58*scale,
			c.metrics.History, font.Kai, colorx.White,
		)
		return
	}
	fontSize, lineHeight := c.metrics.History, c.metrics.History*1.35
	descriptionLines := wrapCollectionText(ref.Description, card.W-40*scale, fontSize)
	maxLines := max(1, int((card.H-142*scale)/lineHeight))
	c.drawer.drawCollectionLines(
		screen, descriptionLines, card.X+20*scale, card.Y+56*scale,
		card.W-40*scale, fontSize, lineHeight, font.Kai, maxLines, colorx.White,
	)
	metaY := card.Y + card.H - 80*scale
	c.drawer.drawText(
		screen, "素材原作者："+ref.Author, card.X+20*scale, metaY,
		c.metrics.Body, font.Kai, color.RGBA{214, 201, 178, 255},
	)
	linkY := metaY + 27*scale
	for idx, link := range ref.Links {
		if idx >= 2 {
			break
		}
		text := fmt.Sprintf("[%d] %s", idx+1, link.Name)
		clr := colorx.White
		area := clickableArea{
			X: card.X + 20*scale, Y: linkY + float64(idx)*c.metrics.Body*1.35,
			W: estimateCollectionTextWidth(text, c.metrics.Body), H: c.metrics.Body * 1.25,
		}
		if isHoverArea(area) {
			clr = colorx.SkyBlue
		}
		c.drawer.drawText(screen, text, area.X, area.Y, c.metrics.Body, font.Kai, clr)
		c.links = append(c.links, collectionLink{Area: area, URL: link.URL})
	}
}

func (c *CollectionUI) drawPlanes(screen *ebiten.Image) {
	viewport := c.layout.PlaneViewport
	if c.planeCanvas == nil || c.planeCanvas.Bounds().Dx() != viewport.Dx() || c.planeCanvas.Bounds().Dy() != viewport.Dy() {
		c.planeCanvas = ebiten.NewImage(viewport.Dx(), viewport.Dy())
	}
	c.planeCanvas.Clear()
	vector.FillRect(c.planeCanvas, 0, 0, float32(viewport.Dx()), float32(viewport.Dy()), color.RGBA{12, 12, 12, 150}, false)

	planes := c.filteredPlanes()
	if len(planes) == 0 {
		c.drawer.drawText(
			c.planeCanvas, "当前筛选条件下没有飞机",
			24*c.metrics.Scale, 30*c.metrics.Scale, c.metrics.CardTitle, font.Kai, colorx.White,
		)
	} else {
		cardWidth, cardHeight := c.metrics.PlaneCardW, c.metrics.PlaneCardH
		cardGap := int(math.Round(18 * c.metrics.Scale))
		contentPadding := int(math.Round(24 * c.metrics.Scale))
		contentWidth := len(planes)*cardWidth + max(0, len(planes)-1)*cardGap + contentPadding
		contentHeight := cardHeight + contentPadding
		offsetX := int(math.Round(float64(max(0, contentWidth-viewport.Dx())) * c.planeScrollX))
		offsetY := int(math.Round(float64(max(0, contentHeight-viewport.Dy())) * c.planeScrollY))
		for idx, plane := range planes {
			x, y := 12+idx*(cardWidth+cardGap)-offsetX, 12-offsetY
			if x+cardWidth < 0 || x > viewport.Dx() || y+cardHeight < 0 || y > viewport.Dy() {
				continue
			}
			c.drawPlaneCard(c.planeCanvas, plane, x, y, cardWidth, cardHeight, viewport.Min)
		}
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(viewport.Min.X), float64(viewport.Min.Y))
	screen.DrawImage(c.planeCanvas, opts)
	vector.StrokeRect(screen, float32(viewport.Min.X), float32(viewport.Min.Y), float32(viewport.Dx()), float32(viewport.Dy()), 2, color.RGBA{214, 201, 178, 180}, false)
}

func (c *CollectionUI) drawPlaneCard(
	screen *ebiten.Image, plane *objUnit.Plane, x, y, width, height int, screenOffset image.Point,
) {
	scale := c.metrics.Scale
	px := func(value float64) float64 { return value * scale }
	vector.FillRect(screen, float32(x), float32(y), float32(width), float32(height), color.RGBA{25, 23, 20, 235}, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(width), float32(height), 1.5, color.RGBA{214, 201, 178, 190}, false)
	c.drawer.drawText(
		screen, plane.DisplayName, float64(x)+px(20), float64(y)+px(20),
		c.metrics.PlaneTitle, font.Hang, colorx.White,
	)
	c.drawer.drawText(
		screen, plane.Nation.ToDisplay()+" · "+plane.Type.ToDisplay(),
		float64(x)+px(20), float64(y)+px(54), c.metrics.Body, font.Kai,
		color.RGBA{175, 165, 150, 255},
	)
	c.drawer.drawCollectionImageFit(
		screen, planeImg.Get(plane.Name, 10), float64(x)+px(24), float64(y)+px(78),
		float64(width)-px(48), px(116), 0, true,
	)

	sectionY := y + int(math.Round(px(208)))
	c.drawPlaneSectionTitle(screen, x, sectionY, width, "基础数据")
	c.drawer.drawText(screen, fmt.Sprintf("耐久 %.0f", plane.TotalHP), float64(x)+px(22), float64(sectionY)+px(34), c.metrics.Body, font.Kai, colorx.White)
	c.drawer.drawText(screen, fmt.Sprintf("减伤 %.0f%%", plane.DamageReduction*100), float64(x)+px(178), float64(sectionY)+px(34), c.metrics.Body, font.Kai, colorx.White)
	c.drawer.drawText(screen, fmt.Sprintf("速度 %.0f km/h", plane.MaxSpeed*5400), float64(x)+px(22), float64(sectionY)+px(60), c.metrics.Body, font.Kai, colorx.White)
	c.drawer.drawText(screen, fmt.Sprintf("航程 %.0f km", plane.Range*14.4), float64(x)+px(178), float64(sectionY)+px(60), c.metrics.Body, font.Kai, colorx.White)

	weaponY := sectionY + int(math.Round(px(94)))
	c.drawPlaneSectionTitle(screen, x, weaponY, width, "武器配置")
	weaponLines := planeWeaponLines(plane)
	for idx, line := range weaponLines {
		if idx >= 6 {
			c.drawer.drawText(screen, "…", float64(x)+px(22), float64(weaponY)+px(34+float64(idx)*24), c.metrics.Body, font.Kai, colorx.White)
			break
		}
		c.drawer.drawText(screen, line, float64(x)+px(22), float64(weaponY)+px(34+float64(idx)*24), c.metrics.Body, font.Kai, colorx.White)
	}

	radarY := weaponY + int(math.Round(px(196)))
	c.drawPlaneSectionTitle(screen, x, radarY, width, "作战能力")
	radarCenterX, radarCenterY := float64(x+width/2), float64(radarY)+px(114)
	hits := c.drawer.drawAbilityRadar(
		screen, radarCenterX, radarCenterY, px(70),
		radarSubject{Name: plane.DisplayName, Power: plane.CombatPower, IsPlane: true},
		c.planeScales, screenOffset, c.metrics.RadarLabel,
	)
	viewportScreen := c.layout.PlaneViewport
	for _, hit := range hits {
		if hit.LabelRect.Overlaps(viewportScreen) {
			c.radarHits = append(c.radarHits, hit)
		}
	}
	c.drawer.drawText(screen, "综合战力", float64(x)+px(22), float64(y+height)-px(48), c.metrics.Body, font.Kai, color.RGBA{214, 201, 178, 255})
	valueText := fmt.Sprintf("%d", plane.CombatPower.Total)
	valueSize := px(26)
	c.drawer.drawText(screen, valueText, float64(x+width)-px(22)-estimateCollectionTextWidth(valueText, valueSize), float64(y+height)-px(52), valueSize, font.Hang, colorx.White)
}

func (c *CollectionUI) drawPlaneSectionTitle(screen *ebiten.Image, x, y, width int, title string) {
	scale := c.metrics.Scale
	c.drawer.drawText(screen, title, float64(x)+20*scale, float64(y), c.metrics.Body, font.Kai, color.RGBA{214, 201, 178, 255})
	vector.StrokeLine(
		screen, float32(float64(x)+100*scale), float32(float64(y)+12*scale),
		float32(float64(x+width)-20*scale), float32(float64(y)+12*scale),
		1, color.RGBA{214, 201, 178, 100}, false,
	)
}

func planeWeaponLines(plane *objUnit.Plane) []string {
	items := planeArmamentItems(plane)
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, item.Label+"  "+item.Value)
	}
	return lines
}

func calculateShipAbilityScales() abilityScales {
	powers := []objUnit.CombatPowerInfo{}
	for _, name := range objUnit.GetAllShipNames() {
		ship := objUnit.ShipMap[name]
		if ship != nil && ship.Nation != objUnit.NationSpecial {
			powers = append(powers, ship.CombatPower)
		}
	}
	return calculateAbilityScales(powers)
}

func calculatePlaneAbilityScales() abilityScales {
	powers := []objUnit.CombatPowerInfo{}
	for _, name := range objUnit.GetAllPlaneNames() {
		plane := objUnit.PlaneMap[name]
		if plane != nil && plane.Nation != objUnit.NationSpecial {
			powers = append(powers, plane.CombatPower)
		}
	}
	return calculateAbilityScales(powers)
}
