package unitpanel

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
)

func TestVerticalSectionsStackInsideRegion(t *testing.T) {
	region := Rect{X: 100, Y: 50, W: 280, H: 420}
	ms := &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				"ship": {Uid: "ship", Name: "bismarck", Type: objUnit.ShipTypeBattleShip, TotalHP: 1000, CurHP: 900},
			},
			Planes: map[string]*objUnit.Plane{},
		},
		Interaction: state.MissionInteractionState{SelectedShips: []string{"ship"}, FocusedShipUid: "ship"},
	}
	panel := New()
	panel.updateWithPointer(ms, region, 0, pointerInput{})
	panel.layout = panel.calcLayout(ms, region, 0)

	if panel.layout.Header.X < 0 || panel.layout.Header.X+panel.layout.Header.W > region.W {
		t.Fatalf("header overflows region width: %+v", panel.layout.Header)
	}
	// 分区自上而下堆叠，不应重叠，也不应超出视口宽度。
	var prev float64 = -1
	for _, s := range []Rect{panel.layout.Header, panel.layout.Visual, panel.layout.Info, panel.layout.Systems} {
		if s.H == 0 {
			continue
		}
		if s.Y < prev {
			t.Fatalf("sections overlap: prev bottom %v, section %+v", prev, s)
		}
		prev = s.Y + s.H
		if s.X < 0 || s.X+s.W > region.W {
			t.Fatalf("section %+v overflows region width %v", s, region.W)
		}
	}
	if panel.layout.contentHeight <= 0 {
		t.Fatalf("content height = %v, want positive", panel.layout.contentHeight)
	}
}

// TestScrollOffsetShiftsSections校验滚动偏移会把所有分区整体上移，且不影响各自尺寸。
func TestScrollOffsetShiftsSections(t *testing.T) {
	region := Rect{X: 0, Y: 0, W: 280, H: 360}
	ms := singleShipState()
	panel := New()
	panel.layout = panel.calcLayout(ms, region, 80)
	if panel.layout.Header.Y >= 0 {
		t.Fatalf("header did not shift up by scroll: Y=%v", panel.layout.Header.Y)
	}
	if panel.layout.Visual.Y >= 0 {
		t.Fatalf("visual did not shift up by scroll: Y=%v", panel.layout.Visual.Y)
	}
}

func TestFiveWeaponRowsFitSystemsColumn(t *testing.T) {
	ms := fullWeaponState()
	panel := New()
	region := Rect{X: 0, Y: 0, W: 280, H: 480}
	panel.updateWithPointer(ms, region, 0, pointerInput{})
	panel.layout = panel.calcLayout(ms, region, 0)
	for index := range 5 {
		assertRectInside(t, panel.weaponToggleRect(index, 5), panel.layout.Systems)
	}
}

func assertRectInside(t *testing.T, inner, outer Rect) {
	t.Helper()
	if inner.X < outer.X || inner.Y < outer.Y ||
		inner.X+inner.W > outer.X+outer.W || inner.Y+inner.H > outer.Y+outer.H {
		t.Fatalf("rect %+v is outside %+v", inner, outer)
	}
}

// TestInfoGroupsFitInfoColumn校验单舰信息分组都落在 Info 分区内，且行高在可读范围。
func TestInfoGroupsFitInfoColumn(t *testing.T) {
	ms := &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				"ship": {
					Uid: "ship", Name: "bismarck", Type: objUnit.ShipTypeBattleShip,
					TotalHP: 10_000, CurHP: 8_200, MaxSpeed: 0.05, CurSpeed: 0.035,
					CurRotation: 65, AttackTarget: "target",
				},
				"target": {Uid: "target", Name: "hood", CurHP: 1},
			},
			Planes: map[string]*objUnit.Plane{},
		},
		Interaction: state.MissionInteractionState{SelectedShips: []string{"ship"}, FocusedShipUid: "ship"},
	}
	panel := New()
	region := Rect{X: 0, Y: 0, W: 280, H: 420}
	panel.updateWithPointer(ms, region, 0, pointerInput{})
	panel.layout = panel.calcLayout(ms, region, 0)

	groups := panel.shipGroups(ms, ms.Arena.Ships["ship"])
	if len(groups) != 1 {
		t.Fatalf("single-ship info groups = %d, want 1 compact section", len(groups))
	}
	content := panel.infoGroupLayout(panel.layout.Info, groups)
	if content.lineH < 10 || content.lineH > 24 {
		t.Fatalf("info line height = %.1f, want within [10,24]", content.lineH)
	}

	if !panel.hasTarget {
		t.Fatal("attack-target ship should expose the center-target button")
	}
	assertRectInside(t, panel.targetButtonRect(), panel.layout.Info)
}

