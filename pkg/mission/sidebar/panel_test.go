package sidebar

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/state"
	mapcfg "github.com/narasux/jutland/pkg/resources/mapcfg"
	"github.com/narasux/jutland/pkg/utils/layout"
)

func newTestPanel(mapAspect float64) *Panel {
	return &Panel{mapAspect: mapAspect}
}

// TestExpandedPanelConsumesPanelHandleAndMinimap 验证展开态只在面板、把手、页签、小地图与单位区消费鼠标。
// 场景：面板停靠右上角、高 86% 屏幕；玩家在面板下方或右缘 12px 细缝拖动镜头时不能被侧栏吞掉。
// 预期：五个面板内点均 true；面板外、面板下方、右缘细缝均 false；失败意味着边缘滚动被阻断或面板漏点。
// 覆盖：calcLayout(1280×720, 展开, aspect=1) 后逐个调用 consumesCursorAt，覆盖 5 个内点与 3 个外点。
func TestExpandedPanelConsumesPanelHandleAndMinimap(t *testing.T) {
	missionState := &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		UI:   state.MissionUIState{SidebarExpanded: true},
	}
	panel := newTestPanel(1)
	ui := calcLayout(missionState.View.Layout, true, panel.mapAspect)
	points := []struct {
		name string
		x, y int
	}{
		{name: "panel", x: int(ui.Panel.X + 10), y: int(ui.Panel.Y + 10)},
		{name: "handle", x: int(ui.Handle.X + ui.Handle.W/2), y: int(ui.Handle.Y + ui.Handle.H/2)},
		{name: "tab bar", x: int(ui.TabBar.X + ui.TabBar.W/2), y: int(ui.TabBar.Y + ui.TabBar.H/2)},
		{name: "minimap", x: int(ui.Map.X + ui.Map.W/2), y: int(ui.Map.Y + ui.Map.H/2)},
		{name: "units area", x: int(ui.Panel.X + 30), y: int(ui.Viewport.Y + ui.Viewport.H/2)},
	}
	for _, point := range points {
		if !panel.consumesCursorAt(missionState, point.x, point.y) {
			t.Fatalf("%s at (%d,%d) did not consume cursor", point.name, point.x, point.y)
		}
	}
	// 面板停靠右上角：面板下方与右缘细缝的战场区域不应被消费
	if panel.consumesCursorAt(missionState, 100, 100) {
		t.Fatal("battlefield point outside sidebar consumed cursor")
	}
	belowPanelY := int(ui.Panel.Y + ui.Panel.H + 10)
	if panel.consumesCursorAt(missionState, 1280-10, belowPanelY) {
		t.Fatal("battlefield point below panel consumed cursor")
	}
	// 右缘 12px 细缝留给镜头边缘滚动
	if panel.consumesCursorAt(missionState, 1280-4, 360) {
		t.Fatal("right screen gap consumed cursor")
	}
}

// TestCollapsedPanelOnlyConsumesHandle 验证收起态面板高度/单位视口为 0，只有顶部把手消费鼠标。
// 场景：玩家收起侧栏后，右上角大部分区域应还给战场，仅保留「下拉」把手。
// 预期：Panel.H=0、Viewport.W=0，Handle.Y=16 贴近顶缘，把手中心 true、原面板坐标 false。
// 覆盖：calcLayout(1280×720,false,1) 检查空面板/视口，再断言把手中心与 (Panel.X+10,Panel.Y+10) 的消费结果。
func TestCollapsedPanelOnlyConsumesHandle(t *testing.T) {
	missionState := &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		View: state.MissionViewState{Layout: layout.ScreenLayout{Width: 1280, Height: 720}},
		UI:   state.MissionUIState{SidebarExpanded: false},
	}
	panel := newTestPanel(1)
	ui := calcLayout(missionState.View.Layout, false, panel.mapAspect)

	if ui.Panel.H != 0 || ui.Viewport.W != 0 {
		t.Fatalf("collapsed panel = %+v, want empty panel and viewport", ui.Panel)
	}
	if ui.Handle.Y > panelMarginTop+handleGap+1 {
		t.Fatalf("collapsed handle Y = %v, want near screen top", ui.Handle.Y)
	}
	if !panel.consumesCursorAt(
		missionState,
		int(ui.Handle.X+ui.Handle.W/2),
		int(ui.Handle.Y+ui.Handle.H/2),
	) {
		t.Fatal("collapsed handle did not consume cursor")
	}
	if panel.consumesCursorAt(missionState, int(ui.Panel.X+10), int(ui.Panel.Y+10)) {
		t.Fatal("collapsed panel area consumed cursor")
	}
}

