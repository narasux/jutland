package game

import (
	"image"
	"testing"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/event"
	uiInput "github.com/ebitenui/ebitenui/input"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/narasux/jutland/pkg/config"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

type collectionTestCursor struct {
	position image.Point
	pressed  bool
	last     bool
	justDown bool
	justUp   bool
}

func (c *collectionTestCursor) Update() {
	c.justDown = c.pressed && !c.last
	c.justUp = !c.pressed && c.last
}

func (c *collectionTestCursor) AfterUpdate() {
	c.last = c.pressed
	c.justDown, c.justUp = false, false
}

func (*collectionTestCursor) Draw(*ebiten.Image)      {}
func (*collectionTestCursor) AfterDraw(*ebiten.Image) {}
func (c *collectionTestCursor) MouseButtonPressed(button ebiten.MouseButton) bool {
	return button == ebiten.MouseButtonLeft && c.pressed
}
func (c *collectionTestCursor) MouseButtonJustPressed(button ebiten.MouseButton) bool {
	return button == ebiten.MouseButtonLeft && c.justDown
}
func (c *collectionTestCursor) MouseButtonJustReleased(button ebiten.MouseButton) bool {
	return button == ebiten.MouseButtonLeft && c.justUp
}
func (c *collectionTestCursor) CursorPosition() (int, int) { return c.position.X, c.position.Y }
func (*collectionTestCursor) GetCursorImage(string) *ebiten.Image {
	return nil
}
func (*collectionTestCursor) GetCursorOffset(string) image.Point { return image.Point{} }

func TestInitializedCombatPower(t *testing.T) {
	if len(objUnit.GetAllPlaneNames()) != len(objUnit.PlaneMap) {
		t.Fatalf("plane name index has %d entries, map has %d", len(objUnit.GetAllPlaneNames()), len(objUnit.PlaneMap))
	}
	for name, current := range objUnit.PlaneMap {
		assertNonNegativePower(t, "plane "+name, current.CombatPower)
	}
	for name, current := range objUnit.ShipMap {
		assertNonNegativePower(t, "ship "+name, current.CombatPower)
	}

	plane := objUnit.PlaneMap["F4F-3"]
	if plane == nil || plane.CombatPower.Total <= 0 {
		t.Fatalf("F4F-3 combat power not initialized: %+v", plane)
	}
	carrier := objUnit.ShipMap["yorktown"]
	if carrier == nil || carrier.CombatPower.Aviation <= 0 {
		t.Fatalf("yorktown aviation power not initialized: %+v", carrier)
	}
	if carrier.CombatPower.Total != carrier.CombatPower.Hull+carrier.CombatPower.Aviation {
		t.Fatalf("yorktown combat power has inconsistent breakdown: %+v", carrier.CombatPower)
	}
}

func assertNonNegativePower(t *testing.T, name string, power objUnit.CombatPowerInfo) {
	t.Helper()
	for field, value := range map[string]int{
		"total": power.Total, "antiShip": power.AntiShip, "antiAir": power.AntiAir,
		"survival": power.Survival, "mobility": power.Mobility,
		"range": power.Range, "burst": power.Burst,
		"hull": power.Hull, "aviation": power.Aviation,
	} {
		if value < 0 {
			t.Fatalf("%s %s power is negative: %+v", name, field, power)
		}
	}
}

func TestDrawCollectionUIAtCommonSizes(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		category collectionCategory
		object   string
	}{
		{name: "ship 720p", width: 1280, height: 720, category: collectionCategoryShip, object: "lowa"},
		{name: "plane 720p", width: 1280, height: 720, category: collectionCategoryPlane, object: "F"},
		{name: "ship 1080p", width: 1920, height: 1080, category: collectionCategoryShip, object: "lowa"},
		{name: "plane 1080p", width: 1920, height: 1080, category: collectionCategoryPlane, object: "F"},
		{name: "ship 2048", width: 2048, height: 1152, category: collectionCategoryShip, object: "lowa"},
		{name: "plane 2048", width: 2048, height: 1152, category: collectionCategoryPlane, object: "F"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			screen := ebiten.NewImage(tt.width, tt.height)
			ui := NewCollectionUI(NewDrawer())
			ui.category = tt.category
			if tt.category == collectionCategoryShip {
				ui.curShip = tt.object
			}
			ui.Draw(screen)
		})
	}
}

func TestCollectionUILayoutStaysInsideScreen(t *testing.T) {
	for _, size := range [][2]int{{1280, 720}, {1920, 1080}, {2048, 1152}} {
		layout := calculateCollectionUILayout(size[0], size[1])
		for name, rect := range map[string]image.Rectangle{
			"blueprint": layout.Blueprint, "toolbar": layout.Toolbar, "plane toolbar": layout.PlaneToolbar,
			"plane viewport": layout.PlaneViewport, "vertical slider": layout.VSlider,
			"horizontal slider": layout.HSlider,
		} {
			if rect.Min.X < 0 || rect.Min.Y < 0 || rect.Max.X > size[0] || rect.Max.Y > size[1] {
				t.Fatalf("%s %v outside %dx%d screen", name, rect, size[0], size[1])
			}
		}
	}
}

