package mapcfg

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/utils/grid"
)

// TestGenPathFromStrandedCoastCell 验证中心点搁浅在海岸格上的舰船能吸附出退出路径。
// 场景：2026-10 存档中 pennsylvania 中心点压在 pearl_harbor (51,88) 海岸格，旧实现从该格直接寻路永远失败，舰船永久搁浅。
// 预期：GenPath((51,88)→(40,80)) 返回非空路径，吸附后的起点与终点都在可航行水面；失败表示搁浅恢复链路再次中断。
// 覆盖：先断言 (51,88)=ChrCoast，再调用 GenPath 并用 Map.IsSea 检查 path[0] 与 path[len-1]。
func TestGenPathFromStrandedCoastCell(t *testing.T) {
	cfg := GetByName("pearl_harbor")
	require.NotNil(t, cfg)
	require.Equal(t, ChrCoast, cfg.Map.Get(51, 88), "前提：该格应为海岸格")

	path := cfg.GenPath(grid.Point{X: 51, Y: 88}, grid.Point{X: 40, Y: 80})
	require.NotEmpty(t, path, "搁浅中心点必须能吸附出退出路径")
	require.True(t, cfg.Map.IsSea(path[0].X, path[0].Y), "吸附后的起点应在水面上")
	require.True(t, cfg.Map.IsSea(path[len(path)-1].X, path[len(path)-1].Y))
}

// TestGenPathSnapsGoalOnCoast 验证玩家点在码头/海岸格时目标会吸附到最近水面并寻路成功。
// 场景：pearl_harbor (50,88) 为海岸格，起点 (54,88) 在水面；旧实现不吸附时会把岸上终点直接交给 A*，导致指令静默失败。
// 预期：GenPath 返回非空路径，且最终点 Map.IsSea=true，玩家点击码头不会毫无反应。
// 覆盖：先断言 (50,88)=ChrCoast，再调用 GenPath((54,88)→(50,88)) 检查路径终点。
func TestGenPathSnapsGoalOnCoast(t *testing.T) {
	cfg := GetByName("pearl_harbor")
	require.NotNil(t, cfg)
	require.Equal(t, ChrCoast, cfg.Map.Get(50, 88), "前提：该格应为海岸格")

	path := cfg.GenPath(grid.Point{X: 54, Y: 88}, grid.Point{X: 50, Y: 88})
	require.NotEmpty(t, path)
	require.True(t, cfg.Map.IsSea(path[len(path)-1].X, path[len(path)-1].Y), "终点应吸附到水面格")
}

// TestGenPathLegsStayOnWater 验证从泊位出发的近域航线逐段都不切过陆地/海岸。
// 场景：以 pearl_harbor 泊位 (54,88) 为起点，扫描 dx∈[-14,14]、dy∈[-16,16] 内所有水面目标，跳过不连通的封闭水域。
// 预期：每条非空路径的相邻点线段按 0.1 格步长采样后都只能经过 ChrSea/ChrDeepSea/ChrShallow，不能压到 ChrCoast/ChrLand。
// 覆盖：双层偏移循环 + GenPath，逐段调用 segmentCrossesLand，失败信息带目标格与路径段。
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

// TestSnapPathEndpointsKeepsStrandedStartInGoalRegion 验证搁浅起点吸附到与终点同一片连通水域。
// 场景：pearl_harbor (59,122) 是海岸格，最近水面 (58,121) 属于与主海域不连通的内湾；旧实现吸附到内湾后 A* 直接失败，玩家点击毫无反应。
// 预期：SnapPathEndpoints 把起点吸附到 (59,140) 所在主海域（WaterRegionAt 相等），终点保持不动；随后 GenPath 非空且 path[0] 也属于主海域。
// 覆盖：先断言 (59,122)=ChrCoast、(58,121) 为内湾且与 (59,140) 不同域，再调用 SnapPathEndpoints 与 GenPath 检查域编号。
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

// TestSnapPathEndpointsKeepsCoastGoalInStartRegion 验证右键点在岸上时终点吸附到起点所在水域。
// 场景：起点 (59,123) 在主海域水面，终点 (59,122) 是海岸格；若终点被吸到别的水域，A* 会失败且右键无反馈。
// 预期：起点原样返回，终点 WaterRegionAt 与起点相同；GenPath 返回非空路径且终点可航行。
// 覆盖：记录 start 的域编号，调用 SnapPathEndpoints 比较 snappedStart 与 goal 域，再调用 GenPath 检查末点。
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

// TestSnapPathEndpointsBothOnLandShareRegion 验证两端都点在陆格上时仍落到同一片水域。
// 场景：pearl_harbor (59,122) 与 (58,122) 都是海岸格，点对点拖拽可能两端都不在水面。
// 预期：先吸附起点得到非零水域，终点被吸到同一水域编号，保证后续寻路有解。
// 覆盖：调用 SnapPathEndpoints 后比较 startRegion 与 goal 的 WaterRegionAt，并断言 startRegion != 0。
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

// TestWaterRegionsMatchGridSearch 验证水域连通域标注与网格寻路完全一致：同域 ⇔ Grid.Search 可达。
// 场景：构造 6×6 含多块陆地的地图（其水面恰好同属一个连通域），枚举所有水面点对；这是“同域吸附”不会把船送进死水的契约。
// 预期：每一对水面点的 WaterRegionAt 相等性与 GenPath 返回非空完全一致，任一方向不符都失败。
// 覆盖：6×6 MapData 转 GridCells，双重遍历水面点对，用 require.Equal 比较 sameRegion 与 reachable。
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

// TestWaterRegionsDoNotConnectThroughLandCorner 验证对角水面不能从两块陆地的夹角“斜穿”成连通。
// 场景：3×3 地图 LLL/LSL/LLS，只有 (1,1) 与 (2,2) 两个水面，对角相邻但两个正交邻格都是陆地。
// 预期：两格 WaterRegionAt 均非 0 且编号不同；若相同说明连通域标注允许从陆角斜穿，与 grid 寻路不一致。
// 覆盖：构造 MapCfg 后分别取两格水域编号，再断言不相等。
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

// TestSnapPathEndpointsPrefersFartherSameRegionWater 验证同域水面更远时仍优先吸附同域，而不是最近但跨域的近水。
// 场景：21×16 地图上方两行与下方两行是海、中间 12 行陆地；起点 (10,0) 在上方海域，终点 (10,11) 深入陆地，最近水面是下方 3 格外的海域，但同域的上方水面在 10 格之外。
// 预期：SnapPathEndpoints 把终点吸到起点同域的上方水面（snapToSeaPreferredRadius=16 内），GenPath 非空且终点可航行。
// 覆盖：先断言 (10,0) 与 (10,14) 不同域，再比较 snappedGoal 与 start 的 WaterRegionAt，最后检查 GenPath。
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
