package sidebar

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/state"
	mapcfg "github.com/narasux/jutland/pkg/resources/mapcfg"
	"github.com/narasux/jutland/pkg/utils/layout"
)

func newTestPanel(tab Tab, mapAspect float64) *Panel {
	return &Panel{tab: tab, mapAspect: mapAspect}
}

func TestExpandedPanelConsumesPanelHandleTabsAndSettings(t *testing.T) {
	for _, tab := range []Tab{TabBattle, TabSettings} {
		missionState := &state.MissionState{
			Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
			View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
			UI:   state.MissionUIState{SidebarExpanded: true},
		}
		panel := newTestPanel(tab, 1)
		ui := calcLayout(missionState.View.Layout, true, tab, panel.mapAspect)
		points := []struct {
			name string
			x, y int
		}{
			{name: "panel", x: int(ui.Panel.X + 10), y: int(ui.Panel.Y + 10)},
			{name: "handle", x: int(ui.Handle.X + ui.Handle.W/2), y: int(ui.Handle.Y + ui.Handle.H/2)},
			{name: "tab bar", x: int(ui.TabBar.X + ui.TabBar.W/2), y: int(ui.TabBar.Y + ui.TabBar.H/2)},
			{name: "minimap", x: int(ui.Map.X + ui.Map.W/2), y: int(ui.Map.Y + ui.Map.H/2)},
			{name: "settings area", x: int(ui.Panel.X + 30), y: int(ui.TabBar.Y + ui.TabBar.H + 40)},
		}
		for _, point := range points {
			if !panel.consumesCursorAt(missionState, point.x, point.y) {
				t.Fatalf("tab %v: %s at (%d,%d) did not consume cursor", tab, point.name, point.x, point.y)
			}
		}
		// 面板停靠右上角：面板下方与右缘细缝的战场区域不应被消费
		if panel.consumesCursorAt(missionState, 100, 100) {
			t.Fatalf("tab %v: battlefield point outside sidebar consumed cursor", tab)
		}
		belowPanelY := int(ui.Panel.Y + ui.Panel.H + 10)
		if panel.consumesCursorAt(missionState, 1280-10, belowPanelY) {
			t.Fatalf("tab %v: battlefield point below panel consumed cursor", tab)
		}
		// 右缘 12px 细缝留给镜头边缘滚动
		if panel.consumesCursorAt(missionState, 1280-4, 360) {
			t.Fatalf("tab %v: right screen gap consumed cursor", tab)
		}
	}
}

func TestCollapsedPanelOnlyConsumesHandle(t *testing.T) {
	missionState := &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		UI:   state.MissionUIState{SidebarExpanded: false},
	}
	panel := newTestPanel(TabBattle, 1)
	ui := calcLayout(missionState.View.Layout, false, TabBattle, panel.mapAspect)

	if ui.Panel.H != 0 || ui.Viewport.W != 0 {
		t.Fatalf("collapsed panel = %+v, want empty panel and viewport", ui.Panel)
	}
	if ui.Handle.Y > panelMarginTop+handleGap+1 {
		t.Fatalf("collapsed handle Y = %v, want near screen top", ui.Handle.Y)
	}
	if !panel.consumesCursorAt(
		missionState,
		int(ui.Handle.X+ui.Handle.W/2),
		int(ui.Handle.Y+ui.Handle.H/2),
	) {
		t.Fatal("collapsed handle did not consume cursor")
	}
	if panel.consumesCursorAt(missionState, int(ui.Panel.X+10), int(ui.Panel.Y+10)) {
		t.Fatal("collapsed panel area consumed cursor")
	}
}

func TestCalcLayoutDockedPanelGeometry(t *testing.T) {
	ui := calcLayout(layout.ScreenLayout{Width: 1280, Height: 720}, true, TabBattle, 1)
	if ui.Panel.W < 260 || ui.Panel.W > 360 {
		t.Fatalf("panel width = %v, want within [260, 360]", ui.Panel.W)
	}
	// 面板贴角停靠：顶缘与右缘只留 12px 细缝，保证镜头边缘滚动可用
	if ui.Panel.Y != panelMarginTop {
		t.Fatalf("panel Y = %v, want %v", ui.Panel.Y, panelMarginTop)
	}
	if gap := 1280 - (ui.Panel.X + ui.Panel.W); gap != panelMarginRight {
		t.Fatalf("right gap = %v, want %v", gap, panelMarginRight)
	}
	if ui.Panel.H <= 0 || ui.Panel.Y+ui.Panel.H >= 720 {
		t.Fatalf("panel bottom = %v, want inside screen", ui.Panel.Y+ui.Panel.H)
	}
	// 把手挂在面板底缘下方
	if ui.Handle.Y < ui.Panel.Y+ui.Panel.H {
		t.Fatalf("handle Y = %v, want below panel bottom %v", ui.Handle.Y, ui.Panel.Y+ui.Panel.H)
	}
	if ui.Viewport.H <= 0 {
		t.Fatalf("viewport height = %v, want positive", ui.Viewport.H)
	}
}

