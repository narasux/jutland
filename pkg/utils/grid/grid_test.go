package grid

import (
	"math"
	"testing"
)

// 对角移动要求两个正交邻格均可通行：贴着墙角不允许斜穿。
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

	nbrs := g.getNeighbors(Point{X: 1, Y: 1})
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

// 线段校验：擦过墙格的线段算受阻；合并出的直线段还必须留出一格余量，
// 否则舰体偏离航线半个格（转向弧线、被拦停后重新起步）就会切进陆格。
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

func TestSearchLeavesShallowCellsUntouched(t *testing.T) {
	cells := make(Cells, 5)
	for y := range cells {
		cells[y] = make([]int, 8)
	}
	cells[2][3] = SD
	g := NewGrid(cells)
	g.Prepare()

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

func TestHeapMatchesLinearOpenSet(t *testing.T) {
	cells := make(Cells, 6)
	for y := range cells {
		cells[y] = make([]int, 8)
	}
	cells[2][3] = W
	cells[3][3] = W
	g := NewGrid(cells)
	g.Prepare()

	start, goal := Point{1, 2}, Point{6, 3}
	heapPath := g.Search(start, goal)
	linearPath := searchLinear(g, start, goal)
	if len(heapPath) == 0 || len(heapPath) != len(linearPath) {
		t.Fatalf("heap %v linear %v", heapPath, linearPath)
	}
	for i := range heapPath {
		if heapPath[i] != linearPath[i] {
			t.Fatalf("heap %v linear %v", heapPath, linearPath)
		}
	}
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

		for _, neighbor := range g.getNeighbors(cur.Point) {
			tentative := gScore[cur.Point] + g.stepCost(cur.Point, neighbor, 1)
			if _, ok := gScore[neighbor]; !ok || tentative < gScore[neighbor] {
				cameFrom[neighbor] = cur.Point
				gScore[neighbor] = tentative
				openSet = append(openSet, Node{neighbor, tentative, tentative + g.heuristic(neighbor, goal)})
			}
		}
		if jp := g.jumpPoints[cur.Point.Y][cur.Point.X]; jp.IsValid() {
			tentative := gScore[cur.Point] + g.stepCost(cur.Point, jp, 1)
			if _, ok := gScore[jp]; !ok || tentative < gScore[jp] {
				cameFrom[jp] = cur.Point
				gScore[jp] = tentative
				openSet = append(openSet, Node{jp, tentative, tentative + g.heuristic(jp, goal)})
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

// 额外代价是软约束：缩放它只改变航线选择，不会让原本可达的目标变得不可达。
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

// 额外代价长度与网格不符时忽略，寻路退回只按距离计算。
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
