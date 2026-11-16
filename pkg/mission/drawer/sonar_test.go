package drawer

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/layout"
)

func totalArcAngle(arcs []sonarArc) float64 {
	total := 0.0
	for _, arc := range arcs {
		total += arc.endAngle - arc.startAngle
	}
	return total
}

// TestSonarCoverageSingleDisk 验证单个探测圆输出为一段 0–2π 完整圆弧，并保留圆心与半径。
// 场景：只选中一艘反潜舰，其 SonarRange 形成一个圆，绘制层需要整圆轮廓。
// 预期：arcs 长度 1，x=10、y=20、radius=30，总角度 2π；失败意味着单圆被裁剪或角度归一化出错。
// 覆盖：sonarCoverageArcs 传入 1 个 sonarDisk{10,20,30}，用 totalArcAngle 累加 end-start 断言。
func TestSonarCoverageSingleDisk(t *testing.T) {
	arcs := sonarCoverageArcs([]sonarDisk{{x: 10, y: 20, radius: 30}})
	require.Len(t, arcs, 1)
	assert.InDelta(t, 10, arcs[0].x, 1e-9)
	assert.InDelta(t, 20, arcs[0].y, 1e-9)
	assert.InDelta(t, 30, arcs[0].radius, 1e-9)
	assert.InDelta(t, 2*math.Pi, totalArcAngle(arcs), 1e-9)
}

// TestSonarCoverageSeparateDisks 验证互不相交的两个探测圆各自保留完整圆周，不互相吞掉。
// 场景：两艘选中舰相距 100 像素、半径各 10，覆盖范围完全分离；并集轮廓应是两个整圆。
// 预期：arcs 长度 2，总角度 4π；失败意味着距离判定把独立圆误判为包含或隐藏。
// 覆盖：两个 sonarDisk（0,0,r10 与 100,0,r10），断言弧段数量与总角度。
func TestSonarCoverageSeparateDisks(t *testing.T) {
	arcs := sonarCoverageArcs([]sonarDisk{
		{x: 0, y: 0, radius: 10},
		{x: 100, y: 0, radius: 10},
	})
	require.Len(t, arcs, 2)
	assert.InDelta(t, 4*math.Pi, totalArcAngle(arcs), 1e-9)
}

// TestSonarCoverageContainedAndIdenticalDisks 验证被包含的小圆与完全重合的圆只保留一份外轮廓。
// 场景：第一组 (100,100,r50) 内套同心 (100,100,r10) 与偏移 10 的 (110,100,r10)，圆心距都不超过 |50-10|；第二组是两个圆心与半径完全相同的 (5,5,r20)。
// 预期：两组都只剩 1 段弧；第一组半径 50（只画大圆），第二组半径 20（distance==0 且 ri==rj 时只隐藏后一个）；失败意味着内部圆重复描边或重合圆画两遍。
// 覆盖：两次 sonarCoverageArcs 调用，分别断言 Len 1 + 半径 50 与 Len 1 + 半径 20（1e-9 容差）。
func TestSonarCoverageContainedAndIdenticalDisks(t *testing.T) {
	arcs := sonarCoverageArcs([]sonarDisk{
		{x: 100, y: 100, radius: 50},
		{x: 100, y: 100, radius: 10},
		{x: 110, y: 100, radius: 10},
	})
	require.Len(t, arcs, 1)
	assert.InDelta(t, 50, arcs[0].radius, 1e-9)

	arcs = sonarCoverageArcs([]sonarDisk{
		{x: 5, y: 5, radius: 20},
		{x: 5, y: 5, radius: 20},
	})
	require.Len(t, arcs, 1)
	assert.InDelta(t, 20, arcs[0].radius, 1e-9)
}

