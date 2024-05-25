package action

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/layout"
	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
)

// 探测游戏地图上的鼠标悬停动作
func DetectCursorHoverOnGameMap(misLayout layout.ScreenLayout) CursorHoverType {
	// FIXME 还没想好如何在选择的时候移动地图，先禁用
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		return DoNothing
	}
	x, y := ebiten.CursorPosition()
	w, h := misLayout.Width, misLayout.Height

	// Hover 上侧
	if y < h/40 {
		// Hover 左侧 3/10
		if 0 < x && x < w/10*3 {
			return HoverScreenTopLeft
		}
		// Hover 右侧 3/10
		if w/10*7 < x && x < w {
			return HoverScreenTopRight
		}
		// Hover 中间 4/10
		return HoverScreenTop
	}

	// Hover 下侧
	if h/40*39 < y {
		// Hover 左侧 3/10
		if x < w/10*3 {
			return HoverScreenBottomLeft
		}
		// Hover 右侧 3/10
		if w/10*7 < x {
			return HoverScreenBottomRight
		}
		// Hover 中间 4/10
		return HoverScreenBottom
	}

	// Hover 左侧
	if x < w/50 {
		return HoverScreenLeft
	}
	// Hover 右侧
	if w/50*49 < x && x < w {
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
		misState.IsAreaSelecting = false
		return nil
	}
	if !misState.IsAreaSelecting {
		sx, sy := ebiten.CursorPosition()
		sArea.StartX, sArea.StartY = sx, sy
		sArea.StartAt.AssignRxy(
			float64(sx)/constants.MapBlockSize+float64(misState.Camera.Pos.MX),
			float64(sy)/constants.MapBlockSize+float64(misState.Camera.Pos.MY),
		)
		misState.IsAreaSelecting = true
	}
	sx, sy := ebiten.CursorPosition()
	sArea.CurX, sArea.CurY = sx, sy
	sArea.CurAt.AssignRxy(
		float64(sx)/constants.MapBlockSize+float64(misState.Camera.Pos.MX),
		float64(sy)/constants.MapBlockSize+float64(misState.Camera.Pos.MY),
	)
	return &sArea
}

// 探测游戏地图上的鼠标按键点击
func DetectMouseButtonClickOnMap(misState *state.MissionState, button ebiten.MouseButton) *obj.MapPos {
	// 鼠标按键没有点击，直接跳过
	if !inpututil.IsMouseButtonJustPressed(button) {
		return nil
	}
	sx, sy := ebiten.CursorPosition()
	rx := float64(misState.Camera.Pos.MX) + float64(sx)/constants.MapBlockSize
	ry := float64(misState.Camera.Pos.MY) + float64(sy)/constants.MapBlockSize
	return lo.ToPtr(obj.NewMapPosR(rx, ry))
}

// 探测键盘按键被压下
func DetectKeyboardKeyJustPressed(key ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(key)
}

// 探测键盘按键被释放
func DetectKeyboardKeyJustReleased(key ebiten.Key) bool {
	return inpututil.IsKeyJustReleased(key)
}

// 键盘按键与组 ID 的映射关系
type KeyGroupIDMapping struct {
	Key     ebiten.Key
	GroupID obj.GroupID
}

var keyGroupIDMappings = []KeyGroupIDMapping{
	{Key: ebiten.KeyDigit1, GroupID: obj.GroupID1},
	{Key: ebiten.KeyDigit2, GroupID: obj.GroupID2},
	{Key: ebiten.KeyDigit3, GroupID: obj.GroupID3},
	{Key: ebiten.KeyDigit4, GroupID: obj.GroupID4},
	{Key: ebiten.KeyDigit5, GroupID: obj.GroupID5},
	{Key: ebiten.KeyDigit6, GroupID: obj.GroupID6},
	{Key: ebiten.KeyDigit7, GroupID: obj.GroupID7},
	{Key: ebiten.KeyDigit8, GroupID: obj.GroupID8},
	{Key: ebiten.KeyDigit9, GroupID: obj.GroupID9},
	{Key: ebiten.KeyDigit0, GroupID: obj.GroupID0},
}

// GetGroupIDByPressedKey 探测按键对应的组 ID
func GetGroupIDByPressedKey() obj.GroupID {
	for _, mapping := range keyGroupIDMappings {
		if inpututil.IsKeyJustPressed(mapping.Key) {
			return mapping.GroupID
		}
	}
	return obj.GroupIDNone
}