// TestCalcLayoutDockedPanelGeometry 验证展开面板停靠几何：宽度 260–360、贴顶贴右各留 12px、底部不出屏。
// 场景：1280×720 下按屏幕 86% 取高，小地图与单位区必须在面板内依次排下。
// 预期：W∈[260,360]、Y=12、右间隙=12、Y+H<720、Handle 在面板底、Viewport.H>0 且紧接小地图下 16px。
// 覆盖：calcLayout(1280×720,true,1) 后断言宽度范围、四条几何关系及 viewport 与小地图的衔接。
func TestCalcLayoutDockedPanelGeometry(t *testing.T) {
	ui := calcLayout(layout.ScreenLayout{Width: 1280, Height: 720}, true, 1)
	if ui.Panel.W < 260 || ui.Panel.W > 360 {
		t.Fatalf("panel width = %v, want within [260, 360]", ui.Panel.W)
	}
	// 面板贴角停靠：顶缘与右缘只留 12px 细缝，保证镜头边缘滚动可用
	if ui.Panel.Y != panelMarginTop {
		t.Fatalf("panel Y = %v, want %v", ui.Panel.Y, panelMarginTop)
	}
	if gap := 1280 - (ui.Panel.X + ui.Panel.W); gap != panelMarginRight {
		t.Fatalf("right gap = %v, want %v", gap, panelMarginRight)
	}
	if ui.Panel.H <= 0 || ui.Panel.Y+ui.Panel.H >= 720 {
		t.Fatalf("panel bottom = %v, want inside screen", ui.Panel.Y+ui.Panel.H)
	}
	// 把手挂在面板底缘下方
	if ui.Handle.Y < ui.Panel.Y+ui.Panel.H {
		t.Fatalf("handle Y = %v, want below panel bottom %v", ui.Handle.Y, ui.Panel.Y+ui.Panel.H)
	}
	if ui.Viewport.H <= 0 {
		t.Fatalf("viewport height = %v, want positive", ui.Viewport.H)
	}
	// 小地图下方直接接战舰信息，中间不再留资金/舰队统计卡
	wantViewportY := ui.Map.Y + ui.Map.H + 16
	if ui.Viewport.Y != wantViewportY {
		t.Fatalf("viewport Y = %v, want %v (immediately below map)", ui.Viewport.Y, wantViewportY)
	}
}

// TestCalcLayoutKeepsMapAspectForNonSquareMap 验证竖长地图的小地图保持比例，未触顶时用满内容宽度。
// 场景：珍珠港等 128×192 地图投到 1280×1440 屏幕；若强行拉成正方形会变形。
// 预期：Map.W/Map.H≈128/192，Map.W=Panel.W-32，X 在内容区居中（此处偏移 0）；失败意味着小地图变形或偏左。
// 覆盖：calcLayout(1280×1440,true,128/192)，断言比例误差<1e-6、宽度为 maxW、X 居中。
func TestCalcLayoutKeepsMapAspectForNonSquareMap(t *testing.T) {
	const (
		screenW = 1280.0
		screenH = 1440.0
		aspect  = 128.0 / 192.0 // 珍珠港等竖长地图
	)
	ui := calcLayout(layout.ScreenLayout{Width: int(screenW), Height: int(screenH)}, true, aspect)

	got := ui.Map.W / ui.Map.H
	if diff := got - aspect; diff < -1e-6 || diff > 1e-6 {
		t.Fatalf("minimap aspect = %v, want %v", got, aspect)
	}
	maxW := ui.Panel.W - 32
	if ui.Map.W != maxW {
		t.Fatalf("minimap width = %v, want %v (under height cap)", ui.Map.W, maxW)
	}
	// 水平居中于 (panelX+16, panelW-32) 的内容区
	wantX := ui.Panel.X + 16 + (maxW-ui.Map.W)/2
	if ui.Map.X != wantX {
		t.Fatalf("minimap X = %v, want %v", ui.Map.X, wantX)
	}
}

