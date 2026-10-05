package grid

import (
	"container/heap"
	"math"

	"github.com/samber/lo"
)

// Grid 地图网格
type Grid struct {
	cells      Cells
	jumpPoints [][]Point
	// extraCost 每格额外通行代价（行优先展平），nil 表示所有格代价相同
	extraCost []float32
}

// NewGrid 创建网格
func NewGrid(cells Cells) *Grid {
	return &Grid{cells: cells}
}

// SetExtraCost 设置每格额外通行代价（行优先展平）。
// 长度与网格不符时忽略，寻路退回只按距离计算代价。
func (g *Grid) SetExtraCost(costs []float32) {
	if len(g.cells) == 0 || len(g.cells[0]) == 0 || len(costs) != len(g.cells)*len(g.cells[0]) {
		return
	}
	g.extraCost = costs
}

// stepCost 从 from 走到 to 的代价：距离 + 目标格的额外代价（按 scale 缩放）。
// 额外代价是软约束（有限值），不会让任何原本可达的格变得不可达。
func (g *Grid) stepCost(from, to Point, scale float64) float64 {
	cost := g.heuristic(from, to)
	if g.extraCost != nil {
		cost += float64(g.extraCost[to.Y*len(g.cells[0])+to.X]) * scale
	}
	return cost
}

// Prepare 按地形预计算跳点。起点和终点不参与，海图保持只读。
func (g *Grid) Prepare() {
	if g.jumpPoints != nil {
		return
	}
	g.preProcess()
}

// Search 搜索可行路径（按基准代价）。
func (g *Grid) Search(start, goal Point) []Point {
	return g.SearchWithCostScale(start, goal, 1)
}

// SearchWithCostScale 搜索可行路径，并按 scale 缩放每格额外代价。
// 大舰需要更多离岸余量（scale > 1），小艇可以贴着岸走（scale < 1）；
// 代价始终是有限值，缩放不会改变可达性。
func (g *Grid) SearchWithCostScale(start, goal Point, scale float64) []Point {
	if !g.validateEndpoints(start, goal) {
		return []Point{}
	}
	g.Prepare()
	if scale <= 0 {
		scale = 1
	}

	openSet := &openHeap{}
	heap.Init(openSet)
	heap.Push(openSet, Node{start, 0, g.heuristic(start, goal)})
	cameFrom := map[Point]Point{}
	gScore := map[Point]float64{}
	gScore[start] = 0

	for openSet.Len() > 0 {
		cur := heap.Pop(openSet).(Node)
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

		neighbors := g.getNeighbors(cur.Point)
		for _, neighbor := range neighbors {
			tentativeGScore := gScore[cur.Point] + g.stepCost(cur.Point, neighbor, scale)
			if _, ok := gScore[neighbor]; !ok || tentativeGScore < gScore[neighbor] {
				cameFrom[neighbor] = cur.Point
				gScore[neighbor] = tentativeGScore
				heap.Push(openSet, Node{neighbor, tentativeGScore, tentativeGScore + g.heuristic(neighbor, goal)})
			}
		}

		if jp := g.jumpPoints[cur.Point.Y][cur.Point.X]; jp.IsValid() {
			tentativeGScore := gScore[cur.Point] + g.stepCost(cur.Point, jp, scale)
			if _, ok := gScore[jp]; !ok || tentativeGScore < gScore[jp] {
				cameFrom[jp] = cur.Point
				gScore[jp] = tentativeGScore
				heap.Push(openSet, Node{jp, tentativeGScore, tentativeGScore + g.heuristic(jp, goal)})
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

// 计算启发式函数值
func (g *Grid) heuristic(a, b Point) float64 {
	h := math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y))
	return lo.Ternary(g.cells[a.Y][a.X] == SD, h+5, h)
}

// 获取邻居。对角移动要求两个正交邻格均可通行，避免从两块陆地的夹角处斜穿，
// 否则合并后的直线路径段会切过陆地格，把舰体中心带进墙里。
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
			if dir.X != 0 && dir.Y != 0 &&
				(g.cells[p.Y][x] == W || g.cells[y][p.X] == W) {
				continue
			}
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

		if g.segmentBlocked(cur, may) {
			mergedPath = append(mergedPath, path[nextIdx])
			curIdx = nextIdx
		}
		nextIdx = idx
	}

	return append(mergedPath, path[pathLen-1])
}

// segmentBlocked 沿 a→b 线段以 0.1 格步长逐点检查扫过的格子是否撞墙。
// 采样点连同其 8 邻格一起检查：合并后的直线段必须留出一格余量。
// 舰体并不总是压在航线格心上（转向弧线、被拦停后重新起步都会偏出半格），
// 只校验中心线会让合并出的直线段贴着岸线，舰体一偏就切进陆格、顶着岸边磨。
func (g *Grid) segmentBlocked(a, b Point) bool {
	distance := math.Hypot(float64(b.X-a.X), float64(b.Y-a.Y))
	if distance == 0 {
		return false
	}
	steps := int(math.Ceil(distance / segmentCheckStep))
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := a.X + int(math.Floor(t*float64(b.X-a.X)))
		y := a.Y + int(math.Floor(t*float64(b.Y-a.Y)))
		if g.cellNearWall(x, y) {
			return true
		}
	}
	return false
}

// cellNearWall 判断格 (x, y) 及其 8 邻格中是否有墙；地图外不算墙
// （舰船位置本身会被 EnsureBorder 夹在地图内，边缘一格仍可正常通行）。
func (g *Grid) cellNearWall(x, y int) bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nx, ny := x+dx, y+dy
			if ny < 0 || ny >= len(g.cells) || nx < 0 || nx >= len(g.cells[ny]) {
				continue
			}
			if g.cells[ny][nx] == W {
				return true
			}
		}
	}
	return false
}

// segmentCheckStep 线段校验的采样步长（格）。足够细以捕捉贴角掠过的路径段。
const segmentCheckStep = 0.1

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
	if direction.X != 0 && direction.Y != 0 {
		if g.cells[p.Y][x] == O {
			if jp := g.getJumpPoint(Point{x, y}, Point{direction.X, 0}); jp.IsValid() {
				return Point{x, y}
			}
		}
		if g.cells[y][p.X] == O {
			if jp := g.getJumpPoint(Point{x, y}, Point{0, direction.Y}); jp.IsValid() {
				return Point{x, y}
			}
		}
	}
	return g.getJumpPoint(Point{x, y}, direction)
}

type openHeap []Node

func (h openHeap) Len() int { return len(h) }

func (h openHeap) Less(i, j int) bool {
	return h[i].G+h[i].H < h[j].G+h[j].H
}

func (h openHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *openHeap) Push(x any) { *h = append(*h, x.(Node)) }

func (h *openHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}
