package bullet

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/loader"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

// shellDiameters 支持的炮弹口径列表
var shellDiameters = []int{
	1024, 510, 500, 460, 457, 406, 381, 356, 343, 305, 283,
	203, 200, 180, 155, 152, 150, 140, 133, 130, 127, 120,
	114, 105, 102, 100, 88, 76, 57, 45, 40, 37, 28, 25, 20,
	13, 8,
}

// shells 炮弹图片映射表
var shells = make(map[int]*ebiten.Image)

// torpedoDiameters 支持的鱼雷口径列表
var torpedoDiameters = []int{324, 450, 533, 570, 610, 622, 1350}

// torpedoes 鱼雷图片映射表
var torpedoes = make(map[int]*ebiten.Image)

// init 预加载炮弹和鱼雷图片资源
func init() {
	// 加载炮弹图片（图片尺寸是实际显示的4倍，加载时按1/4缩放）
	for _, diameter := range shellDiameters {
		path := fmt.Sprintf("/bullets/shells/%d.png", diameter)
		img, err := loader.LoadImage(path)
		if err != nil {
			log.Fatalf("missing %s: %s", path, err)
		}
		// 缩放到原图的 1/4
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		opts.GeoM.Scale(1/4.0, 1/4.0)
		zoomImg := ebiten.NewImage(img.Bounds().Dx()/4, img.Bounds().Dy()/4)
		zoomImg.DrawImage(img, opts)
		shells[diameter] = zoomImg
	}

	// 加载鱼雷图片
	for _, diameter := range torpedoDiameters {
		path := fmt.Sprintf("/bullets/torpedoes/%d.png", diameter)
		img, err := loader.LoadImage(path)
		if err != nil {
			log.Fatalf("missing %s: %s", path, err)
		}
		// 缩放到原图的 1/5 FIXME 后续看下更合适的缩放方式，直接 1/5 好粗暴
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		opts.GeoM.Scale(1/5.0, 1/5.0)
		zoomImg := ebiten.NewImage(img.Bounds().Dx()/5, img.Bounds().Dy()/5)
		zoomImg.DrawImage(img, opts)
		torpedoes[diameter] = zoomImg
	}
}

var (
	bb70  = ebutil.NewImageWithColor(2, 4, colorx.Silver)
	bb160 = ebutil.NewImageWithColor(3, 5, colorx.Silver)
	bb273 = ebutil.NewImageWithColor(3, 7, colorx.Silver)
	bb280 = ebutil.NewImageWithColor(3, 8, colorx.Silver)
	bb356 = ebutil.NewImageWithColor(4, 9, colorx.Silver)
	bb380 = ebutil.NewImageWithColor(4, 10, colorx.Silver)
	bb457 = ebutil.NewImageWithColor(5, 12, colorx.Silver)
)

var laser50 = ebutil.NewImageWithColor(2, 1000, colorx.Green)

var NotFount = ebutil.NewImageWithColor(10, 20, colorx.Red)

// GetShell 获取炮弹弹药图片
func GetShell(diameter int) *ebiten.Image {
	if img, ok := shells[diameter]; ok {
		return img
	}
	// 找不到就暴露出来
	return NotFount
}

// GetTorpedo 获取鱼雷弹药图片
func GetTorpedo(diameter int) *ebiten.Image {
	if img, ok := torpedoes[diameter]; ok {
		return img
	}
	// 找不到就暴露出来
	return NotFount
}

// GetBomb 获取炸弹弹药图片
func GetBomb(diameter int) *ebiten.Image {
	switch diameter {
	case 70:
		return bb70
	case 160:
		return bb160
	case 273:
		return bb273
	case 280:
		return bb280
	case 356:
		return bb356
	case 380:
		return bb380
	case 457:
		return bb457
	}
	// 找不到就暴露出来
	return NotFount
}

// GetLaser 获取镭射弹药图片
func GetLaser(diameter int) *ebiten.Image {
	switch diameter {
	case 50:
		return laser50
	}
	// 找不到就暴露出来
	return NotFount
}
