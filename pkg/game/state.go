package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/narasux/jutland/pkg/utils/layout"
)

type objStates struct {
	MenuButton       *menuButtonStates
	LoadingInterface *loadingInterface
	MissionSelectUI  *missionSelectUI
}

type menuButtonStates struct {
	// 菜单设计字号；窗口自适应缩放不能覆盖该基准值。
	BaseFontSize float64
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
	// 文本尺寸
	FontSize float64
	// 文本字体
	Font *text.GoTextFaceSource
	// 文本颜色
	Color color.Color
	// 渲染出的按钮尺寸
	Width  float64
	Height float64
}

const (
	menuFontSize          = 36.0
	menuHorizontalPadding = 24.0
	menuMinimumGap        = 48.0
)

// AutoUpdateMenuButtonStates 根据菜单按钮文本 & 字体尺寸，自动计算位置等信息
func (s *objStates) AutoUpdateMenuButtonStates(screen *ebiten.Image) {
	screenWidth, screenHeight := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	buttons := []*menuButton{
		s.MenuButton.MissionSelect,
		s.MenuButton.Collection,
		s.MenuButton.GameSetting,
		s.MenuButton.ExitGame,
	}

	baseSize := s.MenuButton.BaseFontSize
	totalWidth := updateMenuButtonSizes(buttons, baseSize)
	gap := menuMinimumGap
	availableWidth := max(screenWidth-2*menuHorizontalPadding, 0)
	minimumWidth := totalWidth + gap*float64(len(buttons)-1)
	if minimumWidth > availableWidth && minimumWidth > 0 {
		scale := availableWidth / minimumWidth
		totalWidth = updateMenuButtonSizes(buttons, baseSize*scale)
		gap *= scale
	}

	// 中文等宽菜单保持原有 20% 分布；英文等长文本则压缩空隙，避免相邻项重叠。
	averageWidth := totalWidth / float64(len(buttons))
	preferredWidth := screenWidth*0.6 + averageWidth*0.4
	groupWidth := max(totalWidth+gap*float64(len(buttons)-1), preferredWidth)
	groupWidth = min(groupWidth, availableWidth)
	gap = (groupWidth - totalWidth) / float64(len(buttons)-1)

	posX := (screenWidth - groupWidth) / 2
	for _, button := range buttons {
		button.PosX = posX
		button.PosY = screenHeight / 5 * 4
		posX += button.Width + gap
	}
}

func updateMenuButtonSizes(buttons []*menuButton, fontSize float64) float64 {
	var totalWidth float64
	for _, button := range buttons {
		button.FontSize = fontSize
		button.Width = layout.CalcTextWidth(button.Text, button.FontSize, button.Font)
		button.Height = button.FontSize
		totalWidth += button.Width
	}
	return totalWidth
}

// 任务加载界面
type loadingInterface struct {
	Ready               bool
	MissionRunningDrawn bool
	LoadedAudioPlayed   bool
}

// Reset 重置加载界面状态，保证每次进入关卡都先显示 loading 再执行阻塞加载。
func (i *loadingInterface) Reset() {
	i.Ready = false
	i.MissionRunningDrawn = false
	i.LoadedAudioPlayed = false
}

// 关卡选择界面 UI
type missionSelectUI struct {
	LeftArrow     clickableArea
	RightArrow    clickableArea
	StartButton   clickableArea
	BackButton    clickableArea
	ClassicButton clickableArea
	TestButton    clickableArea
}

type clickableArea struct {
	X, Y, W, H float64
}
