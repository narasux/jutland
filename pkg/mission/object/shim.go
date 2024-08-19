package object

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/resources/images/bullet"
)

// GetBulletImg 获取弹药图片
func GetBulletImg(btType BulletType, diameter int) *ebiten.Image {
	switch btType {
	case BulletTypeShell:
		return bullet.GetShellBulletImg(diameter)
	case BulletTypeTorpedo:
		return bullet.GetTorpedoBulletImg(diameter)
	}
	return bullet.NotFountImg
}

var BulletImgWidthMap = map[string]int{}

// GetImgWidth 获取弹药图片宽度（虽然可能价值不大，总之先加一点缓存 :）
func GetImgWidth(btName string, btType BulletType, diameter int) int {
	if width, ok := BulletImgWidthMap[btName]; ok {
		return width
	}
	width := GetBulletImg(btType, diameter).Bounds().Dx()
	BulletImgWidthMap[btName] = width
	return width
}