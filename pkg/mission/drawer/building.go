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

func (d *Drawer) drawShipDesignGraph(screen *ebiten.Image, ms *state.MissionState) {
	designGraphImg := bgImg.MissionWindowParchment
	designGraphImgWidth, designGraphImgHeight := designGraphImg.Bounds().Dx(), designGraphImg.Bounds().Dy()
	exceptedWidth, exceptedHeight := float64(ms.Layout.Width)/5*3, float64(ms.Layout.Height)/5*3

	xOffset, yOffset := float64(75), float64(75)
	opts := d.genDefaultDrawImageOptions()
	opts.GeoM.Scale(
		exceptedWidth/float64(designGraphImgWidth),
		exceptedHeight/float64(designGraphImgHeight),
	)
	opts.GeoM.Translate(xOffset, yOffset)
	screen.DrawImage(designGraphImg, opts)

	// 缩略地图添加银色边框
	strokeWidth := float32(5)
	vector.StrokeRect(
		screen,
		float32(xOffset),
		float32(yOffset)+strokeWidth,
		float32(exceptedWidth),
		float32(exceptedHeight)-2*strokeWidth,
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

	rfp, ok := ms.ReinforcePoints[ms.SelectedReinforcePointUid]
	if !ok {
		return
	}

	selectedShipName := rfp.ProvidedShipNames[rfp.CurSelectedShipIndex]
	xOffset, yOffset := float64(75), float64(ms.Layout.Height)/5*3+120

	text := fmt.Sprintf(
		"战舰：%s (%d/%d)",
		object.GetShipDisplayName(selectedShipName),
		rfp.CurSelectedShipIndex+1,
		len(rfp.ProvidedShipNames),
	)
	d.drawText(screen, text, xOffset, yOffset, 48, font.Hang, colorx.White)

	for _, line := range object.GetShipDesc(selectedShipName) {
		yOffset += 80
		d.drawText(screen, line, xOffset, yOffset, 32, font.Hang, colorx.White)
	}
}
