package action

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/layout"
)

// 探测游戏地图上的鼠标悬停动作
func DetectCursorHoverOnGameMap(misLayout layout.ScreenLayout) CursorHoverType {
	// FIXME 还没想好如何在选择的时候移动地图，先禁用
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		return DoNothing
	}
	x, y := ebiten.CursorPosition()
	w, h := misLayout.Width, misLayout.Height
	rightEdgeWidth := 50

	// Hover 上侧
	if y < 5 {
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
	if y > h-50 {
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
	if x < 5 {
		return HoverScreenLeft
	}
	// Hover 右侧
	if x > w-rightEdgeWidth {
		return HoverScreenRight
	}

	// Hover 在中间
	if 5 < x && x < w-rightEdgeWidth && 5 < y && y < h-5 {
		return HoverScreenMiddle
	}
	return DoNothing
}

// 游戏地图上的选区
var sArea = SelectedArea{}

// 探测游戏地图上的鼠标选区
func DetectCursorSelectArea(misState *state.MissionState) *SelectedArea {
	if misState.UI.SidebarConsumesCursor {
		misState.Interaction.IsAreaSelecting = false
		return nil
	}
	// 左键没有被压下，直接跳过
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		misState.Interaction.IsAreaSelecting = false
		return nil
	}
	if !misState.Interaction.IsAreaSelecting {
		sx, sy := ebiten.CursorPosition()
		sArea.StartX, sArea.StartY = sx, sy
		sArea.StartAt = misState.ScreenToCameraPos(float64(sx), float64(sy))
		misState.Interaction.IsAreaSelecting = true
	}
	sx, sy := ebiten.CursorPosition()
	sArea.CurX, sArea.CurY = sx, sy
	sArea.CurAt = misState.ScreenToCameraPos(float64(sx), float64(sy))
	return &sArea
}

// 探测游戏地图上的鼠标按键点击
func DetectMouseButtonClickOnMap(misState *state.MissionState, button ebiten.MouseButton) *objPos.MapPos {
	if misState.UI.SidebarConsumesCursor {
		return nil
	}
	// 鼠标按键没有点击，直接跳过
	if !inpututil.IsMouseButtonJustPressed(button) {
		return nil
	}
	return DetectCursorPosOnMap(misState)
}

// 探测当前鼠标在地图上的位置
func DetectCursorPosOnMap(misState *state.MissionState) *objPos.MapPos {
	sx, sy := ebiten.CursorPosition()
	return lo.ToPtr(misState.ScreenToCameraPos(float64(sx), float64(sy)))
}

// 键盘按键与组 ID 的映射关系
type KeyGroupIDMapping struct {
	Key     ebiten.Key
	GroupID object.GroupID
}

var keyGroupIDMap = []KeyGroupIDMapping{
	{Key: ebiten.KeyDigit1, GroupID: object.GroupID1},
	{Key: ebiten.KeyDigit2, GroupID: object.GroupID2},
	{Key: ebiten.KeyDigit3, GroupID: object.GroupID3},
	{Key: ebiten.KeyDigit4, GroupID: object.GroupID4},
	{Key: ebiten.KeyDigit5, GroupID: object.GroupID5},
	{Key: ebiten.KeyDigit6, GroupID: object.GroupID6},
	{Key: ebiten.KeyDigit7, GroupID: object.GroupID7},
	{Key: ebiten.KeyDigit8, GroupID: object.GroupID8},
	{Key: ebiten.KeyDigit9, GroupID: object.GroupID9},
	{Key: ebiten.KeyDigit0, GroupID: object.GroupID0},
}

// GetGroupIDByPressedKey 探测按键对应的组 ID
func GetGroupIDByPressedKey() object.GroupID {
	for _, mapping := range keyGroupIDMap {
		if inpututil.IsKeyJustPressed(mapping.Key) {
			return mapping.GroupID
		}
	}
	return object.GroupIDNone
}
