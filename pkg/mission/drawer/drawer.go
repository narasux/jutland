package drawer

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/controller/cheat"
	"github.com/narasux/jutland/pkg/mission/layout"
	md "github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/state"
	abbrMapImg "github.com/narasux/jutland/pkg/resources/images/abbrmap"
	mapBlockImg "github.com/narasux/jutland/pkg/resources/images/mapblock"
)

// Drawer 任务运行中图像绘制器
type Drawer struct {
	mission string
	abbrMap *ebiten.Image
}

// NewDrawer ...
func NewDrawer(mission string) *Drawer {
	missionMD := md.Get(mission)
	if err := mapBlockImg.SceneBlockCache.Init(missionMD.MapCfg); err != nil {
		log.Fatal("failed to load map scene blocks: ", err)
	}

	misLayout := layout.NewScreenLayout()
	abbrMap := ebiten.NewImage(misLayout.Height, misLayout.Height)

	bg := abbrMapImg.Background
	w, h := bg.Bounds().Dx(), bg.Bounds().Dy()

	scaleX := float64(misLayout.Height) / float64(w)
	scaleY := float64(misLayout.Height) / float64(h)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scaleX, scaleY)

	abbrMap.DrawImage(abbrMapImg.Background, opts)
	abbrMap.DrawImage(abbrMapImg.Get(missionMD.MapCfg.Name), opts)

	return &Drawer{mission: mission, abbrMap: abbrMap}
}

// Draw 绘制任务关卡图像
func (d *Drawer) Draw(
	screen *ebiten.Image,
	misState *state.MissionState,
	terminal *cheat.Terminal,
) {
	switch misState.MissionStatus {
	case state.MissionInTerminal:
		// 绘制终端模式，不需要绘制其他对象
		d.drawTerminal(screen, misState, terminal)
	case state.MissionInMap:
		// 全屏展示地图模式，不需要绘制地图外的对象
		d.drawAbbreviationMap(screen, misState)
	default:
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
}

func (d *Drawer) genDefaultDrawImageOptions() *ebiten.DrawImageOptions {
	return &ebiten.DrawImageOptions{
		// 线性过滤：通过计算周围像素的加权平均值来进行插值，可使得边缘 & 色彩转换更加自然
		Filter: ebiten.FilterLinear,
	}
}
