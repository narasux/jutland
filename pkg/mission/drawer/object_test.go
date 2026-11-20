package drawer

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objTrail "github.com/narasux/jutland/pkg/mission/object/trail"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	diveImg "github.com/narasux/jutland/pkg/resources/images/dive"
	weaponImg "github.com/narasux/jutland/pkg/resources/images/weapon"
)

// TestTrailHiddenByFog 验证敌舰尾流在迷雾外被隐藏，圈内/己方/无归属尾流照常绘制。
// 场景：16×16 视野用 Stamp(5,5,1) 点亮 (5,5) 附近的格子；敌方 ComputerAlpha 的尾流分别放在圆外 (9,9) 与圆内 (5,5)，己方尾流和无归属尾流也放在 (9,9)。
// 预期：圆外敌舰尾流返回 true 被跳过，圆内敌舰尾流、己方尾流、BelongPlayer 为空的尾流都返回 false；失败意味着顺着尾流就能看出迷雾里的舰队在哪。
// 覆盖：trailHiddenByFog 四组输入——敌方 (9,9)、敌方 (5,5)、己方 (9,9)、无归属 (9,9)，逐一断言 bool；测试尾流不带 OwnerUid，只走 ConcealsEnemy 分支。
func TestTrailHiddenByFog(t *testing.T) {
	vision := state.NewFactionVision(16, 16)
	vision.Stamp(5, 5, 1)
	ms := &state.MissionState{
		Core: state.MissionCoreState{FogOfWar: true},
		Player: state.MissionPlayerState{
			CurPlayer: faction.HumanAlpha,
			Visions:   map[faction.Player]*state.FactionVision{faction.HumanAlpha: vision},
		},
	}

	assert.True(t,
		trailHiddenByFog(ms, &objTrail.Trail{BelongPlayer: faction.ComputerAlpha, Pos: objPos.New(9, 9)}),
		"unseen enemy wake should be hidden")
	assert.False(t,
		trailHiddenByFog(ms, &objTrail.Trail{BelongPlayer: faction.ComputerAlpha, Pos: objPos.New(5, 5)}),
		"enemy wake inside the sight circle should still draw")
	assert.False(t,
		trailHiddenByFog(ms, &objTrail.Trail{BelongPlayer: faction.HumanAlpha, Pos: objPos.New(9, 9)}),
		"own wake is never hidden")
	assert.False(t, trailHiddenByFog(ms, &objTrail.Trail{Pos: objPos.New(9, 9)}),
		"trail without an owner stays as before")
}

// TestSonarRangesHiddenByDefault 验证按住 V 才显示的声呐圈在默认关闭时立即返回，不绘制也不崩。
// 场景：空 MissionState 的 UI.ShowSonarRanges 为 false（该开关只在按住 V 时置 true），相机 10×10 内有一艘选中的驱逐舰 dd，绘制入口收到 nil 屏幕。
// 预期：drawSonarRanges 在第一道 !ShowSonarRanges 判断处返回，不读取 nil 屏幕与选中舰；失败意味着松开 V 后仍在画圈，或空绘制状态下崩溃。
// 覆盖：手搓 ms（Arena.PutShip 一艘 ShipTypeDestroyer/CurHP=100 的 dd + SelectedShips=[dd]），assert.NotPanics 调用 (&Drawer{}).drawSonarRanges(nil, ms)。
func TestSonarRangesHiddenByDefault(t *testing.T) {
	ms := &state.MissionState{}
	ms.View.Camera.Width = 10
	ms.View.Camera.Height = 10
	ms.Arena.PutShip(&objUnit.BattleShip{
		Uid: "dd", Type: objUnit.ShipTypeDestroyer, CurHP: 100,
	})
	ms.Interaction.SelectedShips = []string{"dd"}

	assert.NotPanics(t, func() {
		(&Drawer{}).drawSonarRanges(nil, ms)
	})
}

// TestRotatedRectangleCorners 验证命中框四角按朝向旋转：长度初始沿 Y 轴，角度增大顺时针转。
// 场景：调试命中框要把中心 (10,20)、长 8、宽 2 的矩形按 0°/45°/90° 旋转，角度取自单位 CurRotation。
// 预期：0° 四角为 (9,16)(11,16)(11,24)(9,24)，90° 长轴落到 X 轴；失败意味着框与舰体错向。
// 覆盖：表驱动 3 个角度，用 1e-9 容差逐项断言四个角的 8 个坐标分量。
func TestRotatedRectangleCorners(t *testing.T) {
	tests := []struct {
		name     string
		rotation float64
		want     [4][2]float64
	}{
		{
			name:     "zero degrees keeps length on Y axis",
			rotation: 0,
			want: [4][2]float64{
				{9, 16},
				{11, 16},
				{11, 24},
				{9, 24},
			},
		},
		{
			name:     "forty-five degrees rotates clockwise",
			rotation: 45,
			want: [4][2]float64{
				{12.121320343559642, 16.464466094067262},
				{13.535533905932738, 17.878679656440358},
				{7.878679656440358, 23.535533905932738},
				{6.464466094067262, 22.121320343559642},
			},
		},
		{
			name:     "ninety degrees rotates length onto X axis",
			rotation: 90,
			want: [4][2]float64{
				{14, 19},
				{14, 21},
				{6, 21},
				{6, 19},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rotatedRectangleCorners(10, 20, 8, 2, tt.rotation)
			for idx := range got {
				assert.InDelta(t, tt.want[idx][0], got[idx][0], 1e-9)
				assert.InDelta(t, tt.want[idx][1], got[idx][1], 1e-9)
			}
		})
	}
}

