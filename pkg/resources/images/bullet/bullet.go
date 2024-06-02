package bullet

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

// FIXME 补充火炮弹药图片素材
var (
	GB460BulletImg = ebutil.NewImageWithColor(4, 9, colorx.Gold)
	GB406BulletImg = ebutil.NewImageWithColor(4, 8, colorx.Silver)
	GB356BulletImg = ebutil.NewImageWithColor(4, 6, colorx.Gold)
	GB305BulletImg = ebutil.NewImageWithColor(3, 5, colorx.Silver)
	GB203BulletImg = ebutil.NewImageWithColor(3, 5, colorx.Gold)
	GB155BulletImg = ebutil.NewImageWithColor(2, 4, colorx.Gold)
	GB152BulletImg = ebutil.NewImageWithColor(2, 4, colorx.Gold)
	GB140BulletImg = ebutil.NewImageWithColor(2, 3, colorx.Silver)
	GB127BulletImg = ebutil.NewImageWithColor(2, 3, colorx.Silver)
)

// FIXME 补充鱼雷弹药图片素材
var (
	TB533BulletImg = ebutil.NewImageWithColor(3, 20, colorx.DarkSilver)
	TB610BulletImg = ebutil.NewImageWithColor(4, 24, colorx.Silver)
)

var NotFountImg = ebutil.NewImageWithColor(10, 20, colorx.Red)

// GetImg 获取弹药图片
func GetImg(name string) *ebiten.Image {
	// FIXME 更合理的方式获取
	name = name[3:9]

	switch name {
	case "GB/460":
		return GB460BulletImg
	case "GB/406":
		return GB406BulletImg
	case "GB/356":
		return GB356BulletImg
	case "GB/305":
		return GB305BulletImg
	case "GB/203":
		return GB203BulletImg
	case "GB/155":
		return GB155BulletImg
	case "GB/152":
		return GB152BulletImg
	case "GB/140":
		return GB140BulletImg
	case "GB/127":
		return GB127BulletImg
	case "TB/533":
		return TB533BulletImg
	case "TB/610":
		return TB610BulletImg
	}
	// 找不到就暴露出来
	return NotFountImg
}

var BulletImgWidthMap = map[string]int{}

// GetImgWidth 获取弹药图片宽度（虽然可能价值不大，总之先加一点缓存 :）
func GetImgWidth(name string) int {
	if width, ok := BulletImgWidthMap[name]; ok {
		return width
	}
	width := GetImg(name).Bounds().Dx()
	BulletImgWidthMap[name] = width
	return width
}
