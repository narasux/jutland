package game

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/resources/font"
)

// TestAutoUpdateMenuButtonStatesDoesNotOverlapEnglishText 验证英文菜单按钮在不同屏宽下均不重叠。
// 场景：JetbrainsMono 的 Mission Select/Collection/Settings/Exit Game 长度不一，在 1024 和 2048 宽下布局。
// 预期：前一项右缘 ≤ 后一项 PosX，首项 ≥0，末项右缘 ≤ 屏宽；失败意味着文字重叠或按钮出屏。
// 覆盖：BaseFontSize=36、1280 高画布，两个子测试 width_2048/width_1024 遍历四按钮断言。
func TestAutoUpdateMenuButtonStatesDoesNotOverlapEnglishText(t *testing.T) {
	for _, screenWidth := range []int{2048, 1024} {
		t.Run(fmt.Sprintf("width_%d", screenWidth), func(t *testing.T) {
			buttons := &menuButtonStates{
				BaseFontSize:  menuFontSize,
				MissionSelect: &menuButton{Text: "Mission Select", Font: font.JetbrainsMono},
				Collection:    &menuButton{Text: "Collection", Font: font.JetbrainsMono},
				GameSetting:   &menuButton{Text: "Settings", Font: font.JetbrainsMono},
				ExitGame:      &menuButton{Text: "Exit Game", Font: font.JetbrainsMono},
			}
			states := &objStates{MenuButton: buttons}
			states.AutoUpdateMenuButtonStates(ebiten.NewImage(screenWidth, 1280))

			ordered := []*menuButton{
				buttons.MissionSelect,
				buttons.Collection,
				buttons.GameSetting,
				buttons.ExitGame,
			}
			for idx := 1; idx < len(ordered); idx++ {
				require.LessOrEqual(t, ordered[idx-1].PosX+ordered[idx-1].Width, ordered[idx].PosX)
			}
			require.GreaterOrEqual(t, ordered[0].PosX, 0.0)
			require.LessOrEqual(
				t,
				ordered[len(ordered)-1].PosX+ordered[len(ordered)-1].Width,
				float64(screenWidth),
			)
		})
	}
}

// TestAutoUpdateMenuButtonStatesKeepsClearGapOnWideScreen 验证宽屏下相邻菜单按钮保留最小间距。
// 场景：2048×1280 屏、OpenSans 英文标签，空间充足时布局会把多余宽度分摊到间隙。
// 预期：任意相邻按钮间隙 ≥ menuMinimumGap(48)；失败意味着宽屏下仍挤在一起。
// 覆盖：四个按钮调用 AutoUpdateMenuButtonStates 后，按 PosX 与 Width 逐一计算 gap 并断言。
func TestAutoUpdateMenuButtonStatesKeepsClearGapOnWideScreen(t *testing.T) {
	buttons := &menuButtonStates{
		BaseFontSize:  menuFontSize,
		MissionSelect: &menuButton{Text: "Mission Select", Font: font.OpenSans},
		Collection:    &menuButton{Text: "Collection", Font: font.OpenSans},
		GameSetting:   &menuButton{Text: "Settings", Font: font.OpenSans},
		ExitGame:      &menuButton{Text: "Exit Game", Font: font.OpenSans},
	}
	states := &objStates{MenuButton: buttons}
	states.AutoUpdateMenuButtonStates(ebiten.NewImage(2048, 1280))

	ordered := []*menuButton{
		buttons.MissionSelect,
		buttons.Collection,
		buttons.GameSetting,
		buttons.ExitGame,
	}
	for idx := 1; idx < len(ordered); idx++ {
		gap := ordered[idx].PosX - ordered[idx-1].PosX - ordered[idx-1].Width
		require.GreaterOrEqual(t, gap, menuMinimumGap)
	}
}

// TestAutoUpdateMenuButtonStatesRestoresBaseFontSizeAfterResize 验证窗口变宽后字号恢复到基准值。
// 场景：BaseFontSize=50 的中文菜单先在 320×200 下被缩小，再切到 2048×1280，不能沿用缩小后的字号。
// 预期：窄屏时 MissionSelect.FontSize<50，宽屏时回到 50（误差 0.001）；失败意味着基准字号被覆写。
// 覆盖：同一组按钮先后用两张 ebiten.Image 调用 AutoUpdate，分别断言 Less 与 InDelta。
func TestAutoUpdateMenuButtonStatesRestoresBaseFontSizeAfterResize(t *testing.T) {
	buttons := &menuButtonStates{
		BaseFontSize:  50,
		MissionSelect: &menuButton{Text: "任务选择", Font: font.Hang},
		Collection:    &menuButton{Text: "游戏图鉴", Font: font.Hang},
		GameSetting:   &menuButton{Text: "游戏设置", Font: font.Hang},
		ExitGame:      &menuButton{Text: "退出游戏", Font: font.Hang},
	}
	states := &objStates{MenuButton: buttons}

	states.AutoUpdateMenuButtonStates(ebiten.NewImage(320, 200))
	require.Less(t, buttons.MissionSelect.FontSize, buttons.BaseFontSize)

	states.AutoUpdateMenuButtonStates(ebiten.NewImage(2048, 1280))
	require.InDelta(t, buttons.BaseFontSize, buttons.MissionSelect.FontSize, 0.001)
}