// TestCalcLayoutCapsOversizedMapHeight 验证竖长地图高度超过屏幕 45% 时等比缩小，避免挤出下方单位区。
// 场景：720 高窗口加载 128×192 地图，按面板宽度算出的小地图高 412.8px，超过上限 324px。
// 预期：Map.H≤324 且宽高比仍为 128/192，Viewport.H>0；失败意味着小地图压扁单位面板或破坏比例。
// 覆盖：calcLayout(1280×720,true,128/192)，断言 mapMaxHeightRatio 上限、比例误差与 viewport 高度。
func TestCalcLayoutCapsOversizedMapHeight(t *testing.T) {
	const screenH = 720.0
	ui := calcLayout(layout.ScreenLayout{Width: 1280, Height: int(screenH)}, true, 128.0/192.0)

	maxH := screenH * mapMaxHeightRatio
	if ui.Map.H > maxH {
		t.Fatalf("minimap height = %v, want <= %v", ui.Map.H, maxH)
	}
	got := ui.Map.W / ui.Map.H
	const aspect = 128.0 / 192.0
	if diff := got - aspect; diff < -1e-6 || diff > 1e-6 {
		t.Fatalf("minimap aspect = %v, want %v (capped height must keep ratio)", got, aspect)
	}
	if ui.Viewport.H <= 0 {
		t.Fatalf("viewport height = %v, want positive", ui.Viewport.H)
	}
}

// TestCenterCameraAtMinimapCentersClickedPoint 验证点击小地图中心后相机把该点放到视野中心，并放弃自动定位。
// 场景：玩家点小地图中央跳转视角；此前相机可能还有未完成的 Target 定位 (999,999)。
// 预期：Camera.Pos.RX≈点击点-5.5（容差 1 格）且 ≥0，Target 清空；失败意味着落点偏移或抢相机。
// 覆盖：128×192 地图、11×7 镜头与 stale target，点 Map 中心后反算 clickedRX 断言偏差/边界/清空。
func TestCenterCameraAtMinimapCentersClickedPoint(t *testing.T) {
	const (
		screenW = 1280.0
		screenH = 720.0
		mapW    = 128.0
		mapH    = 192.0
	)
	staleTarget := objPos.NewR(999, 999)
	ms := &state.MissionState{
		Core: state.MissionCoreState{
			MissionStatus: state.MissionRunning,
			MissionMD:     metadata.MissionMetadata{MapCfg: &mapcfg.MapCfg{Width: int(mapW), Height: int(mapH)}},
		},
		View: state.MissionViewState{
			Layout: layout.ScreenLayout{Width: int(screenW), Height: int(screenH)},
			Camera: state.Camera{Width: 11, Height: 7, Target: &staleTarget},
		},
	}
	panel := newTestPanel(mapW / mapH)
	panel.layout = calcLayout(ms.View.Layout, true, panel.mapAspect)

	// 点击小地图某点：该点应落在镜头视野中心
	cx := int(panel.layout.Map.X + panel.layout.Map.W/2)
	cy := int(panel.layout.Map.Y + panel.layout.Map.H/2)
	panel.centerCameraAtMinimap(ms, cx, cy)

	clickedRX := (float64(cx) - panel.layout.Map.X) / panel.layout.Map.W * mapW
	wantRX := clickedRX - float64(ms.View.Camera.Width)/2
	// 点击坐标取整会引入至多 1 屏幕像素（约 0.6 地图格）的偏差
	if diff := math.Abs(ms.View.Camera.Pos.RX - wantRX); diff > 1 {
		t.Fatalf("camera RX = %v, want %v (clicked point centered on camera)", ms.View.Camera.Pos.RX, wantRX)
	}
	if ms.View.Camera.Pos.RX < 0 {
		t.Fatalf("camera RX = %v, want clamped >= 0", ms.View.Camera.Pos.RX)
	}
	// 手动点击小地图要放弃进行中的相机定位，否则两者会抢相机。
	if ms.View.Camera.Target != nil {
		t.Fatalf("camera target not cleared by minimap click: %+v", ms.View.Camera.Target)
	}
}
