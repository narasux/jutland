package object

import (
	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/utils/geometry"
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
func NewBullets(name BulletName, CurPos, TargetPos MapPos, Speed int) *Bullet {
	b := deepcopy.Copy(*bullets[name]).(Bullet)
	b.Uid = uuid.New().String()
	b.CurPosition = CurPos
	b.TargetPosition = TargetPos
	b.Rotation = int(geometry.CalcAngleBetweenPoints(CurPos.RX, CurPos.RY, TargetPos.RX, TargetPos.RY))
	b.Speed = Speed
	return &b
}

// GetBulletImg 获取弹药图片
func GetBulletImg(name BulletName) *ebiten.Image {
	return bulletImages[name]
}

// gb 表示是火炮弹药，tb 表示鱼雷弹药
var gb127T1 = &Bullet{Name: Gb127T1, Damage: 100}
