package state

import (
	"image"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
	"github.com/narasux/jutland/pkg/utils/layout"
)

// newZoomTestState 构造一个位于地图中部、相机不贴边的任务状态，
// 避免 RefreshCameraSize 的边界夹取干扰缩放锚点断言。
func newZoomTestState() *MissionState {
	return &MissionState{
		Core: MissionCoreState{
			MissionMD: metadata.MissionMetadata{
				MapCfg: &mapcfg.MapCfg{Width: 1000, Height: 1000},
			},
		},
		View: MissionViewState{
			Layout: layout.ScreenLayout{Width: 1280, Height: 720},
			Camera: Camera{Pos: objPos.NewR(100, 100), Width: 11, Height: 7},
		},
		UI: MissionUIState{GameOpts: GameOptions{Zoom: DefaultZoom()}},
	}
}

// TestNormalizeZoomFallsBackToDefault 验证非法缩放值不会流入坐标换算。
// 场景：旧存档、调试秘籍或配置可能写入 0、3、100 等非档位值；
// blockSize 由 zoom 计算，若返回非法值可能得到 0 并在屏幕/地图换算时除零。
// 预期：合法档位原样返回；其他值统一回退 DefaultZoom（4）。
// 覆盖：表驱动遍历 1/2/4/8/16 与 0/3/-1/100，逐个断言 NormalizeZoom。
func TestNormalizeZoomFallsBackToDefault(t *testing.T) {
	for _, zoom := range AvailableZooms {
		require.Equal(t, zoom, NormalizeZoom(zoom), "合法档位应原样返回")
	}
	for _, zoom := range []int{0, 3, 5, 100, -1} {
		require.Equal(t, DefaultZoom(), NormalizeZoom(zoom), "非法档位应回退默认值")
	}
}

// TestScreenAndCameraPositionsRoundTrip 验证屏幕坐标与地图坐标互为逆变换。
// 场景：框选、点击移动、缩略图定位都先把屏幕点转成地图点；相机位置任意，
// 若两套换算的 blockSize 或相机偏移不一致，点击位置会整体漂移。
// 预期：CameraPosToScreen 后再 ScreenToCameraPos 回到原 RX/RY；
// ScreenPointToCameraPos 与浮点版本对同一整数点给出相同结果。
// 覆盖：在 1x/4x/16x 三个档位下，对若干地图点和屏幕整数点做往返断言。
func TestScreenAndCameraPositionsRoundTrip(t *testing.T) {
	for _, zoom := range []int{1, 4, 16} {
		ms := newZoomTestState()
		ms.UI.GameOpts.Zoom = zoom

		for _, pos := range []objPos.MapPos{
			objPos.NewR(100, 100),
			objPos.NewR(123.25, 87.75),
			objPos.NewR(300.5, 220.5),
		} {
			sx, sy := ms.CameraPosToScreen(pos)
			got := ms.ScreenToCameraPos(sx, sy)
			require.InDelta(t, pos.RX, got.RX, 1e-9)
			require.InDelta(t, pos.RY, got.RY, 1e-9)
		}

		point := image.Pt(640, 360)
		require.Equal(t, ms.ScreenToCameraPos(640, 360), ms.ScreenPointToCameraPos(point))
	}
}

// TestStepZoomAtScreenPointKeepsAnchor 验证缩放时鼠标下的地图位置保持不动。
// 场景：玩家在战场某点滚轮/双指缩放，期望放大的就是鼠标所指区域；
// 若只改 zoom 不反算相机位置，视野会向地图左上角跳动。
// 预期：缩放前后同一屏幕点对应的地图坐标 RX/RY 保持不变（不贴边时）。
// 覆盖：记录 4x->8x、8x->4x 两种方向缩放前后的锚点，断言 RX/RY 在 1e-9 内相等。
func TestStepZoomAtScreenPointKeepsAnchor(t *testing.T) {
	ms := newZoomTestState()
	const sx, sy = 640, 360

	anchor := ms.ScreenToCameraPos(sx, sy)
	ms.StepZoomAtScreenPoint(1, sx, sy)
	require.Equal(t, 8, ms.UI.GameOpts.Zoom, "4x 向上一档应到 8x")
	zoomedIn := ms.ScreenToCameraPos(sx, sy)
	require.InDelta(t, anchor.RX, zoomedIn.RX, 1e-9)
	require.InDelta(t, anchor.RY, zoomedIn.RY, 1e-9)

	ms.StepZoomAtScreenPoint(-1, sx, sy)
	require.Equal(t, 4, ms.UI.GameOpts.Zoom, "8x 向下一档应回 4x")
	zoomedOut := ms.ScreenToCameraPos(sx, sy)
	require.InDelta(t, anchor.RX, zoomedOut.RX, 1e-9)
	require.InDelta(t, anchor.RY, zoomedOut.RY, 1e-9)
}

// TestStepZoomAtScreenPointStopsAtLimits 验证到达最小/最大档位后不再移动相机。
// 场景：玩家在 1x 继续向下滚，或在 16x 继续向上滚；若没有索引保护，
// nextIndex 会越界并可能回绕到另一端或改动相机。
// 预期：缩放档位与相机位置都保持不变。
// 覆盖：分别从 1x 和 16x 向越界方向调用，断言 zoom 与相机坐标不变。
func TestStepZoomAtScreenPointStopsAtLimits(t *testing.T) {
	for _, tc := range []struct {
		name      string
		zoom      int
		direction int
	}{
		{name: "minimum", zoom: AvailableZooms[0], direction: -1},
		{name: "maximum", zoom: AvailableZooms[len(AvailableZooms)-1], direction: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ms := newZoomTestState()
			ms.UI.GameOpts.Zoom = tc.zoom
			before := ms.View.Camera.Pos

			ms.StepZoomAtScreenPoint(tc.direction, 640, 360)

			require.Equal(t, tc.zoom, ms.UI.GameOpts.Zoom)
			require.Equal(t, before, ms.View.Camera.Pos)
		})
	}
}
