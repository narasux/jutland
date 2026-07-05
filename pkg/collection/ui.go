package collection

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	uiImage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/pkg/browser"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/i18n"
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

type collectionCategory int

const (
	collectionCategoryShip collectionCategory = iota
	collectionCategoryPlane
)

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
		return i18n.Text(i18n.MsgShipTypeCarrier)
	case shipClassBattleship:
		return i18n.Text(i18n.MsgShipTypeBattleship)
	case shipClassCruiser:
		return i18n.Text(i18n.MsgShipTypeCruiser)
	case shipClassDestroyer:
		return i18n.Text(i18n.MsgShipTypeDestroyer)
	case shipClassFrigate:
		return i18n.Text(i18n.MsgShipTypeFrigate)
	case shipClassTorpedo:
		return i18n.Text(i18n.MsgShipTypeTorpedoBoat)
	case shipClassAuxiliary:
		return i18n.Text(i18n.MsgCollectionAuxiliary)
	case shipClassSpecial:
		return i18n.Text(i18n.MsgCollectionSpecial)
	default:
		return i18n.Text(i18n.MsgCollectionAll)
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
		return i18n.Text(i18n.MsgPlaneTypeFighter)
	case planeTypeDive:
		return i18n.Text(i18n.MsgPlaneTypeDiveBomber)
	case planeTypeTorpedo:
		return i18n.Text(i18n.MsgPlaneTypeTorpedoBomber)
	default:
		return i18n.Text(i18n.MsgCollectionAll)
	}
}

var planeTypeFilters = []planeTypeFilter{planeTypeAll, planeTypeFighter, planeTypeDive, planeTypeTorpedo}

type collectionUILayout struct {
	// Blueprint 是舰船页上半部分的主蓝图区域，只负责展示船体图，不塞额外信息。
	Blueprint image.Rectangle
	// Toolbar 是舰船页共用筛选栏。
	Toolbar image.Rectangle
	// PlaneToolbar 是飞机页的专用筛选栏，和舰船页分开计算高度，避免按钮挤压。
	PlaneToolbar image.Rectangle
	// 下面三个卡片组成舰船页下半部分，从左到右分别承载档案、作战能力和来源信息。
	ShipArchive collectionCard
	ShipCombat  collectionCard
	ShipSource  collectionCard
	// 飞机页使用完整宽度的独立视口，卡片按视口高度整体缩放。
	PlaneViewport image.Rectangle
}

type collectionMetrics struct {
	// Scale 是统一缩放因子，所有字号、间距、卡片尺寸都基于它派生。
	Scale       float64
	ToolbarFont float64
	CardTitle   float64
	ShipName    float64
	Body        float64
	History     float64
	RadarLabel  float64
	Tooltip     float64
}

// calculateCollectionMetrics 根据当前分辨率计算图鉴的统一缩放参数。
// 这里故意只放一个总缩放系数，避免字号、卡片、滑条各自漂移后难以维护。
func calculateCollectionMetrics(width, height int) collectionMetrics {
	scale := clampFloat(min(float64(width)/1600, float64(height)/900), 1, 1.30)
	return collectionMetrics{
		Scale: scale, ToolbarFont: 20 * scale, CardTitle: 22 * scale,
		ShipName: 30 * scale, Body: 18 * scale, History: 18 * scale,
		RadarLabel: 14 * scale, Tooltip: 17 * scale,
	}
}

type collectionLink struct {
	// Area 是图鉴中的点击命中区域，Draw 阶段根据它判断是否打开外部链接。
	Area clickableArea
	URL  string
}

type collectionNameOption struct {
	Name  string
	Label string
}

type collectionDropdown int

const (
	// 下拉框状态是手动维护的，原因是 EbitenUI 的临时窗口和全局鼠标状态会被图鉴自己的输入层穿透。
	collectionDropdownNone collectionDropdown = iota
	collectionDropdownNation
	collectionDropdownType
	collectionDropdownName
)

// CollectionUI 管理图鉴的整套交互状态。
// 它同时持有筛选条件、飞机首项位置、当前选中对象和本帧重建标记。
// 这样做的目的，是让页面切换只替换根容器，而不是销毁共享 EbitenUI 实例。
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

	planeFirstIndex int

	shipScales  abilityScales
	planeScales abilityScales
	radarHits   []radarHitArea
	powerHits   []combatPowerHitArea
	links       []collectionLink

	planeCanvas    *ebiten.Image
	comboButtons   []*widget.ListComboButton
	tabButtons     map[collectionCategory]*widget.Button
	nationCombo    *widget.ListComboButton
	typeCombo      *widget.ListComboButton
	nameCombo      *widget.ListComboButton
	planeCount     *widget.Label
	pendingRebuild bool
	openDropdown   collectionDropdown
	dropdownOffset int
}

// UI 是游戏图鉴界面的对外别名，保留给 game 包直接持有。
type UI = CollectionUI

// New 创建图鉴界面，供上层以默认绘制器初始化使用。
func New() *UI { return NewCollectionUI() }

// NewCollectionUI 创建图鉴界面。
// drawer 参数允许测试注入更轻量的绘制器；生产代码通常不需要传参。
func NewCollectionUI(drawers ...*Drawer) *CollectionUI {
	drawer := NewDrawer()
	if len(drawers) > 0 && drawers[0] != nil {
		drawer = drawers[0]
	}
	ui := &CollectionUI{
		drawer:      drawer,
		category:    collectionCategoryShip,
		shipNation:  objUnit.NationAll,
		shipClass:   shipClassAll,
		curShip:     "lowa",
		planeNation: objUnit.NationAll,
		planeType:   planeTypeAll,
	}
	ui.refreshShipAbilityScales()
	ui.refreshPlaneAbilityScales()
	screen := layout.NewScreenLayout()
	ui.resize(screen.Width, screen.Height)
	return ui
}

func (c *CollectionUI) Update() {
	// 输入处理顺序固定为：Tab 切换分类，方向键/滚轮导航，最后交给手动工具栏输入。
	// 这样可以保证单帧内的状态更新是可预测的，也便于和共享 EbitenUI 输入层共存。
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		c.toggleCategory()
	}

	if c.category == collectionCategoryShip {
		_, wheelY := ebiten.Wheel()
		cursor := image.Pt(ebiten.CursorPosition())
		wheelNavigates := !cursor.In(c.layout.Toolbar) && !c.comboExpanded()
		up := inpututil.IsKeyJustPressed(ebiten.KeyArrowUp)
		down := inpututil.IsKeyJustPressed(ebiten.KeyArrowDown)
		if up || down {
			if up {
				c.moveShipClass(-1)
			} else {
				c.moveShipClass(1)
			}
		} else {
			offset := unitNavigationOffset(
				inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft),
				inpututil.IsKeyJustPressed(ebiten.KeyArrowRight),
				wheelY, wheelNavigates,
			)
			if offset != 0 {
				c.moveShip(offset)
			}
		}
	} else if !c.comboExpanded() {
		_, wheelY := ebiten.Wheel()
		cursor := image.Pt(ebiten.CursorPosition())
		up := inpututil.IsKeyJustPressed(ebiten.KeyArrowUp)
		down := inpututil.IsKeyJustPressed(ebiten.KeyArrowDown)
		if up || down {
			if up {
				c.movePlaneType(-1)
			} else {
				c.movePlaneType(1)
			}
		} else {
			offset := unitNavigationOffset(
				inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft),
				inpututil.IsKeyJustPressed(ebiten.KeyArrowRight),
				wheelY, !cursor.In(c.layout.PlaneToolbar),
			)
			if offset != 0 {
				c.movePlane(offset)
			}
		}
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

