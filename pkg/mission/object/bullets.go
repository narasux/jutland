package object

import (
	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mohae/deepcopy"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/resources/colorx"
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
var defaultBulletImg = ebiten.NewImage(2, 4)

func init() {
	defaultBulletImg.Fill(colorx.White)
}

var bulletImages = map[BulletName]*ebiten.Image{
	Gb127T1: defaultBulletImg,
}

// NewBullets 新建弹药
func NewBullets(
	name BulletName,
	curPos, targetPos MapPos,
	speed float64,
	shipUid string,
	player faction.Player,
) *Bullet {
	b := deepcopy.Copy(*bullets[name]).(Bullet)
	b.Uid = uuid.New().String()
	b.CurPos = curPos
	b.TargetPos = targetPos
	b.Rotation = geometry.CalcAngleBetweenPoints(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	b.Speed = speed
	b.BelongShip = shipUid
	b.BelongPlayer = player
	return &b
}

// GetBulletImg 获取弹药图片
func GetBulletImg(name BulletName) *ebiten.Image {
	return bulletImages[name]
}

// gb (gun bullet) 表示是火炮弹药，tb (torpedo bullet) 表示鱼雷弹药
var gb127T1 = &Bullet{Name: Gb127T1, Damage: 100}
