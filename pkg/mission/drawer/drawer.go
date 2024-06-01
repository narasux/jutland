package drawer

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/common/constants"
	md "github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/images/mapblock"
)

// Drawer 任务运行中图像绘制器
type Drawer struct {
	mapImg     *ebiten.Image
	curViewImg *ebiten.Image
}

// NewDrawer ...
func NewDrawer(mission string) *Drawer {
	missionMD := md.Get(mission)
	mapW, mapH := missionMD.MapCfg.Width, missionMD.MapCfg.Height
	// 预先渲染好整个地图，逐帧渲染的时候裁剪即可
	mapImg := ebiten.NewImage(mapW*constants.MapBlockSize, mapH*constants.MapBlockSize)
	for x := 0; x < mapW; x++ {
		for y := 0; y < mapH; y++ {
			char := missionMD.MapCfg.Map.Get(x, y)
			opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			opts.GeoM.Translate(float64(x)*constants.MapBlockSize, float64(y)*constants.MapBlockSize)
			mapImg.DrawImage(mapblock.GetByCharAndPos(char, x, y), opts)
		}
	}
	return &Drawer{mapImg: mapImg}
}

// Draw 绘制任务关卡图像
func (d *Drawer) Draw(screen *ebiten.Image, misState *state.MissionState) {
	// 相机视野
	d.drawCameraViewRealTimeRender(screen, misState)
	// 地图元素
	d.drawBuildings(screen, misState)
	d.drawShotBullets(screen, misState)
	d.drawShipTrails(screen, misState)
	d.drawBattleShips(screen, misState)
	d.drawDestroyedShips(screen, misState)
	// 用户行为
	d.drawArrowOnMapWhenHover(screen, misState)
	d.drawSelectedArea(screen, misState)
	d.drawMarks(screen, misState)
	d.drawTips(screen, misState)
}

func (d *Drawer) genDefaultDrawImageOptions() *ebiten.DrawImageOptions {
	return &ebiten.DrawImageOptions{
		// 线性过滤：通过计算周围像素的加权平均值来进行插值，可使得边缘 & 色彩转换更加自然
		Filter: ebiten.FilterLinear,
	}
}
