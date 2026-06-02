package drawer

import (
	"fmt"
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/common/constants"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objRef "github.com/narasux/jutland/pkg/mission/object/reference"
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
	reinforcePanelFill    = color.RGBA{R: 9, G: 24, B: 29, A: 212}
	reinforcePreviewFill  = color.RGBA{R: 222, G: 218, B: 204, A: 238}
	reinforceMapFill      = color.RGBA{R: 8, G: 22, B: 27, A: 172}
	reinforcePanelBorder  = color.RGBA{R: 105, G: 124, B: 128, A: 198}
	reinforceText         = color.RGBA{R: 226, G: 233, B: 231, A: 255}
	reinforceMutedText    = color.RGBA{R: 172, G: 185, B: 186, A: 255}
	reinforceAccent       = color.RGBA{R: 122, G: 165, B: 178, A: 255}
	reinforceAccentMuted  = color.RGBA{R: 91, G: 123, B: 132, A: 255}
	reinforceCardFill     = color.RGBA{R: 18, G: 38, B: 45, A: 210}
	reinforceCardActive   = color.RGBA{R: 28, G: 57, B: 66, A: 228}
	reinforceProgressBase = color.RGBA{R: 61, G: 77, B: 80, A: 255}
	reinforceProgressFill = color.RGBA{R: 107, G: 151, B: 166, A: 255}
)

// drawBuildingsInCamera 绘制镜头范围内的建筑对象和建筑状态。
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

// drawBuildingInterface 绘制增援点交互界面。
func (d *Drawer) drawBuildingInterface(screen *ebiten.Image, ms *state.MissionState) {
	if ms.MissionStatus != state.MissionInBuilding {
		return
	}
	ui := calcReinforceUILayout(ms)
	d.drawBuildingBackground(screen, ms)
	d.drawReinforcePanel(screen, ui.Preview, reinforcePreviewFill, true)
	d.drawReinforcePanel(screen, ui.Map, reinforceMapFill, false)
	d.drawReinforcePanel(screen, ui.Console, reinforcePanelFill, false)
	d.drawAbbrMapInRPInterface(screen, ms, ui)
	d.drawSelectedProvidedShips(screen, ms, ui)
	d.drawSummonOperationTips(screen, ui, ms.CurFunds)
}

// drawBuildingBackground 绘制增援界面的底图和暗色遮罩。
func (d *Drawer) drawBuildingBackground(screen *ebiten.Image, ms *state.MissionState) {
	windowImg := bgImg.MissionWindow
	windowWidth, windowHeight := windowImg.Bounds().Dx(), windowImg.Bounds().Dy()

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(float64(ms.Layout.Width)/float64(windowWidth), float64(ms.Layout.Height)/float64(windowHeight))
	screen.DrawImage(windowImg, opts)
	vector.FillRect(
		screen, 0, 0, float32(ms.Layout.Width), float32(ms.Layout.Height),
		color.RGBA{R: 3, G: 12, B: 16, A: 116}, false,
	)
}

// 在增援点界面画缩略地图
func (d *Drawer) drawAbbrMapInRPInterface(screen *ebiten.Image, ms *state.MissionState, ui reinforceUILayout) {
	abbrMapWidth, abbrMapHeight := d.abbrMap.Bounds().Dx(), d.abbrMap.Bounds().Dy()
	xOffset, yOffset := ui.Map.X, ui.Map.Y
	mapW, mapH := ui.Map.W, ui.Map.H

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(
		mapW/float64(abbrMapWidth),
		mapH/float64(abbrMapHeight),
	)
	opts.GeoM.Translate(xOffset, yOffset)
	screen.DrawImage(d.abbrMap, opts)
	vector.StrokeRect(
		screen,
		float32(xOffset),
		float32(yOffset),
		float32(mapW),
		float32(mapH),
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

		xIndex := rp.Pos.RX / float64(ms.MissionMD.MapCfg.Width) * mapW
		yIndex := rp.Pos.RY / float64(ms.MissionMD.MapCfg.Height) * mapH

		opts.GeoM.Translate(xIndex+xOffset, yIndex+yOffset)
		screen.DrawImage(img, opts)
	}

	// 绘制当前选中增援点的集结点标记
	if rp, ok := ms.ReinforcePoints[ms.SelectedReinforcePointUid]; ok && rp.BelongPlayer == ms.CurPlayer {
		rallyX := float64(rp.RallyPos.MX)/float64(ms.MissionMD.MapCfg.Width)*mapW + xOffset
		rallyY := float64(rp.RallyPos.MY)/float64(ms.MissionMD.MapCfg.Height)*mapH + yOffset
		ebutil.DrawCrossMarker(screen, rallyX, rallyY, 6, 2, colorx.Green)
		// 集结点陆地失败提示
		if ms.RallySetFailedTick > 0 {
			d.drawText(screen, "陆地不可设集结点", rallyX-44, rallyY-18, 14, font.Kai, colorx.Red)
		}
	}
}

