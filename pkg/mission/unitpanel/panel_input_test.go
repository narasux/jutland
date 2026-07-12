package unitpanel

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
)

func TestHandleClickExpandsAndCollapsesPanel(t *testing.T) {
	missionState := panelInteractionState()
	panel := New()

	collapsed := calcLayout(missionState.View.Layout, false, 0).Handle
	panel.updateWithPointer(missionState, 0, pointerInput{
		X:           int(collapsed.X + collapsed.W/2),
		Y:           int(collapsed.Y + collapsed.H/2),
		JustPressed: true,
	})
	if !missionState.UI.UnitPanelExpanded {
		t.Fatal("clicking the visible collapsed handle did not expand the panel")
	}

	expanded := calcLayout(missionState.View.Layout, true, 0).Handle
	panel.updateWithPointer(missionState, 0, pointerInput{
		X:           int(expanded.X + expanded.W/2),
		Y:           int(expanded.Y + expanded.H/2),
		JustPressed: true,
	})
	if missionState.UI.UnitPanelExpanded {
		t.Fatal("clicking the visible expanded handle did not collapse the panel")
	}
}

func TestHandleDoesNotTriggerOutsideRenderedBounds(t *testing.T) {
	missionState := panelInteractionState()
	panel := New()
	handle := calcLayout(missionState.View.Layout, false, 0).Handle
	panel.updateWithPointer(missionState, 0, pointerInput{
		X:           int(handle.X) - 1,
		Y:           int(handle.Y + handle.H/2),
		JustPressed: true,
	})
	if missionState.UI.UnitPanelExpanded {
		t.Fatal("click outside the rendered handle expanded the panel")
	}
	if !panel.consumesCursorAt(missionState, 0, int(handle.X+1), int(handle.Y+1)) {
		t.Fatal("cursor inside rendered handle was not consumed")
	}
	if panel.consumesCursorAt(missionState, 0, int(handle.X)-1, int(handle.Y+1)) {
		t.Fatal("cursor outside rendered handle was consumed")
	}
}

func TestRenderedHandleAndHitRegionShareRectangle(t *testing.T) {
	missionState := panelInteractionState()
	panel := New()
	panel.updateWithPointer(missionState, 0, pointerInput{})
	if len(panel.hits) == 0 || panel.hits[0].Kind != hitHandle {
		t.Fatal("handle hit region was not built")
	}
	if panel.hits[0].Rect != panel.layout.Handle {
		t.Fatalf("hit rect = %+v, rendered rect = %+v", panel.hits[0].Rect, panel.layout.Handle)
	}
}

func TestDefaultTabFollowsFocusedShipType(t *testing.T) {
	missionState := panelInteractionState()
	carrier := &objUnit.BattleShip{Uid: "carrier", CurHP: 1, Type: objUnit.ShipTypeAircraftCarrier}
	battleship := &objUnit.BattleShip{Uid: "battleship", CurHP: 1, Type: objUnit.ShipTypeBattleShip}
	missionState.Arena.Ships = map[string]*objUnit.BattleShip{carrier.Uid: carrier, battleship.Uid: battleship}
	missionState.Interaction.SelectedShips = []string{carrier.Uid}
	missionState.Interaction.FocusedShipUid = carrier.Uid
	panel := New()
	panel.updateWithPointer(missionState, 0, pointerInput{})
	if panel.tab != TabAircraft {
		t.Fatalf("carrier default tab = %v, want aircraft", panel.tab)
	}

	missionState.Interaction.SelectedShips = []string{battleship.Uid}
	missionState.Interaction.FocusedShipUid = battleship.Uid
	panel.updateWithPointer(missionState, 0, pointerInput{})
	if panel.tab != TabWeapons {
		t.Fatalf("battleship default tab = %v, want weapons", panel.tab)
	}
}

func TestTargetButtonProducesExplicitCenterActionWithoutChangingFocus(t *testing.T) {
	missionState := panelInteractionState()
	missionState.UI.UnitPanelExpanded = true
	ship := &objUnit.BattleShip{Uid: "ship", CurHP: 1, AttackTarget: "target"}
	target := &objUnit.BattleShip{Uid: "target", CurHP: 1}
	missionState.Arena.Ships = map[string]*objUnit.BattleShip{ship.Uid: ship, target.Uid: target}
	missionState.Interaction.SelectedShips = []string{ship.Uid}
	missionState.Interaction.FocusedShipUid = ship.Uid
	panel := New()
	panel.updateWithPointer(missionState, 0, pointerInput{})
	button := panel.targetButtonRect()
	actions := panel.updateWithPointer(missionState, 0, pointerInput{
		X:           int(button.X + button.W/2),
		Y:           int(button.Y + button.H/2),
		JustPressed: true,
	})
	if len(actions) != 1 || actions[0].Kind != ActionCenterTarget || actions[0].TargetUid != target.Uid {
		t.Fatalf("actions = %+v", actions)
	}
	if missionState.Interaction.FocusedShipUid != ship.Uid {
		t.Fatalf("target action changed focus to %q", missionState.Interaction.FocusedShipUid)
	}
}

func TestAircraftActionIgnoresShipsWithoutAircraft(t *testing.T) {
	carrier := &objUnit.BattleShip{Uid: "carrier", CurHP: 1, Aircraft: objUnit.ShipAircraft{HasPlane: true, Disable: true}}
	escort := &objUnit.BattleShip{Uid: "escort", CurHP: 1}
	missionState := panelTestState(carrier, escort)
	action := New().aircraftAction(missionState)
	if !action.Enable || len(action.ShipUids) != 1 || action.ShipUids[0] != carrier.Uid {
		t.Fatalf("action = %+v", action)
	}
}

func panelInteractionState() *state.MissionState {
	return &state.MissionState{
		Core:  state.MissionCoreState{MissionStatus: state.MissionRunning},
		View:  state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		Arena: state.MissionArenaState{Ships: map[string]*objUnit.BattleShip{}, Planes: map[string]*objUnit.Plane{}},
	}
}
