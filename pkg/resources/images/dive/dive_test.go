package dive

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

// 水面与上浮的绿色图标、水下青色图标和过渡金色覆盖层都必须在启动期加载成功，
// 且尺寸与武器图标档位一致。
func TestGetAllStatesAndZooms(t *testing.T) {
	getters := []struct {
		name   string
		get    func(State, int) *ebiten.Image
		states []State
	}{
		{
			name: "normal", get: Get,
			states: []State{Surface, Surfacing},
		},
		{name: "underwater", get: GetUnderwater, states: []State{Diving, Submerged}},
		{name: "transition", get: GetTransition, states: []State{Diving, Surfacing}},
	}
	for _, getter := range getters {
		for _, state := range getter.states {
			for _, zoom := range []int{1, 2, 4} {
				img := getter.get(state, zoom)
				require.NotNil(t, img, "%s %s zoom %d", getter.name, state, zoom)
				require.Equal(
					t, 16*zoom, img.Bounds().Dx(), "%s %s zoom %d", getter.name, state, zoom,
				)
				require.Equal(
					t, 16*zoom, img.Bounds().Dy(), "%s %s zoom %d", getter.name, state, zoom,
				)
			}
		}
	}
}