// drawSelectedProvidedShips 绘制当前可召唤舰船的预览图和信息面板。
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
	d.drawReinforceShipInfo(
		screen,
		selectedShipName,
		rp.CurSelectedShipIndex+1,
		len(rp.ProvidedShipNames),
		ship,
		ui.Info,
	)
	d.drawReinforceQueue(screen, rp.OncomingShips, ui.Queue)
}

// drawReinforceShipInfo 绘制舰船档案和武装配置两张信息卡。
func (d *Drawer) drawReinforceShipInfo(
	screen *ebiten.Image,
	selectedShipName string,
	shipIndex int,
	shipCount int,
	ship *objUnit.BattleShip,
	panel reinforceUIPanel,
) {
	cardInsetX := 16.0
	cardInsetY := 18.0
	cardGap := 36.0
	cardY := panel.Y + cardInsetY
	cardH := panel.H - cardInsetY*2
	archiveW := panel.W * 0.43
	armamentW := panel.W - archiveW - cardGap
	archiveCard := reinforceUIPanel{X: panel.X + cardInsetX, Y: cardY, W: archiveW - cardInsetX, H: cardH}
	armamentCard := reinforceUIPanel{
		X: archiveCard.X + archiveCard.W + cardGap,
		Y: cardY,
		W: armamentW - cardInsetX,
		H: cardH,
	}

	d.drawReinforceInfoCard(screen, archiveCard, "舰船档案")
	d.drawReinforceInfoCard(screen, armamentCard, "武装配置")

	indexText := fmt.Sprintf("%d/%d", shipIndex, shipCount)
	d.drawText(
		screen,
		indexText,
		archiveCard.X+archiveCard.W-76,
		archiveCard.Y+18,
		18,
		font.JetbrainsMono,
		reinforceMutedText,
	)

	titleFontSize := float64(34)
	if len(objUnit.GetShipDisplayName(selectedShipName)) > 6 {
		titleFontSize = 30
	}
	d.drawText(
		screen,
		objUnit.GetShipDisplayName(selectedShipName),
		archiveCard.X+24,
		archiveCard.Y+58,
		titleFontSize,
		font.Hang,
		reinforceText,
	)

	if ship != nil {
		items := []struct {
			label string
			value string
		}{
			{label: "类型", value: fmt.Sprintf("%s / %s", ship.TypeAbbr, ship.Type.ToDisplay())},
			{label: "HP", value: fmt.Sprintf("%.0f", ship.TotalHP)},
			{label: "速度", value: fmt.Sprintf("%.1f 节", ship.MaxSpeed*600)},
			{label: "费用", value: fmt.Sprintf("$%d / %ds", ship.FundsCost, ship.TimeCost)},
		}
		for idx, item := range items {
			lineY := archiveCard.Y + 112 + float64(idx)*32
			d.drawText(screen, item.label, archiveCard.X+24, lineY, 20, font.Kai, reinforceAccent)
			d.drawText(screen, item.value, archiveCard.X+92, lineY, 20, font.Kai, reinforceText)
		}
	}

	drawn := 0
	if ref := objRef.GetReference(selectedShipName); ref != nil {
		for _, armament := range ref.Armaments {
			if drawn >= 6 {
				break
			}
			line := fmt.Sprintf("%s：%s", armament.Label, armament.Value)
			d.drawText(
				screen,
				line,
				armamentCard.X+24,
				armamentCard.Y+58+float64(drawn)*32,
				20,
				font.Kai,
				reinforceText,
			)
			drawn++
		}
	}
}

// drawReinforceInfoCard 绘制增援控制台内统一样式的信息卡。
func (d *Drawer) drawReinforceInfoCard(screen *ebiten.Image, panel reinforceUIPanel, title string) {
	vector.FillRect(
		screen,
		float32(panel.X),
		float32(panel.Y),
		float32(panel.W),
		float32(panel.H),
		color.RGBA{R: 12, G: 28, B: 34, A: 178},
		false,
	)
	vector.StrokeRect(
		screen,
		float32(panel.X),
		float32(panel.Y),
		float32(panel.W),
		float32(panel.H),
		1.5,
		reinforcePanelBorder,
		false,
	)
	d.drawText(screen, title, panel.X+20, panel.Y+16, 20, font.Kai, reinforceAccent)
	vector.StrokeLine(
		screen,
		float32(panel.X+20), float32(panel.Y+44),
		float32(panel.X+panel.W-20), float32(panel.Y+44),
		1,
		reinforceAccentMuted,
		false,
	)
}

