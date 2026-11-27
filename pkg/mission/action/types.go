package action

import objPos "github.com/narasux/jutland/pkg/mission/object/position"

// CursorHoverType 鼠标悬停动作
type CursorHoverType string

const (
	// 无动作
	DoNothing CursorHoverType = "doNothing"

	// HoverScreenMiddle 悬停在屏幕中间
	HoverScreenMiddle CursorHoverType = "hoverScreenMiddle"

	// HoverScreenTop 悬停在屏幕顶部
	HoverScreenTop CursorHoverType = "hoverScreenTop"

	// HoverScreenBottom 悬停在屏幕底部
	HoverScreenBottom CursorHoverType = "hoverScreenBottom"

	// HoverScreenLeft 悬停在屏幕左侧
	HoverScreenLeft CursorHoverType = "hoverScreenLeft"

	// HoverScreenRight 悬停在屏幕右侧
	HoverScreenRight CursorHoverType = "hoverScreenRight"

	// HoverScreenTopLeft 悬停在屏幕左上角
	HoverScreenTopLeft CursorHoverType = "hoverScreenTopLeft"

	// HoverScreenTopRight 悬停在屏幕右上角
	HoverScreenTopRight CursorHoverType = "hoverScreenTopRight"

	// HoverScreenBottomLeft 悬停在屏幕左下角
	HoverScreenBottomLeft CursorHoverType = "hoverScreenBottomLeft"

	// HoverScreenBottomRight 悬停在屏幕右下角
	HoverScreenBottomRight CursorHoverType = "hoverScreenBottomRight"
)

// SelectedArea 选中的区域
type SelectedArea struct {
	// 地图位置
	StartAt, CurAt objPos.MapPos
	// 屏幕位置
	StartX, StartY, CurX, CurY int
}

// Bounds 返回选区在地图坐标下的轴对齐包围盒
func (a *SelectedArea) Bounds() (left, top, right, bottom float64) {
	left, right = min(a.StartAt.RX, a.CurAt.RX), max(a.StartAt.RX, a.CurAt.RX)
	top, bottom = min(a.StartAt.RY, a.CurAt.RY), max(a.StartAt.RY, a.CurAt.RY)
	return
}

// Contain 判断某个位置是否在选中区域内
func (a *SelectedArea) Contain(pos objPos.MapPos) bool {
	left, top, right, bottom := a.Bounds()
	return pos.RX >= left && pos.RX <= right && pos.RY >= top && pos.RY <= bottom
}

// Overlaps 判断选区与给定的轴对齐矩形是否重叠（含边界）
func (a *SelectedArea) Overlaps(left, top, right, bottom float64) bool {
	areaLeft, areaTop, areaRight, areaBottom := a.Bounds()
	return areaRight >= left && areaLeft <= right && areaBottom >= top && areaTop <= bottom
}
