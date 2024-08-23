package texture

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

/*
尾流是一个尺寸为 Size，透明度为 Life 的白色圆形，应该支持缓存而不是每次都重新生成

注：尝试过 golang-lru，比 Map 慢 5% 以内，但如果出现换页则性能直线下降
如果 LRU 还要避免换页，不如直接 Map，反正吃不了多少内存 :D
*/
var trailImgCache = map[string]*ebiten.Image{}

// 不合法的尾流应该直接暴露出来
var invalidTrail = ebutil.NewImageWithColor(25, 25, colorx.Red)

type TrailShape int

const (
	// 圆形尾流
	TrailShapeCircle TrailShape = iota
	// 矩形尾流（长度 = 宽度 15 倍，折半）
	TrailShapeRect
)

// GetTrail 获取尾流图片
func GetTrail(shape TrailShape, size, life float64, clr color.Color) *ebiten.Image {
	if size < 0 || life < 0 {
		return invalidTrail
	}

	key := fmt.Sprintf("%d:%d:%d", shape, int(size), int(life))
	if clr != nil {
		r, g, b, _ := clr.RGBA()
		key += fmt.Sprintf(":%d:%d:%d", r, g, b)
	}
	// 尝试从缓存取
	if img, ok := trailImgCache[key]; ok {
		return img
	}

	// 缓存取不到，则重新生成并且加入到缓存
	if shape == TrailShapeCircle {
		trailImgCache[key] = getCircle(size, life, clr)
	} else if shape == TrailShapeRect {
		trailImgCache[key] = getHalfRect(size, 15, life, clr)
	}

	return trailImgCache[key]
}

func getCircle(diameter, life float64, clr color.Color) *ebiten.Image {
	radius := float32(diameter / 2)
	trailImg := ebiten.NewImage(int(diameter), int(diameter))
	// 默认颜色
	if clr == nil {
		clr = colorx.White
	}
	// 基于基础颜色，添加透明度
	r, g, b, _ := clr.RGBA()
	clr = color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(life)}
	// 绘制圆形
	vector.DrawFilledCircle(trailImg, radius, radius, radius, clr, false)
	return trailImg
}

func getHalfRect(width, multipleWidthAsHeight, life float64, clr color.Color) *ebiten.Image {
	height := int(width * multipleWidthAsHeight)
	trailImg := ebiten.NewImage(int(width), height*2)
	// 默认颜色
	if clr == nil {
		clr = colorx.White
	}
	// 基于基础颜色，添加透明度
	r, g, b, _ := clr.RGBA()
	clr = color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(life)}
	// 绘制矩形
	vector.DrawFilledRect(trailImg, 0, float32(height), float32(width), float32(height), clr, false)
	return trailImg
}
