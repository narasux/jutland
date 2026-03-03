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

// bombDiameters 支持的炸弹口径列表
var bombDiameters = []int{457, 380, 356, 280, 273, 160, 70}

// bombs 炸弹图片映射表
var bombs = make(map[int]*ebiten.Image)

// init 预加载炮弹和鱼雷图片资源
func init() {
	// FIXME 后续支持加载 1， 1/2， 1/4 三种尺寸
	// 加载炮弹图片（图片尺寸是实际显示的4倍，加载时按1/4缩放）
	for _, diameter := range shellDiameters {
		path := fmt.Sprintf("/bullets/shells/%d.png", diameter)
		img, err := loader.LoadImage(path)
		if err != nil {
			log.Fatalf("missing %s: %s", path, err)
		}
		// 缩放到原图的 1/4
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		opts.GeoM.Scale(0.25, 0.25)
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
		// 缩放到原图的 1/4
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		opts.GeoM.Scale(0.25, 0.25)
		zoomImg := ebiten.NewImage(img.Bounds().Dx()/5, img.Bounds().Dy()/5)
		zoomImg.DrawImage(img, opts)
		torpedoes[diameter] = zoomImg
	}

	// 加载炸弹图片（图片尺寸是实际显示的4倍，加载时按1/4缩放）
	for _, diameter := range bombDiameters {
		path := fmt.Sprintf("/bullets/bombs/%d.png", diameter)
		img, err := loader.LoadImage(path)
		if err != nil {
			log.Fatalf("missing %s: %s", path, err)
		}
		// 缩放到原图的 1/4
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		opts.GeoM.Scale(0.25, 0.25)
		zoomImg := ebiten.NewImage(img.Bounds().Dx()/4, img.Bounds().Dy()/4)
		zoomImg.DrawImage(img, opts)
		bombs[diameter] = zoomImg
	}
}

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
	if img, ok := bombs[diameter]; ok {
		return img
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
