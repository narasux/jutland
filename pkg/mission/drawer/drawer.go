package drawer

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	md "github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/images/mapblock"
)

// Drawer 任务运行中图像绘制器
type Drawer struct {
	mission string
}

// NewDrawer ...
func NewDrawer(mission string) *Drawer {
	missionMD := md.Get(mission)
	if err := mapblock.LoadMapSceneBlocks(missionMD.MapCfg.Name); err != nil {
		log.Fatal("failed to load map scene blocks: ", err)
	}
	return &Drawer{mission: mission}
}

// Draw 绘制任务关卡图像
func (d *Drawer) Draw(screen *ebiten.Image, misState *state.MissionState) {
	// 相机视野
	d.drawCameraView(screen, misState)
	// 地图元素
	d.drawBuildings(screen, misState)
	d.drawShotBullets(screen, misState)
	d.drawObjectTrails(screen, misState)
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
