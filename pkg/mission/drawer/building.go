package drawer

import (
	"fmt"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/common/constants"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
	buildingImg "github.com/narasux/jutland/pkg/resources/images/building"
	shipImg "github.com/narasux/jutland/pkg/resources/images/ship"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

type reinforceUILayout struct {
	Preview reinforceUIPanel
	Map     reinforceUIPanel
	Console reinforceUIPanel
	Info    reinforceUIPanel
	Queue   reinforceUIPanel
	Help    reinforceUIPanel
}

type reinforceUIPanel struct {
	X, Y, W, H float64
}

var (
	reinforcePanelFill    = color.RGBA{9, 24, 29, 212}
	reinforcePreviewFill  = color.RGBA{222, 218, 204, 238}
	reinforceMapFill      = color.RGBA{8, 22, 27, 172}
	reinforcePanelBorder  = color.RGBA{105, 124, 128, 198}
	reinforceText         = color.RGBA{226, 233, 231, 255}
	reinforceMutedText    = color.RGBA{172, 185, 186, 255}
	reinforceAccent       = color.RGBA{122, 165, 178, 255}
	reinforceAccentMuted  = color.RGBA{91, 123, 132, 255}
	reinforceCardFill     = color.RGBA{18, 38, 45, 210}
	reinforceCardActive   = color.RGBA{28, 57, 66, 228}
	reinforceProgressBase = color.RGBA{61, 77, 80, 255}
	reinforceProgressFill = color.RGBA{107, 151, 166, 255}
)

func (d *Drawer) drawBuildingsInCamera(screen *ebiten.Image, ms *state.MissionState) {
	// 增援点（只有在屏幕中的才渲染）
	for _, rp := range ms.ReinforcePoints {
		if !ms.Camera.Contains(rp.Pos) {
			continue
		}
		img := lo.Ternary(
			rp.BelongPlayer == ms.CurPlayer,
			buildingImg.ReinforcePoint,
			buildingImg.EnemyReinforcePoint,
		)
		opts := d.genDefaultDrawImageOptions()
		ebutil.SetOptsCenterRotation(opts, img, rp.Rotation)
		opts.GeoM.Translate(
			(rp.Pos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-float64(img.Bounds().Dx()/2),
			(rp.Pos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-float64(img.Bounds().Dy()/2),
		)
		screen.DrawImage(img, opts)

		if process := rp.Progress(); process > 0 {
			d.drawText(
				screen,
				strconv.Itoa(process),
				(rp.Pos.RX-ms.Camera.Pos.RX)*constants.MapBlockSize-10,
				(rp.Pos.RY-ms.Camera.Pos.RY)*constants.MapBlockSize-12,
				20,
				font.Hang,
				colorx.White,
			)
		}
	}
	// 油井
	for _, op := range ms.OilPlatforms {
		if !ms.Camera.Contains(op.Pos) {
			continue
		}
		img := buildingImg.OilPlatform
		x := (op.Pos.RX - ms.Camera.Pos.RX) * constants.MapBlockSize
		y := (op.Pos.RY - ms.Camera.Pos.RY) * constants.MapBlockSize
		opts := d.genDefaultDrawImageOptions()
		opts.GeoM.Translate(x-float64(img.Bounds().Dx()/2), y-float64(img.Bounds().Dy()/2))
		screen.DrawImage(img, opts)
		// 绘制范围圈
		vector.StrokeCircle(
			screen,
			float32(x),
			float32(y),
			float32(op.Radius*constants.MapBlockSize),
			2,
			colorx.Green,
			false,
		)
	}
}

func (d *Drawer) drawBuildingInterface(screen *ebiten.Image, ms *state.MissionState) {
	if ms.MissionStatus != state.MissionInBuilding {
		return
	}
	ui := calcReinforceUILayout(ms)
	d.drawBuildingBackground(screen, ms)
	d.drawReinforcePanel(screen, ui.Preview, "舰船预览", reinforcePreviewFill, true)
	d.drawReinforcePanel(screen, ui.Map, "战区地图", reinforceMapFill, false)
	d.drawReinforcePanel(screen, ui.Console, "补给控制台", reinforcePanelFill, false)
	d.drawAbbrMapInRPInterface(screen, ms, ui)
	d.drawSelectedProvidedShips(screen, ms, ui)
	d.drawSummonOperationTips(screen, ui)
}

func (d *Drawer) drawBuildingBackground(screen *ebiten.Image, ms *state.MissionState) {
	windowImg := bgImg.MissionWindow
	windowWidth, windowHeight := windowImg.Bounds().Dx(), windowImg.Bounds().Dy()

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(float64(ms.Layout.Width)/float64(windowWidth), float64(ms.Layout.Height)/float64(windowHeight))
	screen.DrawImage(windowImg, opts)
	vector.FillRect(
		screen, 0, 0, float32(ms.Layout.Width), float32(ms.Layout.Height),
		color.RGBA{3, 12, 16, 116}, false,
	)
}

// 在增援点界面画缩略地图
func (d *Drawer) drawAbbrMapInRPInterface(screen *ebiten.Image, ms *state.MissionState, ui reinforceUILayout) {
	abbrMapWidth, abbrMapHeight := d.abbrMap.Bounds().Dx(), d.abbrMap.Bounds().Dy()
	mapSize := min(ui.Map.W-32, ui.Map.H-64)
	xOffset, yOffset := ui.Map.X+(ui.Map.W-mapSize)/2, ui.Map.Y+48

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(
		mapSize/float64(abbrMapWidth),
		mapSize/float64(abbrMapHeight),
	)
	opts.GeoM.Translate(xOffset, yOffset)
	screen.DrawImage(d.abbrMap, opts)

	vector.StrokeRect(
		screen,
		float32(xOffset),
		float32(yOffset),
		float32(mapSize),
		float32(mapSize),
		2,
		reinforcePanelBorder,
		false,
	)

	// 把当前选中的增援点展示到地图上
	for _, rp := range ms.ReinforcePoints {
		// 只会画出属于己方的增援点
		if rp.BelongPlayer != ms.CurPlayer {
			continue
		}
		// 选中的是实心绿色，否则是空心绿色
		img := lo.Ternary(
			rp.Uid == ms.SelectedReinforcePointUid,
			textureImg.AbbrSelectedReinforcePoint,
			textureImg.AbbrReinforcePoint,
		)

		opts = d.genDefaultDrawImageOptions()
		ebutil.SetOptsCenterRotation(opts, img, rp.Rotation)

		xIndex := rp.Pos.RX / float64(ms.MissionMD.MapCfg.Width) * mapSize
		yIndex := rp.Pos.RY / float64(ms.MissionMD.MapCfg.Height) * mapSize

		opts.GeoM.Translate(xIndex+xOffset, yIndex+yOffset)
		screen.DrawImage(img, opts)
	}
}

func (d *Drawer) drawSelectedProvidedShips(screen *ebiten.Image, ms *state.MissionState, ui reinforceUILayout) {
	rp, ok := ms.ReinforcePoints[ms.SelectedReinforcePointUid]
	if !ok {
		return
	}

	selectedShipName := rp.ProvidedShipNames[rp.CurSelectedShipIndex]

	// 侧视图 & 俯视图
	zoom := lo.Ternary(ui.Preview.W > 900, 4, 2)
	sideImg := shipImg.GetSide(selectedShipName, zoom)
	sideImgDx, sideImgDy := sideImg.Bounds().Dx(), sideImg.Bounds().Dy()

	topImg := shipImg.GetTop(selectedShipName, zoom)
	// x, y 互换，因为需要顺时针旋转 90 度
	topImgDx, topImgDy := float64(topImg.Bounds().Dy()), float64(topImg.Bounds().Dx())

	contentW, contentH := ui.Preview.W*0.78, ui.Preview.H*0.72
	sideScale := min(1, min(contentW/float64(sideImgDx), contentH*0.35/float64(sideImgDy)))
	topScale := min(1, min(contentW/topImgDx, contentH*0.48/topImgDy))
	sideW, sideH := float64(sideImgDx)*sideScale, float64(sideImgDy)*sideScale
	topW, topH := topImgDx*topScale, topImgDy*topScale
	centerX := ui.Preview.X + ui.Preview.W/2
	sideY := ui.Preview.Y + ui.Preview.H*0.34
	topY := ui.Preview.Y + ui.Preview.H*0.68

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(sideScale, sideScale)
	opts.GeoM.Translate(centerX-sideW/2, sideY-sideH/2)
	screen.DrawImage(sideImg, opts)

	opts = d.genDefaultDrawImageOptions()
	ebutil.SetOptsCenterRotation(opts, topImg, 90)
	opts.GeoM.Scale(topScale, topScale)
	opts.GeoM.Translate(centerX-topW/2, topY-topH/2)
	opts.GeoM.Translate((topW-topH)/2, (topH-topW)/2)
	screen.DrawImage(topImg, opts)

	ship := objUnit.ShipMap[selectedShipName]
	titleFontSize := float64(34)
	if len(objUnit.GetShipDisplayName(selectedShipName)) > 6 {
		titleFontSize = 30
	}
	d.drawText(
		screen,
		fmt.Sprintf(
			"%s (%d/%d)",
			objUnit.GetShipDisplayName(selectedShipName),
			rp.CurSelectedShipIndex+1,
			len(rp.ProvidedShipNames),
		),
		ui.Info.X+24, ui.Info.Y+42, titleFontSize, font.Hang, reinforceText,
	)

	if ship != nil {
		stats := []string{
			fmt.Sprintf("%s / %s", ship.TypeAbbr, ship.Type),
			fmt.Sprintf("HP %.0f", ship.TotalHP),
			fmt.Sprintf("速度 %.1f 节", ship.MaxSpeed*600),
			fmt.Sprintf("费用 $%d / %ds", ship.FundsCost, ship.TimeCost),
		}
		for idx, stat := range stats {
			d.drawText(screen, stat, ui.Info.X+24, ui.Info.Y+86+float64(idx)*28, 18, font.Kai, reinforceMutedText)
		}
	}

	descY := ui.Info.Y + 200
	d.drawText(screen, "武装摘要", ui.Info.X+24, descY, 18, font.Kai, reinforceAccent)
	for idx, line := range objUnit.GetShipDesc(selectedShipName) {
		if idx >= 4 {
			break
		}
		d.drawText(screen, line, ui.Info.X+24, descY+34+float64(idx)*28, 18, font.Kai, reinforceText)
	}

	d.drawText(screen, fmt.Sprintf("当前资金：%d", ms.CurFunds), ui.Queue.X+24, ui.Queue.Y+42, 28, font.Hang, reinforceText)
	d.drawReinforceQueue(screen, rp.OncomingShips, ui.Queue)
}

func (d *Drawer) drawReinforceQueue(screen *ebiten.Image, ships []*objBuilding.OncomingShip, panel reinforceUIPanel) {
	d.drawText(screen, "增援队列", panel.X+24, panel.Y+90, 18, font.Kai, reinforceAccent)
	if len(ships) == 0 {
		d.drawText(screen, "待命", panel.X+24, panel.Y+134, 26, font.Hang, reinforceMutedText)
		return
	}

	maxItems := min(6, len(ships))
	cardW := (panel.W - 48 - 16) / 2
	cardH := 58.0
	for idx := 0; idx < maxItems; idx++ {
		col, row := idx%2, idx/2
		x := panel.X + 24 + float64(col)*(cardW+16)
		y := panel.Y + 126 + float64(row)*(cardH+14)
		d.drawQueueCard(screen, ships[idx], x, y, cardW, cardH, idx == 0)
	}
	if len(ships) > maxItems {
		d.drawText(screen, fmt.Sprintf("+%d", len(ships)-maxItems), panel.X+panel.W-56, panel.Y+panel.H-28, 18, font.JetbrainsMono, reinforceMutedText)
	}
}

func (d *Drawer) drawQueueCard(screen *ebiten.Image, ship *objBuilding.OncomingShip, x, y, w, h float64, active bool) {
	bgColor := reinforceCardFill
	borderColor := reinforcePanelBorder
	if active {
		bgColor = reinforceCardActive
		borderColor = reinforceAccentMuted
	}
	vector.FillRect(screen, float32(x), float32(y), float32(w), float32(h), bgColor, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 1.5, borderColor, false)
	d.drawText(screen, objUnit.GetShipDisplayName(ship.Name), x+12, y+12, 18, font.Kai, reinforceText)
	if active {
		progress := max(0, min(100, ship.Progress))
		barX, barY, barW := x+12, y+h-16, w-24
		vector.FillRect(screen, float32(barX), float32(barY), float32(barW), 5, reinforceProgressBase, false)
		vector.FillRect(screen, float32(barX), float32(barY), float32(barW*progress/100), 5, reinforceProgressFill, false)
		d.drawText(screen, fmt.Sprintf("%.0f%%", progress), x+w-58, y+12, 16, font.JetbrainsMono, reinforceAccent)
		return
	}
	d.drawText(screen, fmt.Sprintf("$%d / %ds", ship.FundsCost, ship.TimeCost), x+12, y+34, 15, font.JetbrainsMono, reinforceMutedText)
}

func (d *Drawer) drawSummonOperationTips(screen *ebiten.Image, ui reinforceUILayout) {
	tips := []string{
		"↑↓ 增援点",
		"←→ 舰船",
	}
	for idx, tip := range tips {
		x := ui.Help.X + 18
		y := ui.Help.Y + 44 + float64(idx)*30
		d.drawText(screen, tip, x, y, 17, font.OpenSans, reinforceMutedText)
	}
}

func calcReinforceUILayout(ms *state.MissionState) reinforceUILayout {
	w, h := float64(ms.Layout.Width), float64(ms.Layout.Height)
	margin, gap := 28.0, 18.0
	bottomMargin := 56.0
	topY := 48.0
	topH := h * 0.55
	consoleY := topY + topH + gap
	consoleH := h - consoleY - bottomMargin
	previewW := w*0.6 - margin - gap/2
	mapW := w - previewW - 2*margin - gap
	consoleW := w - 2*margin
	infoW := consoleW * 0.36
	queueW := consoleW * 0.42
	helpW := consoleW - infoW - queueW - 2*gap
	return reinforceUILayout{
		Preview: reinforceUIPanel{X: margin, Y: topY, W: previewW, H: topH},
		Map:     reinforceUIPanel{X: margin + previewW + gap, Y: topY, W: mapW, H: topH},
		Console: reinforceUIPanel{X: margin, Y: consoleY, W: consoleW, H: consoleH},
		Info:    reinforceUIPanel{X: margin, Y: consoleY, W: infoW, H: consoleH},
		Queue:   reinforceUIPanel{X: margin + infoW + gap, Y: consoleY, W: queueW, H: consoleH},
		Help:    reinforceUIPanel{X: margin + infoW + queueW + 2*gap, Y: consoleY, W: helpW, H: consoleH},
	}
}

func (d *Drawer) drawReinforcePanel(screen *ebiten.Image, panel reinforceUIPanel, title string, fill color.Color, textured bool) {
	vector.FillRect(
		screen, float32(panel.X), float32(panel.Y), float32(panel.W), float32(panel.H),
		fill, false,
	)
	if textured {
		texture := bgImg.MissionWindowParchment
		opts := d.genDefaultDrawImageOptions()
		opts.GeoM.Scale(panel.W/float64(texture.Bounds().Dx()), panel.H/float64(texture.Bounds().Dy()))
		opts.GeoM.Translate(panel.X, panel.Y)
		opts.ColorScale.ScaleAlpha(0.32)
		screen.DrawImage(texture, opts)
	}
	vector.StrokeRect(
		screen, float32(panel.X), float32(panel.Y), float32(panel.W), float32(panel.H),
		2, reinforcePanelBorder, false,
	)
	d.drawText(screen, title, panel.X+18, panel.Y+14, 16, font.OpenSans, reinforceAccent)
}
