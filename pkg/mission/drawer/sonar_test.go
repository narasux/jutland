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

func TestSonarCoverageSingleDisk(t *testing.T) {
	arcs := sonarCoverageArcs([]sonarDisk{{x: 10, y: 20, radius: 30}})
	require.Len(t, arcs, 1)
	assert.InDelta(t, 10, arcs[0].x, 1e-9)
	assert.InDelta(t, 20, arcs[0].y, 1e-9)
	assert.InDelta(t, 30, arcs[0].radius, 1e-9)
	assert.InDelta(t, 2*math.Pi, totalArcAngle(arcs), 1e-9)
}

func TestSonarCoverageSeparateDisks(t *testing.T) {
	arcs := sonarCoverageArcs([]sonarDisk{
		{x: 0, y: 0, radius: 10},
		{x: 100, y: 0, radius: 10},
	})
	require.Len(t, arcs, 2)
	assert.InDelta(t, 4*math.Pi, totalArcAngle(arcs), 1e-9)
}

// 小圆被包含在大圆里时只保留外圆；两个完全重合的圆也只画一个。
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

// 两个相交圆各贡献 240° 外轮廓，重叠的内部弧不能再画。
func TestSonarCoverageOverlappingDisks(t *testing.T) {
	arcs := sonarCoverageArcs([]sonarDisk{
		{x: 0, y: 0, radius: 50},
		{x: 50, y: 0, radius: 50},
	})
	require.NotEmpty(t, arcs)
	assert.InDelta(t, 8*math.Pi/3, totalArcAngle(arcs), 1e-6)
}

// 不同半径的相交圆：大圆被小圆盖住一小段，小圆被大圆盖住大半，
// 余下弧段之和应等于两圆各自未被覆盖的角度。
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

// 三个圆围成一圈时会留下中心空洞，内边界也必须画出来，不能只画外轮廓。
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

// 小圆本身没有被任一大圆完全包含，但圆弧已被两圆的并集完整盖住，
// 此时不应再输出小圆的轮廓。
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

func TestSonarPathUsesCyan(t *testing.T) {
	opts := sonarPathOptions()
	require.True(t, opts.AntiAlias)
	c := colorx.SonarCyan
	assert.InDelta(t, float64(c.R)/255, float64(opts.ColorScale.R()), 1e-6)
	assert.InDelta(t, float64(c.G)/255, float64(opts.ColorScale.G()), 1e-6)
	assert.InDelta(t, float64(c.B)/255, float64(opts.ColorScale.B()), 1e-6)
	assert.InDelta(t, float64(c.A)/255, float64(opts.ColorScale.A()), 1e-6)
}

func TestDrawSingleShipSonarRangeDoesNotPanic(t *testing.T) {
	screen := ebiten.NewImage(400, 300)
	assert.NotPanics(t, func() {
		(&Drawer{}).drawSonarRanges(screen, newSonarDrawState(1))
	})
}

func TestDrawMergedSonarRangesDoesNotPanic(t *testing.T) {
	screen := ebiten.NewImage(400, 300)
	assert.NotPanics(t, func() {
		(&Drawer{}).drawSonarRanges(screen, newSonarDrawState(2))
	})
}
