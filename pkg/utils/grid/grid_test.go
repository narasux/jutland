package grid

import "testing"

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
			tentative := gScore[cur.Point] + g.heuristic(cur.Point, neighbor)
			if _, ok := gScore[neighbor]; !ok || tentative < gScore[neighbor] {
				cameFrom[neighbor] = cur.Point
				gScore[neighbor] = tentative
				openSet = append(openSet, Node{neighbor, tentative, tentative + g.heuristic(neighbor, goal)})
			}
		}
		if jp := g.jumpPoints[cur.Point.Y][cur.Point.X]; jp.IsValid() {
			tentative := gScore[cur.Point] + g.heuristic(cur.Point, jp)
			if _, ok := gScore[jp]; !ok || tentative+g.heuristic(cur.Point, jp) < gScore[jp] {
				cameFrom[jp] = cur.Point
				gScore[jp] = tentative
				openSet = append(openSet, Node{jp, tentative, tentative + g.heuristic(jp, goal)})
			}
		}
	}
	return []Point{}
}