// TestSonarCoverageOverlappingDisks 验证两圆相交时只保留各自未被对方覆盖的外侧弧。
// 场景：(0,0,r50) 与 (50,0,r50) 圆心距 50，交点对应半角 beta=acos(0.5)=60°，每个圆各被盖住 120°。
// 预期：每个圆只贡献 240°（4π/3）外轮廓，总弧角 8π/3；失败意味着内部重叠弧也被画出，声呐圈出现交叉线。
// 覆盖：sonarCoverageArcs 返回非空后，用 totalArcAngle 汇总 end-start，按 1e-6 容差断言 8π/3。
func TestSonarCoverageOverlappingDisks(t *testing.T) {
	arcs := sonarCoverageArcs([]sonarDisk{
		{x: 0, y: 0, radius: 50},
		{x: 50, y: 0, radius: 50},
	})
	require.NotEmpty(t, arcs)
	assert.InDelta(t, 8*math.Pi/3, totalArcAngle(arcs), 1e-6)
}

// TestSonarCoverageDifferentRadii 验证不同半径的圆相交时，各圆剩余弧角按余弦定理算出。
// 场景：(0,0,r100) 与 (90,0,r20) 圆心距 90，小圆未被大圆完全包含（90 > |100-20|=80）；大圆只被小圆盖住 2·bigBeta≈20.9°，小圆则大半被大圆盖住。
// 预期：剩余弧角 = (2π-2·bigBeta) + (2π-2·smallBeta)，其中 bigBeta=acos((90²+100²-20²)/(2·90·100))、smallBeta=acos((90²+20²-100²)/(2·90·20))；失败意味着半径差参与了距离判定却算错。
// 覆盖：sonarCoverageArcs 返回非空后，测试内复算两个 beta 拼出期望值，用 totalArcAngle 按 1e-9 容差比较。
func TestSonarCoverageDifferentRadii(t *testing.T) {
	arcs := sonarCoverageArcs([]sonarDisk{
		{x: 0, y: 0, radius: 100},
		{x: 90, y: 0, radius: 20},
	})
	require.NotEmpty(t, arcs)
	bigBeta := math.Acos(float64(90*90+100*100-20*20) / float64(2*90*100))
	smallBeta := math.Acos(float64(90*90+20*20-100*100) / float64(2*90*20))
	want := (2*math.Pi - 2*bigBeta) + (2*math.Pi - 2*smallBeta)
	assert.InDelta(t, want, totalArcAngle(arcs), 1e-9)
}

// TestSonarCoverageKeepsHoleBoundary 验证三圆围合时中心空洞的内边界也要输出，不能只画外轮廓。
// 场景：半径 95 的三个圆圆心为 (100,0)、(-50,86.6025)、(-50,-86.6025)，构成外接圆半径 100 的正三角形（边长 173.2）；三圆两两相交并围住原点，中心留有未被覆盖的空洞，内边界距原点最近约 5 像素。
// 预期：arcs 非空，且存在弧段中点落在半径 95 以内（math.Hypot(px,py)<95），说明朝内的圆弧被画到；失败意味着只描外圈，空洞边界漏画。
// 覆盖：sonarCoverageArcs 后遍历每段弧，取 (start+end)/2 角度算中点坐标，foundInner 为真则断言通过。
func TestSonarCoverageKeepsHoleBoundary(t *testing.T) {
	arcs := sonarCoverageArcs([]sonarDisk{
		{x: 100, y: 0, radius: 95},
		{x: -50, y: 86.6025, radius: 95},
		{x: -50, y: -86.6025, radius: 95},
	})
	require.NotEmpty(t, arcs)
	foundInner := false
	for _, arc := range arcs {
		mid := (arc.startAngle + arc.endAngle) / 2
		px := arc.x + arc.radius*math.Cos(mid)
		py := arc.y + arc.radius*math.Sin(mid)
		if math.Hypot(px, py) < 95 {
			foundInner = true
		}
	}
	assert.True(t, foundInner, "three overlapping circles should leave a hole boundary")
}

