package dive

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

// TestGetAllStatesAndZooms 验证潜艇深度图标的三种配色与三档缩放都能在启动期加载。
// 场景：绿色 Get 覆盖 Surface/Surfacing，青色 GetUnderwater 覆盖 Diving/Submerged，金色 GetTransition 覆盖 Diving/Surfacing；每档缩放 1/2/4 对应 16/32/64px，与武器图标档位一致。
// 预期：每个状态×缩放都返回非 nil，且图像边长严格等于 16×zoom；失败意味着 128px 源图缺失或 genZoomImages 的逐级减半被破坏。
// 覆盖：三组 getter 表驱动遍历各自状态和 []int{1,2,4}，同时断言行宽与行高。
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
