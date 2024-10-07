package layout

import "github.com/hajimehoshi/ebiten/v2"

// CalcTextWidth 计算文本宽度
func CalcTextWidth(text string, fontSize float64) float64 {
	// 字体原因，宽度大致是 0.35 的文字高度
	return fontSize * float64(len(text)) / 20 * 7
}

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
