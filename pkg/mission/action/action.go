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

// edgeScrollZone 是屏幕四边触发相机滚屏的判定带宽度（像素）。
// 四条边共用同一宽度，角落区域是相邻两条判定带的交集：只有同时进入两条带才斜向滚屏。
const edgeScrollZone = 50

// HoverTypeAt 按光标屏幕坐标判定边缘滚屏动作，w、h 为战场逻辑尺寸。
// 纯函数，不读 Ebiten 输入，方便覆盖角落、判定带边界这些容易出错的点。
func HoverTypeAt(x, y, w, h int) CursorHoverType {
	left, right := x < edgeScrollZone, x > w-edgeScrollZone
	top, bottom := y < edgeScrollZone, y > h-edgeScrollZone

	switch {
	case top && left:
		return HoverScreenTopLeft
	case top && right:
		return HoverScreenTopRight
	case bottom && left:
		return HoverScreenBottomLeft
	case bottom && right:
		return HoverScreenBottomRight
	case top:
		return HoverScreenTop
	case bottom:
		return HoverScreenBottom
	case left:
		return HoverScreenLeft
	case right:
		return HoverScreenRight
	}
	return HoverScreenMiddle
}

// 探测游戏地图上的鼠标悬停动作
func DetectCursorHoverOnGameMap(misLayout layout.ScreenLayout) CursorHoverType {
	// FIXME 还没想好如何在选择的时候移动地图，先禁用
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		return DoNothing
	}
	x, y := ebiten.CursorPosition()
	return HoverTypeAt(x, y, misLayout.Width, misLayout.Height)
}

// 游戏地图上的选区
var sArea = SelectedArea{}

// CurrentSelectArea 返回 Update 里已经探测好的选区。没有在框选时返回 nil。
func CurrentSelectArea(misState *state.MissionState) *SelectedArea {
	if !misState.Interaction.IsAreaSelecting {
		return nil
	}
	return &sArea
}

// DetectCursorSelectArea 探测游戏地图上的鼠标选区，并更新是否正在框选。
func DetectCursorSelectArea(misState *state.MissionState) *SelectedArea {
	if misState.UI.UIConsumesCursor {
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
	if misState.UI.UIConsumesCursor {
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
