package mission

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/narasux/jutland/pkg/resources/mapblock"
)

// Drawer 任务运行中图像绘制器
type Drawer struct{}

// NewDrawer ...
func NewDrawer() *Drawer {
	return &Drawer{}
}

// Draw 绘制任务图像（仅相机页面）
func (d *Drawer) Draw(screen *ebiten.Image, state *MissionState) {
	d.drawBackground(screen, state)
	d.drawBuildings(screen, state)
	d.drawBattleShips(screen, state)
	d.drawShotBullets(screen, state)
	d.drawUI(screen, state)
}

// 绘制背景（地图资源）
func (d *Drawer) drawBackground(screen *ebiten.Image, state *MissionState) {
	// 多绘制一行 & 列，避免出现黑边
	blockCountX := screen.Bounds().Dx()/mapblock.BlockSize + 1
	blockCountY := screen.Bounds().Dy()/mapblock.BlockSize + 1

	for x := 0; x < blockCountX; x++ {
		for y := 0; y < blockCountY; y++ {
			opts := d.genDefaultDrawImageOptions()
			opts.GeoM.Translate(float64(x)*mapblock.BlockSize, float64(y)*mapblock.BlockSize)

			mapX, mapY := state.CameraPos.X+x, state.CameraPos.Y+y
			char := state.MissionMD.MapCfg.Map.Get(mapX, mapY)
			screen.DrawImage(mapblock.GetByCharAndPos(char, mapX, mapY), opts)
		}
	}
}

// 绘制建筑物
func (d *Drawer) drawBuildings(screen *ebiten.Image, state *MissionState) {
}

// 绘制战舰
func (d *Drawer) drawBattleShips(screen *ebiten.Image, state *MissionState) {
}

// 绘制已发射的弹丸
func (d *Drawer) drawShotBullets(screen *ebiten.Image, state *MissionState) {
}

// 绘制 UI
func (d *Drawer) drawUI(screen *ebiten.Image, state *MissionState) {
}

func (d *Drawer) genDefaultDrawImageOptions() *ebiten.DrawImageOptions {
	return &ebiten.DrawImageOptions{
		// 线性过滤：通过计算周围像素的加权平均值来进行插值，可使得边缘 & 色彩转换更加自然
		Filter: ebiten.FilterLinear,
	}
}
