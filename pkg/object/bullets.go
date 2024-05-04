package object

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mohae/deepcopy"
)

type BulletName string

const (
	// 火炮弹药 130mm 类型一
	Gb127T1 BulletName = "Gb127T1"
)

var bullets = map[BulletName]*Bullet{
	Gb127T1: gb127T1,
}

// FIXME 补充火炮弹药图片素材
var bulletImages = map[BulletName]*ebiten.Image{
	Gb127T1: ebiten.NewImage(2, 4),
}

// NewBullets 新建弹药
func NewBullets(name BulletName, CurPosition, TargetPosition MapPos, Rotate int, Speed int) *Bullet {
	b := deepcopy.Copy(*bullets[name]).(Bullet)
	b.CurPosition = CurPosition
	b.TargetPosition = TargetPosition
	b.Rotate = Rotate
	b.Speed = Speed
	return &b
}

// GetBulletImg 获取弹药图片
func GetBulletImg(name BulletName) *ebiten.Image {
	return bulletImages[name]
}

// gb 表示是火炮弹药，tb 表示鱼雷弹药
var gb127T1 = &Bullet{Name: Gb127T1, Damage: 100}
