package sidebar

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
)

func TestExpandedSidebarConsumesPanelHandleAndButtons(t *testing.T) {
	missionState := &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		UI:   state.MissionUIState{SidebarExpanded: true},
	}
	panel := &Panel{}
	ui := calcLayout(missionState.View.Layout, true)
	points := []struct {
		name string
		x, y int
	}{
		{name: "panel", x: int(ui.Panel.X + 10), y: 10},
		{name: "handle", x: int(ui.Handle.X + ui.Handle.W/2), y: int(ui.Handle.Y + ui.Handle.H/2)},
		{name: "settings button area", x: int(ui.Panel.X + 30), y: int(ui.Map.Y + ui.Map.H + 150)},
	}
	for _, point := range points {
		if !panel.consumesCursorAt(missionState, point.x, point.y) {
			t.Fatalf("%s at (%d,%d) did not consume cursor", point.name, point.x, point.y)
		}
	}
	if panel.consumesCursorAt(missionState, 100, 100) {
		t.Fatal("battlefield point outside sidebar consumed cursor")
	}
}
