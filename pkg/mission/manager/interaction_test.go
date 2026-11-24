package manager

import (
	"slices"
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestConsumeZoomInputLimitsMouseWheelToOneStep 验证鼠标滚轮每 8 帧最多产生 1 次缩放方向。
// 场景：玩家滚动滚轮，Ebiten 连续给出 wheelY=12 的事件；同一帧序列里反复调用 consumeZoomInput。
// 预期：首次返回 1，随后 wheelZoomCooldownTicks-1 帧返回 0，冷却结束再返回 1；
// 失败意味着原始增量 12 被当成 12 档缩放，画面会瞬间跳变甚至越界。
// 覆盖：固定 wheelY=12、magnification=0，逐帧调用并区分冷却中与冷却后的返回值。
func TestConsumeZoomInputLimitsMouseWheelToOneStep(t *testing.T) {
	manager := &MissionManager{}

	if direction := manager.consumeZoomInput(12, 0); direction != 1 {
		t.Fatalf("first wheel direction = %d, want 1", direction)
	}
	for frame := 1; frame < wheelZoomCooldownTicks; frame++ {
		if direction := manager.consumeZoomInput(12, 0); direction != 0 {
			t.Fatalf("wheel direction during cooldown at frame %d = %d, want 0", frame, direction)
		}
	}
	if direction := manager.consumeZoomInput(12, 0); direction != 1 {
		t.Fatalf("wheel direction after cooldown = %d, want 1", direction)
	}
}

// TestConsumeZoomInputAccumulatesTrackpadMagnification 验证触控板捏合按累积量触发缩放步。
// 场景：单次捏合事件 magnification=0.3，远小于 1 的阈值，需要连续两次事件才够一步。
// 预期：第一次返回 0（只累加 0.6），第二次累加后超过 1.0 返回 1；失败则一次轻微捏合就缩放，
// 触控板会过于灵敏，或累积量永远达不到阈值导致无法缩放。
// 覆盖：固定 wheelY=0，连续两次 consumeZoomInput(0, 0.3) 并断言方向 0 → 1。
func TestConsumeZoomInputAccumulatesTrackpadMagnification(t *testing.T) {
	manager := &MissionManager{}

	if direction := manager.consumeZoomInput(0, 0.3); direction != 0 {
		t.Fatalf("first pinch direction = %d, want 0", direction)
	}
	if direction := manager.consumeZoomInput(0, 0.3); direction != 1 {
		t.Fatalf("accumulated pinch direction = %d, want 1", direction)
	}
}

// newGroupShip 造一艘属于我方某编组的战舰。
func newGroupShip(uid string, groupID object.GroupID, hp float64, pos objPos.MapPos) *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Uid:          uid,
		Name:         uid,
		CurHP:        hp,
		CurPos:       pos,
		GroupID:      groupID,
		BelongPlayer: faction.HumanAlpha,
	}
}

// newGroupSelectManager 造编组选舰测试用的管理器与状态。
func newGroupSelectManager(ships ...*objUnit.BattleShip) (*MissionManager, *state.MissionState) {
	ms := cameraTestState()
	ms.Player.CurPlayer = faction.HumanAlpha
	ms.Arena.Ships = map[string]*objUnit.BattleShip{}
	for _, ship := range ships {
		ms.Arena.Ships[ship.Uid] = ship
	}
	return &MissionManager{state: ms}, ms
}

// pressGroup 连按同一个编组键 count 次，用于跨过“第一次只选中”的语义。
func pressGroup(manager *MissionManager, groupID object.GroupID, count int) {
	for range count {
		manager.selectShipsByGroup(groupID)
	}
}

