package action

type ActionType string

const (
	// 无动作
	DoNothing ActionType = "doNothing"

	// HoverScreenMiddle 悬停在屏幕中间
	HoverScreenMiddle ActionType = "hoverScreenMiddle"

	// HoverScreenTop 悬停在屏幕顶部
	HoverScreenTop ActionType = "hoverScreenTop"

	// HoverScreenBottom 悬停在屏幕底部
	HoverScreenBottom ActionType = "hoverScreenBottom"

	// HoverScreenLeft 悬停在屏幕左侧
	HoverScreenLeft ActionType = "hoverScreenLeft"

	// HoverScreenRight 悬停在屏幕右侧
	HoverScreenRight ActionType = "hoverScreenRight"

	// HoverScreenTopLeft 悬停在屏幕左上角
	HoverScreenTopLeft ActionType = "hoverScreenTopLeft"

	// HoverScreenTopRight 悬停在屏幕右上角
	HoverScreenTopRight ActionType = "hoverScreenTopRight"

	// HoverScreenBottomLeft 悬停在屏幕左下角
	HoverScreenBottomLeft ActionType = "hoverScreenBottomLeft"

	// HoverScreenBottomRight 悬停在屏幕右下角
	HoverScreenBottomRight ActionType = "hoverScreenBottomRight"
)
