package action

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/layout"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/images/mapblock"
)

// 探测游戏地图上的鼠标悬停动作
func DetectCursorHoverOnGameMap(misLayout layout.ScreenLayout) CursorHoverType {
	// FIXME 还没想好如何在选择的时候移动地图，先禁用
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		return DoNothing
	}
	x, y := ebiten.CursorPosition()
	w, h := misLayout.Camera.Width, misLayout.Camera.Height

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

// 游戏地图上的选区
var selectedArea = SelectedArea{}

// 探测游戏地图上的鼠标选区
func DetectCursorSelectArea(misState *state.MissionState) *SelectedArea {
	// 左键没有点击，直接跳过
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		selectedArea.Selecting = false
		return &selectedArea
	}
	if !selectedArea.Selecting {
		sx, sy := ebiten.CursorPosition()
		selectedArea.StartX = sx
		selectedArea.StartY = sy
		selectedArea.TopLeft.RX = float64(sx)/mapblock.BlockSize + float64(misState.Camera.Pos.MX)
		selectedArea.TopLeft.RY = float64(sy)/mapblock.BlockSize + float64(misState.Camera.Pos.MY)
		selectedArea.Selecting = true
	}
	sx, sy := ebiten.CursorPosition()
	selectedArea.CurX = sx
	selectedArea.CurY = sy
	// FIXME 考虑各种拉取的方式，如果是右下拉左上，这里 width，height 都是负数
	selectedArea.Width = float64(sx)/mapblock.BlockSize + float64(misState.Camera.Pos.MX) - selectedArea.TopLeft.RX
	selectedArea.Height = float64(sy)/mapblock.BlockSize + float64(misState.Camera.Pos.MY) - selectedArea.TopLeft.RY
	return &selectedArea
}