// TestSelectShipsByGroupFirstPressOnlySelects 验证编组键首次按下只选中舰船、不移动相机。
// 场景：编组 1 有 "one"(400,400) 与 "two"(620,500)，相机初始在 (10,10)；随后再按下另一个编组 2。
// 预期：SelectedGroupID 切到所按编组、SelectedShips 收录该编组全部 2 艘，Camera.Target 仍为 nil、
// Camera.Pos 与初始值完全一致；失败意味着按一下编组键画面就跳走，玩家会丢失当前视野。
// 覆盖：newGroupSelectManager 构造状态，分别按 GroupID1 与 GroupID2 并逐项断言相机与选中集合。
func TestSelectShipsByGroupFirstPressOnlySelects(t *testing.T) {
	one := newGroupShip("one", object.GroupID1, 100, objPos.NewR(400, 400))
	two := newGroupShip("two", object.GroupID1, 100, objPos.NewR(620, 500))
	manager, ms := newGroupSelectManager(one, two)
	initialCamera := ms.View.Camera.Pos

	// 第一次按下编组 1：只选中，不动相机
	manager.selectShipsByGroup(object.GroupID1)
	if ms.Interaction.SelectedGroupID != object.GroupID1 {
		t.Fatalf("selected group = %v, want GroupID1", ms.Interaction.SelectedGroupID)
	}
	if len(ms.Interaction.SelectedShips) != 2 {
		t.Fatalf("selected ships = %v, want both group ships", ms.Interaction.SelectedShips)
	}
	if ms.View.Camera.Target != nil {
		t.Fatalf("camera target = %+v, want nil on first press", ms.View.Camera.Target)
	}
	if ms.View.Camera.Pos != initialCamera {
		t.Fatalf("camera moved from %+v to %+v on first press", initialCamera, ms.View.Camera.Pos)
	}

	// 第一次按下另一个编组同样不移动相机
	three := newGroupShip("three", object.GroupID2, 100, objPos.NewR(300, 700))
	ms.Arena.Ships[three.Uid] = three
	manager.selectShipsByGroup(object.GroupID2)
	if ms.Interaction.SelectedGroupID != object.GroupID2 {
		t.Fatalf("selected group = %v, want GroupID2", ms.Interaction.SelectedGroupID)
	}
	if ms.View.Camera.Target != nil {
		t.Fatalf("camera target = %+v, want nil on first press of another group", ms.View.Camera.Target)
	}
	if ms.View.Camera.Pos != initialCamera {
		t.Fatalf("camera moved from %+v to %+v on first press of another group", initialCamera, ms.View.Camera.Pos)
	}
}

// TestRepeatedGroupPressTargetsFormationCenter 验证再次按下编组键会把相机定位到编队包围盒中心。
// 场景：编组 1 两舰位于 (400,400) 与 (620,500)，第二次按下编组键时触发 centerCameraOn；
// 之后继续连按 16 次，模拟玩家反复敲同一个编组键。
// 预期：Camera.Target 等于 centeredCameraPos(ms, (510,450))，且按下当帧 Pos 不变（由逐帧推进）；
// 连续 16 次重按目标始终不变。失败意味着定位点随按击次数在编队内乱跳或直接瞬移。
// 覆盖：pressGroup 连按 2 次与 16 次两段，比较 *(Camera.Target) 与编队中心换算结果。
func TestRepeatedGroupPressTargetsFormationCenter(t *testing.T) {
	one := newGroupShip("one", object.GroupID1, 100, objPos.NewR(400, 400))
	two := newGroupShip("two", object.GroupID1, 100, objPos.NewR(620, 500))
	manager, ms := newGroupSelectManager(one, two)
	initialCamera := ms.View.Camera.Pos

	// 第二次按下编组 1：目标是编队包围盒中心（两舰中点），一次定位带进整个编队。
	pressGroup(manager, object.GroupID1, 2)
	want := centeredCameraPos(ms, objPos.NewR(510, 450))
	if target := ms.View.Camera.Target; target == nil || *target != want {
		t.Fatalf("camera target = %v, want formation center %+v", target, want)
	}
	// 定位是渐进的：按下当帧不瞬移，由 updateCameraPosition 逐帧推进。
	if ms.View.Camera.Pos != initialCamera {
		t.Fatalf("camera teleported from %+v to %+v on group press", initialCamera, ms.View.Camera.Pos)
	}

	// 回归：继续连按不能再编队内乱跳，目标必须始终是编队中心。
	for range 16 {
		pressGroup(manager, object.GroupID1, 1)
		if target := ms.View.Camera.Target; target == nil || *target != want {
			t.Fatalf("repeat press retargeted camera to %v, want stable %+v", target, want)
		}
	}
}

// TestGroupPressIgnoresDestroyedShip 验证编队中心只统计存活舰船，沉船不参与包围盒。
// 场景：编组 1 里 "alive"(400,400) 仍在，另一艘 "sunk" 的 CurHP=0 却停在 (700,600)；
// 连按两次编组键后 formationCenter 需要跳过沉船。
// 预期：Camera.Target 等于以 (400,400) 为中心的 centeredCameraPos，而不是两船中点 (550,500)；
// 失败意味着沉船把镜头拖向残骸，编队定位偏离主力。
// 覆盖：newGroupShip 分别造 100 HP 与 0 HP 的同编组舰，按 2 次后与单舰定位结果比较。
func TestGroupPressIgnoresDestroyedShip(t *testing.T) {
	alive := newGroupShip("alive", object.GroupID1, 100, objPos.NewR(400, 400))
	sunk := newGroupShip("sunk", object.GroupID1, 0, objPos.NewR(700, 600))
	manager, ms := newGroupSelectManager(alive, sunk)

	pressGroup(manager, object.GroupID1, 2)
	// 包围盒只统计存活舰船，沉船不能把定位目标拖走。
	want := centeredCameraPos(ms, alive.CurPos)
	if target := ms.View.Camera.Target; target == nil || *target != want {
		t.Fatalf("camera target = %v, want alive ship center %+v", target, want)
	}
}

