package bullet

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/resources/colorx"
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
	name = name[3:8]

	switch name {
	case "GB460":
		return GB460BulletImg
	case "GB406":
		return GB406BulletImg
	case "GB356":
		return GB356BulletImg
	case "GB305":
		return GB305BulletImg
	case "GB203":
		return GB203BulletImg
	case "GB155":
		return GB155BulletImg
	case "GB152":
		return GB152BulletImg
	case "GB140":
		return GB140BulletImg
	case "GB127":
		return GB127BulletImg
	case "TB533":
		return TB533BulletImg
	case "TB610":
		return TB610BulletImg
	}
	// 找不到就暴露出来
	return NotFountImg
}
