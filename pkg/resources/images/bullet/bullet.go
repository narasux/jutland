package bullet

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/resources/colorx"
)

// FIXME 补充火炮弹药图片素材
var GB406BulletImg = ebiten.NewImage(4, 8)

var (
	GB356BulletImg = ebiten.NewImage(4, 6)
	GB152BulletImg = ebiten.NewImage(2, 3)
	GB127BulletImg = ebiten.NewImage(2, 3)
)

func init() {
	GB406BulletImg.Fill(colorx.Silver)
	GB356BulletImg.Fill(colorx.Gold)
	GB152BulletImg.Fill(colorx.Gold)
	GB127BulletImg.Fill(colorx.Silver)
}

// GetImg 获取弹药图片
func GetImg(name string) *ebiten.Image {
	// FIXME 更合理的方式获取
	name = name[:5]

	switch name {
	case "GB406":
		return GB406BulletImg
	case "GB356":
		return GB356BulletImg
	case "GB152":
		return GB152BulletImg
	case "GB127":
		return GB127BulletImg
	}
	return GB127BulletImg
}
