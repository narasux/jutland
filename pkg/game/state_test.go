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
