package game

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/resources/font"
)

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
