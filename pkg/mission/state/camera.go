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

// ContainsMargin 判断坐标是否在视野内，四周各外扩 margin 个地图格。
func (c *Camera) ContainsMargin(pos objPos.MapPos, margin int) bool {
	return !(pos.MX < c.Pos.MX-margin ||
		pos.MX > c.Pos.MX+c.Width+margin ||
		pos.MY < c.Pos.MY-margin ||
		pos.MY > c.Pos.MY+c.Height+margin)
}