// TestPlaneIconStatus 验证舰载机图标的绿/黄/灰三态由甲板备机、空中数与起飞开关共同决定。
// 场景：planeIconStatus 依次看 ShipAircraft.Disable、StandbyCount()（各机组 CurCount 之和）与空中飞机数；表驱动覆盖甲板备机 3 架、甲板空但空中 6 架、甲板空且全灭、关闭但甲板 12 架且空中 4 架。
// 预期：分别得到 WeaponStatusLoaded（绿，可起飞）、WeaponStatusReloading（黄，已出击或返航）、WeaponStatusDisabled（灰，无机可用）；Disable 优先于备机数，满甲板也显示禁用。
// 覆盖：4 行表驱动调用 planeIconStatus(&aircraft, airborne)，用 assert.Equal 比对 weaponImg.WeaponStatus 枚举。
func TestPlaneIconStatus(t *testing.T) {
	tests := []struct {
		name     string
		aircraft objUnit.ShipAircraft
		airborne int
		want     weaponImg.WeaponStatus
	}{
		{
			name:     "甲板有备机时可以起飞",
			aircraft: objUnit.ShipAircraft{Groups: []objUnit.PlaneGroup{{Name: "F", CurCount: 3}}},
			want:     weaponImg.WeaponStatusLoaded,
		},
		{
			name:     "甲板已空但飞机在空或返航",
			aircraft: objUnit.ShipAircraft{Groups: []objUnit.PlaneGroup{{Name: "F", CurCount: 0}}},
			airborne: 6,
			want:     weaponImg.WeaponStatusReloading,
		},
		{
			name:     "甲板已空且飞机全部损失",
			aircraft: objUnit.ShipAircraft{Groups: []objUnit.PlaneGroup{{Name: "F", CurCount: 0}}},
			want:     weaponImg.WeaponStatusDisabled,
		},
		{
			name: "起飞被关闭时满甲板也显示禁用",
			aircraft: objUnit.ShipAircraft{
				Disable: true,
				Groups:  []objUnit.PlaneGroup{{Name: "F", CurCount: 12}},
			},
			airborne: 4,
			want:     weaponImg.WeaponStatusDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, planeIconStatus(&tt.aircraft, tt.airborne))
		})
	}
}

// TestAirbornePlaneCounts 验证空中飞机按 BelongShip 聚合，回收后已从集合移除的飞机不再计入。
// 场景：Arena.Planes 有 p1/p2 属舰 a、p3 属舰 b、p4 为 nil 占位；飞机返航入库后会从集合删除，因此无需再扣减。
// 预期：{"a":2,"b":1}，nil 项被跳过；传入 nil 飞机表返回空 map（非 nil），assert.Empty 成立；失败意味着机队行或图标状态的空中数错位。
// 覆盖：airbornePlaneCounts(4 项含 nil) 断言映射内容，再对 nil 输入断言 Empty。
func TestAirbornePlaneCounts(t *testing.T) {
	planes := map[string]*objUnit.Plane{
		"p1": {Uid: "p1", BelongShip: "a"},
		"p2": {Uid: "p2", BelongShip: "a"},
		"p3": {Uid: "p3", BelongShip: "b"},
		"p4": nil,
	}

	assert.Equal(t, map[string]int{"a": 2, "b": 1}, airbornePlaneCounts(planes))
	assert.Empty(t, airbornePlaneCounts(nil))
}

