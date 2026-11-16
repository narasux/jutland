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

// TestCenteredCameraPosCentersTargetOnScreen 验证 centeredCameraPos 把地图目标摆到屏幕正中。
// 场景：1000x1000 地图、1280x720 逻辑布局、相机 33x19 格、默认缩放，目标取 (100,80)，
// 该位置离地图边缘足够远，不会触发边界钳制。
// 预期：返回的相机左上角等于 (100-1280/blockSize/2, 80-720/blockSize/2)，误差不超过 1e-9，
// 且缩放档位保持 state.DefaultZoom()；失败意味着居中偏半屏或定位顺带改缩放，点击舰船后目标不在中心。
// 覆盖：按 MapBlockDisplaySize() 计算期望值，分别比较 RX/RY 与缩放字段。
func TestCenteredCameraPosCentersTargetOnScreen(t *testing.T) {
	missionState := &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{MapCfg: &mapcfg.MapCfg{Width: 1000, Height: 1000}},
		},
		View: state.MissionViewState{
			Layout: layout.ScreenLayout{Width: 1280, Height: 720},
			Camera: state.Camera{Width: 33, Height: 19},
		},
		UI: state.MissionUIState{GameOpts: state.GameOptions{Zoom: state.DefaultZoom()}},
	}
	target := objPos.NewR(100, 80)
	result := centeredCameraPos(missionState, target)
	blockSize := missionState.MapBlockDisplaySize()
	wantX := target.RX - 1280.0/blockSize/2
	wantY := target.RY - 720.0/blockSize/2
	if math.Abs(result.RX-wantX) > 1e-9 || math.Abs(result.RY-wantY) > 1e-9 {
		t.Fatalf("camera = (%v,%v), want (%v,%v)", result.RX, result.RY, wantX, wantY)
	}
	if missionState.UI.GameOpts.Zoom != state.DefaultZoom() {
		t.Fatal("centering changed zoom")
	}
}

// TestCenteredCameraPosClampsAtMapBorder 验证目标贴近地图左上角时相机被钳制到地图原点。
// 场景：100x100 小地图、1280x720 布局，目标 (1,1) 居中所需的相机坐标为负数，
// 会越出地图边界，必须由 EnsureBorder(CameraPosBorder()) 截断。
// 预期：返回 (0,0)；失败意味着相机可以移出地图，屏幕会出现黑边或画出地图外的地块。
// 覆盖：复用同一套布局/缩放配置，只断言 result.RX==0 且 result.RY==0。
func TestCenteredCameraPosClampsAtMapBorder(t *testing.T) {
	missionState := &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{MapCfg: &mapcfg.MapCfg{Width: 100, Height: 100}},
		},
		View: state.MissionViewState{
			Layout: layout.ScreenLayout{Width: 1280, Height: 720},
			Camera: state.Camera{Width: 33, Height: 19},
		},
		UI: state.MissionUIState{GameOpts: state.GameOptions{Zoom: state.DefaultZoom()}},
	}
	result := centeredCameraPos(missionState, objPos.NewR(1, 1))
	if result.RX != 0 || result.RY != 0 {
		t.Fatalf("camera = (%v,%v), want map origin", result.RX, result.RY)
	}
}

// TestGameCameraDoesNotMoveWhileUICapturesCursor 验证 UI 接管光标时游戏模式不再触发边缘滚屏。
// 场景：任务状态里 UIConsumesCursor=true，模拟玩家正在拖动侧栏/单位面板，光标停在屏幕边缘。
// 预期：getNextCameraPosInGameMode 返回 nil；失败意味着拖拽 UI 时相机跟着光标漂移，
// 面板操作会顺手把战场视野推走。
// 覆盖：只构造带 UI 状态的最小 MissionState，直接调用取下一帧相机位置的函数并断言空指针。
func TestGameCameraDoesNotMoveWhileUICapturesCursor(t *testing.T) {
	missionManager := &MissionManager{state: &state.MissionState{
		UI: state.MissionUIState{UIConsumesCursor: true},
	}}
	if next := missionManager.getNextCameraPosInGameMode(); next != nil {
		t.Fatalf("next camera position = %v, want nil while UI captures cursor", next)
	}
}

