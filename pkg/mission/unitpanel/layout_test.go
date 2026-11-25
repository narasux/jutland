package unitpanel

import (
	"testing"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
)

// TestVerticalSectionsStackInsideRegion 验证单舰的 Header/Visual/Info/Systems 自上而下堆叠、不重叠且不越宽。
// 场景：右侧单位面板只有 280px 宽，战列舰同时有侧视图、基础信息与系统区三块内容。
// 预期：Header.X≥0 且右缘≤280，各非空分区 Y≥上一分区底、X 在界内，contentHeight>0；失败意味着分区重叠或溢出。
// 覆盖：region(100,50,280,420)、bismarck（1000/900）选中聚焦，updateWithPointer + calcLayout 后遍历四分区。
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

// TestScrollOffsetShiftsSections 验证滚动偏移会把整个内容区上移，Header/Visual 都会移出视口顶部。
// 场景：region(0,0,280,360)、scrollY=80，singleShipState 选中并聚焦 bismarck（战列舰，带视图区与基础信息）。
// 预期：Header.Y=contentPad(14)-80=-66，Visual.Y 在其下方继续上移，两者都小于 0，说明 -scrollY 统一作用到所有分区；失败意味着滚动只移动部分分区或完全没生效。
// 覆盖：直接 calcLayout(ms, region, 80)（绕过 ensureLayout 缓存），断言两分区 Y<0。
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

// TestFiveWeaponRowsFitSystemsColumn 验证装备全部 5 种武器的战列舰，每行武器开关都落在 Systems 分区内。
// 场景：Bismarck 同时有主炮/副炮/防空/鱼雷/火箭共 5 行；高度算错时最后一行会越界点不到。
// 预期：index 0–4 的 weaponToggleRect 均被 layout.Systems 包含；失败意味着武器开关越界、命中检测失效。
// 覆盖：fullWeaponState 五类武器各 1 门，region(0,0,280,480)，循环 5 行用 assertRectInside 检查四边。
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

// TestInfoGroupsFitInfoColumn 验证单舰信息分组只压成一节、行高可读，且「→」居中按钮落在 Info 分区内。
// 场景：bismarck 选中并聚焦，TotalHP 10000/CurHP 8200、MaxSpeed 0.05、CurSpeed 0.035、航向 65°、AttackTarget=target；信息行需与右侧目标按钮共处在 280×420 面板。
// 预期：shipGroups 只返回 1 个紧凑分组；infoGroupLayout 的 lineH 落在 [10,24]（实际 infoLineH=17）；hasTarget=true 且 targetButtonRect 完全包含在 layout.Info 内，按钮高度不超过单行高度减 2（旧实现 20px 高会顶到上下两行）；失败意味着行高被压扁、按钮越界点不到或箭头再次压到相邻数字。
// 覆盖：updateWithPointer + calcLayout 后断言分组数、行高区间、hasTarget 与按钮高度，再用 assertRectInside 检查按钮四边。
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
	button := panel.targetButtonRect()
	assertRectInside(t, button, panel.layout.Info)
	if button.H > content.lineH-2 {
		t.Fatalf("target button height = %.1f, want <= %.1f to stay inside one row", button.H, content.lineH-2)
	}
}

// TestFleetGroupIsSingleSection 验证多选舰队只汇总成一个分组节，便于阅读。
// 场景：选中 bismarck 与 essex 两艘、聚焦 bismarck；buildInfoGroups 在 len(ships)>1 时走 fleetGroup 汇总卡，而不是逐舰展开。
// 预期：只返回 1 个分组，items 恰 4 项（名称、总数、平均 HP、平均航速）；失败意味着舰队信息被拆成多节或汇总字段丢失。
// 覆盖：构造两舰 Arena 并选中后，直接调用 buildInfoGroups(ms, [a,b])，断言分组数与 items 长度。
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
				"ship": {
					Uid:     "ship",
					Name:    "bismarck",
					Type:    objUnit.ShipTypeBattleShip,
					TotalHP: 10_000,
					CurHP:   8_200,
				},
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
