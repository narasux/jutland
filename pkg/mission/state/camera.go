package state

import objPos "github.com/narasux/jutland/pkg/mission/object/position"

// Camera 相机（当前视野）
type Camera struct {
	// 相机左上角位置
	Pos           objPos.MapPos
	Width         int
	Height        int
	BaseMoveSpeed float64
}

// Contains 判断坐标是否在视野内
func (c *Camera) Contains(pos objPos.MapPos) bool {
	return !(pos.MX < c.Pos.MX ||
		pos.MX > c.Pos.MX+c.Width ||
		pos.MY < c.Pos.MY ||
		pos.MY > c.Pos.MY+c.Height)
}

// GetVisibleRange 获取相机视野范围（用于迷雾渲染优化）
// 返回相机视野的左上角和右下角坐标
func (c *Camera) GetVisibleRange() (x1, y1, x2, y2 int) {
	x1 = c.Pos.MX
	y1 = c.Pos.MY
	x2 = c.Pos.MX + c.Width
	y2 = c.Pos.MY + c.Height
	return x1, y1, x2, y2
}
