package unitpanel

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
)

func airfieldTestState(uid string) *state.MissionState {
	airfield := &objBuilding.Airfield{
		Uid:          uid,
		Pos:          objPos.NewR(53, 65),
		Rotation:     50,
		RunwayLength: 10.2,
		RunwayWidth:  0.8,
		BelongPlayer: faction.HumanAlpha,
		Aircraft: objUnit.ShipAircraft{
			TakeOffTime: 2.5,
			HasPlane:    true,
			Groups: []objUnit.PlaneGroup{
				{Name: "F4F-3", MaxCount: 6, CurCount: 4},
				{Name: "SBD-3", MaxCount: 6, CurCount: 6},
			},
		},
		Losses: map[string]int64{"F4F-3": 2},
	}
	ms := &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		Arena: state.MissionArenaState{
			Airfields: map[string]*objBuilding.Airfield{uid: airfield},
			Planes: map[string]*objUnit.Plane{
				"p1": {Uid: "p1", Name: "F4F-3", BelongShip: uid, CurHP: 10, TotalHP: 35},
			},
		},
		Interaction: state.MissionInteractionState{SelectedAirfieldUid: uid},
	}
	return ms
}

func TestAirfieldPanelLayoutStacksSections(t *testing.T) {
	region := Rect{X: 100, Y: 50, W: 280, H: 420}
	ms := airfieldTestState("af-test")
	panel := New()
	panel.updateWithPointer(ms, region, 0, pointerInput{})
	panel.layout = panel.calcLayout(ms, region, 0)

	// 机场面板没有视图区与系统页签，只有一张信息卡
	if panel.layout.Visual.H != 0 || panel.layout.Systems.H != 0 {
		t.Fatalf("airfield panel should have no visual/systems sections: %+v", panel.layout)
	}
	if panel.layout.Info.H == 0 {
		t.Fatal("airfield panel should have an info section")
	}
	if panel.layout.contentHeight <= panel.layout.Header.H {
		t.Fatalf("content height = %v, want more than header only", panel.layout.contentHeight)
	}
	// 可点击区域：机场状态开关 1 个 + 生产机型选择行 2 个
	if len(panel.hits) != 3 {
		t.Fatalf("airfield panel should register 3 hit regions, got %d", len(panel.hits))
	}
	if panel.hits[0].Kind != hitAirfieldToggle || panel.hits[0].AirfieldUid != "af-test" {
		t.Fatalf("first hit = %+v, want airfield toggle", panel.hits[0])
	}
	if panel.hits[1].Kind != hitAirfieldProduce || panel.hits[1].PlaneName != "F4F-3" ||
		panel.hits[2].PlaneName != "SBD-3" {
		t.Fatalf("produce hits = %+v, %+v, want F4F-3 then SBD-3", panel.hits[1], panel.hits[2])
	}
	// 信息卡：起飞间隔 1 行 + 开关 1 行 + 节标题 1 行 + 表头 1 行 + 两机型 2 行 = 6 行
	wantInfo := airfieldInfoPad + 6*airfieldLineH
	if diff := panel.layout.Info.H - wantInfo; diff < -0.01 || diff > 0.01 {
		t.Fatalf("info height = %v, want %v", panel.layout.Info.H, wantInfo)
	}
}

// TestAirfieldPanelActions 机场开关与生产机型行的点击产出对应类型化动作。
func TestAirfieldPanelActions(t *testing.T) {
	region := Rect{X: 100, Y: 50, W: 280, H: 420}
	ms := airfieldTestState("af-test")
	panel := New()
	panel.updateWithPointer(ms, region, 0, pointerInput{})

	// 点击机场状态开关 → ActionToggleAirfield
	actions := panel.activate(ms, panel.hits[0])
	if len(actions) != 1 || actions[0].Kind != ActionToggleAirfield || actions[0].AirfieldUid != "af-test" {
		t.Fatalf("toggle actions = %+v, want ActionToggleAirfield", actions)
	}
	// 点击机型行 → ActionSetProducing
	actions = panel.activate(ms, panel.hits[2])
	if len(actions) != 1 || actions[0].Kind != ActionSetProducing ||
		actions[0].AirfieldUid != "af-test" || actions[0].PlaneName != "SBD-3" {
		t.Fatalf("produce actions = %+v, want ActionSetProducing SBD-3", actions)
	}
}

