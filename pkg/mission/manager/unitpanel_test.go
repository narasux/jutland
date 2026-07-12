package manager

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/sidebar"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/unitpanel"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
	"github.com/narasux/jutland/pkg/utils/layout"
)

func TestCenteredCameraPosUsesVisibleBattlefield(t *testing.T) {
	missionState := &state.MissionState{
		Core: state.MissionCoreState{MissionMD: metadata.MissionMetadata{MapCfg: &mapcfg.MapCfg{Width: 1000, Height: 1000}}},
		View: state.MissionViewState{
			Layout: layout.ScreenLayout{Width: 1280, Height: 720},
			Camera: state.Camera{Width: 33, Height: 19},
		},
		UI: state.MissionUIState{GameOpts: state.GameOptions{Zoom: state.DefaultZoom()}},
	}
	target := objPos.NewR(100, 80)
	result := centeredCameraPos(missionState, target, 300, 240)
	blockSize := missionState.MapBlockDisplaySize()
	wantX := target.RX - (1280.0-300)/blockSize/2
	wantY := target.RY - (720.0-240)/blockSize/2
	if math.Abs(result.RX-wantX) > 1e-9 || math.Abs(result.RY-wantY) > 1e-9 {
		t.Fatalf("camera = (%v,%v), want (%v,%v)", result.RX, result.RY, wantX, wantY)
	}
	if missionState.UI.GameOpts.Zoom != state.DefaultZoom() {
		t.Fatal("centering changed zoom")
	}
}

func TestCenteredCameraPosClampsAtMapBorder(t *testing.T) {
	missionState := &state.MissionState{
		Core: state.MissionCoreState{MissionMD: metadata.MissionMetadata{MapCfg: &mapcfg.MapCfg{Width: 100, Height: 100}}},
		View: state.MissionViewState{
			Layout: layout.ScreenLayout{Width: 1280, Height: 720},
			Camera: state.Camera{Width: 33, Height: 19},
		},
		UI: state.MissionUIState{GameOpts: state.GameOptions{Zoom: state.DefaultZoom()}},
	}
	result := centeredCameraPos(missionState, objPos.NewR(1, 1), 0, 0)
	if result.RX != 0 || result.RY != 0 {
		t.Fatalf("camera = (%v,%v), want map origin", result.RX, result.RY)
	}
}

func TestGameCameraDoesNotMoveWhileUICapturesCursor(t *testing.T) {
	missionManager := &MissionManager{state: &state.MissionState{
		UI: state.MissionUIState{UIConsumesCursor: true},
	}}
	if next := missionManager.getNextCameraPosInGameMode(); next != nil {
		t.Fatalf("next camera position = %v, want nil while UI captures cursor", next)
	}
}

func TestFocusShipActionDoesNotMoveCameraOrClearSelection(t *testing.T) {
	first := &objUnit.BattleShip{Uid: "first", CurHP: 100}
	second := &objUnit.BattleShip{Uid: "second", CurHP: 100}
	initialCamera := objPos.NewR(12.5, 18.5)
	missionState := &state.MissionState{
		View: state.MissionViewState{Camera: state.Camera{Pos: initialCamera}},
		Interaction: state.MissionInteractionState{
			SelectedShips:  []string{first.Uid, second.Uid},
			FocusedShipUid: first.Uid,
		},
		Arena: state.MissionArenaState{Ships: map[string]*objUnit.BattleShip{
			first.Uid: first, second.Uid: second,
		}},
	}
	missionManager := &MissionManager{state: missionState}
	missionManager.handleUnitPanelActions([]unitpanel.Action{{
		Kind: unitpanel.ActionFocusShip, FocusUid: second.Uid,
	}})
	if missionState.Interaction.FocusedShipUid != second.Uid {
		t.Fatalf("focused ship = %q, want %q", missionState.Interaction.FocusedShipUid, second.Uid)
	}
	if missionState.View.Camera.Pos != initialCamera {
		t.Fatalf("camera moved from %+v to %+v", initialCamera, missionState.View.Camera.Pos)
	}
	if len(missionState.Interaction.SelectedShips) != 2 {
		t.Fatalf("selection changed: %v", missionState.Interaction.SelectedShips)
	}
}

