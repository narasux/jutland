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
