package grid

import (
	"math"
	"testing"
)

// TestDiagonalNeighborsRequireBothOrthogonalOpen 验证对角移动要求两个正交邻格都开放，不能贴墙角斜穿。
// 场景：3x3 网格上下正中是墙，中心格只剩左右直行；若允许斜穿，单位会切过墙的角。
// 预期：中心格的邻居只有 (0,1)/(2,1)，四个对角邻居都被挡住。
// 覆盖：构造 W 在 (1,0)/(1,2) 的网格，调用 getNeighbors 并逐点比较。
func TestDiagonalNeighborsRequireBothOrthogonalOpen(t *testing.T) {
	cells := make(Cells, 3)
	for y := range cells {
		cells[y] = make([]int, 3)
	}
	// 布局（上下正中是墙）：
	// . W .
	// . . .
	// . W .
	cells[0][1] = W
	cells[2][1] = W
	g := NewGrid(cells)

	var buf [8]Point
	nbrs := buf[:g.getNeighbors(Point{X: 1, Y: 1}, &buf)]
	// 只剩左右两个直行邻居，四个对角全被正交墙挡住
	want := []Point{{X: 0, Y: 1}, {X: 2, Y: 1}}
	if len(nbrs) != len(want) {
		t.Fatalf("neighbors = %v, want %v", nbrs, want)
	}
	for i := range want {
		if nbrs[i] != want[i] {
			t.Fatalf("neighbors = %v, want %v", nbrs, want)
		}
	}
}

// TestSegmentBlockedKeepsOneCellClearance 验证线段校验会拦下擦墙线段，并保留一格余量。
// 场景：舰体沿合并后的航点直线行驶，转向弧线/被拦停重启会偏离半个格；若只按格心判碰撞，会切进陆格。
// 预期：擦过墙格 (2,2) 的线段受阻；离墙一格受阻；离墙两格以上放行。
// 覆盖：5x6 网格放一堵墙，三组线段调用 segmentBlocked 断言。
func TestSegmentBlockedKeepsOneCellClearance(t *testing.T) {
	cells := make(Cells, 5)
	for y := range cells {
		cells[y] = make([]int, 6)
	}
	// 墙在 (2,2)
	cells[2][2] = W
	g := NewGrid(cells)

	if !g.segmentBlocked(Point{X: 4, Y: 4}, Point{X: 0, Y: 0}) {
		t.Fatal("擦过墙格的线段应当被判定为受阻")
	}
	if !g.segmentBlocked(Point{X: 3, Y: 4}, Point{X: 3, Y: 0}) {
		t.Fatal("离墙只有一格的线段应当被判定为受阻")
	}
	if g.segmentBlocked(Point{X: 5, Y: 4}, Point{X: 5, Y: 0}) {
		t.Fatal("离墙两格以上的线段不应被判为受阻")
	}
}

// TestSearchLeavesShallowCellsUntouched 验证寻路不会改写网格里的浅海标记，重复搜索结果稳定。
// 场景：5×8 网格仅 (3,2) 为浅海 SD，起点 (1,2)、终点 (6,2) 的直路正好经过它。
// 预期：两次 Search 后 cells[2][3] 仍为 SD；两条合并路径非空且逐点一致；
// 失败表示寻路污染地图数据（SD 被覆盖后浅海代价失效），或池化工作区残留导致结果漂移。
// 覆盖：构造 5 行 × 8 列 Cells，设置 SD 后连续两次 Search(Point{1,2}, Point{6,2})，先比长度再逐点比。
func TestSearchLeavesShallowCellsUntouched(t *testing.T) {
	cells := make(Cells, 5)
	for y := range cells {
		cells[y] = make([]int, 8)
	}
	cells[2][3] = SD
	g := NewGrid(cells)

	first := g.Search(Point{1, 2}, Point{6, 2})
	second := g.Search(Point{1, 2}, Point{6, 2})
	if cells[2][3] != SD {
		t.Fatalf("shallow cell = %d, want %d", cells[2][3], SD)
	}
	if len(first) == 0 {
		t.Fatal("search found no path")
	}
	if len(first) != len(second) {
		t.Fatalf("paths differ in length: %v vs %v", first, second)
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("paths differ: %v vs %v", first, second)
		}
	}
}