// TestAirfieldSquadRowsMarkProducing 生产机型自动切换：待命 + 出击 < 上限
// 才生产；指定机型满编后顺延到下一个未满编机型，全部满编则不标记。
func TestAirfieldSquadRowsMarkProducing(t *testing.T) {
	ms := airfieldTestState("af-test")
	af := ms.Arena.Airfields["af-test"]
	// 指定 SBD（满编 6/6）：自动顺延到未满编的 F4F
	af.CurProducing = "SBD-3"
	rows := airfieldSquadRows(ms, af)
	if !rows[0].Producing || rows[1].Producing {
		t.Fatalf("producing flags = %v/%v, want F4F-3 true, SBD-3 false", rows[0].Producing, rows[1].Producing)
	}
	if rows[0].Progress < 0 || rows[0].Progress > 100 {
		t.Fatalf("progress = %d, want 0-100", rows[0].Progress)
	}

	// F4F 也满编：全部满编，不标记任何机型，提示容量已满
	af.Aircraft.Groups[0].CurCount = 6
	rows = airfieldSquadRows(ms, af)
	if rows[0].Producing || rows[1].Producing {
		t.Fatal("capacity full should mark no type as producing")
	}
	if !airfieldCapacityFull(rows) {
		t.Fatal("all types full should report capacity full")
	}
}

// TestAirfieldFundsLowHint 资金不足提示：己方机场未满编且资金低于当前机型
// 单价时提示（停用不影响生产，照常提示）；满编 / 敌方机场不提示。
func TestAirfieldFundsLowHint(t *testing.T) {
	// 覆盖机型费用模板：F4F-3 单价 10（原模板可能不存在或费用不同）
	oldTemplate, hadTemplate := objUnit.PlaneMap["F4F-3"]
	objUnit.PlaneMap["F4F-3"] = &objUnit.Plane{Name: "F4F-3", FundsCost: 10, TimeCost: 4}
	t.Cleanup(func() {
		if hadTemplate {
			objUnit.PlaneMap["F4F-3"] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, "F4F-3")
		}
	})

	ms := airfieldTestState("af-test")
	ms.Player = state.MissionPlayerState{CurFunds: 5, CurPlayer: faction.HumanAlpha}
	af := ms.Arena.Airfields["af-test"]
	if !fundsLow(ms, af) {
		t.Fatal("5 funds below cost 10 should flag funds low")
	}

	ms.Player.CurFunds = 15
	if fundsLow(ms, af) {
		t.Fatal("enough funds should not flag")
	}

	af.Disabled = true
	// 停用不影响生产：资金不足照常提示
	ms.Player.CurFunds = 5
	if !fundsLow(ms, af) {
		t.Fatal("disabled airfield still produces, funds low should still flag")
	}
	af.Disabled = false

	// 满编（6/6）不提示
	af.Aircraft.Groups[0].CurCount = 6
	if fundsLow(ms, af) {
		t.Fatal("full group should not flag")
	}
	af.Aircraft.Groups[0].CurCount = 4

	// 敌方机场不按玩家资金提示
	af.BelongPlayer = faction.ComputerAlpha
	if fundsLow(ms, af) {
		t.Fatal("enemy airfield should not flag")
	}
}

func TestAirfieldSquadRowsCountFlyingPlanes(t *testing.T) {
	ms := airfieldTestState("af-test")
	rows := airfieldSquadRows(ms, ms.Arena.Airfields["af-test"])
	if len(rows) != 2 {
		t.Fatalf("squad rows = %d, want 2", len(rows))
	}
	if rows[0].Name != "F4F-3" || rows[0].Stock != 4 || rows[0].Max != 6 ||
		rows[0].Flying != 1 || rows[0].Lost != 2 {
		t.Fatalf("F4F-3 row = %+v, want stock 4/6 flying 1 lost 2", rows[0])
	}
	if rows[1].Name != "SBD-3" || rows[1].Flying != 0 || rows[1].Lost != 0 {
		t.Fatalf("SBD-3 row = %+v, want flying 0 lost 0", rows[1])
	}
}