func TestCalcLayoutKeepsMapAspectForNonSquareMap(t *testing.T) {
	const (
		screenW = 1280.0
		screenH = 1440.0
		aspect  = 128.0 / 192.0 // 珍珠港等竖长地图
	)
	ui := calcLayout(layout.ScreenLayout{Width: int(screenW), Height: int(screenH)}, true, TabBattle, aspect)

	got := ui.Map.W / ui.Map.H
	if diff := got - aspect; diff < -1e-6 || diff > 1e-6 {
		t.Fatalf("minimap aspect = %v, want %v", got, aspect)
	}
	maxW := ui.Panel.W - 32
	if ui.Map.W != maxW {
		t.Fatalf("minimap width = %v, want %v (under height cap)", ui.Map.W, maxW)
	}
	// 水平居中于 (panelX+16, panelW-32) 的内容区
	wantX := ui.Panel.X + 16 + (maxW-ui.Map.W)/2
	if ui.Map.X != wantX {
		t.Fatalf("minimap X = %v, want %v", ui.Map.X, wantX)
	}
}

func TestCalcLayoutCapsOversizedMapHeight(t *testing.T) {
	const screenH = 720.0
	ui := calcLayout(layout.ScreenLayout{Width: 1280, Height: int(screenH)}, true, TabBattle, 128.0/192.0)

	maxH := screenH * mapMaxHeightRatio
	if ui.Map.H > maxH {
		t.Fatalf("minimap height = %v, want <= %v", ui.Map.H, maxH)
	}
	got := ui.Map.W / ui.Map.H
	const aspect = 128.0 / 192.0
	if diff := got - aspect; diff < -1e-6 || diff > 1e-6 {
		t.Fatalf("minimap aspect = %v, want %v (capped height must keep ratio)", got, aspect)
	}
	if ui.Viewport.H <= 0 {
		t.Fatalf("viewport height = %v, want positive", ui.Viewport.H)
	}
}

func TestTabAtDetectsTabs(t *testing.T) {
	panel := newTestPanel(TabBattle, 1)
	panel.layout = calcLayout(layout.ScreenLayout{Width: 1280, Height: 720}, true, TabBattle, 1)
	if tab, ok := panel.tabAt(int(panel.layout.TabBar.X+10), int(panel.layout.TabBar.Y+10)); !ok || tab != TabBattle {
		t.Fatalf("left tab = %v, ok %v", tab, ok)
	}
	if tab, ok := panel.tabAt(int(panel.layout.TabBar.X+panel.layout.TabBar.W-10), int(panel.layout.TabBar.Y+10)); !ok ||
		tab != TabSettings {
		t.Fatalf("right tab = %v, ok %v", tab, ok)
	}
	if _, ok := panel.tabAt(int(panel.layout.TabBar.X), int(panel.layout.TabBar.Y+panel.layout.TabBar.H+5)); ok {
		t.Fatal("point below tab bar should not be a tab")
	}
}

func TestUpdateSettingsTogglesGameOptions(t *testing.T) {
	ms := &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		UI: state.MissionUIState{
			SidebarExpanded: true,
			GameOpts:        state.GameOptions{ForceDisplayState: true, DisplayDamageNumber: true},
		},
	}
	panel := newTestPanel(TabSettings, 1)
	panel.layout = calcLayout(ms.View.Layout, true, TabSettings, 1)

	row0 := panel.settingsRowRect(0)
	panel.updateSettings(ms, int(row0.X+row0.W/2), int(row0.Y+row0.H/2), true)
	if ms.UI.GameOpts.ForceDisplayState {
		t.Fatal("clicking row 0 did not toggle ForceDisplayState")
	}
	if !ms.UI.GameOpts.DisplayDamageNumber {
		t.Fatal("row 1 should be untouched")
	}

	row1 := panel.settingsRowRect(1)
	panel.updateSettings(ms, int(row1.X+row1.W/2), int(row1.Y+row1.H/2), true)
	if ms.UI.GameOpts.DisplayDamageNumber {
		t.Fatal("clicking row 1 did not toggle DisplayDamageNumber")
	}
}

func TestCenterCameraAtMinimapCentersClickedPoint(t *testing.T) {
	const (
		screenW = 1280.0
		screenH = 720.0
		mapW    = 128.0
		mapH    = 192.0
	)
	ms := &state.MissionState{
		Core: state.MissionCoreState{
			MissionStatus: state.MissionRunning,
			MissionMD:     metadata.MissionMetadata{MapCfg: &mapcfg.MapCfg{Width: int(mapW), Height: int(mapH)}},
		},
		View: state.MissionViewState{
			Layout: layout.ScreenLayout{Width: int(screenW), Height: int(screenH)},
			Camera: state.Camera{Width: 11, Height: 7},
		},
	}
	panel := newTestPanel(TabBattle, mapW/mapH)
	panel.layout = calcLayout(ms.View.Layout, true, TabBattle, panel.mapAspect)

	// 点击小地图某点：该点应落在镜头视野中心
	cx := int(panel.layout.Map.X + panel.layout.Map.W/2)
	cy := int(panel.layout.Map.Y + panel.layout.Map.H/2)
	panel.centerCameraAtMinimap(ms, cx, cy)

	clickedRX := (float64(cx) - panel.layout.Map.X) / panel.layout.Map.W * mapW
	wantRX := clickedRX - float64(ms.View.Camera.Width)/2
	// 点击坐标取整会引入至多 1 屏幕像素（约 0.6 地图格）的偏差
	if diff := math.Abs(ms.View.Camera.Pos.RX - wantRX); diff > 1 {
		t.Fatalf("camera RX = %v, want %v (clicked point centered on camera)", ms.View.Camera.Pos.RX, wantRX)
	}
	if ms.View.Camera.Pos.RX < 0 {
		t.Fatalf("camera RX = %v, want clamped >= 0", ms.View.Camera.Pos.RX)
	}
}
