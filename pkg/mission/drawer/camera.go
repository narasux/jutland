package drawer

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/images/mapblock"
)

// 绘制相机视野（实时渲染）
func (d *Drawer) drawCameraViewRealTimeRender(screen *ebiten.Image, ms *state.MissionState) {
	for x := 0; x < ms.Camera.Width; x++ {
		for y := 0; y < ms.Camera.Height; y++ {
			opts := d.genDefaultDrawImageOptions()
			opts.GeoM.Translate(float64(x)*constants.MapBlockSize, float64(y)*constants.MapBlockSize)

			mapX, mapY := ms.Camera.Pos.MX+x, ms.Camera.Pos.MY+y
			char := ms.MissionMD.MapCfg.Map.Get(mapX, mapY)
			screen.DrawImage(mapblock.GetByCharAndPos(char, mapX, mapY), opts)
		}
	}
}

// 绘制相机视野（裁剪的方式）
func (d *Drawer) drawCameraViewCropSprite(screen *ebiten.Image, ms *state.MissionState) {
	x1 := ms.Camera.Pos.MX * constants.MapBlockSize
	y1 := ms.Camera.Pos.MY * constants.MapBlockSize
	x2 := x1 + ms.Camera.Width*constants.MapBlockSize
	y2 := y1 + ms.Camera.Height*constants.MapBlockSize
	cropRect := image.Rect(x1, y1, x2, y2)
	d.curViewImg = d.mapImg.SubImage(cropRect).(*ebiten.Image)
	screen.DrawImage(d.curViewImg, nil)
}
