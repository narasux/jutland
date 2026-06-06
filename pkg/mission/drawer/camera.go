package drawer

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/state"
	mapBlockImg "github.com/narasux/jutland/pkg/resources/images/mapblock"
)

// 绘制相机视野（实时渲染）
func (d *Drawer) drawCameraView(screen *ebiten.Image, ms *state.MissionState) {
	xAccOffset := ms.View.Camera.Pos.RX - float64(ms.View.Camera.Pos.MX)
	yAccOffset := ms.View.Camera.Pos.RY - float64(ms.View.Camera.Pos.MY)
	blockSize := ms.MapBlockDisplaySize()
	for x := 0; x <= ms.View.Camera.Width; x++ {
		for y := 0; y <= ms.View.Camera.Height; y++ {
			opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
			opts.GeoM.Translate(
				(float64(x)-xAccOffset)*blockSize,
				(float64(y)-yAccOffset)*blockSize,
			)

			mapX, mapY := ms.View.Camera.Pos.MX+x, ms.View.Camera.Pos.MY+y
			char := ms.Core.MissionMD.MapCfg.Map.Get(mapX, mapY)
			images := mapBlockImg.GetByCharAndPosZoom(char, mapX, mapY, ms.UI.GameOpts.Zoom)
			for _, img := range images {
				if img == nil {
					continue
				}
				screen.DrawImage(img, opts)
			}
		}
	}
}
