package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

type objStates struct {
	MenuButton *menuButtonStates
}

type menuButtonStates struct {
	// 任务选择
	MissionSelect *menuButton
	// 游戏图鉴
	Collection *menuButton
	// 游戏设置
	GameSetting *menuButton
	// 退出游戏
	ExitGame *menuButton
}

type menuButton struct {
	// 菜单按钮文本
	Text string
	// 对应的游戏模式
	Mode GameMode
	// top-left 位置
	PosX float64
	PosY float64
	// 文本尺寸，有默认值，鼠标 hover 会改变
	FontSize float64
	// 文本字体
	Font *text.GoTextFaceSource
	// 文本颜色
	Color color.Color
	// 渲染出的按钮尺寸
	Width  float64
	Height float64
}

// 根据菜单按钮文本 & 字体尺寸，自动计算位置等信息
func (s *objStates) AutoUpdateMenuButtonStates(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	for idx, button := range []*menuButton{
		s.MenuButton.MissionSelect,
		s.MenuButton.Collection,
		s.MenuButton.GameSetting,
		s.MenuButton.ExitGame,
	} {
		button.PosX = (float64(screenWidth) - ebutil.CalcTextWidth(button.Text, button.FontSize)) * 0.2 * float64(idx+1)
		button.PosY = float64(screenHeight / 5 * 4)
		button.Width = ebutil.CalcTextWidth(button.Text, button.FontSize)
		button.Height = button.FontSize
	}
}
