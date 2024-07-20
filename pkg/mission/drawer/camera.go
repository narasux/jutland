package drawer

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/images/mapblock"
)

// 绘制相机视野（实时渲染）
func (d *Drawer) drawCameraView(screen *ebiten.Image, ms *state.MissionState) {
	xAccOffset := ms.Camera.Pos.RX - float64(ms.Camera.Pos.MX)
	yAccOffset := ms.Camera.Pos.RY - float64(ms.Camera.Pos.MY)
	for x := 0; x < ms.Camera.Width; x++ {
		for y := 0; y < ms.Camera.Height; y++ {
			opts := d.genDefaultDrawImageOptions()
			opts.GeoM.Translate(
				(float64(x)-xAccOffset)*constants.MapBlockSize,
				(float64(y)-yAccOffset)*constants.MapBlockSize,
			)

			mapX, mapY := ms.Camera.Pos.MX+x, ms.Camera.Pos.MY+y
			char := ms.MissionMD.MapCfg.Map.Get(mapX, mapY)
			images := mapblock.GetByCharAndPos(char, mapX, mapY)
			for _, img := range images {
				screen.DrawImage(img, opts)
			}
		}
	}
}
