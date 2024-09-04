package drawer

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/font"
	bgImg "github.com/narasux/jutland/pkg/resources/images/background"
	buildingImg "github.com/narasux/jutland/pkg/resources/images/building"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
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
		setOptsCenterRotation(opts, img, rp.Rotation)
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
}

func (d *Drawer) drawBuildingInterface(screen *ebiten.Image, ms *state.MissionState) {
	if ms.MissionStatus != state.MissionInBuilding {
		return
	}
	d.drawBuildingBackground(screen, ms)
	d.drawAbbrMapInRPInterface(screen, ms)
	d.drawShipDesignGraph(screen, ms)
	d.drawProvidedShips(screen, ms)
}

func (d *Drawer) drawBuildingBackground(screen *ebiten.Image, ms *state.MissionState) {
	windowImg := bgImg.MissionWindow
	windowWidth, windowHeight := windowImg.Bounds().Dx(), windowImg.Bounds().Dy()

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(float64(ms.Layout.Width)/float64(windowWidth), float64(ms.Layout.Height)/float64(windowHeight))
	screen.DrawImage(windowImg, opts)
}

// 在增援点界面画缩略地图
func (d *Drawer) drawAbbrMapInRPInterface(screen *ebiten.Image, ms *state.MissionState) {
	abbrMapWidth, abbrMapHeight := d.abbrMap.Bounds().Dx(), d.abbrMap.Bounds().Dy()
	exceptedWidth, exceptedHeight := float64(ms.Layout.Height)/5*3, float64(ms.Layout.Height)/5*3

	padding := float64(ms.Layout.Width/5*2-ms.Layout.Height/5*3) / 3
	xOffset, yOffset := float64(ms.Layout.Width)/5*3+2*padding, float64(50)

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(
		exceptedWidth/float64(abbrMapWidth),
		exceptedHeight/float64(abbrMapHeight),
	)
	opts.GeoM.Translate(xOffset, yOffset)
	screen.DrawImage(d.abbrMap, opts)

	// 缩略地图添加银色边框
	strokeWidth := float32(5)
	vector.StrokeRect(
		screen,
		float32(xOffset),
		float32(yOffset),
		float32(exceptedWidth),
		float32(exceptedHeight),
		strokeWidth,
		colorx.Silver,
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
		setOptsCenterRotation(opts, img, rp.Rotation)

		xIndex := rp.Pos.RX / float64(ms.MissionMD.MapCfg.Width) * float64(abbrMapWidth)
		yIndex := rp.Pos.RY / float64(ms.MissionMD.MapCfg.Height) * float64(abbrMapHeight)

		opts.GeoM.Translate(xIndex+xOffset, yIndex+yOffset)
		screen.DrawImage(img, opts)
	}
}

func (d *Drawer) drawShipDesignGraph(screen *ebiten.Image, ms *state.MissionState) {
	designGraphImg := bgImg.MissionWindowParchment
	designGraphImgWidth, designGraphImgHeight := designGraphImg.Bounds().Dx(), designGraphImg.Bounds().Dy()
	exceptedWidth, exceptedHeight := float64(ms.Layout.Width)/5*3, float64(ms.Layout.Height)/5*3

	xOffset, yOffset := float64(ms.Layout.Width/5*2-ms.Layout.Height/5*3)/3, float64(50)

	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(
		exceptedWidth/float64(designGraphImgWidth),
		exceptedHeight/float64(designGraphImgHeight),
	)
	opts.GeoM.Translate(xOffset, yOffset)
	screen.DrawImage(designGraphImg, opts)

	// 设计图添加银色边框
	strokeWidth := float32(5)
	vector.StrokeRect(
		screen,
		float32(xOffset),
		float32(yOffset),
		float32(exceptedWidth),
		float32(exceptedHeight),
		strokeWidth,
		colorx.Silver,
		false,
	)
}

func (d *Drawer) drawProvidedShips(screen *ebiten.Image, ms *state.MissionState) {
	// FIXME 移除这段测试逻辑
	keys := lo.Keys(ms.ReinforcePoints)
	slices.Sort(keys)
	ms.SelectedReinforcePointUid = keys[0]

	rp, ok := ms.ReinforcePoints[ms.SelectedReinforcePointUid]
	if !ok {
		return
	}

	selectedShipName := rp.ProvidedShipNames[rp.CurSelectedShipIndex]
	xOffset, yOffset := float64(75), float64(ms.Layout.Height)/5*3+120

	text := fmt.Sprintf(
		"战舰：%s (%d/%d)",
		object.GetShipDisplayName(selectedShipName),
		rp.CurSelectedShipIndex+1,
		len(rp.ProvidedShipNames),
	)
	d.drawText(screen, text, xOffset, yOffset, 40, font.Hang, colorx.White)

	text = fmt.Sprintf("当前资金：%d", ms.CurFunds)
	d.drawText(screen, text, xOffset+1000, yOffset, 40, font.Hang, colorx.White)

	yOffset += 80
	for _, line := range object.GetShipDesc(selectedShipName) {
		d.drawText(screen, line, xOffset, yOffset, 24, font.Hang, colorx.White)
		yOffset += 60
	}
}
