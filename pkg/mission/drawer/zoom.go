package drawer

import (
	"github.com/hajimehoshi/ebiten/v2"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/state"
)

// degToRad 将角度转换为弧度，供 Ebiten GeoM.Rotate 使用。
const degToRad = 0.017453292519943295

// drawImageAtScale 在指定屏幕左上角绘制缩放图片。
// 这里直接在 GeoM 中缩放并平移，供 HP、武器图标等非中心锚点元素复用。
func drawImageAtScale(screen *ebiten.Image, img *ebiten.Image, x, y, scale float64) {
	if img == nil {
		return
	}
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(x, y)
	screen.DrawImage(img, opts)
}

// drawImageCentered 以指定屏幕坐标为中心绘制缩放和旋转后的图片。
// 实现上先把图片原点移到中心，再旋转、缩放并移动到目标屏幕位置。
func drawImageCentered(
	screen *ebiten.Image, img *ebiten.Image, centerX, centerY, rotation, scale float64,
) {
	if img == nil {
		return
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	opts.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	opts.GeoM.Rotate(rotation * degToRad)
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(centerX, centerY)
	screen.DrawImage(img, opts)
}

// drawImageCenteredAtMapPos 以地图坐标为中心绘制图片。
// 它先通过任务视图换算得到屏幕坐标，再复用中心绘制逻辑。
func drawImageCenteredAtMapPos(
	screen *ebiten.Image, ms *state.MissionState, img *ebiten.Image, pos objPos.MapPos, rotation, scale float64,
) {
	x, y := ms.CameraPosToScreen(pos)
	drawImageCentered(screen, img, x, y, rotation, scale)
}
