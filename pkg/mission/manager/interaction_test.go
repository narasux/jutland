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

func TestSelectShipsByGroupFirstPressDoesNotMoveCamera(t *testing.T) {
	one := newGroupShip("one", object.GroupID1, 100, objPos.NewR(400, 400))
	two := newGroupShip("two", object.GroupID1, 100, objPos.NewR(620, 500))
	manager, ms := newGroupSelectManager(one, two)
	initialCamera := ms.View.Camera.Pos

	// 第一次按下编组 1：只选中，不移动相机
	manager.selectShipsByGroup(object.GroupID1)
	if ms.Interaction.SelectedGroupID != object.GroupID1 {
		t.Fatalf("selected group = %v, want GroupID1", ms.Interaction.SelectedGroupID)
	}
	if len(ms.Interaction.SelectedShips) != 2 {
		t.Fatalf("selected ships = %v, want both group ships", ms.Interaction.SelectedShips)
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
	if ms.View.Camera.Pos != initialCamera {
		t.Fatalf("camera moved from %+v to %+v on first press of another group", initialCamera, ms.View.Camera.Pos)
	}
}

func TestRepeatedGroupPressMovesCameraToRandomGroupShip(t *testing.T) {
	one := newGroupShip("one", object.GroupID1, 100, objPos.NewR(400, 400))
	two := newGroupShip("two", object.GroupID1, 100, objPos.NewR(620, 500))
	manager, ms := newGroupSelectManager(one, two)
	initialCamera := ms.View.Camera.Pos

	manager.selectShipsByGroup(object.GroupID1)
	if ms.View.Camera.Pos != initialCamera {
		t.Fatalf("camera moved on first press: %+v", ms.View.Camera.Pos)
	}

	want := map[objPos.MapPos]bool{
		centeredCameraPos(ms, one.CurPos): true,
		centeredCameraPos(ms, two.CurPos): true,
	}
	hit := map[objPos.MapPos]bool{}
	for range 64 {
		manager.selectShipsByGroup(object.GroupID1)
		if !want[ms.View.Camera.Pos] {
			t.Fatalf("camera = %+v, want centered on one of the group ships", ms.View.Camera.Pos)
		}
		hit[ms.View.Camera.Pos] = true
	}
	if len(hit) != 2 {
		t.Fatalf("camera targets = %d, want both group ships reachable (random pick)", len(hit))
	}
}

func TestGroupPressIgnoresDestroyedShip(t *testing.T) {
	alive := newGroupShip("alive", object.GroupID1, 100, objPos.NewR(400, 400))
	sunk := newGroupShip("sunk", object.GroupID1, 0, objPos.NewR(700, 600))
	manager, ms := newGroupSelectManager(alive, sunk)

	manager.selectShipsByGroup(object.GroupID1)
	for range 8 {
		manager.selectShipsByGroup(object.GroupID1)
		if ms.View.Camera.Pos != centeredCameraPos(ms, alive.CurPos) {
			t.Fatalf("camera = %+v, want alive ship position", ms.View.Camera.Pos)
		}
	}
}