// TestFocusShipActionDoesNotMoveCameraOrClearSelection 验证“聚焦舰船”动作只改焦点、不动相机也不改选中集。
// 场景：单位面板中同时选中 "first"/"second"，焦点原本在 "first"，相机停在 (12.5,18.5)；
// 玩家点击面板里的第二艘舰触发 unitpanel.ActionFocusShip。
// 预期：FocusedShipUid 变成 "second"，Camera.Pos 与初始值相等，SelectedShips 仍为 2 艘；
// 失败意味着点开某艘船会把镜头拽走并清空多选，编队指挥操作被打断。
// 覆盖：构造 Ships 映射与 Interaction 状态，调用 handleUnitPanelActions 后逐项断言。
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

// TestSyncFocusedShipDoesNotMoveCamera 验证选中集变化后 syncFocusedShip 只补齐焦点、不推动相机。
// 场景：只选中 "ship" 一艘，FocusedShipUid 为空（例如选中集刚被框选刷新），
// 相机停在 (7.25,9.75)，调用 syncFocusedShip。
// 预期：焦点被补成 "ship"，Camera.Pos 保持 (7.25,9.75)；失败意味着补焦点时顺手做了居中，
// 每一次选中都会把视野拉向所选舰船。
// 覆盖：单舰 MissionState 直接调用 syncFocusedShip，比较焦点 UID 与相机坐标。
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

// TestExplicitTargetActionMovesCameraWithoutChangingFocus 验证“居中到目标”动作走平滑定位且不改选中与焦点。
// 场景：相机取 cameraTestState，选中并聚焦 "ally"，敌舰 "target" 在 (100,90)；
// 触发 unitpanel.ActionCenterTarget 后先看当帧，再循环 advanceCameraMove 直到收敛。
// 预期：当帧 Camera.Target 非空但 Pos 未变（不瞬移）；收敛后 Pos 等于 centeredCameraPos(target 位置)；
// FocusedShipUid 仍为 "ally"、SelectedShips 仍只有 1 艘。失败意味着居中绕过了缓动，
// 或者查看敌情会替换掉玩家当前指挥的舰船。
// 覆盖：handleUnitPanelActions 后检查 Target/Pos，随后逐帧推进并比较居中坐标与选择状态。
func TestExplicitTargetActionMovesCameraWithoutChangingFocus(t *testing.T) {
	ally := &objUnit.BattleShip{Uid: "ally", CurHP: 100}
	target := &objUnit.BattleShip{Uid: "target", CurHP: 100, CurPos: objPos.NewR(100, 90)}
	missionState := cameraTestState()
	missionState.Interaction.SelectedShips = []string{ally.Uid}
	missionState.Interaction.FocusedShipUid = ally.Uid
	missionState.Arena.Ships = map[string]*objUnit.BattleShip{ally.Uid: ally, target.Uid: target}
	initialCamera := missionState.View.Camera.Pos
	missionManager := &MissionManager{
		state:   missionState,
		sidebar: &sidebar.Panel{},
	}
	missionManager.handleUnitPanelActions([]unitpanel.Action{{
		Kind: unitpanel.ActionCenterTarget, TargetUid: target.Uid,
	}})
	// 定位是渐进的：这一帧只设目标，随后逐帧推进到居中。
	if missionState.View.Camera.Target == nil {
		t.Fatal("explicit target action did not set camera target")
	}
	if missionState.View.Camera.Pos != initialCamera {
		t.Fatalf("explicit target action teleported camera to %+v", missionState.View.Camera.Pos)
	}
	for missionManager.advanceCameraMove() {
	}
	if want := centeredCameraPos(missionState, target.CurPos); missionState.View.Camera.Pos != want {
		t.Fatalf("camera = %+v, want centered on target %+v", missionState.View.Camera.Pos, want)
	}
	if missionState.Interaction.FocusedShipUid != ally.Uid || len(missionState.Interaction.SelectedShips) != 1 {
		t.Fatalf("explicit target action changed selection: %+v", missionState.Interaction)
	}
}

// TestResizeUsesLogicalScreenDimensions 验证窗口尺寸变化会同步逻辑布局并刷新相机视野格数。
// 场景：初始 1280x720 的任务状态收到 Ebiten.Resize 事件，逻辑尺寸变为 1920x1080。
// 预期：View.Layout 更新为 1920x1080，且 Camera.Width/Height 被 RefreshCameraSize 重算为正数；
// 失败意味着布局与相机视野脱节，窗口放大后可见范围还是旧的，点击/框选坐标会错位。
// 覆盖：cameraTestState 后调用 Resize(1920,1080)，断言宽高字段与相机尺寸大于 0。
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