func unitNavigationOffset(left, right bool, wheelY float64, wheelAllowed bool) int {
	// 左右方向键和滚轮共用一个结果，返回 -1 / 1 表示向前或向后切换条目。
	// 这让 Update 里只需要处理一次移动逻辑。
	if left || (wheelAllowed && wheelY > 0) {
		return -1
	}
	if right || (wheelAllowed && wheelY < 0) {
		return 1
	}
	return 0
}

func (c *CollectionUI) applyPendingRebuild() {
	// 控件回调只修改状态，不直接重建 UI。
	// 这样可以避开按钮释放事件、下拉框弹层和滚轮状态在同一帧内被打断的问题。
	if !c.pendingRebuild {
		return
	}
	c.pendingRebuild = false
	c.buildUI()
}

// Container 返回由游戏共享 EbitenUI 主实例承载的图鉴容器。
func (c *CollectionUI) Container() widget.Containerer { return c.root }

// ReloadLanguage 标记图鉴控件在下次更新时按当前语言重建。
func (c *CollectionUI) ReloadLanguage() { c.pendingRebuild = true }

func (c *CollectionUI) comboExpanded() bool {
	return c.openDropdown != collectionDropdownNone
}

func (c *CollectionUI) updateManualToolbarInput() {
	// 这里是 EbitenUI 的补偿路径。
	// 某些按钮和列表控件在共享 UI 容器切换后会失去稳定的点击命中，所以图鉴把关键筛选动作再做一层原生命中处理。
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
	// 手动点击处理按命中优先级从高到低排列：
	// 1. 下拉列表内容；2. 分类按钮；3. 筛选下拉框。
	// 这样可以避免一次点击同时触发多个状态变更。
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
		c.selectCategory(collectionCategoryShip)
		return
	}
	if button := c.tabButtons[collectionCategoryPlane]; button != nil && point.In(button.GetWidget().Rect) {
		c.selectCategory(collectionCategoryPlane)
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
	c.openDropdown = collectionDropdownNone
}

func (c *CollectionUI) toggleDropdown(dropdown collectionDropdown) {
	// 同一个下拉框再次点击时收起；切换到其他下拉框时直接打开目标框并重置滚动偏移。
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
	// 下拉框内容完全由当前页面状态派生，避免额外缓存导致切换分类后出现陈旧条目。
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
		for _, ship := range ships {
			entries = append(entries, collectionNameOption{
				Name: ship.Name, Label: objUnit.GetShipDisplayName(ship.Name),
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
	// 选择项后只更新状态，并延迟重建界面。
	// 这样可以让当前帧的 UI 事件先完成，再刷新按钮、标签和下拉列表内容。
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
	// 下拉列表默认跟按钮等宽；舰名列表按最长条目加左右边距自适应。
	button := c.dropdownButton(dropdown)
	if button == nil {
		return image.Rectangle{}
	}
	buttonRect := button.GetWidget().Rect
	width := buttonRect.Dx()
	if dropdown == collectionDropdownName {
		contentWidth := 0.0
		for _, entry := range c.dropdownEntries(dropdown) {
			contentWidth = max(
				contentWidth,
				estimateCollectionTextWidth(c.dropdownEntryLabel(entry), c.metrics.ToolbarFont),
			)
		}
		width = max(width, int(math.Ceil(contentWidth+24*c.metrics.Scale)))
		width = min(width, max(buttonRect.Dx(), c.width-buttonRect.Min.X-12))
	}
	height := c.dropdownVisibleRows(dropdown) * c.dropdownRowHeight()
	return image.Rect(buttonRect.Min.X, buttonRect.Max.Y+2, buttonRect.Min.X+width, buttonRect.Max.Y+2+height)
}

func (c *CollectionUI) DrawOverlay(screen *ebiten.Image) {
	// hover 提示与下拉列表都在共享 UI 之后绘制，保证舰船和战机工具栏视觉一致。
	point := image.Pt(ebiten.CursorPosition())
	if rect, ok := c.hoveredTab(point); ok {
		c.drawToolbarHover(screen, rect)
	}
	if rect, ok := c.hoveredComboRect(point); ok {
		c.drawToolbarHover(screen, rect)
	}
	if c.openDropdown == collectionDropdownNone {
		return
	}
	rect := c.dropdownListRect(c.openDropdown)
	vector.FillRect(
		screen,
		float32(rect.Min.X),
		float32(rect.Min.Y),
		float32(rect.Dx()),
		float32(rect.Dy()),
		color.RGBA{20, 19, 17, 252},
		false,
	)
	vector.StrokeRect(
		screen,
		float32(rect.Min.X),
		float32(rect.Min.Y),
		float32(rect.Dx()),
		float32(rect.Dy()),
		1.5,
		colorx.Gold,
		false,
	)
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
			vector.FillRect(
				screen,
				float32(rowRect.Min.X),
				float32(rowRect.Min.Y),
				float32(rowRect.Dx()),
				float32(rowRect.Dy()),
				color.RGBA{80, 69, 52, 255},
				false,
			)
		}
		c.drawer.drawText(
			screen, c.dropdownEntryLabel(entries[index]), float64(rowRect.Min.X)+12*c.metrics.Scale,
			float64(rowRect.Min.Y)+6*c.metrics.Scale, c.metrics.ToolbarFont, font.LocalizedUI(font.Kai), colorx.White,
		)
	}
}

func (c *CollectionUI) drawToolbarHover(screen *ebiten.Image, rect image.Rectangle) {
	// 分类按钮和下拉框使用同一层轻量提示，避免默认高亮颜色过强或不同容器表现不一致。
	if rect.Empty() {
		return
	}
	vector.FillRect(
		screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()),
		color.RGBA{R: 145, G: 126, B: 92, A: 42}, false,
	)
	vector.StrokeRect(
		screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()),
		1, color.RGBA{R: 190, G: 176, B: 148, A: 125}, false,
	)
}

func (c *CollectionUI) hoveredTab(point image.Point) (image.Rectangle, bool) {
	// 当前分类已经有选中底色，只为另一个分类绘制 hover，保证舰船页和战机页行为对称。
	for category, button := range c.tabButtons {
		if category == c.category || button == nil {
			continue
		}
		rect := button.GetWidget().Rect
		if !rect.Empty() && point.In(rect) {
			return rect, true
		}
	}
	return image.Rectangle{}, false
}

func (c *CollectionUI) hoveredComboRect(point image.Point) (image.Rectangle, bool) {
	// 不依赖 EbitenUI 的 hover 状态，直接用当前控件几何命中，避免不同根容器下表现不一致。
	for _, combo := range c.comboButtons {
		if combo == nil {
			continue
		}
		rect := combo.GetWidget().Rect
		if !rect.Empty() && point.In(rect) {
			return rect, true
		}
	}
	return image.Rectangle{}, false
}

