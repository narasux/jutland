package unitpanel

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
)

// testRegion 是输入测试使用的滚动视口（原点为 0,0，局部坐标与屏幕坐标一致）。
var testRegion = Rect{X: 0, Y: 0, W: 280, H: 480}

// TestDefaultTabFollowsFocusedShipType 验证焦点舰切换时默认页签跟随舰种：航母进航空页，其余进武器页。
// 场景：玩家先选航母看舰载机，再切到战列舰；不同舰种常用操作不同，需要自动切页。
// 预期：航母焦点时 tab=TabAircraft，换战列舰后 tab=TabWeapons；失败意味着页签停在上一舰种。
// 覆盖：两舰放入 Arena，先选 carrier 调 updateWithPointer 断言 tab，再改 Selected/Focused 为 battleship。
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

// TestTargetButtonProducesExplicitCenterActionWithoutChangingFocus 验证「→」按钮只发一次居中且不改焦点舰。
// 场景：焦点舰有攻击目标，信息卡「目标」行出现居中按钮；玩家点它把镜头移到目标，但选中仍应留在本舰。
// 预期：actions 仅 1 条且 Kind=ActionCenterTarget、TargetUid=target，焦点仍为 ship；失败意味着焦点被切走。
// 覆盖：panelInteractionState 加 ship（AttackTarget=target）与 target，取 targetButtonRect 中心 JustPressed。
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

// TestAircraftActionIgnoresShipsWithoutAircraft 验证舰载机批量动作只收集 HasPlane 的舰，有禁用时转为允许。
// 场景：编队里航母起飞被关闭、护航驱逐没有飞机；玩家点航空总开关，不能给无飞机舰发指令。
// 预期：Enable=true，ShipUids 仅 [carrier]；失败意味着动作下发到无飞机舰或开关方向反了。
// 覆盖：panelTestState(carrier,escort)，New().aircraftAction(ms)，断言 Enable、长度与首个 UID。
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

// TestSubmarineDiveButtonProducesToggleAction 验证潜艇「下潜/上浮」按钮命中后产生单个 ActionToggleDive。
// 场景：选中一艘水面潜艇，信息区「潜航状态」行右侧出现按钮；点击应请求切换潜航状态。
// 预期：按钮宽>0；actions 长度 1、Kind=ActionToggleDive、ShipUids=[sub]；失败意味着按钮为空或动作未带目标。
// 覆盖：panelInteractionState 放入潜艇并聚焦，updateWithPointer 后取 computeDiveRect 中心点击并断言动作。
func TestSubmarineDiveButtonProducesToggleAction(t *testing.T) {
	missionState := panelInteractionState()
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 1, TotalHP: 1, Type: objUnit.ShipTypeSubmarine,
	}
	missionState.Arena.Ships = map[string]*objUnit.BattleShip{sub.Uid: sub}
	missionState.Interaction.SelectedShips = []string{sub.Uid}
	missionState.Interaction.FocusedShipUid = sub.Uid
	panel := New()
	panel.updateWithPointer(missionState, testRegion, 0, pointerInput{})

	button := panel.computeDiveRect(missionState, sub)
	if button.W == 0 {
		t.Fatal("submarine dive button rect is empty")
	}
	actions := panel.updateWithPointer(missionState, testRegion, 0, pointerInput{
		X:           int(button.X + button.W/2),
		Y:           int(button.Y + button.H/2),
		JustPressed: true,
	})
	if len(actions) != 1 || actions[0].Kind != ActionToggleDive {
		t.Fatalf("actions = %+v", actions)
	}
	if len(actions[0].ShipUids) != 1 || actions[0].ShipUids[0] != sub.Uid {
		t.Fatalf("action ships = %+v", actions[0].ShipUids)
	}
}

// TestSubmarineDiveButtonWorksDuringTransition 验证潜航过渡中「下潜/上浮」按钮仍可点，以最后一次指令为准。
// 场景：潜艇 sub 处于 DiveStateDiving，选中并聚焦；NextDiveState 对 Diving 返回 Surface，面板仍应注册 hitDiveToggle。
// 预期：computeDiveRect 宽>0（过渡中不禁用按钮）；点击按钮中心只产生 1 条 ActionToggleDive；失败意味着过渡期间按钮消失或点击被吞。
// 覆盖：panelInteractionState + sub（Diving），testRegion(0,0,280,480) 下 updateWithPointer，取 computeDiveRect 中心 JustPressed 点击并断言动作。
func TestSubmarineDiveButtonWorksDuringTransition(t *testing.T) {
	missionState := panelInteractionState()
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 1, TotalHP: 1, Type: objUnit.ShipTypeSubmarine,
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateDiving},
	}
	missionState.Arena.Ships = map[string]*objUnit.BattleShip{sub.Uid: sub}
	missionState.Interaction.SelectedShips = []string{sub.Uid}
	missionState.Interaction.FocusedShipUid = sub.Uid
	panel := New()
	panel.updateWithPointer(missionState, testRegion, 0, pointerInput{})

	button := panel.computeDiveRect(missionState, sub)
	if button.W == 0 {
		t.Fatal("dive button should stay available while diving")
	}
	actions := panel.updateWithPointer(missionState, testRegion, 0, pointerInput{
		X:           int(button.X + button.W/2),
		Y:           int(button.Y + button.H/2),
		JustPressed: true,
	})
	if len(actions) != 1 || actions[0].Kind != ActionToggleDive {
		t.Fatalf("actions = %+v", actions)
	}
}