// TestSonarCoverageUnionCoversSmallDisk 验证小圆边界被两圆并集完整覆盖时，不再输出小圆轮廓。
// 场景：(0,0,r5) 与 (±8,0,r10) 两圆相交；原点小圆与每个大圆的圆心距 8 落在 (|10-5|, 10+5) 内，没有被任何单圆完全包含。
// 预期：返回的 arcs 非空，但不存在 arc.radius==5，即小圆整圈已被并集盖住；失败意味着声呐圈中心多出一圈多余轮廓。
// 覆盖：sonarCoverageArcs 后遍历全部弧段，用 assert.NotEqual 断言半径不等于 5.0。
func TestSonarCoverageUnionCoversSmallDisk(t *testing.T) {
	arcs := sonarCoverageArcs([]sonarDisk{
		{x: 0, y: 0, radius: 5},
		{x: -8, y: 0, radius: 10},
		{x: 8, y: 0, radius: 10},
	})
	require.NotEmpty(t, arcs)
	for _, arc := range arcs {
		assert.NotEqual(t, 5.0, arc.radius)
	}
}

func newSonarDrawState(selectedCount int) *state.MissionState {
	ms := &state.MissionState{
		View: state.MissionViewState{
			Layout: layout.ScreenLayout{Width: 400, Height: 300},
			Camera: state.Camera{Width: 20, Height: 20},
		},
		UI: state.MissionUIState{
			ShowSonarRanges: true,
			GameOpts:        state.GameOptions{Zoom: state.DefaultZoom()},
		},
		Player: state.MissionPlayerState{CurPlayer: faction.HumanAlpha},
		Arena:  state.MissionArenaState{Ships: map[string]*objUnit.BattleShip{}},
	}
	for i := 0; i < selectedCount; i++ {
		ship := &objUnit.BattleShip{
			Uid: string(rune('a' + i)), BelongPlayer: faction.HumanAlpha,
			CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeDestroyer,
			CurPos: objPos.NewR(5+float64(i)*3, 5), Length: 64, Width: 8,
		}
		ms.Arena.PutShip(ship)
		ms.Interaction.SelectedShips = append(ms.Interaction.SelectedShips, ship.Uid)
	}
	return ms
}

// TestSonarPathUsesCyan 验证声呐轮廓显式染成 SonarCyan 并开启抗锯齿，避免默认白色。
// 场景：多选声呐并集用 vector.StrokePath 绘制；DrawPathOptions 零值 ColorScale 是白色，漏染色会画成白圈。
// 预期：AntiAlias=true，ColorScale 的 RGBA 分别等于 SonarCyan/255；失败意味着声呐圈与主题色不一致。
// 覆盖：调用 sonarPathOptions()，按 1e-6 容差逐通道对比 colorx.SonarCyan。
func TestSonarPathUsesCyan(t *testing.T) {
	opts := sonarPathOptions()
	require.True(t, opts.AntiAlias)
	c := colorx.SonarCyan
	assert.InDelta(t, float64(c.R)/255, float64(opts.ColorScale.R()), 1e-6)
	assert.InDelta(t, float64(c.G)/255, float64(opts.ColorScale.G()), 1e-6)
	assert.InDelta(t, float64(c.B)/255, float64(opts.ColorScale.B()), 1e-6)
	assert.InDelta(t, float64(c.A)/255, float64(opts.ColorScale.A()), 1e-6)
}

// TestDrawSonarRangesDoesNotPanic 验证单舰圆与多舰并集两条绘制分支都不会因裁剪/合并越界 panic。
// 场景：400×300 屏幕边缘显示 1 艘或 2 艘相交声呐圈；单圆走裁剪，多圆走并集轮廓。
// 预期：drawSonarRanges 两种输入都正常返回；失败意味着圆弧裁剪/并集在边缘坐标产生非法路径。
// 覆盖：表驱动 newSonarDrawState(1)/newSonarDrawState(2)，各自在 400×300 图像上 assert.NotPanics。
func TestDrawSonarRangesDoesNotPanic(t *testing.T) {
	for _, tc := range []struct {
		name  string
		ships int
	}{
		{name: "single ship", ships: 1},
		{name: "merged ships", ships: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			screen := ebiten.NewImage(400, 300)
			assert.NotPanics(t, func() {
				(&Drawer{}).drawSonarRanges(screen, newSonarDrawState(tc.ships))
			})
		})
	}
}