func TestCollectionWidgetActions(t *testing.T) {
	ui := NewCollectionUI(NewDrawer())
	ui.resize(1920, 1080)
	event.ExecuteDeferred()

	oldRoot := ui.Container()
	ui.tabButtons[collectionCategoryPlane].Click()
	event.ExecuteDeferred()
	if ui.category != collectionCategoryPlane || !ui.pendingRebuild {
		t.Fatalf("plane tab did not update state: category=%v pending=%v", ui.category, ui.pendingRebuild)
	}
	ui.applyPendingRebuild()
	if ui.Container() == oldRoot {
		t.Fatal("pending category rebuild did not replace the root container")
	}
	event.ExecuteDeferred()

	ui.tabButtons[collectionCategoryShip].Click()
	event.ExecuteDeferred()
	ui.applyPendingRebuild()
	event.ExecuteDeferred()
	ui.nationCombo.EntrySelectedEvent.Fire(&widget.ListComboButtonEntrySelectedEventArgs{
		Button: ui.nationCombo, Entry: objUnit.NationUS, PreviousEntry: objUnit.NationAll,
	})
	event.ExecuteDeferred()
	ui.applyPendingRebuild()
	event.ExecuteDeferred()
	if ui.shipNation != objUnit.NationUS {
		t.Fatalf("nation combo selected %q, want US", ui.shipNation)
	}

	ui.typeCombo.EntrySelectedEvent.Fire(&widget.ListComboButtonEntrySelectedEventArgs{
		Button: ui.typeCombo, Entry: shipClassCruiser, PreviousEntry: shipClassAll,
	})
	event.ExecuteDeferred()
	ui.applyPendingRebuild()
	event.ExecuteDeferred()
	ships := ui.filteredShips()
	if len(ships) < 2 {
		t.Fatalf("expected at least two filtered ships, got %d", len(ships))
	}
	target := ships[1]
	option := collectionNameOption{Name: target.Name}
	ui.nameCombo.EntrySelectedEvent.Fire(&widget.ListComboButtonEntrySelectedEventArgs{
		Button: ui.nameCombo, Entry: option,
	})
	event.ExecuteDeferred()
	ui.applyPendingRebuild()
	event.ExecuteDeferred()
	if ui.curShip != target.Name {
		t.Fatalf("name combo selected %q, want %q", ui.curShip, target.Name)
	}

	before := ui.curShip
	ui.nextButton.Click()
	event.ExecuteDeferred()
	ui.applyPendingRebuild()
	event.ExecuteDeferred()
	if ui.curShip == before {
		t.Fatal("next button did not move to another filtered ship")
	}
	ui.previousButton.Click()
	event.ExecuteDeferred()
	ui.applyPendingRebuild()
	if ui.curShip != before {
		t.Fatalf("previous button returned to %q, want %q", ui.curShip, before)
	}
}

func TestCollectionControlsHandleRealMouseSequence(t *testing.T) {
	cursor := &collectionTestCursor{}
	uiInput.SetCursorUpdater(cursor)
	t.Cleanup(func() { uiInput.SetCursorUpdater(nil) })

	collection := NewCollectionUI(NewDrawer())
	collection.resize(2048, 1152)
	host := &ebitenui.UI{Container: collection.Container(), DisableDefaultFocus: true}
	screen := ebiten.NewImage(2048, 1152)
	host.Draw(screen)

	click := func(rect image.Rectangle) {
		if rect.Empty() {
			t.Fatal("control has an empty input rectangle")
		}
		cursor.position = image.Pt((rect.Min.X+rect.Max.X)/2, (rect.Min.Y+rect.Max.Y)/2)
		cursor.pressed = true
		host.Update()
		cursor.pressed = false
		host.Update()
	}

	click(collection.tabButtons[collectionCategoryPlane].GetWidget().Rect)
	if collection.category != collectionCategoryPlane {
		t.Fatal("real mouse press/release did not select the plane category")
	}
	collection.applyPendingRebuild()
	host.Container = collection.Container()
	host.Draw(screen)

	click(collection.nationCombo.GetWidget().Rect)
	if !collection.nationCombo.ContentVisible() {
		t.Fatal("real mouse press/release did not open the nation combo")
	}
	collection.nationCombo.SetContentVisible(false)
	click(collection.typeCombo.GetWidget().Rect)
	if !collection.typeCombo.ContentVisible() {
		t.Fatal("real mouse press/release did not open the plane type combo")
	}
}

