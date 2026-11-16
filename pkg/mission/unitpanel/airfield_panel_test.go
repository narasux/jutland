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

// TestAirfieldPanelLayoutStacksSections 验证机场面板只有信息卡（无视图/系统页签），并生成 1 开关 + 2 生产行。
// 场景：选中驻有 F4F-3/SBD-3 的机场；机场没有舰体视图与武器页签，布局需改排为单卡。
// 预期：Visual/Systems 高 0、Info 高 140、contentHeight>Header；hits 恰 3 个且顺序为开关、F4F-3、SBD-3。
// 覆盖：airfieldTestState + region(100,50,280,420)，查分区高度、3 个 hit 的 Kind/Uid/PlaneName 与 Info 容差。
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

// TestAirfieldPanelActions 验证机场面板的点击命中会转换为带机场 uid 的类型化动作。
// 场景：选中驻有 F4F-3(4/6) 与 SBD-3(6/6) 的 af-test，面板按 region(100,50,280,420) 重建 hits；hits[0] 是机场状态开关，hits[2] 是 SBD-3 生产行。
// 预期：activate(hits[0]) 只返回 1 条 ActionToggleAirfield（AirfieldUid=af-test）；activate(hits[2]) 只返回 1 条 ActionSetProducing（PlaneName=SBD-3）；失败意味着开关与选机型点错目标或动作重复。
// 覆盖：updateWithPointer 生成 hits 后直接调用 activate 两次，逐字段断言 actions 长度、Kind、AirfieldUid、PlaneName。
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

// TestAirfieldSelectionInvalidatesLayoutCache 验证选中机场必须让布局缓存失效。
// 场景：机场面板与「未选中任何单位」的布局在舰船/焦点字段上同值；先以 SelectedAirfieldUid="" 填充缓存（Info.H=0），再选中 af-test，最后取消选中。
// 预期：空选中 Info.H=0；选中后即使其它键值不变也要重算，Info.H=airfieldInfoPad(8)+6*airfieldLineH(22)=140；取消选中回到 0；失败意味着缓存键缺 airfield 字段会复用空布局，面板只剩标题、信息卡画不出来。
// 覆盖：同一个 Panel 连续三次 updateWithPointer（ensureLayout 用 unitLayoutKey 缓存），断言三次 Info.H，容差 ±0.01。
func TestAirfieldSelectionInvalidatesLayoutCache(t *testing.T) {
	region := Rect{X: 100, Y: 50, W: 280, H: 420}
	ms := airfieldTestState("af-test")
	panel := New()

	// 先在未选中机场的状态下填充缓存（复现点机场前的空布局）
	ms.Interaction.SelectedAirfieldUid = ""
	panel.updateWithPointer(ms, region, 0, pointerInput{})
	if panel.layout.Info.H != 0 {
		t.Fatalf("empty selection should have no info section: %+v", panel.layout.Info)
	}

	// 选中机场：即使其它键值不变，也必须重算出信息卡
	ms.Interaction.SelectedAirfieldUid = "af-test"
	panel.updateWithPointer(ms, region, 0, pointerInput{})
	if panel.layout.Info.H == 0 {
		t.Fatal("airfield selection must recompute layout with an info card")
	}
	// 信息卡：起飞间隔 1 行 + 开关 1 行 + 节标题 1 行 + 表头 1 行 + 两机型 2 行 = 6 行
	wantInfo := airfieldInfoPad + 6*airfieldLineH
	if diff := panel.layout.Info.H - wantInfo; diff < -0.01 || diff > 0.01 {
		t.Fatalf("info height = %v, want %v", panel.layout.Info.H, wantInfo)
	}

	// 取消选中：缓存同样要失效，回到空布局
	ms.Interaction.SelectedAirfieldUid = ""
	panel.updateWithPointer(ms, region, 0, pointerInput{})
	if panel.layout.Info.H != 0 {
		t.Fatalf("deselecting airfield should restore empty layout: %+v", panel.layout.Info)
	}
}

// TestAirfieldSquadRowsMarkProducing 验证生产机型标记：待命+出击 < 上限才生产，指定机型满编后自动顺延。
// 场景：af-test 的 F4F-3 为 4/6、SBD-3 为 6/6，CurProducing 却指定已满编的 SBD-3，应顺延到未满编的 F4F-3。
// 预期：rows[0](F4F-3).Producing=true、rows[1](SBD-3)=false，Progress 落在 0–100；把 F4F-3 也补到 6/6 后两组都不标记，airfieldCapacityFull(rows)=true；失败意味着满编机型仍在吞生产或容量判断失效。
// 覆盖：airfieldSquadRows 两次调用 + airfieldCapacityFull，断言 Producing、Progress 区间与满编提示。
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

// TestAirfieldFundsLowHint 验证资金不足提示：己方机场未满编且资金低于当前生产机型单价时提示。
// 场景：临时把 PlaneMap["F4F-3"] 覆盖为 FundsCost=10、TimeCost=4（结束还原或删除）；af-test 的 F4F-3 为 4/6、SBD-3 满编，生产目标顺延到 F4F-3，玩家资金 5。
// 预期：资金 5<10 提示；15 不提示；Disabled=true 时仍提示（停用不影响生产，照常开工）；F4F-3 补到 6/6 满编不提示；机场改为 ComputerAlpha 时不按玩家资金提示；失败意味着用错机型单价或给敌方机场提示。
// 覆盖：依次修改 CurFunds/Disabled/CurCount/BelongPlayer 共五次调用 fundsLow，断言 true/false，Cleanup 还原 PlaneMap 原值。
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

// TestAirfieldSquadRowsCountFlyingPlanes 验证机队行按 BelongShip 统计空中飞机，库存/出击/损失分列。
// 场景：机场驻 F4F-3 与 SBD-3，Arena.Planes 中有 1 架 F4F-3 仍在空，F4F-3 已损失 2 架。
// 预期：F4F-3 行 Stock4 Max6 Flying1 Lost2，SBD-3 行 Flying0 Lost0；失败意味着空中数串机场或损失对不上。
// 覆盖：airfieldTestState 构造 Groups 与 Losses，airfieldSquadRows 返回后断言行数、Name 与四个计数。
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
