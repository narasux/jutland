package drawer

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/state"
)

// FogImageCache 蒙层贴图缓存：img 是渲染分辨率下的贴图，stamp 是已上传的蒙层版本。
// 主视图与小地图各持一份；同一份像素只在重新合成后上传一次。
type FogImageCache struct {
	img   *ebiten.Image
	stamp int64
}

// BlitFog 把当前玩家的蒙层画到屏幕。迷雾关闭时什么都不做。
func BlitFog(
	screen *ebiten.Image,
	ms *state.MissionState,
	cache *FogImageCache,
	scaleX, scaleY, translateX, translateY float64,
) {
	// 当前玩家没开迷雾，或这一拍还没生成蒙层，就不画。
	if cache == nil || !ms.UsesFog(ms.Player.CurPlayer) {
		return
	}
	vision := ms.Player.Visions[ms.Player.CurPlayer]
	if vision == nil || len(vision.Shade) == 0 {
		return
	}
	// 蒙层是每格 4 像素。尺寸变了才重建贴图，避免每拍重新分配。
	rw, rh := vision.Width*4, vision.Height*4
	if cache.img == nil || cache.img.Bounds().Dx() != rw || cache.img.Bounds().Dy() != rh {
		cache.img = ebiten.NewImage(rw, rh)
		cache.stamp = 0
	}
	// 蒙层按降频节奏重新合成，没有新版本就不用再上传一次同样的像素。
	if cache.stamp != vision.ShadeStamp {
		cache.img.WritePixels(vision.Shade)
		cache.stamp = vision.ShadeStamp
	}
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	opts.GeoM.Scale(scaleX, scaleY)
	opts.GeoM.Translate(translateX, translateY)
	screen.DrawImage(cache.img, opts)
}

func (d *Drawer) drawFog(screen *ebiten.Image, ms *state.MissionState) {
	block := ms.MapBlockDisplaySize()
	BlitFog(
		screen, ms, &d.fog,
		block/4, block/4,
		-ms.View.Camera.Pos.RX*block, -ms.View.Camera.Pos.RY*block,
	)
}

func (d *Drawer) drawAbbrFog(screen *ebiten.Image, ms *state.MissionState) {
	if ms.Core.MissionMD.MapCfg == nil {
		return
	}
	abbrW := float64(d.abbrMap.Bounds().Dx())
	abbrH := float64(d.abbrMap.Bounds().Dy())
	mapW := float64(ms.Core.MissionMD.MapCfg.Width)
	mapH := float64(ms.Core.MissionMD.MapCfg.Height)
	xOffset := float64(ms.View.Layout.Width-d.abbrMap.Bounds().Dx()) / 2
	BlitFog(
		screen, ms, &d.fog,
		abbrW/(mapW*4), abbrH/(mapH*4),
		xOffset, 0,
	)
}