// TestHeapMatchesLinearOpenSet 验证堆开集 A* 与线性扫描参照实现给出相同最优代价。
// 场景：开放区存在大量等代价平局，逐点路径可以不同，但最低代价必须一致；
// 下半区加罚让最优路径唯一，堆弹出次序错误会立刻暴露。
// 预期：两条路径非空，且 pathCost 都等于 8；若选中被罚的下路，代价会变成 18+。
// 覆盖：6x8 网格放两堵墙、下半区加 10 代价，分别跑 Search 与 searchLinear，比较路径代价。
func TestHeapMatchesLinearOpenSet(t *testing.T) {
	cells := make(Cells, 6)
	for y := range cells {
		cells[y] = make([]int, 8)
	}
	cells[2][3] = W
	cells[3][3] = W
	g := NewGrid(cells)
	costs := make([]float32, 6*8)
	for y := 4; y < 6; y++ {
		for x := 0; x < 8; x++ {
			costs[y*8+x] = 10
		}
	}
	g.SetExtraCost(costs)

	start, goal := Point{1, 2}, Point{6, 3}
	heapPath := g.Search(start, goal)
	linearPath := searchLinear(g, start, goal)
	if len(heapPath) == 0 || len(linearPath) == 0 {
		t.Fatalf("heap %v linear %v", heapPath, linearPath)
	}
	heapCost := pathCost(g, start, heapPath)
	linearCost := pathCost(g, start, linearPath)
	// 绕上路是唯一最优（代价 8），绕下路被罚到 18+：两边都必须给出最优代价。
	// 平局允许走不同的等代价走廊，但代价必须精确等于最优值。
	if heapCost != 8 || linearCost != 8 {
		t.Fatalf("heap cost %v path %v, linear cost %v path %v, want 8",
			heapCost, heapPath, linearCost, linearPath)
	}
}

// pathCost 沿合并后的航点折线按 segmentCheckStep 采样，累计与搜索一致的 stepCost。
func pathCost(g *Grid, start Point, merged []Point) float64 {
	total := 0.0
	cur := start
	for _, wp := range merged {
		dx, dy := wp.X-cur.X, wp.Y-cur.Y
		steps := int(math.Ceil(math.Hypot(float64(dx), float64(dy)) / segmentCheckStep))
		prev := cur
		for i := 1; i <= steps; i++ {
			t := float64(i) / float64(steps)
			cell := Point{
				X: cur.X + int(math.Floor(t*float64(dx))),
				Y: cur.Y + int(math.Floor(t*float64(dy))),
			}
			if cell != prev {
				total += g.stepCost(prev, cell, 1)
				prev = cell
			}
		}
		cur = wp
	}
	return total
}

// searchLinear 用切片线性找最小开集，方便和堆实现对照。
func searchLinear(g *Grid, start, goal Point) []Point {
	if !g.validateEndpoints(start, goal) {
		return []Point{}
	}
	openSet := []Node{{start, 0, g.heuristic(start, goal)}}
	cameFrom := map[Point]Point{}
	gScore := map[Point]float64{start: 0}

	for len(openSet) > 0 {
		cur := openSet[0]
		for _, node := range openSet {
			if node.G+node.H < cur.G+cur.H {
				cur = node
			}
		}
		kept := make([]Node, 0, len(openSet))
		removed := false
		for _, node := range openSet {
			if !removed && node.Point == cur.Point && node.G == cur.G && node.H == cur.H {
				removed = true
				continue
			}
			kept = append(kept, node)
		}
		openSet = kept
		if known, ok := gScore[cur.Point]; ok && cur.G > known {
			continue
		}
		if cur.Point == goal {
			path := []Point{cur.Point}
			for cur.Point != start {
				cur.Point = cameFrom[cur.Point]
				path = append([]Point{cur.Point}, path...)
			}
			return g.mergePathWithCheckPoint(g.mergePathWithSameM(path))
		}

		var buf [8]Point
		nbCount := g.getNeighbors(cur.Point, &buf)
		for _, neighbor := range buf[:nbCount] {
			tentative := gScore[cur.Point] + g.stepCost(cur.Point, neighbor, 1)
			if _, ok := gScore[neighbor]; !ok || tentative < gScore[neighbor] {
				cameFrom[neighbor] = cur.Point
				gScore[neighbor] = tentative
				openSet = append(openSet, Node{neighbor, tentative, g.heuristic(neighbor, goal)})
			}
		}
	}
	return []Point{}
}

