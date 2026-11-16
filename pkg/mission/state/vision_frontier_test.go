package state

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// markExplored 把圆心附近的格子并入已探索，并清掉当前可见，模拟多拍侦察后的视野图。
func markExplored(v *FactionVision, x, y, radius float64) {
	v.Stamp(x, y, radius)
	v.CommitExplored()
	v.ClearVisible()
}

// TestFarthestExploredPicksFarthestCell 验证选点取已探索格中距起点最远的一格。
// 场景：侦察机完成一次巡逻后需要折返去最远的已探索区域继续搜索；
// 若按行优先/最近点返回，飞机会在近处反复绕圈。
// 预期：返回格必须已探索，且到起点的距离等于全图已探索格的最大距离。
// 覆盖：在 20x20 视野中标记 (2,2)/(8,8)/(15,15) 三片区域，扫描全图算出最大距离后与返回值比较。
func TestFarthestExploredPicksFarthestCell(t *testing.T) {
	vision := NewFactionVision(20, 20)
	for _, p := range [][3]float64{{2, 2, 1}, {8, 8, 1}, {15, 15, 1}} {
		markExplored(vision, p[0], p[1], p[2])
	}
	from := objPos.New(0, 0)

	wantDist := -1.0
	for y := 0; y < vision.Height; y++ {
		for x := 0; x < vision.Width; x++ {
			if !vision.ExploredAt(x, y) {
				continue
			}
			wantDist = max(wantDist, math.Hypot(float64(x)-from.RX, float64(y)-from.RY))
		}
	}
	require.Greater(t, wantDist, 0.0)

	got, ok := vision.FarthestExplored(from, nil)
	require.True(t, ok)
	require.True(t, vision.ExploredAt(got.MX, got.MY))
	require.InDelta(t, wantDist, got.Distance(from), 1e-9)
}

// TestFarthestExploredSkipsCellsCoveredByOtherScouts 验证其他侦察机视距内的格子不参与选点。
// 场景：多架侦察机同时搜索，若已有机组覆盖的区域仍被选为最远点，会造成重复侦察和航程浪费。
// 预期：blocked 点 36 格内的已探索格全部跳过，返回 remaining 区域中的最远点。
// 覆盖：80x80 视野标记 (2,2)/(40,40)/(75,75)，blocked=(40,40)；断言结果落在 (75,75) 附近。
func TestFarthestExploredSkipsCellsCoveredByOtherScouts(t *testing.T) {
	vision := NewFactionVision(80, 80)
	for _, p := range [][3]float64{{2, 2, 1}, {40, 40, 1}, {75, 75, 1}} {
		markExplored(vision, p[0], p[1], p[2])
	}

	got, ok := vision.FarthestExplored(
		objPos.New(0, 0),
		[]objPos.MapPos{objPos.New(40, 40)},
	)
	require.True(t, ok)
	require.False(t, coveredByScout(got.MX, got.MY, []objPos.MapPos{objPos.New(40, 40)}))
	require.GreaterOrEqual(t, got.MX, 74)
	require.GreaterOrEqual(t, got.MY, 74)
}

// TestFarthestExploredReturnsFalseWhenNothingAvailable 验证空视野/全部被覆盖时返回 false。
// 场景：开局尚未侦察，或所有已探索格都在其他侦察机视距内；调用方必须能区分“没有目标”和“目标在 (0,0)”。
// 预期：nil 视野、空视野、唯一已探索格被 blocked 覆盖，三种情况都返回 ok=false。
// 覆盖：分别构造 nil、NewFactionVision(8,8)、标记 (2,2) 后 blocked=(2,2) 的视野并调用。
func TestFarthestExploredReturnsFalseWhenNothingAvailable(t *testing.T) {
	var nilVision *FactionVision
	_, ok := nilVision.FarthestExplored(objPos.New(0, 0), nil)
	require.False(t, ok)

	empty := NewFactionVision(8, 8)
	_, ok = empty.FarthestExplored(objPos.New(0, 0), nil)
	require.False(t, ok)

	covered := NewFactionVision(8, 8)
	markExplored(covered, 2, 2, 1)
	_, ok = covered.FarthestExplored(
		objPos.New(0, 0),
		[]objPos.MapPos{objPos.New(2, 2)},
	)
	require.False(t, ok)
}

// TestCoveredByScoutUsesClosedRadius 验证侦察机覆盖半径是 36 格闭区间（含边界）。
// 场景：多机搜索去重依赖这个半径；若边界处理成开区间，最近的两架飞机会在 36 格处各多搜一圈。
// 预期：距离 36 返回 true，距离 37 返回 false；对角 25,25（约 35.36）true，26,26（约 36.77）false。
// 覆盖：直接调用 coveredByScout，传入单点 blocked=(0,0) 与四个边界坐标。
func TestCoveredByScoutUsesClosedRadius(t *testing.T) {
	blocked := []objPos.MapPos{objPos.New(0, 0)}

	require.True(t, coveredByScout(36, 0, blocked))
	require.False(t, coveredByScout(37, 0, blocked))
	require.True(t, coveredByScout(0, -36, blocked))
	require.False(t, coveredByScout(0, -37, blocked))
	require.True(t, coveredByScout(25, 25, blocked))
	require.False(t, coveredByScout(26, 26, blocked))
}
