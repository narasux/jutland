package bullet

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/resources/colorx"
)

// FIXME 补充火炮弹药图片素材
var DefaultBulletImg = ebiten.NewImage(2, 4)

func init() {
	DefaultBulletImg.Fill(colorx.White)
}

// GetImg 获取弹药图片
func GetImg(name string) *ebiten.Image {
	// FIXME 应该加载正确的图片
	return DefaultBulletImg
}