// TestCameraMoveEasesTowardTargetAndSnaps 验证相机定位是单调收敛的缓动并在终点精确吸附。
// 场景：单舰编组按两次设定 Target 后，逐帧调用 advanceCameraMove，最多观察 600 帧。
// 预期：每帧到目标的距离不得增大；至少走 2 帧（排除瞬移）；收敛后 Pos 与 Target 完全相等、
// Target 被清空。失败分别意味着相机远离目标抖动、定位退化成瞬移，或留下浮点差导致永不结束。
// 覆盖：记录首次距离 prev 与帧数 frames，循环 advanceCameraMove 并比较 Distance，最后断言 Pos/Target。
func TestCameraMoveEasesTowardTargetAndSnaps(t *testing.T) {
	one := newGroupShip("one", object.GroupID1, 100, objPos.NewR(400, 400))
	manager, ms := newGroupSelectManager(one)
	pressGroup(manager, object.GroupID1, 2)

	target := ms.View.Camera.Target
	if target == nil {
		t.Fatal("group press did not set a camera target")
	}

	prev := ms.View.Camera.Pos.Distance(*target)
	frames := 0
	for manager.advanceCameraMove() {
		frames++
		if frames > 600 {
			t.Fatal("camera move did not converge")
		}
		if dist := ms.View.Camera.Pos.Distance(*target); dist > prev {
			t.Fatalf("camera moved away from target: %v -> %v", prev, dist)
		} else {
			prev = dist
		}
	}

	if frames < 2 {
		t.Fatalf("camera snapped in %d frames, want a gradual move", frames)
	}
	if ms.View.Camera.Pos != *target {
		t.Fatalf("camera = %+v, want landed exactly on %+v", ms.View.Camera.Pos, *target)
	}
	if ms.View.Camera.Target != nil {
		t.Fatalf("camera target not cleared after arrival: %+v", ms.View.Camera.Target)
	}
}

// TestAreaSelectingCancelsCameraMove 验证框选期间会放弃正在进行的相机定位。
// 场景：编组按键已设置 Camera.Target，此时玩家按住拖拽开始框选，IsAreaSelecting 置为 true，
// 又调了一帧 advanceCameraMove。
// 预期：返回 false 表示不再移动，且 Camera.Target 被清空；失败意味着镜头在框选时继续滑动，
// 选区的地图坐标会随相机跳变，选中范围与玩家看到的框不一致。
// 覆盖：按两次编组键制造定位目标，设置框选状态后断言返回值与 Target 均为空。
func TestAreaSelectingCancelsCameraMove(t *testing.T) {
	one := newGroupShip("one", object.GroupID1, 100, objPos.NewR(400, 400))
	manager, ms := newGroupSelectManager(one)
	pressGroup(manager, object.GroupID1, 2)

	// 框选依赖相机静止，此时必须放弃定位，否则选区地图坐标会跳变。
	ms.Interaction.IsAreaSelecting = true
	if manager.advanceCameraMove() {
		t.Fatal("camera kept moving while area selecting")
	}
	if ms.View.Camera.Target != nil {
		t.Fatalf("camera target not cleared while area selecting: %+v", ms.View.Camera.Target)
	}
}

