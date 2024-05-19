package object

import (
	"fmt"
	"math"

	"github.com/narasux/jutland/pkg/utils/geometry"
)

// MapPos 位置
type MapPos struct {
	// 地图位置（用于通用计算，如小地图等）
	MX, MY int
	// 真实位置（用于计算屏幕位置，如不需要可不初始化）
	RX, RY float64
}

// NewMapPos ...
func NewMapPos(mx, my int) MapPos {
	return MapPos{MX: mx, MY: my, RX: float64(mx), RY: float64(my)}
}

// MEqual 判断位置是否相等（用地图位置判断，RX，RY 太准确一直没法到）
func (p *MapPos) MEqual(other MapPos) bool {
	return p.MX == other.MX && p.MY == other.MY
}

// Near 判断位置是否在指定范围内
func (p *MapPos) Near(other MapPos, distance int) bool {
	return geometry.CalcDistance(p.RX, p.RY, other.RX, other.RY) <= float64(distance)
}

// String ...
func (p *MapPos) String() string {
	return fmt.Sprintf("(%f/%d, %f/%d)", p.RX, p.MX, p.RY, p.MY)
}

// AssignRxy 重新赋值 RX，RY，并计算 MX，MY
func (p *MapPos) AssignRxy(rx, ry float64) {
	p.RX, p.RY = rx, ry
	p.MX, p.MY = int(math.Floor(rx)), int(math.Floor(ry))
}

// AssignMxy 重新赋值 MX，MY，同时计算 RX，RY
func (p *MapPos) AssignMxy(mx, my int) {
	p.MX, p.MY = mx, my
	p.RX, p.RY = float64(mx), float64(my)
}

// AddRx 修改 Rx，同时计算 Mx
func (p *MapPos) AddRx(rx float64) {
	p.RX += rx
	p.MX = int(math.Floor(p.RX))
}

// SubRx 修改 Rx，同时计算 Mx
func (p *MapPos) SubRx(rx float64) {
	p.RX -= rx
	p.MX = int(math.Floor(p.RX))
}

// AddRy 修改 Ry，同时计算 My
func (p *MapPos) AddRy(ry float64) {
	p.RY += ry
	p.MY = int(math.Floor(p.RY))
}

// SubRy 修改 Ry，同时计算 My
func (p *MapPos) SubRy(ry float64) {
	p.RY -= ry
	p.MY = int(math.Floor(p.RY))
}

// EnsureBorder 边界检查
func (p *MapPos) EnsureBorder(borderX, borderY float64) {
	p.RX = max(min(p.RX, borderX), 0)
	p.RY = max(min(p.RY, borderY), 0)
	p.MX = int(math.Floor(p.RX))
	p.MY = int(math.Floor(p.RY))
}

// Copy 复制 MapPos 对象
func (p *MapPos) Copy() MapPos {
	return MapPos{MX: p.MX, MY: p.MY, RX: p.RX, RY: p.RY}
}
