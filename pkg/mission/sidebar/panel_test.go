package sidebar

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
)

func TestExpandedSidebarConsumesPanelHandleTabsAndSettings(t *testing.T) {
	for _, tab := range []Tab{TabBattle, TabSettings} {
		missionState := &state.MissionState{
			Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
			View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
			UI:   state.MissionUIState{SidebarExpanded: true},
		}
		panel := &Panel{tab: tab}
		ui := calcLayout(missionState.View.Layout, true, tab)
		points := []struct {
			name string
			x, y int
		}{
			{name: "panel", x: int(ui.Panel.X + 10), y: 10},
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
		if panel.consumesCursorAt(missionState, 100, 100) {
			t.Fatalf("tab %v: battlefield point outside sidebar consumed cursor", tab)
		}
	}
}

func TestCalcLayoutExpandedWidthMatchesSidebar(t *testing.T) {
	ui := calcLayout(layout.ScreenLayout{Width: 1280, Height: 720}, true, TabBattle)
	if ui.Panel.W < 260 || ui.Panel.W > 360 {
		t.Fatalf("panel width = %v, want within [260, 360]", ui.Panel.W)
	}
	if ui.Panel.X != 1280-ui.Panel.W {
		t.Fatalf("panel X = %v, want %v", ui.Panel.X, 1280-ui.Panel.W)
	}
	if ui.Viewport.H <= 0 {
		t.Fatalf("viewport height = %v, want positive", ui.Viewport.H)
	}
}

func TestTabAtDetectsTabs(t *testing.T) {
	panel := &Panel{tab: TabBattle}
	panel.layout = calcLayout(layout.ScreenLayout{Width: 1280, Height: 720}, true, TabBattle)
	if tab, ok := panel.tabAt(int(panel.layout.TabBar.X+10), int(panel.layout.TabBar.Y+10)); !ok || tab != TabBattle {
		t.Fatalf("left tab = %v, ok %v", tab, ok)
	}
	if tab, ok := panel.tabAt(int(panel.layout.TabBar.X+panel.layout.TabBar.W-10), int(panel.layout.TabBar.Y+10)); !ok || tab != TabSettings {
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
		UI: state.MissionUIState{SidebarExpanded: true,
			GameOpts: state.GameOptions{ForceDisplayState: true, DisplayDamageNumber: true}},
	}
	panel := &Panel{tab: TabSettings}
	panel.layout = calcLayout(ms.View.Layout, true, TabSettings)

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
