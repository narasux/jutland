package state

import obj "github.com/narasux/jutland/pkg/mission/object"

// Camera 相机（当前视野）
type Camera struct {
	// 相机左上角位置
	Pos           obj.MapPos
	Width         int
	Height        int
	BaseMoveSpeed float64
}

// Contains 判断坐标是否在视野内
func (c *Camera) Contains(pos obj.MapPos) bool {
	return !(pos.MX < c.Pos.MX ||
		pos.MX > c.Pos.MX+c.Width ||
		pos.MY < c.Pos.MY ||
		pos.MY > c.Pos.MY+c.Height)
}