// TestGameCameraCornerPanMovesBothAxesWithZoomScaledStep 验证光标压在屏幕左上角时两轴同时滚屏，
// 且步长按缩放换算、小数位原样保留。
// 场景：测试进程里 Ebiten 尚未开局，CursorPosition 固定返回 (0,0)，正好是屏幕左上角；
// 相机停在 (10.125,10.125)（不是 0.25 的整数倍，模拟刚缩放或平滑定位之后的停留位置），
// BaseMoveSpeed=0.25，地图 1000x1000、视野 33x19 不会碰边界。
// 旧实现有两处问题：0 < x 让左上角退化成 HoverScreenTop，只滚竖直轴；
// updateCameraPosition 又把新坐标截断到 BaseMoveSpeed 的整数倍，第一帧会多跳一段。
// 预期：1x 档（zoom=4）相机到 (9.875,9.875)，4x 档（zoom=16）到 (10.0625,10.0625)；
// 失败意味着角落掉了一个滚屏轴、步长没跟缩放走，或坐标被截断回步长整数倍。
// 覆盖：表驱动 zoom=4/16 两档，调用 updateCameraPosition 后比较 Camera.Pos 的 RX/RY。
func TestGameCameraCornerPanMovesBothAxesWithZoomScaledStep(t *testing.T) {
	for _, tc := range []struct {
		zoom         int
		wantX, wantY float64
	}{
		{zoom: 4, wantX: 9.875, wantY: 9.875},
		{zoom: 16, wantX: 10.0625, wantY: 10.0625},
	} {
		ms := cameraTestState()
		ms.UI.GameOpts.Zoom = tc.zoom
		ms.View.Camera.BaseMoveSpeed = 0.25
		ms.View.Camera.Pos = objPos.NewR(10.125, 10.125)
		manager := &MissionManager{state: ms}

		manager.updateCameraPosition()

		pos := ms.View.Camera.Pos
		if pos.RX != tc.wantX || pos.RY != tc.wantY {
			t.Fatalf(
				"zoom %d: camera pos = (%v,%v), want (%v,%v)",
				tc.zoom, pos.RX, pos.RY, tc.wantX, tc.wantY,
			)
		}
	}
}

// TestGroupKeySelectionWorksWhilePanelConsumesCursor 验证光标停在面板上时编组键仍能切换编队。
// 场景：侧栏展开、鼠标停在小地图上（UIConsumesCursor=true），玩家用键盘按编组键 1；
// 旧实现把编组键处理写在“光标未被 UI 占用”的分支里，面板一挡住光标数字键就完全失效。
// 预期：选中集变成编队 1 的两艘舰并按舰名稳定排序为 one/two，焦点补齐为 one，
// 面板占用时遗留的框选状态被复位；按住 Ctrl（IsGrouping）时数字键只写编组、不切换选中。
// 失败意味着面板又把键盘输入一起挡掉了，或者编组模式下误切了选中集。
// 覆盖：cameraTestState + 编队 1 两舰与编队 2 一舰，表驱动“面板占用 / 光标空闲 / 编组中”
// 三种状态，调用 updateShipSelectionForTick(GroupID1) 后断言选中集、焦点与 IsAreaSelecting。
// 注：按下的数字键由 Update 传给本函数，真实按键检测与接线无法在无窗口测试里注入，需手动验证。
func TestGroupKeySelectionWorksWhilePanelConsumesCursor(t *testing.T) {
	for _, tc := range []struct {
		name           string
		consumesCursor bool
		grouping       bool
		wantSelected   []string
		wantFocused    string
	}{
		{
			name:           "panel consumes cursor",
			consumesCursor: true,
			wantSelected:   []string{"one", "two"},
			wantFocused:    "one",
		},
		{
			name:         "free cursor",
			wantSelected: []string{"one", "two"},
			wantFocused:  "one",
		},
		{
			name:           "grouping mode",
			consumesCursor: true,
			grouping:       true,
			wantSelected:   nil,
			wantFocused:    "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			one := newGroupShip("one", object.GroupID1, 100, objPos.NewR(400, 400))
			two := newGroupShip("two", object.GroupID1, 100, objPos.NewR(620, 500))
			other := newGroupShip("other", object.GroupID2, 100, objPos.NewR(300, 300))

			ms := cameraTestState()
			ms.Core.MissionStatus = state.MissionRunning
			ms.Player.CurPlayer = faction.HumanAlpha
			ms.UI.UIConsumesCursor = tc.consumesCursor
			ms.Interaction.IsGrouping = tc.grouping
			// 模拟上一帧遗留的框选：面板接管光标时必须复位，否则选区卡在拖动状态。
			ms.Interaction.IsAreaSelecting = tc.consumesCursor
			ms.Arena.Ships = map[string]*objUnit.BattleShip{
				one.Uid: one, two.Uid: two, other.Uid: other,
			}
			manager := &MissionManager{state: ms}

			manager.updateShipSelectionForTick(object.GroupID1)

			if got := ms.Interaction.SelectedShips; !slices.Equal(got, tc.wantSelected) {
				t.Fatalf("selected ships = %v, want %v", got, tc.wantSelected)
			}
			if ms.Interaction.FocusedShipUid != tc.wantFocused {
				t.Fatalf("focused ship = %q, want %q", ms.Interaction.FocusedShipUid, tc.wantFocused)
			}
			if ms.Interaction.IsAreaSelecting {
				t.Fatal("area selecting not reset while UI consumes cursor")
			}
		})
	}
}
