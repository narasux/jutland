package mission

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// 探测游戏地图上的鼠标悬停动作
func detectCursorHoverOnGameMap(layout ScreenLayout) ActionType {
	x, y := ebiten.CursorPosition()
	w, h := layout.Camera.Width, layout.Camera.Height

	// Hover 上侧
	if 0 < y && y < h/15 {
		// Hover 左侧 3/10
		if 0 < x && x < w/10*3 {
			return HoverScreenTopLeft
		}
		// Hover 右侧 3/10
		if w/10*7 < x && x < w {
			return HoverScreenTopRight
		}
		// Hover 中间 4/10
		if w/10*3 < x && x < w/10*7 {
			return HoverScreenTop
		}
	}

	// Hover 下侧
	if h/15*14 < y && y < h {
		// Hover 左侧 3/10
		if 0 < x && x < w/10*3 {
			return HoverScreenBottomLeft
		}
		// Hover 右侧 3/10
		if w/10*7 < x && x < w {
			return HoverScreenBottomRight
		}
		// Hover 中间 4/10
		if w/10*4 < x && x < w/10*6 {
			return HoverScreenBottom
		}
	}

	// Hover 左侧
	if 0 < x && x < w/20 {
		return HoverScreenLeft
	}
	// Hover 右侧
	if w/20*19 < x && x < w {
		return HoverScreenRight
	}

	// Hover 在中间
	if 0 < x && x < w && 0 < y && y < h {
		return HoverScreenMiddle
	}
	return DoNothing
}