// drawReinforceQueue 绘制当前增援队列和队列中的舰船卡片。
func (d *Drawer) drawReinforceQueue(screen *ebiten.Image, ships []*objBuilding.OncomingShip, panel reinforceUIPanel) {
	cardInsetX := 16.0
	cardInsetY := 18.0
	card := reinforceUIPanel{
		X: panel.X + cardInsetX,
		Y: panel.Y + cardInsetY,
		W: panel.W - cardInsetX*2,
		H: panel.H - cardInsetY*2,
	}
	d.drawReinforceInfoCard(screen, card, "增援队列")

	if len(ships) == 0 {
		d.drawText(screen, "待命", card.X+24, card.Y+64, 24, font.Kai, reinforceMutedText)
		return
	}

	maxItems := calcReinforceQueueMaxItems(card.H, len(ships))
	cardGap := 16.0
	cardW := (card.W - 48 - cardGap) / 2
	cardH := 58.0
	for idx := 0; idx < maxItems; idx++ {
		col, row := idx%2, idx/2
		x := card.X + 24 + float64(col)*(cardW+cardGap)
		y := card.Y + 62 + float64(row)*(cardH+14)
		d.drawQueueCard(screen, ships[idx], x, y, cardW, cardH, idx == 0)
	}
	if len(ships) > maxItems {
		d.drawText(
			screen,
			fmt.Sprintf("+%d", len(ships)-maxItems),
			card.X+card.W-42,
			card.Y+card.H-24,
			18,
			font.JetbrainsMono,
			reinforceMutedText,
		)
	}
}

// drawQueueCard 绘制单个队列舰船的建造进度或费用信息。
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
		vector.FillRect(
			screen,
			float32(barX),
			float32(barY),
			float32(barW*progress/100),
			5,
			reinforceProgressFill,
			false,
		)
		d.drawText(screen, fmt.Sprintf("%.0f%%", progress), x+w-58, y+12, 16, font.JetbrainsMono, reinforceAccent)
		return
	}
	d.drawText(
		screen,
		fmt.Sprintf("$%d / %ds", ship.FundsCost, ship.TimeCost),
		x+12,
		y+34,
		15,
		font.JetbrainsMono,
		reinforceMutedText,
	)
}

// calcReinforceQueueMaxItems 根据队列面板可用高度动态计算最多可展示的卡片数量。
func calcReinforceQueueMaxItems(panelH float64, count int) int {
	titleAndDivider := 62.0
	bottomReserve := 36.0
	rowH := 58.0 + 14.0
	maxRows := max(1, int((panelH-titleAndDivider-bottomReserve)/rowH))
	return min(2*maxRows, count)
}

// drawSummonOperationTips 绘制当前资金和增援操作提示。
func (d *Drawer) drawSummonOperationTips(screen *ebiten.Image, ui reinforceUILayout, curFunds int64) {
	cardInsetX := 16.0
	cardInsetY := 18.0
	card := reinforceUIPanel{
		X: ui.Help.X + cardInsetX,
		Y: ui.Help.Y + cardInsetY,
		W: ui.Help.W - cardInsetX*2,
		H: ui.Help.H - cardInsetY*2,
	}
	d.drawReinforceInfoCard(screen, card, "控制状态")

	d.drawText(screen, "当前资金", card.X+20, card.Y+58, 17, font.Kai, reinforceAccent)
	d.drawText(screen, fmt.Sprintf("%d", curFunds), card.X+20, card.Y+84, 24, font.Kai, reinforceText)

	dividerY := card.Y + 124
	vector.StrokeLine(
		screen,
		float32(card.X+20), float32(dividerY),
		float32(card.X+card.W-20), float32(dividerY),
		1,
		reinforceAccentMuted,
		false,
	)

	tips := []string{
		"↑ ↓ 增援点",
		"← → 舰船",
		"Enter 召唤",
		"退格 取消增援",
		"点击地图 设集结点",
	}
	for idx, tip := range tips {
		x := card.X + 20
		y := dividerY + 28 + float64(idx)*30
		d.drawText(screen, tip, x, y, 18, font.Kai, reinforceMutedText)
	}
}

