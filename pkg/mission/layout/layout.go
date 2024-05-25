package layout

import "github.com/hajimehoshi/ebiten/v2"

// ScreenPos 屏幕坐标
type ScreenPos struct {
	SX int
	SY int
}

// Screen 关卡屏幕
type ScreenLayout struct {
	// 屏幕宽高（一般是屏幕分辨率）
	Width  int
	Height int
}

// NewScreenLayout ...
func NewScreenLayout() ScreenLayout {
	width, height := ebiten.Monitor().Size()
	return ScreenLayout{Width: width, Height: height}
}
