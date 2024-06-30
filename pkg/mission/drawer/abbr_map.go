package drawer

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/images/texture"
)

// 绘制缩略地图
func (d *Drawer) drawAbbreviationMap(screen *ebiten.Image, ms *state.MissionState) {
	abbrMapWidth, abbrMapHeight := d.abbrMap.Bounds().Dx(), d.abbrMap.Bounds().Dy()

	opts := d.genDefaultDrawImageOptions()
	// 居中展示缩略地图
	xOffset := float64(ms.Layout.Width-abbrMapWidth) / 2
	opts.GeoM.Translate(xOffset, 0)
	screen.DrawImage(d.abbrMap, opts)

	// 绘制地图元素
	// 敌我战舰
	for _, ship := range ms.Ships {
		shipImg := texture.GetAbbrShipImg(ship.Tonnage, ship.BelongPlayer != ms.CurPlayer)
		opts = d.genDefaultDrawImageOptions()
		setOptsCenterRotation(opts, shipImg, ship.CurRotation)

		xIndex := ship.CurPos.RX / float64(ms.MissionMD.MapCfg.Width) * float64(abbrMapWidth)
		yIndex := ship.CurPos.RY / float64(ms.MissionMD.MapCfg.Height) * float64(abbrMapHeight)

		opts.GeoM.Translate(xIndex+xOffset, yIndex)
		screen.DrawImage(shipImg, opts)
	}
}
