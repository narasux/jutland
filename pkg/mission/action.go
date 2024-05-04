package mission

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// ActionDetector 动作检测（用户在页面上的行为）
type ActionDetector struct {
}

// NewActionDetector ...
func NewActionDetector() *ActionDetector {
	return &ActionDetector{}
}

// DetectCursorHover ...
func (d *ActionDetector) DetectCursorHover() ActionType {
	posX, posY := ebiten.CursorPosition()
	width, height := ebiten.Monitor().Size()

	// Hover 上侧
	if posY < height/15 {
		// Hover 左侧 3/10
		if posX < width/10*3 {
			return HoverScreenTopLeft
		}
		// Hover 右侧 3/10
		if posX > width/10*7 {
			return HoverScreenTopRight
		}
		// Hover 中间 4/10
		return HoverScreenTop
	}

	// Hover 下侧
	if posY > height/15*14 {
		// Hover 左侧 3/10
		if posX < width/10*3 {
			return HoverScreenBottomLeft
		}
		// Hover 右侧 3/10
		if posX > width/10*7 {
			return HoverScreenBottomRight
		}
		// Hover 中间 4/10
		return HoverScreenBottom
	}

	// Hover 左侧
	if posX < width/20 {
		return HoverScreenLeft
	}
	// Hover 右侧
	if posX > width/20*19 {
		return HoverScreenRight
	}

	// Hover 在中间
	return HoverScreenMiddle
}
