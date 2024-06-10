package bullet

import (
	"github.com/hajimehoshi/ebiten/v2"

	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/ebutil"
)

// FIXME 补充火炮弹药图片素材
var (
	GB1024BulletImg = ebutil.NewImageWithColor(6, 14, colorx.DarkRed)
	GB460BulletImg  = ebutil.NewImageWithColor(4, 9, colorx.Gold)
	GB406BulletImg  = ebutil.NewImageWithColor(4, 8, colorx.Silver)
	GB356BulletImg  = ebutil.NewImageWithColor(4, 6, colorx.Gold)
	GB305BulletImg  = ebutil.NewImageWithColor(3, 5, colorx.Silver)
	GB203BulletImg  = ebutil.NewImageWithColor(3, 5, colorx.Gold)
	GB155BulletImg  = ebutil.NewImageWithColor(2, 4, colorx.Gold)
	GB152BulletImg  = ebutil.NewImageWithColor(2, 4, colorx.Gold)
	GB140BulletImg  = ebutil.NewImageWithColor(2, 3, colorx.Silver)
	GB127BulletImg  = ebutil.NewImageWithColor(2, 3, colorx.Silver)
)

// FIXME 补充鱼雷弹药图片素材
var (
	TB533BulletImg = ebutil.NewImageWithColor(3, 20, colorx.DarkSilver)
	TB610BulletImg = ebutil.NewImageWithColor(4, 24, colorx.Silver)
)

var NotFountImg = ebutil.NewImageWithColor(10, 20, colorx.Red)

// GetImg 获取弹药图片
func GetImg(btType obj.BulletType, diameter int) *ebiten.Image {
	if btType == obj.BulletTypeShell {
		switch diameter {
		case 1024:
			return GB1024BulletImg
		case 460:
			return GB460BulletImg
		case 406:
			return GB406BulletImg
		case 356:
			return GB356BulletImg
		case 305:
			return GB305BulletImg
		case 203:
			return GB203BulletImg
		case 155:
			return GB155BulletImg
		case 152:
			return GB152BulletImg
		case 140:
			return GB140BulletImg
		case 127:
			return GB127BulletImg
		}
	} else if btType == obj.BulletTypeTorpedo {
		switch diameter {
		case 533:
			return TB533BulletImg
		case 610:
			return TB610BulletImg
		}
	}

	// 找不到就暴露出来
	return NotFountImg
}

var BulletImgWidthMap = map[string]int{}

// GetImgWidth 获取弹药图片宽度（虽然可能价值不大，总之先加一点缓存 :）
func GetImgWidth(btName string, btType obj.BulletType, diameter int) int {
	if width, ok := BulletImgWidthMap[btName]; ok {
		return width
	}
	width := GetImg(btType, diameter).Bounds().Dx()
	BulletImgWidthMap[btName] = width
	return width
}
