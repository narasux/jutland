package grid

import (
	"math"
	"sync"

	"github.com/samber/lo"
)

// Grid 地图网格
type Grid struct {
	cells Cells
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
	if scale <= 0 {
		scale = 1
	}
	width := len(g.cells[0])

	s := searchScratches.Get().(*searchScratch)
	defer searchScratches.Put(s)
	s.begin(width * len(g.cells))

	startIdx := int32(start.Y*width + start.X)
	goalIdx := int32(goal.Y*width + goal.X)
	s.gGen[startIdx] = s.gen
	s.gScore[startIdx] = 0
	s.push(heapItem{idx: startIdx, g: 0, f: g.heuristic(start, goal)})

	var buf [8]Point
	for len(s.heap) > 0 {
		cur := s.pop()
		// 陈旧条目：同格已有更小的 g（gGen 对上才说明是本次写的）
		if s.gGen[cur.idx] == s.gen && cur.g > s.gScore[cur.idx] {
			continue
		}
		if cur.idx == goalIdx {
			path := reconstructPath(s.parent, width, startIdx, goalIdx)
			return g.mergePathWithCheckPoint(g.mergePathWithSameM(path))
		}

		curPt := Point{X: int(cur.idx) % width, Y: int(cur.idx) / width}
		n := g.getNeighbors(curPt, &buf)
		for _, neighbor := range buf[:n] {
			nbIdx := int32(neighbor.Y*width + neighbor.X)
			tentativeGScore := s.gScore[cur.idx] + g.stepCost(curPt, neighbor, scale)
			if s.gGen[nbIdx] != s.gen || tentativeGScore < s.gScore[nbIdx] {
				s.gGen[nbIdx] = s.gen
				s.gScore[nbIdx] = tentativeGScore
				s.parent[nbIdx] = cur.idx
				// f 按旧 openHeap 的 G+H 原样计算：当年 H 字段存的是 G+h，
				// 即 G 被计入两次（实际优先值 2g+h）。这是既有寻路行为，
				// 改它会改变航线，属于行为修正，留待单独处理。
				h := g.heuristic(neighbor, goal)
				s.push(heapItem{
					idx: nbIdx, g: tentativeGScore,
					f: tentativeGScore + (tentativeGScore + h),
				})
			}
		}
	}
	return []Point{}
}

// reconstructPath 沿 parent 回溯生成从起点到终点的路径。
// 先数长度再一次填好，O(L)（旧实现每步前插是 O(L²)）；
// 返回值是新切片，不与 sync.Pool 复用的工作区共享底层数组。
func reconstructPath(parent []int32, width int, startIdx, goalIdx int32) []Point {
	n := 1
	for idx := goalIdx; idx != startIdx; idx = parent[idx] {
		n++
	}
	path := make([]Point, n)
	idx := goalIdx
	for i := n - 1; i >= 0; i-- {
		path[i] = Point{X: int(idx) % width, Y: int(idx) / width}
		if i > 0 {
			idx = parent[idx]
		}
	}
	return path
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

// 计算启发式函数值（热路径：每次入堆都调用，直接 if，不走 lo.Ternary）
func (g *Grid) heuristic(a, b Point) float64 {
	h := math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y))
	if g.cells[a.Y][a.X] == SD {
		return h + 5
	}
	return h
}

// neighborDirs 邻居方向与扩展顺序固定，保证堆的插入顺序与历史实现一致。
var neighborDirs = [8]Point{
	{-1, 0},
	{1, 0},
	{0, -1},
	{0, 1},
	{-1, -1},
	{-1, 1},
	{1, -1},
	{1, 1},
}

// getNeighbors 把 p 的可通行邻居写入 buf 并返回数量（至多 8 个）。
// 由调用方提供 buf，避免每次节点扩展都分配切片。
// 对角移动要求两个正交邻格均可通行，避免从两块陆地的夹角处斜穿，
// 否则合并后的直线路径段会切过陆地格，把舰体中心带进墙里。
func (g *Grid) getNeighbors(p Point, buf *[8]Point) int {
	n := 0
	for _, dir := range neighborDirs {
		x, y := p.X+dir.X, p.Y+dir.Y
		if y >= 0 && y < len(g.cells) && x >= 0 && x < len(g.cells[0]) && g.cells[y][x] != W {
			if dir.X != 0 && dir.Y != 0 &&
				(g.cells[p.Y][x] == W || g.cells[y][p.X] == W) {
				continue
			}
			buf[n] = Point{x, y}
			n++
		}
	}
	return n
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

// heapItem 开集条目：idx 是行优先展平后的格子下标（地图规模远小于 int32 上限）。
type heapItem struct {
	idx int32
	g   float64
	f   float64
}

// searchScratch 单次搜索的可复用工作区。
// Grid 会被后台寻路线程并发搜索，工作区不能挂在 Grid 上，
// 统一放 sync.Pool：搜索前 Get、结束后归还，摊销后每次搜索接近零分配。
type searchScratch struct {
	// gen 是本次搜索的代号；gGen 等于它才表示这一格是本次写的。
	// 用代号代替“每次搜索把整张表清零”：大图一次清零就要几百微秒。
	gen    uint32
	gGen   []uint32
	gScore []float64
	parent []int32 // 回溯链：每格来路的下标
	heap   []heapItem
}

var searchScratches = sync.Pool{New: func() any { return new(searchScratch) }}

// begin 准备 n 格的工作区并切到新一代，O(1) 复位。
func (s *searchScratch) begin(n int) {
	if cap(s.gGen) < n {
		s.gGen = make([]uint32, n)
		s.gScore = make([]float64, n)
		s.parent = make([]int32, n)
	} else {
		s.gGen = s.gGen[:n]
		s.gScore = s.gScore[:n]
		s.parent = s.parent[:n]
	}
	s.heap = s.heap[:0]
	s.gen++
	if s.gen == 0 { // 代号回绕：清零，避免把旧数据误判成本次写过
		clear(s.gGen)
		s.gen = 1
	}
}

// push 入堆（上浮）。与 container/heap 的 up 同构，
// 保持与旧实现完全相同的弹出序列，路径逐点不变。
func (s *searchScratch) push(it heapItem) {
	s.heap = append(s.heap, it)
	i := len(s.heap) - 1
	for i > 0 {
		p := (i - 1) / 2
		if s.heap[i].f >= s.heap[p].f {
			break
		}
		s.heap[i], s.heap[p] = s.heap[p], s.heap[i]
		i = p
	}
}

// pop 弹出堆顶（与末尾交换后下沉）。与 container/heap 的 Pop 同构。
func (s *searchScratch) pop() heapItem {
	n := len(s.heap) - 1
	s.heap[0], s.heap[n] = s.heap[n], s.heap[0]
	i := 0
	for {
		l := 2*i + 1
		if l >= n {
			break
		}
		j := l
		if r := l + 1; r < n && s.heap[r].f < s.heap[l].f {
			j = r
		}
		if s.heap[j].f >= s.heap[i].f {
			break
		}
		s.heap[i], s.heap[j] = s.heap[j], s.heap[i]
		i = j
	}
	it := s.heap[n]
	s.heap = s.heap[:n]
	return it
}