func TestCollectionNavigationWrapsAndComboBlocksWheel(t *testing.T) {
	ui := NewCollectionUI(NewDrawer())
	ships := ui.filteredShips()
	if len(ships) < 2 {
		t.Fatal("expected multiple ships")
	}
	ui.curShip = ships[0].Name
	ui.moveShip(-1)
	if ui.curShip != ships[len(ships)-1].Name {
		t.Fatalf("previous navigation selected %q, want last ship", ui.curShip)
	}

	if ui.comboExpanded() {
		t.Fatal("combo unexpectedly expanded initially")
	}
	ui.toggleDropdown(collectionDropdownNation)
	if !ui.comboExpanded() {
		t.Fatal("expanded combo was not detected")
	}
	ui.toggleDropdown(collectionDropdownNation)
	if ui.comboExpanded() {
		t.Fatal("closed combo still blocks wheel navigation")
	}
}

func TestManualToolbarFallbackSelectsCategoryAndFilters(t *testing.T) {
	collection := NewCollectionUI(NewDrawer())
	collection.resize(2048, 1152)
	host := &ebitenui.UI{Container: collection.Container(), DisableDefaultFocus: true}
	screen := ebiten.NewImage(2048, 1152)
	host.Draw(screen)

	center := func(rect image.Rectangle) image.Point {
		return image.Pt((rect.Min.X+rect.Max.X)/2, (rect.Min.Y+rect.Max.Y)/2)
	}
	collection.handleToolbarClick(center(collection.tabButtons[collectionCategoryPlane].GetWidget().Rect))
	collection.applyPendingRebuild()
	host.Container = collection.Container()
	host.Draw(screen)
	if collection.category != collectionCategoryPlane {
		t.Fatal("manual toolbar fallback did not switch to planes")
	}

	collection.handleToolbarClick(center(collection.nationCombo.GetWidget().Rect))
	listRect := collection.dropdownListRect(collectionDropdownNation)
	jpRow := 2
	collection.handleToolbarClick(image.Pt(
		listRect.Min.X+10, listRect.Min.Y+jpRow*collection.dropdownRowHeight()+collection.dropdownRowHeight()/2,
	))
	collection.applyPendingRebuild()
	if collection.planeNation != objUnit.NationJP {
		t.Fatalf("manual nation dropdown selected %q, want JP", collection.planeNation)
	}

	host.Container = collection.Container()
	host.Draw(screen)
	collection.handleToolbarClick(center(collection.typeCombo.GetWidget().Rect))
	typeRect := collection.dropdownListRect(collectionDropdownType)
	fighterRow := 1
	collection.handleToolbarClick(image.Pt(
		typeRect.Min.X+10, typeRect.Min.Y+fighterRow*collection.dropdownRowHeight()+collection.dropdownRowHeight()/2,
	))
	collection.applyPendingRebuild()
	if collection.planeType != planeTypeFighter {
		t.Fatalf("manual type dropdown selected %q, want fighter", collection.planeType)
	}

	host.Container = collection.Container()
	host.Draw(screen)
	collection.handleToolbarClick(center(collection.tabButtons[collectionCategoryShip].GetWidget().Rect))
	collection.applyPendingRebuild()
	host.Container = collection.Container()
	host.Draw(screen)
	collection.handleToolbarClick(center(collection.typeCombo.GetWidget().Rect))
	classRect := collection.dropdownListRect(collectionDropdownType)
	cruiserRow := 3
	collection.handleToolbarClick(image.Pt(
		classRect.Min.X+10, classRect.Min.Y+cruiserRow*collection.dropdownRowHeight()+collection.dropdownRowHeight()/2,
	))
	collection.applyPendingRebuild()
	if collection.shipClass != shipClassCruiser {
		t.Fatalf("manual class dropdown selected %q, want cruiser", collection.shipClass)
	}

	host.Container = collection.Container()
	host.Draw(screen)
	ships := collection.filteredShips()
	if len(ships) < 2 {
		t.Fatal("expected at least two cruisers for name selection")
	}
	collection.handleToolbarClick(center(collection.nameCombo.GetWidget().Rect))
	nameRect := collection.dropdownListRect(collectionDropdownName)
	nameRow := 1
	collection.handleToolbarClick(image.Pt(
		nameRect.Min.X+10, nameRect.Min.Y+nameRow*collection.dropdownRowHeight()+collection.dropdownRowHeight()/2,
	))
	collection.applyPendingRebuild()
	if collection.curShip != ships[nameRow].Name {
		t.Fatalf("manual name dropdown selected %q, want %q", collection.curShip, ships[nameRow].Name)
	}
}

