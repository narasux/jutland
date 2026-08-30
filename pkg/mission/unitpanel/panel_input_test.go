package unitpanel

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
)

// testRegion 是输入测试使用的滚动视口（原点为 0,0，局部坐标与屏幕坐标一致）。
var testRegion = Rect{X: 0, Y: 0, W: 280, H: 480}

func TestDefaultTabFollowsFocusedShipType(t *testing.T) {
	missionState := panelInteractionState()
	carrier := &objUnit.BattleShip{Uid: "carrier", CurHP: 1, Type: objUnit.ShipTypeAircraftCarrier}
	battleship := &objUnit.BattleShip{Uid: "battleship", CurHP: 1, Type: objUnit.ShipTypeBattleShip}
	missionState.Arena.Ships = map[string]*objUnit.BattleShip{carrier.Uid: carrier, battleship.Uid: battleship}
	missionState.Interaction.SelectedShips = []string{carrier.Uid}
	missionState.Interaction.FocusedShipUid = carrier.Uid
	panel := New()
	panel.updateWithPointer(missionState, testRegion, 0, pointerInput{})
	if panel.tab != TabAircraft {
		t.Fatalf("carrier default tab = %v, want aircraft", panel.tab)
	}

	missionState.Interaction.SelectedShips = []string{battleship.Uid}
	missionState.Interaction.FocusedShipUid = battleship.Uid
	panel.updateWithPointer(missionState, testRegion, 0, pointerInput{})
	if panel.tab != TabWeapons {
		t.Fatalf("battleship default tab = %v, want weapons", panel.tab)
	}
}

func TestTargetButtonProducesExplicitCenterActionWithoutChangingFocus(t *testing.T) {
	missionState := panelInteractionState()
	ship := &objUnit.BattleShip{Uid: "ship", CurHP: 1, AttackTarget: "target"}
	target := &objUnit.BattleShip{Uid: "target", CurHP: 1}
	missionState.Arena.Ships = map[string]*objUnit.BattleShip{ship.Uid: ship, target.Uid: target}
	missionState.Interaction.SelectedShips = []string{ship.Uid}
	missionState.Interaction.FocusedShipUid = ship.Uid
	panel := New()
	panel.updateWithPointer(missionState, testRegion, 0, pointerInput{})
	button := panel.targetButtonRect()
	actions := panel.updateWithPointer(missionState, testRegion, 0, pointerInput{
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
	carrier := &objUnit.BattleShip{
		Uid:      "carrier",
		CurHP:    1,
		Aircraft: objUnit.ShipAircraft{HasPlane: true, Disable: true},
	}
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