func TestSyncFocusedShipDoesNotMoveCamera(t *testing.T) {
	ship := &objUnit.BattleShip{Uid: "ship", Name: "ship", CurHP: 100}
	initialCamera := objPos.NewR(7.25, 9.75)
	missionState := &state.MissionState{
		View:        state.MissionViewState{Camera: state.Camera{Pos: initialCamera}},
		Interaction: state.MissionInteractionState{SelectedShips: []string{ship.Uid}},
		Arena:       state.MissionArenaState{Ships: map[string]*objUnit.BattleShip{ship.Uid: ship}},
	}
	missionManager := &MissionManager{state: missionState}
	missionManager.syncFocusedShip()
	if missionState.Interaction.FocusedShipUid != ship.Uid {
		t.Fatalf("focused ship = %q, want %q", missionState.Interaction.FocusedShipUid, ship.Uid)
	}
	if missionState.View.Camera.Pos != initialCamera {
		t.Fatalf("camera moved from %+v to %+v", initialCamera, missionState.View.Camera.Pos)
	}
}

func TestExplicitTargetActionMovesCameraWithoutChangingFocus(t *testing.T) {
	ally := &objUnit.BattleShip{Uid: "ally", CurHP: 100}
	target := &objUnit.BattleShip{Uid: "target", CurHP: 100, CurPos: objPos.NewR(100, 90)}
	missionState := cameraTestState()
	missionState.Interaction.SelectedShips = []string{ally.Uid}
	missionState.Interaction.FocusedShipUid = ally.Uid
	missionState.Arena.Ships = map[string]*objUnit.BattleShip{ally.Uid: ally, target.Uid: target}
	initialCamera := missionState.View.Camera.Pos
	missionManager := &MissionManager{
		state:     missionState,
		sidebar:   &sidebar.Panel{},
		unitPanel: unitpanel.New(),
	}
	missionManager.handleUnitPanelActions([]unitpanel.Action{{
		Kind: unitpanel.ActionCenterTarget, TargetUid: target.Uid,
	}})
	if missionState.View.Camera.Pos == initialCamera {
		t.Fatal("explicit target action did not move camera")
	}
	if missionState.Interaction.FocusedShipUid != ally.Uid || len(missionState.Interaction.SelectedShips) != 1 {
		t.Fatalf("explicit target action changed selection: %+v", missionState.Interaction)
	}
}

func TestResizeUsesLogicalScreenDimensions(t *testing.T) {
	missionState := cameraTestState()
	missionManager := &MissionManager{state: missionState}
	missionManager.Resize(1920, 1080)
	if missionState.View.Layout.Width != 1920 || missionState.View.Layout.Height != 1080 {
		t.Fatalf("layout = %+v", missionState.View.Layout)
	}
	if missionState.View.Camera.Width <= 0 || missionState.View.Camera.Height <= 0 {
		t.Fatalf("camera size was not refreshed: %+v", missionState.View.Camera)
	}
}

func cameraTestState() *state.MissionState {
	return &state.MissionState{
		Core: state.MissionCoreState{MissionMD: metadata.MissionMetadata{
			MapCfg: &mapcfg.MapCfg{Width: 1000, Height: 1000},
		}},
		View: state.MissionViewState{
			Layout: layout.ScreenLayout{Width: 1280, Height: 720},
			Camera: state.Camera{Pos: objPos.NewR(10, 10), Width: 33, Height: 19},
		},
		Arena: state.MissionArenaState{Ships: map[string]*objUnit.BattleShip{}},
		UI:    state.MissionUIState{GameOpts: state.GameOptions{Zoom: state.DefaultZoom()}},
	}
}
