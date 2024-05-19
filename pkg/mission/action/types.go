package action

import obj "github.com/narasux/jutland/pkg/mission/object"

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
	StartAt, CurAt obj.MapPos
	// 屏幕位置
	StartX, StartY, CurX, CurY int
}

// Contain 判断某个位置是否在选中区域内
func (a *SelectedArea) Contain(pos obj.MapPos) bool {
	topLeftX, topLeftY := min(a.StartAt.RX, a.CurAt.RX), min(a.StartAt.RY, a.CurAt.RY)
	bottomRightX, bottomRightY := max(a.StartAt.RX, a.CurAt.RX), max(a.StartAt.RY, a.CurAt.RY)

	return pos.RX >= topLeftX && pos.RX <= bottomRightX && pos.RY >= topLeftY && pos.RY <= bottomRightY
}
