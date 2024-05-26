package bullet

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/resources/colorx"
)

// FIXME 补充火炮弹药图片素材
var (
	GB406BulletImg = ebiten.NewImage(4, 8)
	GB356BulletImg = ebiten.NewImage(4, 6)
	GB203BulletImg = ebiten.NewImage(3, 4)
	GB152BulletImg = ebiten.NewImage(2, 3)
	GB140BulletImg = ebiten.NewImage(2, 3)
	GB127BulletImg = ebiten.NewImage(2, 3)
)

// FIXME 补充鱼雷弹药图片素材
var (
	TB533BulletImg = ebiten.NewImage(3, 20)
	TB610BulletImg = ebiten.NewImage(4, 24)
)

var NotFountImg = ebiten.NewImage(10, 20)

func init() {
	GB406BulletImg.Fill(colorx.Silver)
	GB356BulletImg.Fill(colorx.Gold)
	GB203BulletImg.Fill(colorx.Gold)
	GB152BulletImg.Fill(colorx.Gold)
	GB140BulletImg.Fill(colorx.Silver)
	GB127BulletImg.Fill(colorx.Silver)

	TB533BulletImg.Fill(colorx.DarkSilver)
	TB610BulletImg.Fill(colorx.Silver)

	NotFountImg.Fill(colorx.Green)
}

// GetImg 获取弹药图片
func GetImg(name string) *ebiten.Image {
	// FIXME 更合理的方式获取
	name = name[3:8]

	switch name {
	case "GB406":
		return GB406BulletImg
	case "GB356":
		return GB356BulletImg
	case "GB203":
		return GB203BulletImg
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
