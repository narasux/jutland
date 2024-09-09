package drawer

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
	shipImg "github.com/narasux/jutland/pkg/resources/images/ship"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// 绘制缩略地图
func (d *Drawer) drawAbbreviationMap(screen *ebiten.Image, ms *state.MissionState) {
	// 绘制左/右背景 + 居中展示缩略地图
	d.drawAbbrMapAndBackground(screen, ms)
	// 舰队概览
	d.drawAbbrFleetOverview(screen, ms)
	// 绘制当前视野范围
	d.drawAbbrCameraBox(screen, ms)
	// 绘制建筑物
	d.drawAbbrBuildings(screen, ms)
	// 绘制敌我战舰
	d.drawAbbrShips(screen, ms)
}

// 绘制左右背景 + 居中展示缩略地图
func (d *Drawer) drawAbbrMapAndBackground(screen *ebiten.Image, ms *state.MissionState) {
	window := bgImg.MissionWindow
	windowWidth, windowHeight := window.Bounds().Dx(), window.Bounds().Dy()
	abbrMapWidth, abbrMapHeight := d.abbrMap.Bounds().Dx(), d.abbrMap.Bounds().Dy()

	xOffset := float64(ms.Layout.Width-abbrMapWidth) / 2

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(xOffset/float64(windowWidth), float64(ms.Layout.Height)/float64(windowHeight))
	screen.DrawImage(window, opts)

	opts.GeoM.Translate(xOffset+float64(abbrMapWidth), 0)
	screen.DrawImage(window, opts)

	opts = d.genDefaultDrawImageOptions()
	opts.GeoM.Translate(xOffset, 0)
	screen.DrawImage(d.abbrMap, opts)
	// 缩略地图添加银色边框
	strokeWidth := float32(4)
	vector.StrokeRect(
		screen,
		float32(xOffset),
		strokeWidth,
		float32(abbrMapWidth),
		float32(abbrMapHeight)-2*strokeWidth,
		strokeWidth,
		colorx.Silver,
		false,
	)
}

func (d *Drawer) drawAbbrFleetOverview(screen *ebiten.Image, ms *state.MissionState) {
	abbrMapWidth, abbrMapHeight := d.abbrMap.Bounds().Dx(), d.abbrMap.Bounds().Dy()
	windowWidth := float64(ms.Layout.Width-abbrMapWidth) / 2

	drawShips := func(player faction.Player, xOffset, yOffset, rotation float64) {
		fleet := ms.Fleet(player)

		// 绘制敌我标识
		d.drawText(
			screen,
			fmt.Sprintf("%s（%d）", lo.Ternary(player == ms.CurPlayer, "我", "敌"), fleet.Total),
			xOffset-float64(windowWidth)/2+25, 20, 48, font.Hang, colorx.White,
		)

		for idx, cls := range fleet.Classes {
			if yOffset > float64(abbrMapHeight)-110 {
				d.drawText(
					screen,
					fmt.Sprintf("%d+ . . . . . .", len(fleet.Classes)-idx-1),
					xOffset-50, yOffset, 20, font.Hang, colorx.White,
				)
				break
			}

			opts := d.genDefaultDrawImageOptions()
			sImg := shipImg.GetTop(cls.Kind.Name, ms.GameOpts.Zoom)
			shipWidth, shipLength := sImg.Bounds().Dx(), sImg.Bounds().Dy()

			setOptsCenterRotation(opts, sImg, rotation)
			opts.GeoM.Translate(xOffset, yOffset-float64(shipLength-shipWidth)/2)
			screen.DrawImage(sImg, opts)
			yOffset += float64(shipWidth) + 15

			d.drawText(
				screen,
				fmt.Sprintf("%s x %d", cls.Kind.DisplayName, cls.Total),
				xOffset-50, yOffset, 20, font.Hang, colorx.White,
			)
			yOffset += 50
		}
	}

	// 己方舰队概览
	drawShips(ms.CurPlayer, float64(windowWidth)/2, 80, 90)
	// 敌方舰队概览
	drawShips(ms.CurEnemy, float64(abbrMapWidth)+float64(windowWidth)*1.5, 80, 270)
}

// 绘制当前视野范围
func (d *Drawer) drawAbbrCameraBox(screen *ebiten.Image, ms *state.MissionState) {
	abbrMapWidth, abbrMapHeight := d.abbrMap.Bounds().Dx(), d.abbrMap.Bounds().Dy()
	xOffset := float64(ms.Layout.Width-abbrMapWidth) / 2

	rx, ry := float32(ms.Camera.Pos.RX), float32(ms.Camera.Pos.RY)
	cameraWidth, cameraHeight := float32(ms.Camera.Width), float32(ms.Camera.Height)
	mapWidth, mapHeight := float32(ms.MissionMD.MapCfg.Width), float32(ms.MissionMD.MapCfg.Height)

	x1 := rx / mapWidth * float32(abbrMapWidth)
	y1 := ry / mapHeight * float32(abbrMapHeight)
	x2 := (rx + cameraWidth) / mapWidth * float32(abbrMapWidth)
	y2 := (ry + cameraHeight) / mapHeight * float32(abbrMapHeight)
	vector.StrokeRect(screen, x1+float32(xOffset), y1, x2-x1, y2-y1, 2, colorx.White, false)
}

// 绘制建筑物
func (d *Drawer) drawAbbrBuildings(screen *ebiten.Image, ms *state.MissionState) {
	abbrMapWidth, abbrMapHeight := d.abbrMap.Bounds().Dx(), d.abbrMap.Bounds().Dy()
	xOffset := float64(ms.Layout.Width-abbrMapWidth) / 2

	for _, rp := range ms.ReinforcePoints {
		img := lo.Ternary(
			rp.BelongPlayer == ms.CurPlayer,
			textureImg.AbbrReinforcePoint,
			textureImg.AbbrEnemyReinforcePoint,
		)
		opts := d.genDefaultDrawImageOptions()
		setOptsCenterRotation(opts, img, rp.Rotation)

		xIndex := rp.Pos.RX / float64(ms.MissionMD.MapCfg.Width) * float64(abbrMapWidth)
		yIndex := rp.Pos.RY / float64(ms.MissionMD.MapCfg.Height) * float64(abbrMapHeight)

		opts.GeoM.Translate(xIndex+xOffset, yIndex)
		screen.DrawImage(img, opts)
	}
}

// 绘制敌我战舰
func (d *Drawer) drawAbbrShips(screen *ebiten.Image, ms *state.MissionState) {
	abbrMapWidth, abbrMapHeight := d.abbrMap.Bounds().Dx(), d.abbrMap.Bounds().Dy()
	xOffset := float64(ms.Layout.Width-abbrMapWidth) / 2

	for _, s := range ms.Ships {
		sImg := textureImg.GetAbbrShip(s.Tonnage, s.BelongPlayer != ms.CurPlayer)
		opts := d.genDefaultDrawImageOptions()
		setOptsCenterRotation(opts, sImg, s.CurRotation)

		xIndex := s.CurPos.RX / float64(ms.MissionMD.MapCfg.Width) * float64(abbrMapWidth)
		yIndex := s.CurPos.RY / float64(ms.MissionMD.MapCfg.Height) * float64(abbrMapHeight)

		opts.GeoM.Translate(xIndex+xOffset, yIndex)
		screen.DrawImage(sImg, opts)
	}
}
