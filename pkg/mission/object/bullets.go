package object

import (
	"math"

	"github.com/google/uuid"
	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

type BulletType string

const (
	// BulletTypeShell 火炮炮弹
	BulletTypeShell BulletType = "shell"
	// BulletTypeTorpedo 鱼雷
	BulletTypeTorpedo BulletType = "torpedo"
)

// 火炮 / 鱼雷弹药
type Bullet struct {
	// 弹药名称
	Name string `json:"name"`
	// 弹药类型
	Type BulletType `json:"type"`
	// 口径
	Diameter int `json:"diameter"`
	// 伤害数值
	Damage float64 `json:"damage"`
	// 生命（前进太多要消亡）
	// FIXME Life 根据实时的距离计算
	Life int `json:"life"`

	// 唯一标识
	Uid string
	// 当前位置
	CurPos MapPos
	// 目标位置
	TargetPos MapPos
	// 旋转角度
	Rotation float64
	// 速度
	Speed float64

	// 所属战舰
	BelongShip string
	// 所属阵营（玩家）
	BelongPlayer faction.Player

	// 是否命中战舰/水面/陆地
	HitShip  bool
	HitWater bool
	HitLand  bool
}

// Forward 弹药前进
func (b *Bullet) Forward() {
	// 修改位置
	b.CurPos.AddRx(math.Sin(b.Rotation*math.Pi/180) * b.Speed)
	b.CurPos.SubRy(math.Cos(b.Rotation*math.Pi/180) * b.Speed)
	// 修改生命
	b.Life--
}

var bulletMap = map[string]*Bullet{}

// NewBullets 新建弹药
func NewBullets(
	name string,
	curPos, targetPos MapPos,
	speed float64,
	shipUid string,
	player faction.Player,
) *Bullet {
	b := deepcopy.Copy(*bulletMap[name]).(Bullet)
	b.Uid = uuid.New().String()
	b.CurPos = curPos
	b.TargetPos = targetPos
	b.Rotation = geometry.CalcAngleBetweenPoints(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	b.Speed = speed
	b.BelongShip = shipUid
	b.BelongPlayer = player
	return &b
}
