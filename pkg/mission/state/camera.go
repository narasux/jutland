package state

import objCommon "github.com/narasux/jutland/pkg/mission/object/common"

// Camera 相机（当前视野）
type Camera struct {
	// 相机左上角位置
	Pos           objCommon.MapPos
	Width         int
	Height        int
	BaseMoveSpeed float64
}

// Contains 判断坐标是否在视野内
func (c *Camera) Contains(pos objCommon.MapPos) bool {
	return !(pos.MX < c.Pos.MX ||
		pos.MX > c.Pos.MX+c.Width ||
		pos.MY < c.Pos.MY ||
		pos.MY > c.Pos.MY+c.Height)
}