func TestShipNavigationInputMapping(t *testing.T) {
	tests := []struct {
		name                  string
		left, up, right, down bool
		wheelY                float64
		wheelAllowed          bool
		want                  int
	}{
		{name: "left", left: true, want: -1},
		{name: "up", up: true, want: -1},
		{name: "right", right: true, want: 1},
		{name: "down", down: true, want: 1},
		{name: "wheel up", wheelY: 1, wheelAllowed: true, want: -1},
		{name: "wheel down", wheelY: -1, wheelAllowed: true, want: 1},
		{name: "blocked wheel", wheelY: -1, wheelAllowed: false, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shipNavigationOffset(
				tt.left, tt.up, tt.right, tt.down, tt.wheelY, tt.wheelAllowed,
			); got != tt.want {
				t.Fatalf("navigation offset = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCollectionMetricsScaleReadably(t *testing.T) {
	base := calculateCollectionMetrics(1280, 720)
	if base.Scale != 1 || base.Body != 18 || base.CardTitle != 22 {
		t.Fatalf("720p metrics = %+v, want readable base sizes", base)
	}
	large := calculateCollectionMetrics(2048, 1152)
	if large.Scale < 1.27 || large.Scale > 1.29 || large.Body < 23 || large.RadarLabel < 20 {
		t.Fatalf("2048x1152 metrics did not scale as expected: %+v", large)
	}
}

func TestGameUsesSingleEbitenUIHost(t *testing.T) {
	oldSettings := config.G
	config.G = &config.GameSettings{SpeedMultiplier: 1}
	t.Cleanup(func() { config.G = oldSettings })
	game := New()
	host := game.ui
	if host == nil || !host.DisableDefaultFocus {
		t.Fatal("shared UI host is missing or still consumes Tab focus")
	}
	game.mode = GameModeCollection
	game.syncUIContainer()
	if game.ui != host || game.ui.Container != game.collectionUI.Container() {
		t.Fatal("collection did not attach to the shared UI host")
	}
	game.mode = GameModeGameSetting
	game.syncUIContainer()
	if game.ui != host || game.ui.Container != game.settingUI.Container() {
		t.Fatal("settings did not attach to the shared UI host")
	}
}

func TestCollectionFilters(t *testing.T) {
	ui := NewCollectionUI(NewDrawer())
	ui.shipNation = objUnit.NationUS
	ui.shipClass = shipClassCruiser
	ships := ui.filteredShips()
	if len(ships) == 0 {
		t.Fatal("expected US cruisers")
	}
	for _, ship := range ships {
		if ship.Nation != objUnit.NationUS || ship.Type != objUnit.ShipTypeCruiser {
			t.Fatalf("unexpected ship in filtered result: %+v", ship)
		}
	}

	ui.planeNation = objUnit.NationJP
	ui.planeType = planeTypeFighter
	planes := ui.filteredPlanes()
	if len(planes) == 0 {
		t.Fatal("expected Japanese fighters")
	}
	for _, plane := range planes {
		if plane.Nation != objUnit.NationJP || plane.Type != objUnit.PlaneTypeFighter {
			t.Fatalf("unexpected plane in filtered result: %+v", plane)
		}
	}
}

func TestInitializedUnitsHaveSupportedNation(t *testing.T) {
	supported := map[objUnit.Nation]bool{}
	for _, nation := range objUnit.AvailableNations() {
		if nation != objUnit.NationAll {
			supported[nation] = true
		}
	}
	for name, ship := range objUnit.ShipMap {
		if !supported[ship.Nation] {
			t.Errorf("ship %s has unsupported nation %q", name, ship.Nation)
		}
	}
	for name, plane := range objUnit.PlaneMap {
		if !supported[plane.Nation] {
			t.Errorf("plane %s has unsupported nation %q", name, plane.Nation)
		}
	}
}

func TestAbilityDimensionsRemainDataDriven(t *testing.T) {
	if len(collectionAbilityDimensions) != 6 {
		t.Fatalf("ability dimensions = %d, want initial six axes", len(collectionAbilityDimensions))
	}
	seen := map[string]bool{}
	for _, dimension := range collectionAbilityDimensions {
		if dimension.ID == "" || dimension.Label == "" || dimension.Value == nil || seen[dimension.ID] {
			t.Fatalf("invalid ability dimension: %+v", dimension)
		}
		seen[dimension.ID] = true
	}
}

func TestAbilityScalesUseNinetyFifthPercentile(t *testing.T) {
	powers := make([]objUnit.CombatPowerInfo, 20)
	for idx := range powers {
		powers[idx].AntiShip = idx + 1
	}
	powers[19].AntiShip = 1000
	scales := calculateAbilityScales(powers)
	if got := scales["anti_ship"]; got != 19 {
		t.Fatalf("anti-ship scale = %.0f, want 95th percentile 19", got)
	}
}
