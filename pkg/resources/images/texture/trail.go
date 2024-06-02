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
var InvalidTrailImg = ebutil.NewImageWithColor(25, 25, colorx.Red)

type TrailShape int

const (
	// 圆形尾流
	TrailShapeCircle TrailShape = iota
	// 矩形尾流（长度 = 宽度 15 倍，折半）
	TrailShapeRect
)

// GetTrailImg 获取尾流图片
func GetTrailImg(shape TrailShape, size, life float64) *ebiten.Image {
	if size < 0 || life < 0 {
		return InvalidTrailImg
	}

	key := fmt.Sprintf("%d:%.2f:%.2f", shape, size, life)
	// 尝试从缓存取
	if img, ok := trailImgCache[key]; ok {
		return img
	}

	// 缓存取不到，则重新生成并且加入到缓存
	if shape == TrailShapeCircle {
		trailImgCache[key] = getCircleImg(size, life)
	} else if shape == TrailShapeRect {
		trailImgCache[key] = getHalfRectImg(size, 15, life)
	}

	return trailImgCache[key]
}

func getCircleImg(diameter, life float64) *ebiten.Image {
	radius := float32(diameter / 2)
	trailImg := ebiten.NewImage(int(diameter), int(diameter))
	clr := color.NRGBA{255, 255, 255, uint8(life)}
	vector.DrawFilledCircle(trailImg, radius, radius, radius, clr, false)
	return trailImg
}

func getHalfRectImg(width, multipleWidthAsHeight, life float64) *ebiten.Image {
	height := int(width * multipleWidthAsHeight)
	trailImg := ebiten.NewImage(int(width), height*2)
	clr := color.NRGBA{255, 255, 255, uint8(life)}
	vector.DrawFilledRect(trailImg, 0, float32(height), float32(width), float32(height), clr, false)
	return trailImg
}
