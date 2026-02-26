package drawer

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/warfog"
)

// FogDrawer 迷雾渲染器
type FogDrawer struct {
	// 缓存的迷雾图像（避免重复创建）
	fogUnexplored *ebiten.Image
	fogExplored   *ebiten.Image
}

// NewFogDrawer 创建迷雾渲染器
func NewFogDrawer() *FogDrawer {
	// 创建 1x1 的纯色图像（会在渲染时缩放）
	fogUnexplored := ebiten.NewImage(1, 1)
	fogUnexplored.Fill(color.RGBA{0, 0, 0, 255}) // 黑色，完全不透明

	fogExplored := ebiten.NewImage(1, 1)
	fogExplored.Fill(color.RGBA{0, 0, 0, 128}) // 黑色，半透明

	return &FogDrawer{
		fogUnexplored: fogUnexplored,
		fogExplored:   fogExplored,
	}
}

// DrawFog 绘制战争迷雾
func (d *FogDrawer) DrawFog(screen *ebiten.Image, misState *state.MissionState) {
	fog := misState.FogOfWar
	if fog == nil {
		return
	}

	// BlackSheepWall 激活时不绘制迷雾
	if fog.BlackSheepWallActive {
		return
	}

	camera := &misState.Camera

	// 只绘制相机视野范围内的迷雾（性能优化）
	x1, y1, x2, y2 := camera.GetVisibleRange()

	// 边界检查
	if x1 < 0 {
		x1 = 0
	}
	if y1 < 0 {
		y1 = 0
	}
	if x2 > fog.MapWidth {
		x2 = fog.MapWidth
	}
	if y2 > fog.MapHeight {
		y2 = fog.MapHeight
	}

	// 绘制每个格子的迷雾
	for x := x1; x < x2; x++ {
		for y := y1; y < y2; y++ {
			fogState := fog.GetFogState(x, y)
			if fogState == warfog.FogStateVisible {
				continue // 可见区域不绘制迷雾
			}

			// 计算屏幕坐标
			screenX := float64((x - camera.Pos.MX) * constants.MapBlockSize)
			screenY := float64((y - camera.Pos.MY) * constants.MapBlockSize)

			// 选择迷雾图像
			var fogImage *ebiten.Image
			if fogState == warfog.FogStateUnexplored {
				fogImage = d.fogUnexplored
			} else {
				fogImage = d.fogExplored
			}

			// 绘制迷雾
			// 注意：GeoM操作顺序非常重要！必须先缩放，后平移
			// 因为 Ebiten 的矩阵操作是从后往前应用的
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Scale(float64(constants.MapBlockSize), float64(constants.MapBlockSize))
			opts.GeoM.Translate(screenX, screenY)
			screen.DrawImage(fogImage, opts)
		}
	}
}
