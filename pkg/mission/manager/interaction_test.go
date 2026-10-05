package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

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
