package grid

import (
	"math"

	"github.com/samber/lo"
)

// Grid 地图网格
type Grid struct {
	cells      Cells
	jumpPoints [][]Point
}

// NewGrid 创建网格
func NewGrid(cells Cells) *Grid {
	return &Grid{cells: cells}
}

// Search 搜索可行路径
func (g *Grid) Search(start, goal Point) []Point {
	if !g.validateEndpoints(start, goal) {
		return []Point{}
	}
	// 预处理
	g.setEndpoints(start, goal)
	g.preProcess()

	openSet := []Node{{start, 0, g.heuristic(start, goal)}}
	cameFrom := map[Point]Point{}
	gScore, fScore := map[Point]float64{}, map[Point]float64{}
	gScore[start], fScore[start] = 0, g.heuristic(start, goal)

	for len(openSet) > 0 {
		cur := openSet[0]
		for _, node := range openSet {
			if node.G+node.H < cur.G+cur.H {
				cur = node
			}
		}

		if cur.Point == goal {
			path := []Point{cur.Point}
			for cur.Point != start {
				cur.Point = cameFrom[cur.Point]
				path = append([]Point{cur.Point}, path...)
			}
			return g.mergePathWithCheckPoint(g.mergePathWithSameM(path))
		}

		// 从列表中移除 Cur
		for i, n := range openSet {
			if n.Point == cur.Point {
				openSet = append(openSet[:i], openSet[i+1:]...)
			}
		}

		neighbors := g.getNeighbors(cur.Point)
		for _, neighbor := range neighbors {
			tentativeGScore := gScore[cur.Point] + g.heuristic(cur.Point, neighbor)
			if _, ok := gScore[neighbor]; !ok || tentativeGScore < gScore[neighbor] {
				cameFrom[neighbor] = cur.Point
				gScore[neighbor] = tentativeGScore
				fScore[neighbor] = tentativeGScore + g.heuristic(neighbor, goal)
				openSet = append(openSet, Node{neighbor, tentativeGScore, fScore[neighbor]})
			}
		}

		if jp := g.jumpPoints[cur.Point.Y][cur.Point.X]; jp.IsValid() {
			tentativeGScore := gScore[cur.Point] + g.heuristic(cur.Point, jp)
			if _, ok := gScore[jp]; !ok || tentativeGScore+g.heuristic(cur.Point, jp) < gScore[jp] {
				cameFrom[jp] = cur.Point
				gScore[jp] = tentativeGScore
				fScore[jp] = tentativeGScore + g.heuristic(jp, goal)
				openSet = append(openSet, Node{jp, tentativeGScore, fScore[jp]})
			}
		}
	}
	return []Point{}
}

// 检查起点和终点是否有效
func (g *Grid) validateEndpoints(start, goal Point) bool {
	if start.Y < 0 || start.Y >= len(g.cells) ||
		start.X < 0 || start.X >= len(g.cells[0]) ||
		g.cells[start.Y][start.X] == W {
		return false
	}
	if goal.Y < 0 || goal.Y >= len(g.cells) ||
		goal.X < 0 || goal.X >= len(g.cells[0]) ||
		g.cells[goal.Y][goal.X] == W {
		return false
	}
	return true
}

func (g *Grid) setEndpoints(start, goal Point) {
	g.cells[start.Y][start.X] = S
	g.cells[goal.Y][goal.X] = E
}

// 计算启发式函数值
func (g *Grid) heuristic(a, b Point) float64 {
	h := math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y))
	return lo.Ternary(g.cells[a.Y][a.X] == SD, h+5, h)
}

// 获取邻居
func (g *Grid) getNeighbors(p Point) []Point {
	directions := []Point{
		{-1, 0},
		{1, 0},
		{0, -1},
		{0, 1},
		{-1, -1},
		{-1, 1},
		{1, -1},
		{1, 1},
	}
	var neighbors []Point
	for _, dir := range directions {
		x, y := p.X+dir.X, p.Y+dir.Y
		if y >= 0 && y < len(g.cells) && x >= 0 && x < len(g.cells[0]) && g.cells[y][x] != W {
			neighbors = append(neighbors, Point{x, y})
		}
	}
	return neighbors
}