// calcReinforceUILayout 计算增援界面中预览、地图和控制台区域布局。
func calcReinforceUILayout(ms *state.MissionState) reinforceUILayout {
	w, h := float64(ms.Layout.Width), float64(ms.Layout.Height)
	margin, topGap, consoleGap := 18.0, 18.0, 4.0
	bottomMargin := 56.0
	topY := 48.0
	topH := h * 0.55
	consoleY := topY + topH + topGap
	consoleH := h - consoleY - bottomMargin
	previewW := w*0.6 - margin - topGap/2
	mapW := w - previewW - 2*margin - topGap
	consoleW := w - 2*margin
	helpW := max(240.0, consoleW*0.16)
	queueW := consoleW * 0.32
	infoW := consoleW - queueW - helpW - 2*consoleGap
	return reinforceUILayout{
		Preview: reinforceUIPanel{X: margin, Y: topY, W: previewW, H: topH},
		Map:     reinforceUIPanel{X: margin + previewW + topGap, Y: topY, W: mapW, H: topH},
		Console: reinforceUIPanel{X: margin, Y: consoleY, W: consoleW, H: consoleH},
		Info:    reinforceUIPanel{X: margin, Y: consoleY, W: infoW, H: consoleH},
		Queue:   reinforceUIPanel{X: margin + infoW + consoleGap, Y: consoleY, W: queueW, H: consoleH},
		Help:    reinforceUIPanel{X: margin + infoW + queueW + 2*consoleGap, Y: consoleY, W: helpW, H: consoleH},
	}
}

// drawReinforcePanel 绘制增援界面外层区域底色和边框。
func (d *Drawer) drawReinforcePanel(screen *ebiten.Image, panel reinforceUIPanel, fill color.Color, textured bool) {
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
}

// drawRallyLine 绘制增援点到集结点的虚线
func (d *Drawer) drawRallyLine(screen *ebiten.Image, ms *state.MissionState) {
	if ms.ShowRallyLinePointUid == "" {
		return
	}
	rp, ok := ms.ReinforcePoints[ms.ShowRallyLinePointUid]
	if !ok || rp.BelongPlayer != ms.CurPlayer {
		return
	}

	// 将地图坐标转换为屏幕坐标
	startX := (rp.Pos.RX - ms.Camera.Pos.RX) * constants.MapBlockSize
	startY := (rp.Pos.RY - ms.Camera.Pos.RY) * constants.MapBlockSize
	endX := (rp.RallyPos.RX - ms.Camera.Pos.RX) * constants.MapBlockSize
	endY := (rp.RallyPos.RY - ms.Camera.Pos.RY) * constants.MapBlockSize

	const (
		rallyFlagPoleHeight = 24.0
		rallyLineEndGap     = 8.0
	)
	startGap := float64(max(buildingImg.ReinforcePoint.Bounds().Dx(), buildingImg.ReinforcePoint.Bounds().Dy()))/2 + 8
	lineStartX, lineStartY, lineEndX, lineEndY, ok := ebutil.TrimLineSegment(
		startX, startY, endX, endY, startGap, rallyLineEndGap,
	)
	if !ok {
		ebutil.DrawFlagMarker(screen, endX, endY, rallyFlagPoleHeight, colorx.Green)
		return
	}

	const dashLen = 8.0
	const gapLen = 6.0

	// 用分段线段绘制虚线
	dx := lineEndX - lineStartX
	dy := lineEndY - lineStartY
	totalDist := math.Sqrt(dx*dx + dy*dy)
	ux, uy := dx/totalDist, dy/totalDist
	progress := 0.0
	drawing := true
	lineColor := color.RGBA{R: 72, G: 206, B: 128, A: 180}

	for progress < totalDist {
		seg := dashLen
		if !drawing {
			seg = gapLen
		}
		if progress+seg > totalDist {
			seg = totalDist - progress
		}
		if drawing {
			segStartX := lineStartX + ux*progress
			segStartY := lineStartY + uy*progress
			segEndX := segStartX + ux*seg
			segEndY := segStartY + uy*seg
			vector.StrokeLine(
				screen,
				float32(segStartX), float32(segStartY),
				float32(segEndX), float32(segEndY),
				2, lineColor, false,
			)
		}
		progress += seg
		drawing = !drawing
	}

	// 在集结点位置绘制旗帜标记
	ebutil.DrawFlagMarker(screen, endX, endY, rallyFlagPoleHeight, colorx.Green)
}