// TestFleetGroupIsSingleSection校验多选舰队信息只汇总为一个分组节，便于阅读。
func TestFleetGroupIsSingleSection(t *testing.T) {
	ms := &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				"a": {Uid: "a", Name: "bismarck", Type: objUnit.ShipTypeBattleShip, TotalHP: 100, CurHP: 80},
				"b": {Uid: "b", Name: "essex", Type: objUnit.ShipTypeAircraftCarrier, TotalHP: 100, CurHP: 90},
			},
			Planes: map[string]*objUnit.Plane{},
		},
		Interaction: state.MissionInteractionState{SelectedShips: []string{"a", "b"}, FocusedShipUid: "a"},
	}
	panel := New()
	panel.updateWithPointer(ms, Rect{X: 0, Y: 0, W: 280, H: 480}, 0, pointerInput{})
	groups := panel.buildInfoGroups(ms, []*objUnit.BattleShip{ms.Arena.Ships["a"], ms.Arena.Ships["b"]})
	if len(groups) != 1 {
		t.Fatalf("fleet info groups = %d, want 1 summary section", len(groups))
	}
	if len(groups[0].items) != 4 {
		t.Fatalf("fleet summary items = %d, want 4", len(groups[0].items))
	}
}

// singleShipState 返回一艘选中单舰的测试状态。
func singleShipState() *state.MissionState {
	return &state.MissionState{
		Core:        state.MissionCoreState{MissionStatus: state.MissionRunning},
		Interaction: state.MissionInteractionState{SelectedShips: []string{"ship"}, FocusedShipUid: "ship"},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				"ship": {Uid: "ship", Name: "bismarck", Type: objUnit.ShipTypeBattleShip, TotalHP: 10_000, CurHP: 8_200},
			},
			Planes: map[string]*objUnit.Plane{},
		},
	}
}

// fullWeaponState 返回装备全部五种武器类型的战舰状态，用于验证武器行都在系统分区内。
func fullWeaponState() *state.MissionState {
	ship := &objUnit.BattleShip{
		Uid: "ship", Name: "bismarck", Type: objUnit.ShipTypeBattleShip,
		TotalHP: 10_000, CurHP: 9_000,
		Weapon: objUnit.ShipWeapon{
			MainGuns:         []*objUnit.Gun{{ReloadTime: 1}},
			SecondaryGuns:    []*objUnit.Gun{{ReloadTime: 1}},
			AntiAircraftGuns: []*objUnit.Gun{{ReloadTime: 1}},
			Torpedoes:        []*objUnit.TorpedoLauncher{{ReloadTime: 1}},
			Rockets:          []*objUnit.RocketLauncher{{ReloadTime: 1}},
		},
	}
	return &state.MissionState{
		Core:        state.MissionCoreState{MissionStatus: state.MissionRunning},
		Interaction: state.MissionInteractionState{SelectedShips: []string{"ship"}, FocusedShipUid: "ship"},
		UI:          state.MissionUIState{},
		Arena: state.MissionArenaState{
			Ships:  map[string]*objUnit.BattleShip{"ship": ship},
			Planes: map[string]*objUnit.Plane{},
		},
	}
}