// 合并路径（根据相同的斜率，即方向）
func (g *Grid) mergePathWithSameM(path []Point) []Point {
	pathLen := len(path)
	if pathLen < 3 {
		return path
	}
	mergedPath := []Point{path[0]}

	// 同一方向的点，只保留起点和终点
	dy := float64(path[1].Y - path[0].Y)
	dx := float64(path[1].X - path[0].X)
	lastM := lo.Ternary(dx == 0, math.Inf(1), dy/dx)

	for idx := 2; idx < pathLen; idx++ {
		dy = float64(path[idx].Y - path[idx-1].Y)
		dx = float64(path[idx].X - path[idx-1].X)
		m := lo.Ternary(dx == 0, math.Inf(1), dy/dx)
		// 斜率发生改变说明存在转向
		if m != lastM {
			mergedPath = append(mergedPath, path[idx-1])
			lastM = m
		}
	}
	return append(mergedPath, path[pathLen-1])
}

// 对于相邻的三个点，如果中间没有障碍物（检查点法），应该跳过中间点
// FIXME 这个函数感觉还是有问题
func (g *Grid) mergePathWithCheckPoint(path []Point) []Point {
	pathLen := len(path)
	if pathLen < 3 {
		return path
	}
	mergedPath := []Point{path[0]}

	curIdx, nextIdx := 0, 1
	for idx := 2; idx < pathLen; idx++ {
		cur := path[curIdx]
		may := path[idx]

		distance := math.Sqrt(math.Pow(float64(may.X-cur.X), 2) + math.Pow(float64(may.Y-cur.Y), 2))
		for i := 1; i < int(distance); i++ {
			x := int(float64(i)/distance*float64(may.X-cur.X) + float64(cur.X))
			y := int(float64(i)/distance*float64(may.Y-cur.Y) + float64(cur.Y))
			if g.cells[y][x] == W {
				mergedPath = append(mergedPath, path[nextIdx])
				curIdx = nextIdx
				break
			}
		}
		nextIdx = idx
	}

	return append(mergedPath, path[pathLen-1])
}

// 预处理
func (g *Grid) preProcess() {
	g.jumpPoints = make([][]Point, len(g.cells))
	for idx := range g.jumpPoints {
		g.jumpPoints[idx] = make([]Point, len(g.cells[0]))
	}
	// 垂直 / 水平四个方向
	hvDirections := []Point{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	// 计算每个点的跳点
	for y := range g.cells {
		for x := range g.cells[y] {
			if g.cells[y][x] != W {
				for _, dir := range hvDirections {
					jumpPoint := g.getJumpPoint(Point{x, y}, dir)
					if jumpPoint.IsValid() {
						g.jumpPoints[y][x] = jumpPoint
					}
				}
			}
		}
	}
}

// 获取跳点
func (g *Grid) getJumpPoint(p Point, direction Point) Point {
	x, y := p.X+direction.X, p.Y+direction.Y

	if y < 0 || y >= len(g.cells) || x < 0 || x >= len(g.cells[0]) || g.cells[y][x] == W {
		return Point{-1, -1}
	}
	if g.cells[y][x] == E {
		return Point{x, y}
	}
	if direction.X != 0 && direction.Y != 0 {
		if g.cells[p.Y][x] == O || g.cells[p.Y][x] == E {
			if jp := g.getJumpPoint(Point{x, y}, Point{direction.X, 0}); jp.IsValid() {
				return Point{x, y}
			}
		}
		if g.cells[y][p.X] == O || g.cells[y][p.X] == E {
			if jp := g.getJumpPoint(Point{x, y}, Point{0, direction.Y}); jp.IsValid() {
				return Point{x, y}
			}
		}
	}
	return g.getJumpPoint(Point{x, y}, direction)
}
