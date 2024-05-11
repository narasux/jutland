package drawer

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/colorx"
)

// 绘制控制台
func (d *Drawer) drawConsole(screen *ebiten.Image, ms *state.MissionState) {
	// 纯银色控制台
	consoleBGImg := ebiten.NewImage(ms.Layout.Console.Width, ms.Layout.Console.Height)
	consoleBGImg.Fill(colorx.DarkSilver)
	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(
		float64(ms.Layout.Console.TopLeft.SX),
		float64(ms.Layout.Console.TopLeft.SY),
	)
	screen.DrawImage(consoleBGImg, opts)

	// 小地图
	opts = d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(
		float64(ms.Layout.Console.SmallMap.TopLeft.SX),
		float64(ms.Layout.Console.SmallMap.TopLeft.SY),
	)
	// FIXME 目前先丢一个黑图片进去，将来应该是考虑地图，战舰，视野等等，渲染一个实时图片
	smallMapImg := ebiten.NewImage(ms.Layout.Console.SmallMap.Width, ms.Layout.Console.SmallMap.Height)
	smallMapImg.Fill(colorx.Black)
	screen.DrawImage(smallMapImg, opts)

	// 菜单
	opts = d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(
		float64(ms.Layout.Console.Menu.TopLeft.SX),
		float64(ms.Layout.Console.Menu.TopLeft.SY),
	)
	// FIXME 目前先丢一个银色图片进去，将来再考虑按钮之类的
	menuImg := ebiten.NewImage(ms.Layout.Console.Menu.Width, ms.Layout.Console.Menu.Height)
	menuImg.Fill(colorx.Silver)
	screen.DrawImage(menuImg, opts)
}
