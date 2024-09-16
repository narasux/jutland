package grid

const (
	// W 墙壁
	W = -1
	// O 空地
	O = 0
	// S 起点
	S = 1
	// E 终点
	E = 2
	// SD 浅海
	SD = 3
)

// Point 坐标
type Point struct {
	X, Y int
}

// IsValid 是否无效
func (p *Point) IsValid() bool {
	return p.X >= 0 || p.Y >= 0
}

// Node 节点
// Point 坐标 X, Y
// G 值（从起点到该点的代价）
// H 值（从该点到终点的代价）
type Node struct {
	Point Point
	G     float64
	H     float64
}

// Cells 网格
type Cells [][]int