// touchedCells 沿合并后的路径段按 0.1 格步长采样，收集实际经过的格子。
func touchedCells(path []Point, start Point) map[Point]bool {
	touched := map[Point]bool{start: true}
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
			touched[Point{
				X: cur.X + int(math.Floor(t*float64(dx))),
				Y: cur.Y + int(math.Floor(t*float64(dy))),
			}] = true
		}
		cur = next
	}
	return touched
}

// TestSearchWithCostScaleSwitchesGap 验证额外代价缩放只改变航线选择，不会让可达目标不可达。
// 场景：x=3 的墙只有 y=2（高代价 20）与 y=7（免费）两个缺口；缩放系数应让寻路在直路/绕路之间切换。
// 预期：0.1 倍走 (3,2)；2 倍绕 (3,7)；100 倍仍能返回非空路径。
// 覆盖：构造 9x7 网格与代价表，调用 SearchWithCostScale 三次，用 touchedCells 判断经过的缺口。
func TestSearchWithCostScaleSwitchesGap(t *testing.T) {
	cells := make(Cells, 9)
	for y := range cells {
		cells[y] = make([]int, 7)
	}
	// x=3 是墙，只有 y=2 与 y=7 两个缺口
	for y := 0; y < 9; y++ {
		cells[y][3] = W
	}
	cells[2][3] = O
	cells[7][3] = O
	// 直路缺口 (3,2) 代价高，绕路缺口 (3,7) 免费
	costs := make([]float32, 7*9)
	costs[2*7+3] = 20
	g := NewGrid(cells)
	g.SetExtraCost(costs)

	start, goal := Point{X: 0, Y: 2}, Point{X: 6, Y: 2}
	direct := touchedCells(g.SearchWithCostScale(start, goal, 0.1), start)
	if !direct[Point{X: 3, Y: 2}] {
		t.Fatal("小代价缩放时应走直路缺口")
	}
	detour := touchedCells(g.SearchWithCostScale(start, goal, 2), start)
	if detour[Point{X: 3, Y: 2}] {
		t.Fatal("大代价缩放时应绕开高代价缺口")
	}
	if !detour[Point{X: 3, Y: 7}] {
		t.Fatal("大代价缩放时应走便宜的绕路缺口")
	}
	if len(g.SearchWithCostScale(start, goal, 100)) == 0 {
		t.Fatal("再大的代价缩放也不能让可达目标变成不可达")
	}
}

// TestSetExtraCostIgnoresMismatchedLength 验证代价表长度与网格不符时被忽略，寻路仍按距离工作。
// 场景：配置/调用方传入错误长度的 ExtraCost（例如只给 3 个格）；若直接索引会越界或污染代价。
// 预期：SetExtraCost 后 Search 仍返回非空路径。
// 覆盖：4x4 空网格设置长度 3 的代价表，调用 Search(0,0)->(3,3) 断言路径非空。
func TestSetExtraCostIgnoresMismatchedLength(t *testing.T) {
	cells := make(Cells, 4)
	for y := range cells {
		cells[y] = make([]int, 4)
	}
	g := NewGrid(cells)
	g.SetExtraCost([]float32{1, 2, 3})
	if path := g.Search(Point{X: 0, Y: 0}, Point{X: 3, Y: 3}); len(path) == 0 {
		t.Fatal("代价表长度不符时不应影响寻路")
	}
}
