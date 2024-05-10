package action

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/layout"
	obj "github.com/narasux/jutland/pkg/mission/object"
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
var sArea = SelectedArea{}

// 探测游戏地图上的鼠标选区
func DetectCursorSelectArea(misState *state.MissionState) *SelectedArea {
	// 左键没有被压下，直接跳过
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		sArea.Selecting = false
		return nil
	}
	if !sArea.Selecting {
		sx, sy := ebiten.CursorPosition()
		sArea.StartX, sArea.StartY = sx, sy
		sArea.StartAt.AssignRxy(
			float64(sx)/mapblock.BlockSize+float64(misState.Camera.Pos.MX),
			float64(sy)/mapblock.BlockSize+float64(misState.Camera.Pos.MY),
		)
		sArea.Selecting = true
	}
	sx, sy := ebiten.CursorPosition()
	sArea.CurX, sArea.CurY = sx, sy
	sArea.CurAt.AssignRxy(
		float64(sx)/mapblock.BlockSize+float64(misState.Camera.Pos.MX),
		float64(sy)/mapblock.BlockSize+float64(misState.Camera.Pos.MY),
	)
	return &sArea
}

// 探测游戏地图上的左键点击
func DetectMouseLeftButtonClickOnMap(misState *state.MissionState) *obj.MapPos {
	return detectMouseButtonClickOnMap(misState, ebiten.MouseButtonLeft)
}

// 探测游戏地图上的右键点击
func DetectMouseRightButtonClickOnMap(misState *state.MissionState) *obj.MapPos {
	return detectMouseButtonClickOnMap(misState, ebiten.MouseButtonRight)
}

// 探测游戏地图上的鼠标按键点击
func detectMouseButtonClickOnMap(misState *state.MissionState, button ebiten.MouseButton) *obj.MapPos {
	// 鼠标按键没有点击，直接跳过
	if !inpututil.IsMouseButtonJustPressed(button) {
		return nil
	}
	sx, sy := ebiten.CursorPosition()
	mx := misState.Camera.Pos.MX + int(float64(sx)/mapblock.BlockSize)
	my := misState.Camera.Pos.MY + int(float64(sy)/mapblock.BlockSize)
	return lo.ToPtr(obj.NewMapPos(mx, my))
}
