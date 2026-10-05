package mapcfg

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/utils/grid"
)

// coastMapCfg 构造 12x8 测试地图：y=0 是贯穿地图的陆地，其余是开阔水域。
// 沿 y=1 走是贴着岸线的直路（离岸 1 格），绕到 y=3 以外是离岸更远的绕路。
func coastMapCfg() *MapCfg {
	rows := MapData{
		"LLLLLLLLLLLL",
		"SSSSSSSSSSSS",
		"SSSSSSSSSSSS",
		"SSSSSSSSSSSS",
		"SSSSSSSSSSSS",
		"SSSSSSSSSSSS",
		"SSSSSSSSSSSS",
		"SSSSSSSSSSSS",
	}
	return &MapCfg{Map: rows, Cells: rows.ToGridCells(), Width: 12, Height: 8}
}

// touchedCells 沿合并后的路径段按 0.1 格步长采样，收集实际经过的格子。
func touchedCells(path []grid.Point, start grid.Point) map[grid.Point]bool {
	touched := map[grid.Point]bool{start: true}
	cur := start
	for _, next := range path {
		dx, dy := next.X-cur.X, next.Y-cur.Y
		steps := int(math.Ceil(math.Hypot(float64(dx), float64(dy)) / 0.1))
		if steps == 0 {
			touched[next] = true
			cur = next
			continue
		}
		for i := 0; i <= steps; i++ {
			t := float64(i) / float64(steps)
			touched[grid.Point{
				X: cur.X + int(math.Floor(t*float64(dx))),
				Y: cur.Y + int(math.Floor(t*float64(dy))),
			}] = true
		}
		cur = next
	}
	return touched
}

// maxTouchedY 返回路径经过的最大 y（越大说明离岸越远）。
func maxTouchedY(path []grid.Point, start grid.Point) int {
	maxY := start.Y
	for p := range touchedCells(path, start) {
		maxY = max(maxY, p.Y)
	}
	return maxY
}

// 离岸距离场：陆格为 0，可航行格为到最近陆格的切比雪夫距离。
func TestClearanceField(t *testing.T) {
	cfg := coastMapCfg()
	require.Equal(t, int16(0), cfg.ClearanceAt(5, 0), "陆地格离岸距离应为 0")
	require.Equal(t, int16(1), cfg.ClearanceAt(5, 1))
	require.Equal(t, int16(2), cfg.ClearanceAt(5, 2))
	require.Equal(t, int16(3), cfg.ClearanceAt(5, 3))
	require.Equal(t, int16(4), cfg.ClearanceAt(5, 4))
}

// 贴岸代价要随舰体尺寸缩放：小艇可以贴着岸线走，大舰要绕到离岸更远的水域。
func TestSearchCostScaleByShipSize(t *testing.T) {
	cfg := coastMapCfg()
	start, goal := grid.Point{X: 1, Y: 1}, grid.Point{X: 10, Y: 1}

	smallPath := cfg.PreparedGrid().SearchWithCostScale(start, goal, 0.1)
	require.Equal(t, 1, maxTouchedY(smallPath, start), "小艇应贴着岸线直走")

	largePath := cfg.PreparedGrid().SearchWithCostScale(start, goal, 3)
	require.GreaterOrEqual(t, maxTouchedY(largePath, start), 3, "大舰应绕到离岸 3 格以外")

	// 代价只是软约束：两种缩放下目标都可达
	require.NotEmpty(t, cfg.GenPath(start, goal))
	require.NotEmpty(t, cfg.PreparedGrid().SearchWithCostScale(start, goal, 100))
}
