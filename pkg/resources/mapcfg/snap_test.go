package mapcfg

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/utils/grid"
)

// 回归：2026-10 存档里 pennsylvania 中心点压在 (51,88) 海岸格上，
// 旧实现从该格寻路永远失败，舰船永久搁浅。吸附后必须能寻出退出路径。
func TestGenPathFromStrandedCoastCell(t *testing.T) {
	cfg := GetByName("pearl_harbor")
	require.NotNil(t, cfg)
	require.Equal(t, ChrCoast, cfg.Map.Get(51, 88), "前提：该格应为海岸格")

	path := cfg.GenPath(grid.Point{X: 51, Y: 88}, grid.Point{X: 40, Y: 80})
	require.NotEmpty(t, path, "搁浅中心点必须能吸附出退出路径")
	require.True(t, cfg.Map.IsSea(path[0].X, path[0].Y), "吸附后的起点应在水面上")
	require.True(t, cfg.Map.IsSea(path[len(path)-1].X, path[len(path)-1].Y))
}

// 玩家点在码头/海岸格上时，目标吸附到最近水面，寻路应成功而不是静默失败。
func TestGenPathSnapsGoalOnCoast(t *testing.T) {
	cfg := GetByName("pearl_harbor")
	require.NotNil(t, cfg)
	require.Equal(t, ChrCoast, cfg.Map.Get(50, 88), "前提：该格应为海岸格")

	path := cfg.GenPath(grid.Point{X: 54, Y: 88}, grid.Point{X: 50, Y: 88})
	require.NotEmpty(t, path)
	require.True(t, cfg.Map.IsSea(path[len(path)-1].X, path[len(path)-1].Y), "终点应吸附到水面格")
}

// 从泊位出发的所有近域路径，逐段密集采样校验都不切过陆地/海岸格。
func TestGenPathLegsStayOnWater(t *testing.T) {
	cfg := GetByName("pearl_harbor")
	require.NotNil(t, cfg)

	berth := grid.Point{X: 54, Y: 88}
	for dx := -14; dx <= 14; dx++ {
		for dy := -16; dy <= 16; dy++ {
			goal := grid.Point{X: berth.X + dx, Y: berth.Y + dy}
			if !cfg.Map.IsSea(goal.X, goal.Y) {
				continue
			}
			path := cfg.GenPath(berth, goal)
			if len(path) == 0 {
				// 不连通的目标（封闭水域等）不在本用例范围内
				continue
			}
			for i := 1; i < len(path); i++ {
				require.False(t, segmentCrossesLand(cfg, path[i-1], path[i]),
					"目标 %v 的路径段 %v→%v 穿过了陆地格", goal, path[i-1], path[i])
			}
		}
	}
}

// segmentCrossesLand 以 0.1 格步长采样，检查线段实际经过的每个格子。
func segmentCrossesLand(cfg *MapCfg, a, b grid.Point) bool {
	length := math.Hypot(float64(b.X-a.X), float64(b.Y-a.Y))
	if length == 0 {
		return false
	}
	steps := int(math.Ceil(length / 0.1))
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Floor(float64(a.X) + t*float64(b.X-a.X)))
		y := int(math.Floor(float64(a.Y) + t*float64(b.Y-a.Y)))
		if chr := cfg.Map.Get(x, y); chr == ChrCoast || chr == ChrLand {
			return true
		}
	}
	return false
}

// 回归：搁浅在海岸格上的舰船，起点必须吸附到“与终点同一片水域”。
// 珍珠港 (59,122) 的最近水面是内湾 (58,121)，与主海域不连通，
// 旧实现会吸附到内湾，A* 直接失败，玩家点击后毫无反应。
func TestSnapPathEndpointsKeepsStrandedStartInGoalRegion(t *testing.T) {
	cfg := GetByName("pearl_harbor")
	require.NotNil(t, cfg)
	require.Equal(t, ChrCoast, cfg.Map.Get(59, 122), "前提：(59,122) 应为海岸格")
	require.True(t, cfg.Map.IsSea(58, 121), "前提：(58,121) 是内湾水面")

	lagoon := cfg.WaterRegionAt(58, 121)
	ocean := cfg.WaterRegionAt(59, 140)
	require.NotEqual(t, lagoon, ocean, "前提：内湾与主海域不连通")

	start, goal := cfg.SnapPathEndpoints(grid.Point{X: 59, Y: 122}, grid.Point{X: 59, Y: 140})
	require.Equal(t, ocean, cfg.WaterRegionAt(start.X, start.Y), "起点应吸附到终点所在水域")
	require.Equal(t, ocean, cfg.WaterRegionAt(goal.X, goal.Y), "终点本来就在主海域，不应被改动")

	path := cfg.GenPath(grid.Point{X: 59, Y: 122}, grid.Point{X: 59, Y: 140})
	require.NotEmpty(t, path, "搁浅舰必须能寻出通往主海域的路径")
	require.Equal(t, ocean, cfg.WaterRegionAt(path[0].X, path[0].Y))
}