func (c *CollectionUI) Draw(screen *ebiten.Image) {
	// Draw 只负责渲染当前状态，不做筛选计算以外的副作用。
	// 任何会改变控件树的操作都放在 Update 的尾部或回调中通过 pendingRebuild 排队处理。
	if c.width != screen.Bounds().Dx() || c.height != screen.Bounds().Dy() {
		c.resize(screen.Bounds().Dx(), screen.Bounds().Dy())
	}
	c.radarHits = c.radarHits[:0]
	c.powerHits = c.powerHits[:0]
	c.links = c.links[:0]

	if c.category == collectionCategoryShip {
		c.drawShip(screen)
	} else {
		c.drawPlanes(screen)
	}
	if hit := hoveredCombatPowerArea(c.powerHits); hit != nil {
		c.drawer.drawCombatPowerTooltip(screen, hit, c.metrics.Tooltip)
	} else {
		c.drawer.drawRadarTooltip(screen, hoveredRadarArea(c.radarHits), c.metrics.Tooltip)
	}
}

func (c *CollectionUI) resize(width, height int) {
	// 分辨率变化时同时重算布局、缩放和飞机视口画布。
	// 这里重新 build UI 是为了让控件尺寸和命中区跟随新的布局矩形。
	c.width, c.height = width, height
	c.metrics = calculateCollectionMetrics(width, height)
	c.layout = calculateCollectionUILayout(width, height)
	c.clampPlaneFirstIndex()
	c.planeCanvas = ebiten.NewImage(c.layout.PlaneViewport.Dx(), c.layout.PlaneViewport.Dy())
	c.buildUI()
}

func calculateCollectionUILayout(width, height int) collectionUILayout {
	// 上半部分给蓝图，下面给信息卡片。
	// 飞机页不复用舰船页布局，因为飞机卡片需要在独立视口内完整纵向展示。
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
	planeTop, planeBottom := planeToolbar.Max.Y+14, height-24
	planeViewport := image.Rect(marginX, planeTop, width-marginX, planeBottom)

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
			X: float64(
				marginX + archiveW + combatW + 2*gap,
			), Y: float64(detailsY), W: float64(sourceW), H: float64(detailsH),
		},
		PlaneViewport: planeViewport,
	}
}

func (c *CollectionUI) buildUI() {
	// 每次重建只替换根容器，不创建新的 ebitenui.UI 实例。
	// 这样可以保留共享输入、焦点和事件分发链路。
	c.comboButtons = c.comboButtons[:0]
	c.tabButtons = map[collectionCategory]*widget.Button{}
	c.nationCombo, c.typeCombo, c.nameCombo = nil, nil, nil
	c.planeCount = nil
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

	c.root = root
}

func (c *CollectionUI) buildToolbar(rect image.Rectangle) *widget.Container {
	if c.category == collectionCategoryPlane {
		return c.buildPlaneToolbar(rect)
	}
	// 工具栏统一使用行布局，方便在宽屏下横向展开，在窄屏下保持一致的命中和间距规则。
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
	shipTab := c.newTabButton(collectionCategoryShip)
	planeTab := c.newTabButton(collectionCategoryPlane)
	c.tabButtons[collectionCategoryShip] = shipTab
	c.tabButtons[collectionCategoryPlane] = planeTab
	panel.AddChild(shipTab)
	panel.AddChild(planeTab)

	c.nationCombo = c.newNationCombo(c.shipNation, func(nation objUnit.Nation) {
		c.setShipNation(nation)
	})
	c.typeCombo = c.newShipClassCombo()
	c.nameCombo = c.newShipNameCombo()
	panel.AddChild(c.nationCombo)
	panel.AddChild(c.typeCombo)
	panel.AddChild(c.nameCombo)
	return panel
}

func (c *CollectionUI) buildPlaneToolbar(rect image.Rectangle) *widget.Container {
	// 飞机数量独立锚定在整条工具栏中央，不受左侧筛选控件宽度影响。
	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(uiImage.NewNineSliceColor(color.RGBA{18, 18, 18, 215})),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(rect.Dx(), rect.Dy())),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	controls := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewRowLayout(
		widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
		widget.RowLayoutOpts.Spacing(int(math.Round(9*c.metrics.Scale))),
		widget.RowLayoutOpts.Padding(&widget.Insets{
			Left: int(math.Round(12 * c.metrics.Scale)), Right: int(math.Round(12 * c.metrics.Scale)),
			Top: int(math.Round(7 * c.metrics.Scale)), Bottom: int(math.Round(7 * c.metrics.Scale)),
		}),
	)))
	shipTab := c.newTabButton(collectionCategoryShip)
	planeTab := c.newTabButton(collectionCategoryPlane)
	c.tabButtons[collectionCategoryShip] = shipTab
	c.tabButtons[collectionCategoryPlane] = planeTab
	c.nationCombo = c.newNationCombo(c.planeNation, func(nation objUnit.Nation) {
		c.setPlaneNation(nation)
	})
	c.typeCombo = c.newPlaneTypeCombo()
	controls.AddChild(shipTab)
	controls.AddChild(planeTab)
	controls.AddChild(c.nationCombo)
	controls.AddChild(c.typeCombo)
	controls.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	panel.AddChild(controls)

	planeCount := len(c.filteredPlanes())
	count := c.newLabel(i18n.Plural(i18n.MsgCollectionPlaneCount, planeCount, map[string]any{
		"Count": planeCount,
	}), c.metrics.ToolbarFont*0.78)
	count.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		Padding: &widget.Insets{
			Right: int(math.Round(18 * c.metrics.Scale)),
		},
	}
	panel.AddChild(count)
	c.planeCount = count
	return panel
}

func collectionCategoryLabel(category collectionCategory) string {
	if category == collectionCategoryPlane {
		return i18n.Text(i18n.MsgCollectionPlane)
	}
	return i18n.Text(i18n.MsgCollectionShip)
}

func (c *CollectionUI) newTabButton(category collectionCategory) *widget.Button {
	// 标签按钮只是分类切换入口，不携带额外状态。
	return c.newButton(collectionCategoryLabel(category), 72, c.category == category, func() {
		c.selectCategory(category)
	})
}

