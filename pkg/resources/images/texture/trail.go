package texture

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/narasux/jutland/pkg/resources/colorx"
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

// GetTrailImg 获取尾流图片
func GetTrailImg(size float64, life int) *ebiten.Image {
	if size < 0 || life < 0 {
		return InvalidTrailImg
	}

	key := fmt.Sprintf("%.2f:%d", size, life)
	// 尝试从缓存取
	if img, ok := trailImgCache[key]; ok {
		return img
	}

	// 缓存取不到，则重新生成
	sideLength := int(math.Floor(size * 2))
	trailImg := ebiten.NewImage(sideLength, sideLength)
	clr := color.NRGBA{255, 255, 255, uint8(life)}
	vector.DrawFilledCircle(trailImg, float32(size), float32(size), float32(size), clr, false)

	// 加入到缓存
	trailImgCache[key] = trailImg
	return trailImg
}