// 右键点在岸上时，终点必须吸附到与起点同一片水域，寻路不能静默失败。
func TestSnapPathEndpointsKeepsCoastGoalInStartRegion(t *testing.T) {
	cfg := GetByName("pearl_harbor")
	require.NotNil(t, cfg)
	require.Equal(t, ChrCoast, cfg.Map.Get(59, 122), "前提：(59,122) 应为海岸格")

	start := grid.Point{X: 59, Y: 123}
	ocean := cfg.WaterRegionAt(start.X, start.Y)
	require.NotZero(t, ocean)

	snappedStart, goal := cfg.SnapPathEndpoints(start, grid.Point{X: 59, Y: 122})
	require.Equal(t, start, snappedStart, "起点在水面时不应被改动")
	require.Equal(t, ocean, cfg.WaterRegionAt(goal.X, goal.Y), "终点应吸附到起点所在水域")

	path := cfg.GenPath(start, grid.Point{X: 59, Y: 122})
	require.NotEmpty(t, path, "点在岸上时必须给出可达航线")
	require.True(t, cfg.Map.IsSea(path[len(path)-1].X, path[len(path)-1].Y), "终点必须是可航行格")
}

// 两端都点在陆格上时，也要落在同一片水域，避免寻路失败。
func TestSnapPathEndpointsBothOnLandShareRegion(t *testing.T) {
	cfg := GetByName("pearl_harbor")
	require.NotNil(t, cfg)
	require.True(t, cfg.Map.IsLand(59, 122), "前提：(59,122) 是海岸格")
	require.True(t, cfg.Map.IsLand(58, 122), "前提：(58,122) 是海岸格")

	start, goal := cfg.SnapPathEndpoints(grid.Point{X: 59, Y: 122}, grid.Point{X: 58, Y: 122})
	startRegion := cfg.WaterRegionAt(start.X, start.Y)
	require.NotZero(t, startRegion)
	require.Equal(t, startRegion, cfg.WaterRegionAt(goal.X, goal.Y), "两端应落在同一片水域")
}

// 水域连通域必须与寻路网格一致：同域 ⇔ Grid.Search 能寻出路径。
// 这条契约保证“同域吸附”不会把起点送进一条根本走不通的水域。
func TestWaterRegionsMatchGridSearch(t *testing.T) {
	rows := MapData{
		"SSSSSS",
		"SLSSLS",
		"SSLLSS",
		"SSSLSS",
		"SSLSSS",
		"SSSSSS",
	}
	cfg := &MapCfg{Map: rows, Cells: rows.ToGridCells(), Width: 6, Height: 6}
	points := []grid.Point{}
	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			if cfg.Map.IsSea(x, y) {
				points = append(points, grid.Point{X: x, Y: y})
			}
		}
	}
	for _, from := range points {
		for _, to := range points {
			sameRegion := cfg.WaterRegionAt(from.X, from.Y) == cfg.WaterRegionAt(to.X, to.Y)
			reachable := len(cfg.GenPath(from, to)) > 0
			require.Equal(
				t, sameRegion, reachable,
				"(%d,%d) → (%d,%d) 同域=%v 可达=%v", from.X, from.Y, to.X, to.Y, sameRegion, reachable,
			)
		}
	}
}

// 夹角处的水面不应被判为连通（对角需要两个正交邻格都可航行）。
func TestWaterRegionsDoNotConnectThroughLandCorner(t *testing.T) {
	rows := MapData{
		"LLL",
		"LSL",
		"LLS",
	}
	cfg := &MapCfg{Map: rows, Cells: rows.ToGridCells(), Width: 3, Height: 3}
	require.NotEqual(t, 0, cfg.WaterRegionAt(1, 1))
	require.NotEqual(t, 0, cfg.WaterRegionAt(2, 2))
	require.NotEqual(
		t,
		cfg.WaterRegionAt(1, 1),
		cfg.WaterRegionAt(2, 2),
		"夹角处的水面不应被判为连通",
	)
}

// 同域水面更远时，仍要优先吸附到同域水面，否则寻路会因跨水域而失败。
func TestSnapPathEndpointsPrefersFartherSameRegionWater(t *testing.T) {
	rows := MapData{
		"SSSSSSSSSSSSSSSSSSSSS",
		"SSSSSSSSSSSSSSSSSSSSS",
		"LLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLL",
		"LLLLLLLLLLLLLLLLLLLLL",
		"SSSSSSSSSSSSSSSSSSSSS",
		"SSSSSSSSSSSSSSSSSSSSS",
	}
	cfg := &MapCfg{Map: rows, Cells: rows.ToGridCells(), Width: 21, Height: 16}
	// 起点在上方海域，终点在陆地深处：最近水面是下方海域（3 格），
	// 但它与起点不连通；同域的上方海域在 10 格外，必须选它。
	start := grid.Point{X: 10, Y: 0}
	goal := grid.Point{X: 10, Y: 11}
	require.NotEqual(t, cfg.WaterRegionAt(10, 0), cfg.WaterRegionAt(10, 14), "前提：上下海域不连通")

	_, snappedGoal := cfg.SnapPathEndpoints(start, goal)
	require.Equal(t, cfg.WaterRegionAt(start.X, start.Y), cfg.WaterRegionAt(snappedGoal.X, snappedGoal.Y))

	path := cfg.GenPath(start, goal)
	require.NotEmpty(t, path, "点在远处陆格上时仍要给出同域航线")
	require.True(t, cfg.Map.IsSea(path[len(path)-1].X, path[len(path)-1].Y), "终点必须是可航行格")
}