func (c *CollectionUI) newButton(label string, width int, selected bool, clicked func()) *widget.Button {
	// 所有按钮共用同一套视觉参数，区别只在是否处于选中态。
	face := collectionFace(c.metrics.ToolbarFont, font.Kai)
	idle := color.RGBA{30, 28, 25, 230}
	textIdle := colorx.White
	if selected {
		idle = color.RGBA{190, 157, 92, 255}
		textIdle = colorx.Black
	}
	scale := c.metrics.Scale
	return widget.NewButton(
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(
			int(math.Round(float64(width)*scale)), int(math.Round(34*scale)),
		)),
		widget.ButtonOpts.Image(&widget.ButtonImage{
			Idle: uiImage.NewNineSliceColor(idle), Hover: uiImage.NewNineSliceColor(idle),
			Pressed: uiImage.NewNineSliceColor(color.RGBA{45, 40, 34, 255}),
		}),
		widget.ButtonOpts.Text(label, face, &widget.ButtonTextColor{
			Idle: textIdle, Hover: textIdle, Pressed: colorx.White,
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
	selected objUnit.Nation, changed func(objUnit.Nation),
) *widget.ListComboButton {
	// 国籍筛选在舰船和飞机页复用同一控件，只是回调目标不同。
	entries := make([]any, 0, len(objUnit.AvailableNations()))
	for _, nation := range objUnit.AvailableNations() {
		entries = append(entries, nation)
	}
	return c.newCombo(
		entries, selected, 132,
		func(entry any) string {
			return i18n.Format(i18n.MsgCollectionNationValue, map[string]any{
				"Value": entry.(objUnit.Nation).ToDisplay(),
			})
		},
		func(entry any) string { return entry.(objUnit.Nation).ToDisplay() },
		func(entry any) { changed(entry.(objUnit.Nation)) },
	)
}

func (c *CollectionUI) newShipClassCombo() *widget.ListComboButton {
	// 舰种筛选按大类归并，和图鉴的历史分类口径一致。
	entries := make([]any, 0, len(shipClassFilters))
	for _, filter := range shipClassFilters {
		entries = append(entries, filter)
	}
	return c.newCombo(
		entries, c.shipClass, 148,
		func(entry any) string {
			return i18n.Format(i18n.MsgCollectionShipClass, map[string]any{
				"Value": entry.(shipClassFilter).display(),
			})
		},
		func(entry any) string { return entry.(shipClassFilter).display() },
		func(entry any) {
			c.setShipClass(entry.(shipClassFilter))
		},
	)
}

func (c *CollectionUI) newPlaneTypeCombo() *widget.ListComboButton {
	// 飞机类型筛选直接映射到配置里的机种枚举。
	entries := make([]any, 0, len(planeTypeFilters))
	for _, filter := range planeTypeFilters {
		entries = append(entries, filter)
	}
	return c.newCombo(
		entries, c.planeType, 160,
		func(entry any) string {
			return i18n.Format(i18n.MsgCollectionPlaneType, map[string]any{
				"Value": entry.(planeTypeFilter).display(),
			})
		},
		func(entry any) string { return entry.(planeTypeFilter).display() },
		func(entry any) {
			c.setPlaneType(entry.(planeTypeFilter))
		},
	)
}

func (c *CollectionUI) newShipNameCombo() *widget.ListComboButton {
	// 舰名下拉框只列出当前筛选结果，保证翻页、键盘和筛选始终指向同一批对象。
	ships := c.filteredShips()
	entries := make([]any, 0, len(ships))
	selected := collectionNameOption{}
	for _, ship := range ships {
		option := collectionNameOption{
			Name: ship.Name, Label: objUnit.GetShipDisplayName(ship.Name),
		}
		entries = append(entries, option)
		if ship.Name == c.curShip {
			selected = option
		}
	}
	if len(entries) == 0 {
		entries = append(entries, collectionNameOption{Label: i18n.Text(i18n.MsgCollectionNoMatchingShip)})
		selected = entries[0].(collectionNameOption)
	}
	return c.newCombo(
		entries, selected, 210,
		func(entry any) string {
			return i18n.Format(i18n.MsgCollectionShipName, map[string]any{
				"Value": entry.(collectionNameOption).Label,
			})
		},
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
	// 切换国籍后，当前舰船若不再命中筛选条件，就会自动跳到第一个可见舰船。
	c.shipNation = nation
	c.ensureCurrentShip()
	c.refreshShipAbilityScales()
	c.pendingRebuild = true
}

func (c *CollectionUI) setShipClass(class shipClassFilter) {
	// 舰种变化和国籍变化一样，需要重新保证当前舰船仍然有效。
	c.shipClass = class
	c.ensureCurrentShip()
	c.refreshShipAbilityScales()
	c.pendingRebuild = true
}

func (c *CollectionUI) setCurrentShip(name string) {
	// 舰名选择器直接指定当前舰船，但仍然走 pendingRebuild，保证卡片内容同步刷新。
	c.curShip = name
	c.pendingRebuild = true
}

func (c *CollectionUI) setPlaneNation(nation objUnit.Nation) {
	// 只有筛选值真正变化时才回到第一架，避免焦点控件的重复事件覆盖键盘导航结果。
	if c.planeNation == nation {
		return
	}
	c.planeNation = nation
	c.resetPlanePosition()
	c.refreshPlaneAbilityScales()
	c.pendingRebuild = true
}

func (c *CollectionUI) setPlaneType(planeType planeTypeFilter) {
	// 机种切换和国籍切换的处理方式一致。
	if c.planeType == planeType {
		return
	}
	c.planeType = planeType
	c.resetPlanePosition()
	c.refreshPlaneAbilityScales()
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
	// 所有下拉框都走同一套样式和选中回调，差异只来自条目生成器。
	faceSource := font.ForText(buttonLabel(selected), font.LocalizedUI(font.Kai))
	faceValue := text.Face(&text.GoTextFace{Source: faceSource, Size: c.metrics.ToolbarFont})
	face := &faceValue
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
		// 方向键由图鉴导航独占；否则获得焦点的下拉框会先展开，再让本帧导航被 comboExpanded 拦截。
		widget.ListComboButtonOpts.DisableDefaultKeys(true),
		widget.ListComboButtonOpts.MaxContentHeight(int(math.Round(280*scale))),
	)
	c.comboButtons = append(c.comboButtons, combo)
	return combo
}

func collectionFace(size float64, source *text.GoTextFaceSource) *text.Face {
	// 统一字体生成入口，避免图鉴不同控件各自拼字号导致视觉不一致。
	face := text.Face(&text.GoTextFace{Source: font.LocalizedUI(source), Size: size})
	return &face
}

func (c *CollectionUI) toggleCategory() {
	// 分类切换后保留各自的筛选状态，便于在舰船和飞机页之间来回查看。
	if c.category == collectionCategoryShip {
		c.selectCategory(collectionCategoryPlane)
		return
	}
	c.selectCategory(collectionCategoryShip)
}

func (c *CollectionUI) selectCategory(category collectionCategory) {
	// 鼠标、EbitenUI 回调和 Tab 键统一走同一状态转换，确保两个分类的切换行为完全一致。
	c.openDropdown = collectionDropdownNone
	if c.category == category {
		return
	}
	c.category = category
	c.pendingRebuild = true
}

func (c *CollectionUI) resetPlanePosition() {
	c.planeFirstIndex = 0
}

type planeCardGeometry struct {
	Scale         float64
	Width, Height int
	Gap, Padding  int
	VisibleCount  int
}

func (c *CollectionUI) calculatePlaneCardGeometry() planeCardGeometry {
	// 340×850 是飞机卡片的设计基准；宽高连同边距整体缩放，保证卡片始终完整落在视口内。
	viewport := c.layout.PlaneViewport
	scale := min(
		c.metrics.Scale,
		float64(viewport.Dy())/(850+24),
		float64(viewport.Dx())/(340+24),
	)
	scale = max(0.1, scale)
	width := max(1, int(math.Round(340*scale)))
	height := max(1, int(math.Round(850*scale)))
	gap := max(1, int(math.Round(18*scale)))
	padding := max(1, int(math.Round(12*scale)))
	availableWidth := max(1, viewport.Dx()-2*padding)
	visibleCount := max(1, (availableWidth+gap)/(width+gap))
	return planeCardGeometry{
		Scale: scale, Width: width, Height: height,
		Gap: gap, Padding: padding, VisibleCount: visibleCount,
	}
}

func (c *CollectionUI) maxPlaneFirstIndex() int {
	return max(0, len(c.filteredPlanes())-c.calculatePlaneCardGeometry().VisibleCount)
}

func (c *CollectionUI) clampPlaneFirstIndex() {
	c.planeFirstIndex = max(0, min(c.planeFirstIndex, c.maxPlaneFirstIndex()))
}

func (c *CollectionUI) movePlane(offset int) {
	// 左右箭头每次移动一个飞机卡片，并在首尾处停止，行为与原水平滚动条一致。
	c.planeFirstIndex = max(0, min(c.planeFirstIndex+offset, c.maxPlaneFirstIndex()))
}

func (c *CollectionUI) movePlaneType(offset int) {
	// 上下方向键切换机种；当前国籍下没有飞机的机种会被跳过，行为对齐舰船页的舰种导航。
	if offset == 0 || len(planeTypeFilters) == 0 {
		return
	}
	current := 0
	for idx, planeType := range planeTypeFilters {
		if planeType == c.planeType {
			current = idx
			break
		}
	}
	direction := 1
	if offset < 0 {
		direction = -1
	}
	for step := 1; step <= len(planeTypeFilters); step++ {
		idx := (current + direction*step + len(planeTypeFilters)) % len(planeTypeFilters)
		c.planeType = planeTypeFilters[idx]
		if len(c.filteredPlanes()) == 0 {
			continue
		}
		c.resetPlanePosition()
		c.refreshPlaneAbilityScales()
		c.pendingRebuild = true
		return
	}
	c.planeType = planeTypeFilters[current]
}

func clampFloat(value, minimum, maximum float64) float64 {
	return min(maximum, max(minimum, value))
}

func (c *CollectionUI) filteredShips() []*objUnit.BattleShip {
	// 舰船过滤顺序与“分类 -> 国籍 -> 名称”工具栏一致，方便用户和代码逻辑对应。
	ships := make([]*objUnit.BattleShip, 0, len(objUnit.AllShipNames))
	for _, name := range objUnit.AllShipNames {
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
	// 图鉴里的“舰种”是大类而不是精细子类，所以这里要把多个底层类型合并到同一筛选项。
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
	// 飞机页只按国籍和机种筛选，保持列表足够短，便于纵列和横向滚动展示。
	planes := make([]*objUnit.Plane, 0, len(objUnit.AllPlaneNames))
	for _, name := range objUnit.AllPlaneNames {
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
	// 飞机机种与配置枚举一一对应，只有“全部”是额外的宽松筛选项。
	return filter == planeTypeAll || string(planeType) == string(filter)
}

func (c *CollectionUI) ensureCurrentShip() {
	// 如果当前舰船被筛掉，就自动跳到筛选结果里的第一艘，避免卡片区显示空壳状态。
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
	// 舰船导航是循环的，前后切换永远停在某一艘有效对象上。
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

func (c *CollectionUI) moveShipClass(offset int) {
	// 上下方向键只切换舰种，不改变国籍；没有当前国籍舰船的舰种会被跳过。
	if offset == 0 || len(shipClassFilters) == 0 {
		return
	}
	current := 0
	for idx, class := range shipClassFilters {
		if class == c.shipClass {
			current = idx
			break
		}
	}
	direction := 1
	if offset < 0 {
		direction = -1
	}
	for step := 1; step <= len(shipClassFilters); step++ {
		idx := (current + direction*step + len(shipClassFilters)) % len(shipClassFilters)
		c.shipClass = shipClassFilters[idx]
		ships := c.filteredShips()
		if len(ships) == 0 {
			continue
		}
		c.curShip = ships[0].Name
		c.refreshShipAbilityScales()
		c.pendingRebuild = true
		return
	}
	c.shipClass = shipClassFilters[current]
}

func (c *CollectionUI) drawShip(screen *ebiten.Image) {
	// 舰船页上半部分是纯蓝图，下半部分是档案、作战能力和来源三块内容。
	ship := objUnit.ShipMap[c.curShip]
	if ship == nil {
		c.drawer.drawText(
			screen,
			i18n.Text(i18n.MsgCollectionNoMatchingShip),
			80,
			float64(c.layout.ShipArchive.Y+40),
			24,
			font.LocalizedUI(font.Kai),
			colorx.White,
		)
		return
	}
	c.drawShipBlueprint(screen, ship)
	c.drawShipArchive(screen, ship)
	c.drawShipCombat(screen, ship)
	c.drawShipSource(screen, ship)
}

func (c *CollectionUI) drawShipBlueprint(screen *ebiten.Image, ship *objUnit.BattleShip) {
	// 蓝图区只放船体侧视和俯视图，不额外塞战力或档案信息。
	rect := c.layout.Blueprint
	bg := bgImg.MissionWindowParchment
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	opts.GeoM.Scale(float64(rect.Dx())/float64(bg.Bounds().Dx()), float64(rect.Dy())/float64(bg.Bounds().Dy()))
	opts.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	screen.DrawImage(bg, opts)
	vector.StrokeRect(
		screen,
		float32(rect.Min.X),
		float32(rect.Min.Y),
		float32(rect.Dx()),
		float32(rect.Dy()),
		4,
		colorx.Silver,
		false,
	)

	innerX, innerY := float64(rect.Min.X+24), float64(rect.Min.Y+18)
	innerW, innerH := float64(rect.Dx()-48), float64(rect.Dy()-36)
	halfHeight := (innerH - 12) / 2
	sideImage := shipImg.GetSide(ship.Name, 4)
	topImage := shipImg.GetTop(ship.Name, 4)
	sharedScale := min(
		collectionImageFitScale(sideImage, innerW, halfHeight, 0, false),
		collectionImageFitScale(topImage, innerW, halfHeight, 90, false),
	)
	c.drawer.drawCollectionImageScaled(screen, sideImage, innerX, innerY, innerW, halfHeight, 0, sharedScale)
	c.drawer.drawCollectionImageScaled(
		screen, topImage, innerX, innerY+halfHeight+12, innerW, halfHeight, 90, sharedScale,
	)
}

func (c *CollectionUI) drawShipArchive(screen *ebiten.Image, ship *objUnit.BattleShip) {
	// 档案卡片优先使用 reference 里的标准资料，缺失时才退回到配置字段。
	card := c.layout.ShipArchive
	scale := c.metrics.Scale
	c.drawer.drawCollectionCard(screen, card, i18n.Text(i18n.MsgCollectionShipArchive), c.metrics)
	ref := objRef.GetReference(ship.Name)
	c.drawer.drawText(
		screen, objUnit.GetShipDisplayName(ship.Name), card.X+20*scale, card.Y+54*scale,
		c.metrics.ShipName, font.LocalizedUI(font.Hang), colorx.White,
	)
	if position := c.currentShipPositionLabel(); position != "" {
		c.drawer.drawText(
			screen, position, card.X+20*scale, card.Y+88*scale,
			c.metrics.Body, font.LocalizedUI(font.Kai), color.RGBA{175, 165, 150, 255},
		)
	}
	c.drawCompactInfoItems(screen, shipArchiveInfoItems(ship, ref), card, card.Y+116*scale)
}

func (c *CollectionUI) currentShipPositionLabel() string {
	position, total := 0, 0
	for _, name := range objUnit.AllShipNames {
		ship := objUnit.ShipMap[name]
		if ship == nil || (c.shipNation != objUnit.NationAll && ship.Nation != c.shipNation) ||
			!matchShipClass(ship.Type, c.shipClass) {
			continue
		}
		total++
		if ship.Name == c.curShip {
			position = total
		}
	}
	if position == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", position, total)
}

func shipArchiveInfoItems(ship *objUnit.BattleShip, ref *objRef.Reference) []objRef.InfoItem {
	// reference 只保留精细舰种名称，其余档案数据全部来自 ships.json5 初始化后的舰船对象。
	return []objRef.InfoItem{
		{Label: i18n.Text(i18n.MsgCollectionNationLabel), Value: ship.Nation.ToDisplay()},
		{Label: i18n.Text(i18n.MsgCollectionTypeLabel), Value: shipTypeDisplayValue(ship, ref)},
		{
			Label: i18n.Text(i18n.MsgCollectionYear),
			Value: lo.Ternary(ship.Year == 0, "--", fmt.Sprintf("%d", ship.Year)),
		},
		{Label: i18n.Text(i18n.MsgCollectionTonnage), Value: fmt.Sprintf("%.0f", ship.Tonnage)},
		{
			Label: i18n.Text(i18n.MsgCollectionSpeed),
			Value: i18n.Format(
				i18n.MsgValueKnots,
				map[string]any{"Value": formatShipArchiveNumber(ship.MaxSpeed * 600)},
			),
		},
		{
			Label: i18n.Text(i18n.MsgCollectionCost),
			Value: i18n.Format(i18n.MsgValueCost, map[string]any{"Funds": ship.FundsCost, "Seconds": ship.TimeCost}),
		},
		{
			Label: i18n.Text(i18n.MsgCollectionHorizontalDR),
			Value: formatShipArchiveNumber(ship.HorizontalDamageReduction*100) + "%",
		},
		{
			Label: i18n.Text(i18n.MsgCollectionVerticalDR),
			Value: formatShipArchiveNumber(ship.VerticalDamageReduction*100) + "%",
		},
	}
}

func shipTypeDisplayValue(ship *objUnit.BattleShip, ref *objRef.Reference) string {
	display := ship.Type.ToDisplay()
	if ref != nil && strings.TrimSpace(ref.Type) != "" {
		display = strings.TrimSpace(ref.Type)
	}
	if ship.TypeAbbr == "" {
		return display
	}
	return i18n.Format(i18n.MsgTypeWithAbbr, map[string]any{"Type": display, "Abbr": ship.TypeAbbr})
}

func formatShipArchiveNumber(value float64) string {
	// 航速最多保留一位小数；同时避免初始化换算带来的浮点尾数。
	return strings.TrimSuffix(fmt.Sprintf("%.1f", value), ".0")
}

func (c *CollectionUI) drawShipCombat(screen *ebiten.Image, ship *objUnit.BattleShip) {
	// 作战卡片左边展示武装，右边展示雷达，顶部补一行总战力，方便快速扫读。
	card := c.layout.ShipCombat
	scale := c.metrics.Scale
	c.drawer.drawCollectionCard(screen, card, i18n.Text(i18n.MsgCollectionCombat), c.metrics)
	label := i18n.Text(i18n.MsgCollectionTotalPower)
	value := fmt.Sprintf("%d", ship.CombatPower.Total)
	labelX, valueX := combatPowerHeaderPositions(card, scale, label, value)
	c.drawer.drawText(
		screen, label, labelX, card.Y+18*scale,
		c.metrics.Body, font.LocalizedUI(font.Kai), color.RGBA{214, 201, 178, 255},
	)
	c.drawer.drawText(
		screen, value, valueX, card.Y+15*scale,
		24*scale, font.JetbrainsMono, colorx.White,
	)
	if ship.Type == objUnit.ShipTypeAircraftCarrier {
		c.powerHits = append(c.powerHits, combatPowerHitArea{
			Rect: image.Rect(
				int(labelX-8*scale), int(card.Y+8*scale),
				int(card.X+card.W-14*scale), int(card.Y+48*scale),
			),
			Subject: radarSubject{Name: objUnit.GetShipDisplayName(ship.Name), Power: ship.CombatPower},
		})
	}

	weaponW := card.W * 0.44
	items := shipArmamentItems(ship)
	weaponCard := collectionCard{X: card.X, Y: card.Y, W: weaponW, H: card.H}
	c.drawCompactInfoItems(screen, items, weaponCard, card.Y+58*scale)

	radius := min(card.H*0.28, (card.W-weaponW)*0.30)
	radius = max(48, radius)
	centerX := card.X + weaponW + (card.W-weaponW)/2
	centerY := card.Y + card.H/2 + 12
	hits := c.drawer.drawAbilityRadar(
		screen,
		centerX,
		centerY,
		radius,
		radarSubject{
			Name:  objUnit.GetShipDisplayName(ship.Name),
			Power: ship.CombatPower,
		},
		c.shipScales,
		image.Point{},
		c.metrics.RadarLabel,
	)
	c.radarHits = append(c.radarHits, hits...)
	c.drawRadarScaleNote(
		screen, card.X+weaponW+12*scale, card.Y+card.H-38*scale,
		card.W-weaponW-24*scale, c.metrics.RadarLabel,
		i18n.Text(i18n.MsgCollectionRelativeNote),
	)
}

func combatPowerHeaderPositions(card collectionCard, scale float64, label, value string) (float64, float64) {
	valueWidth := layout.CalcTextWidth(value, 24*scale, font.JetbrainsMono)
	valueX := card.X + card.W - 20*scale - valueWidth
	labelWidth := estimateCollectionTextWidth(label, 18*scale)
	labelX := valueX - 14*scale - labelWidth
	return labelX, valueX
}

func (c *CollectionUI) drawRadarScaleNote(
	screen *ebiten.Image, x, y, width, fontSize float64, note string,
) {
	lineHeight := fontSize * 1.25
	for idx, line := range wrapCollectionText(note, width, fontSize) {
		if idx >= 2 {
			break
		}
		lineWidth := estimateCollectionTextWidth(line, fontSize)
		c.drawer.drawText(
			screen, line, x+(width-lineWidth)/2, y+float64(idx)*lineHeight,
			fontSize, font.LocalizedUI(font.Kai), color.RGBA{175, 165, 150, 255},
		)
	}
}

func (c *CollectionUI) drawCompactInfoItems(
	screen *ebiten.Image, items []objRef.InfoItem, card collectionCard, startY float64,
) {
	// 这里不是完整表格，只是紧凑的键值排版，所以会对文本长度做保守截断。
	fontSize := c.metrics.Body
	lineHeight := fontSize * 1.35
	scale := c.metrics.Scale
	maxRows := max(1, int((card.Y+card.H-startY-10*scale)/lineHeight))
	for idx, item := range items {
		if idx >= maxRows {
			if idx < len(items) {
				c.drawer.drawText(
					screen, "…", card.X+20*scale, startY+float64(idx-1)*lineHeight,
					fontSize, font.LocalizedUI(font.Kai), colorx.White,
				)
			}
			break
		}
		y := startY + float64(idx)*lineHeight
		c.drawer.drawText(
			screen, item.Label, card.X+20*scale, y,
			fontSize, font.LocalizedUI(font.Kai), color.RGBA{214, 201, 178, 255},
		)
		valueX := compactInfoValueX(item.Label, card, fontSize, scale)
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
		c.drawer.drawText(screen, value, valueX, y, fontSize, font.LocalizedUI(font.Kai), colorx.White)
	}
}

func compactInfoValueX(label string, card collectionCard, fontSize, scale float64) float64 {
	// value 始终放在 label 实际宽度之后；长舰载机名称不再受固定列宽上限影响。
	labelX := card.X + 20*scale
	return max(card.X+76*scale, labelX+estimateCollectionTextWidth(label, fontSize)+16*scale)
}

func shipArmamentItems(ship *objUnit.BattleShip) []objRef.InfoItem {
	// 舰船武装直接采用 reference 摘要；航母的舰载机已经按角色写入配置，不再从编组重复追加。
	items := []objRef.InfoItem{}
	ref := objRef.GetReference(ship.Name)
	if ref != nil {
		items = append(items, ref.Armaments...)
	}
	if len(items) == 0 {
		items = append(items, objRef.InfoItem{
			Label: i18n.Text(i18n.MsgCollectionArmaments),
			Value: i18n.Text(i18n.MsgCollectionNone),
		})
	}
	return items
}

func (c *CollectionUI) drawShipSource(screen *ebiten.Image, ship *objUnit.BattleShip) {
	// 来源卡片不做外链抓取，只展示本地配置和 reference 数据，保持图鉴离线可用。
	card := c.layout.ShipSource
	scale := c.metrics.Scale
	c.drawer.drawCollectionCard(screen, card, i18n.Text(i18n.MsgCollectionHistorySource), c.metrics)
	ref := objRef.GetReference(ship.Name)
	if ref == nil {
		c.drawer.drawText(
			screen, i18n.Text(i18n.MsgCollectionNoHistory), card.X+20*scale, card.Y+58*scale,
			c.metrics.History, font.LocalizedUI(font.Kai), colorx.White,
		)
		return
	}
	fontSize := c.metrics.History
	if i18n.CurrentLanguage() == i18n.LanguageEnglish {
		fontSize *= 0.82
	}
	lineHeight := fontSize * 1.35
	contentWidth := card.W - 40*scale
	descriptionLines := wrapCollectionText(ref.Description, card.W-40*scale, fontSize)
	authorLines := []string{}
	if ref.Author != "" {
		authorText := i18n.Format(i18n.MsgCollectionAssetAuthor, map[string]any{"Author": ref.Author})
		authorLines = wrapCollectionText(authorText, contentWidth, fontSize)
	}
	linkLines := make([][]string, min(2, len(ref.Links)))
	metadataLines := len(authorLines)
	for idx := range linkLines {
		linkLines[idx] = wrapCollectionText(
			fmt.Sprintf("[%d] %s", idx+1, ref.Links[idx].Name), contentWidth, fontSize,
		)
		metadataLines += len(linkLines[idx])
	}
	descriptionY := card.Y + 56*scale
	// 历史正文之后固定留一行空白，再按内容流顺序绘制作者和链接。
	reservedHeight := float64(metadataLines+1)*lineHeight + 12*scale
	maxLines := max(1, int((card.Y+card.H-descriptionY-reservedHeight)/lineHeight))
	c.drawer.drawCollectionLines(
		screen, descriptionLines, card.X+20*scale, descriptionY,
		card.W-40*scale, fontSize, lineHeight, font.LocalizedUI(font.Kai), maxLines, colorx.White,
	)
	drawnLines := min(maxLines, len(descriptionLines))
	metaY := descriptionY + float64(drawnLines+1)*lineHeight
	for _, line := range authorLines {
		c.drawer.drawText(
			screen, line, card.X+20*scale, metaY, fontSize,
			font.LocalizedUI(font.Kai), color.RGBA{214, 201, 178, 255},
		)
		metaY += lineHeight
	}
	for idx, lines := range linkLines {
		for _, line := range lines {
			area := clickableArea{
				X: card.X + 20*scale, Y: metaY,
				W: estimateCollectionTextWidth(line, fontSize), H: fontSize * 1.25,
			}
			clr := colorx.White
			if isHoverArea(area) {
				clr = colorx.SkyBlue
			}
			c.drawer.drawText(screen, line, area.X, area.Y, fontSize, font.LocalizedUI(font.Kai), clr)
			c.links = append(c.links, collectionLink{Area: area, URL: ref.Links[idx].URL})
			metaY += lineHeight
		}
	}
}

func (c *CollectionUI) drawPlanes(screen *ebiten.Image) {
	// 飞机页只横向排列完整卡片；可见范围由两侧箭头按单架飞机步进。
	viewport := c.layout.PlaneViewport
	if c.planeCanvas == nil || c.planeCanvas.Bounds().Dx() != viewport.Dx() ||
		c.planeCanvas.Bounds().Dy() != viewport.Dy() {
		c.planeCanvas = ebiten.NewImage(viewport.Dx(), viewport.Dy())
	}
	c.planeCanvas.Clear()
	vector.FillRect(
		c.planeCanvas,
		0,
		0,
		float32(viewport.Dx()),
		float32(viewport.Dy()),
		color.RGBA{12, 12, 12, 150},
		false,
	)

	planes := c.filteredPlanes()
	if len(planes) == 0 {
		c.drawer.drawText(
			c.planeCanvas, i18n.Text(i18n.MsgCollectionNoMatchingPlane),
			24*c.metrics.Scale, 30*c.metrics.Scale, c.metrics.CardTitle, font.LocalizedUI(font.Kai), colorx.White,
		)
	} else {
		geometry := c.calculatePlaneCardGeometry()
		maxFirst := max(0, len(planes)-geometry.VisibleCount)
		c.planeFirstIndex = max(0, min(c.planeFirstIndex, maxFirst))
		visibleCount := min(geometry.VisibleCount, len(planes)-c.planeFirstIndex)
		contentWidth := visibleCount*geometry.Width + max(0, visibleCount-1)*geometry.Gap
		startX := (viewport.Dx() - contentWidth) / 2
		y := (viewport.Dy() - geometry.Height) / 2
		for column := 0; column < visibleCount; column++ {
			plane := planes[c.planeFirstIndex+column]
			x := startX + column*(geometry.Width+geometry.Gap)
			c.drawPlaneCard(
				c.planeCanvas, plane, x, y, geometry.Width, geometry.Height,
				geometry.Scale, viewport.Min,
			)
		}
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(viewport.Min.X), float64(viewport.Min.Y))
	screen.DrawImage(c.planeCanvas, opts)
	vector.StrokeRect(
		screen,
		float32(viewport.Min.X),
		float32(viewport.Min.Y),
		float32(viewport.Dx()),
		float32(viewport.Dy()),
		2,
		color.RGBA{214, 201, 178, 180},
		false,
	)
}

func (c *CollectionUI) drawPlaneCard(
	screen *ebiten.Image, plane *objUnit.Plane, x, y, width, height int,
	scale float64, screenOffset image.Point,
) {
	// 每张飞机卡片从上到下固定分为：标题、素材、基础数据、武器配置、雷达图和总战力。
	px := func(value float64) float64 { return value * scale }
	vector.FillRect(
		screen,
		float32(x),
		float32(y),
		float32(width),
		float32(height),
		color.RGBA{25, 23, 20, 235},
		false,
	)
	vector.StrokeRect(
		screen,
		float32(x),
		float32(y),
		float32(width),
		float32(height),
		1.5,
		color.RGBA{214, 201, 178, 190},
		false,
	)
	c.drawer.drawText(
		screen, objUnit.GetPlaneDisplayName(plane.Name), float64(x)+px(20), float64(y)+px(20),
		px(27), font.LocalizedUI(font.Hang), colorx.White,
	)
	c.drawer.drawText(
		screen, plane.Nation.ToDisplay()+" · "+plane.Type.ToDisplay(),
		float64(x)+px(20), float64(y)+px(54), px(18), font.LocalizedUI(font.Kai),
		color.RGBA{175, 165, 150, 255},
	)
	c.drawer.drawCollectionImageAtBaseScale(
		screen, planeImg.GetOriginal(plane.Name), float64(x)+px(24), float64(y)+px(78),
		float64(width)-px(48), px(116), 0, planeImg.GetDisplayScale(plane.Name),
	)

	sectionY := y + int(math.Round(px(208)))
	c.drawPlaneSectionTitle(screen, x, sectionY, width, scale, i18n.Text(i18n.MsgCollectionBasicData))
	bodyFont := font.LocalizedUI(font.Kai)
	basicLines := []string{
		i18n.Format(i18n.MsgCollectionDurability, map[string]any{"Value": fmt.Sprintf("%.0f", plane.TotalHP)}),
		i18n.Format(
			i18n.MsgCollectionDamageReduction,
			map[string]any{"Value": fmt.Sprintf("%.0f", plane.DamageReduction*100)},
		),
		i18n.Format(i18n.MsgCollectionPlaneSpeed, map[string]any{"Value": fmt.Sprintf("%.0f", plane.MaxSpeed*5400)}),
		i18n.Format(i18n.MsgCollectionRange, map[string]any{"Value": fmt.Sprintf("%.0f", plane.Range*14.4)}),
	}
	for idx, line := range basicLines {
		c.drawer.drawText(
			screen, line, float64(x)+px(22), float64(sectionY)+px(34+float64(idx)*22),
			px(18), bodyFont, colorx.White,
		)
	}

	weaponY := sectionY + int(math.Round(px(128)))
	c.drawPlaneSectionTitle(screen, x, weaponY, width, scale, i18n.Text(i18n.MsgCollectionWeaponConfig))
	weaponLines := planeWeaponLines(plane)
	for idx, line := range weaponLines {
		if idx >= 6 {
			c.drawer.drawText(
				screen,
				"…",
				float64(x)+px(22),
				float64(weaponY)+px(34+float64(idx)*24),
				px(18),
				bodyFont,
				colorx.White,
			)
			break
		}
		c.drawer.drawText(
			screen,
			line,
			float64(x)+px(22),
			float64(weaponY)+px(34+float64(idx)*24),
			px(18),
			bodyFont,
			colorx.White,
		)
	}

	radarY := weaponY + int(math.Round(px(184)))
	c.drawPlaneSectionTitle(screen, x, radarY, width, scale, i18n.Text(i18n.MsgCollectionCombatAbility))
	radarCenterX, radarCenterY := float64(x+width/2), float64(radarY)+px(114)
	hits := c.drawer.drawAbilityRadar(
		screen, radarCenterX, radarCenterY, px(70),
		radarSubject{Name: objUnit.GetPlaneDisplayName(plane.Name), Power: plane.CombatPower, IsPlane: true},
		c.planeScales, screenOffset, px(14),
	)
	viewportScreen := c.layout.PlaneViewport
	for _, hit := range hits {
		if hit.LabelRect.Overlaps(viewportScreen) {
			c.radarHits = append(c.radarHits, hit)
		}
	}
	c.drawRadarScaleNote(
		screen, float64(x)+px(20), float64(radarY)+px(208), float64(width)-px(40), px(14),
		i18n.Format(i18n.MsgCollectionFormationNote, map[string]any{"Count": plane.CombatPower.FormationSize}),
	)
	formationLabel := i18n.Format(
		i18n.MsgCollectionFormationPower,
		map[string]any{"Count": plane.CombatPower.FormationSize},
	)
	valueText := fmt.Sprintf("%d", plane.CombatPower.Total)
	valueSize := px(26)
	valueX := float64(x+width) - px(22) - estimateCollectionTextWidth(valueText, valueSize)
	labelX := float64(x) + px(22)
	labelSize := px(18)
	labelMaxWidth := valueX - labelX - px(12)
	if labelWidth := estimateCollectionTextWidth(formationLabel, labelSize); labelWidth > labelMaxWidth {
		labelSize *= labelMaxWidth / labelWidth
	}
	c.drawer.drawText(
		screen,
		formationLabel,
		labelX,
		float64(y+height)-px(48),
		labelSize,
		bodyFont,
		color.RGBA{214, 201, 178, 255},
	)
	c.drawer.drawText(
		screen,
		valueText,
		valueX,
		float64(y+height)-px(52),
		valueSize,
		font.JetbrainsMono,
		colorx.White,
	)
}

func (c *CollectionUI) drawPlaneSectionTitle(
	screen *ebiten.Image, x, y, width int, scale float64, title string,
) {
	// 小节标题统一使用短横线分隔，保持飞机卡片的纵向阅读节奏。
	titleX := float64(x) + 20*scale
	titleSize := 18 * scale
	c.drawer.drawText(
		screen,
		title,
		titleX,
		float64(y),
		titleSize,
		font.LocalizedUI(font.Kai),
		color.RGBA{214, 201, 178, 255},
	)
	lineStartX := titleX + estimateCollectionTextWidth(title, titleSize) + 12*scale
	lineEndX := float64(x+width) - 20*scale
	if lineStartX >= lineEndX {
		return
	}
	vector.StrokeLine(
		screen, float32(lineStartX), float32(float64(y)+12*scale),
		float32(lineEndX), float32(float64(y)+12*scale),
		1, color.RGBA{214, 201, 178, 100}, false,
	)
}

func planeWeaponLines(plane *objUnit.Plane) []string {
	// 飞机武器只做“标签 + 值”的直接映射，具体分组和压缩由 planeArmamentItems 负责。
	items := planeArmamentItems(plane)
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, item.Label+"  "+item.Value)
	}
	return lines
}

func (c *CollectionUI) refreshShipAbilityScales() {
	// 舰船雷达只和当前国籍、舰种筛选结果比较；舰名选择不会改变比较池。
	ships := c.filteredShips()
	powers := make([]objUnit.CombatPowerInfo, 0, len(ships))
	for _, ship := range ships {
		powers = append(powers, ship.CombatPower)
	}
	c.shipScales = calculateAbilityScales(powers)
}

func (c *CollectionUI) refreshPlaneAbilityScales() {
	// 飞机雷达使用独立的当前筛选池，避免把舰炮与飞机的投送距离放在同一尺度比较。
	planes := c.filteredPlanes()
	powers := make([]objUnit.CombatPowerInfo, 0, len(planes))
	for _, plane := range planes {
		powers = append(powers, plane.CombatPower)
	}
	c.planeScales = calculateAbilityScales(powers)
}