// TestSubmarineDepthVisualEndpoints 验证潜航淡出的两端取值：水面全不透明，下潜完成用固定透明度与缩放。
// 场景：submarineDepthVisual 按 ship.DiveProgress() 线性插值，水面状态 progress=0、Submerged 状态 progress=1。
// 预期：水面 alpha=1.0、scale=1.0；水下 alpha=submarineSubmergedAlpha(0.55)、scale=submarineSubmergedScale(0.86)；失败意味着潜航时舰体不变淡或缩放过量。
// 覆盖：一艘 BattleShip 依次把 Submarine.DiveState 改为 Surface/Submerged，两次调用后各用 1e-9 容差断言 alpha 与 scale。
func TestSubmarineDepthVisualEndpoints(t *testing.T) {
	ship := &objUnit.BattleShip{
		Type: objUnit.ShipTypeSubmarine, Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSurface},
	}
	alpha, scale := submarineDepthVisual(ship)
	assert.InDelta(t, 1.0, alpha, 1e-9)
	assert.InDelta(t, 1.0, scale, 1e-9)

	ship.Submarine.DiveState = objUnit.DiveStateSubmerged
	alpha, scale = submarineDepthVisual(ship)
	assert.InDelta(t, submarineSubmergedAlpha, alpha, 1e-9)
	assert.InDelta(t, submarineSubmergedScale, scale, 1e-9)
}

// TestDiveBadgeLayoutStates 验证潜航状态牌：潜艇四种深度各有对应图标，非潜艇不显示，尺寸随缩放。
// 场景：newDiveBadgeLayout 同时供绘制与武器图标行使用；表驱动覆盖 Surface/Diving/Submerged/Surfacing 四种 DiveState，另加一艘驱逐舰作反例，ms 使用 DefaultZoom。
// 预期：潜艇 visible=true 且 state 依次为 diveImg.Surface/Diving/Submerged/Surfacing；驱逐舰 visible=false 直接返回；可见时 width = weaponIconBaseSize(16) * ms.ZoomScale()。
// 覆盖：5 行表驱动构造 BattleShip（CurHP=100）调用 newDiveBadgeLayout，断言 visible/state/width，非可见行提前 return。
func TestDiveBadgeLayoutStates(t *testing.T) {
	ms := &state.MissionState{
		UI: state.MissionUIState{GameOpts: state.GameOptions{Zoom: state.DefaultZoom()}},
	}
	tests := []struct {
		name     string
		shipType objUnit.ShipType
		state    objUnit.DiveState
		want     diveImg.State
		visible  bool
	}{
		{
			name: "水面显示水面图标", shipType: objUnit.ShipTypeSubmarine,
			state: objUnit.DiveStateSurface, want: diveImg.Surface, visible: true,
		},
		{
			name: "下潜中显示下潜图标", shipType: objUnit.ShipTypeSubmarine,
			state: objUnit.DiveStateDiving, want: diveImg.Diving, visible: true,
		},
		{
			name: "水下显示水下图标", shipType: objUnit.ShipTypeSubmarine,
			state: objUnit.DiveStateSubmerged, want: diveImg.Submerged, visible: true,
		},
		{
			name: "上浮中显示上浮图标", shipType: objUnit.ShipTypeSubmarine,
			state: objUnit.DiveStateSurfacing, want: diveImg.Surfacing, visible: true,
		},
		{name: "非潜艇不显示", shipType: objUnit.ShipTypeDestroyer, state: objUnit.DiveStateSubmerged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ship := &objUnit.BattleShip{
				Type: tt.shipType, CurHP: 100,
				Submarine: objUnit.SubmarineState{DiveState: tt.state},
			}
			layout := newDiveBadgeLayout(ms, ship)
			assert.Equal(t, tt.visible, layout.visible)
			if !tt.visible {
				return
			}
			assert.Equal(t, tt.want, layout.state)
			assert.InDelta(t, weaponIconBaseSize*ms.ZoomScale(), layout.width, 1e-9)
		})
	}
}

// TestDrawDiveOverlays 验证下潜/上浮覆盖层在空、半满、满进度下都能绘制且不崩溃。
// 场景：下潜用青色 diveImg.GetUnderwater(Diving,2) 从底部往上贴，上浮用绿色 diveImg.Get(Surfacing,2) 从顶部往下贴；进度取 0、0.1、0.5、1。
// 预期：drawDiveOverlayBottom/drawDiveOverlayTop 对 80×80 屏幕、偏移 (10,10)、scale 0.5 均正常返回；progress=0 时 fillRows=0 提前返回，1 时铺满图标高度。
// 覆盖：ebiten.NewImage(80,80) + 双层循环（progress × bottom/top）包在 assert.NotPanics 中；只验证绘制路径，不做像素断言。
func TestDrawDiveOverlays(t *testing.T) {
	screen := ebiten.NewImage(80, 80)
	cyan := diveImg.GetUnderwater(diveImg.Diving, 2)
	green := diveImg.Get(diveImg.Surfacing, 2)
	assert.NotPanics(t, func() {
		for _, progress := range []float64{0, 0.1, 0.5, 1} {
			drawDiveOverlayBottom(screen, cyan, progress, 10, 10, 0.5)
			drawDiveOverlayTop(screen, green, progress, 10, 10, 0.5)
		}
	})
}
