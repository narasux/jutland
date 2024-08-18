package object

import (
	"log"
	"math"

	"github.com/google/uuid"
	"github.com/mohae/deepcopy"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

type BulletType string

const (
	// BulletTypeShell 火炮炮弹
	BulletTypeShell BulletType = "shell"
	// BulletTypeTorpedo 鱼雷
	BulletTypeTorpedo BulletType = "torpedo"
)

type BulletShotType int

const (
	// BulletShotTypeDirect 直射
	BulletShotTypeDirect BulletShotType = iota
	// BulletShotTypeArcing 曲射（抛物线射击）
	BulletShotTypeArcing
)

type CriticalType int

const (
	// CriticalTypeNone 没有暴击
	CriticalTypeNone CriticalType = iota
	// CriticalTypeThreeTimes 三倍暴击
	CriticalTypeThreeTimes
	// CriticalTypeTenTimes 十倍暴击
	CriticalTypeTenTimes
)

type HitObjectType int

const (
	// HitObjectTypeNone 无
	HitObjectTypeNone HitObjectType = iota
	// HitObjectTypeShip 战舰
	HitObjectTypeShip
	// HitObjectTypeWater 水面
	HitObjectTypeWater
	// HitObjectTypeLand 陆地
	HitObjectTypeLand
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
	// 暴击概率（理论上口径越大越容易被暴击，但是暴击率不应该太高）
	CriticalRate float64 `json:"criticalRate"`
	// 生命（前进太多要消亡）
	Life int

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
	// 射击方式
	ShotType BulletShotType
	// 前进周期数
	ForwardAge int

	// 所属战舰
	BelongShip string
	// 所属阵营（玩家）
	BelongPlayer faction.Player

	// 实际造成的伤害
	RealDamage float64
	// 造成暴击类型
	CriticalType CriticalType
	// 击中的对象类型
	HitObjectType HitObjectType
}

// Forward 弹药前进
func (b *Bullet) Forward() {
	// 修改位置
	nextPos := b.CurPos.Copy()
	nextPos.AddRx(math.Sin(b.Rotation*math.Pi/180) * b.Speed)
	nextPos.SubRy(math.Cos(b.Rotation*math.Pi/180) * b.Speed)

	// 直射的弹药只要一直塔塔开就好了，曲射的要考虑的就多了去了 :）
	if b.ShotType == BulletShotTypeDirect {
		b.CurPos = nextPos
	} else if b.ShotType == BulletShotTypeArcing {
		curDist := geometry.CalcDistance(b.CurPos.RX, b.CurPos.RY, b.TargetPos.RX, b.TargetPos.RY)
		nextDist := geometry.CalcDistance(nextPos.RX, nextPos.RY, b.TargetPos.RX, b.TargetPos.RY)
		// 离目标地点越来越远，说明下一个位置已经过了，曲射就是已经命中
		b.CurPos = lo.Ternary(nextDist > curDist, b.TargetPos, nextPos)
	} else {
		log.Fatal("unknown bullet shot type: ", b.ShotType)
	}

	// 修改生命 & 前进周期数
	b.Life--
	b.ForwardAge++
}

// GenTrail 生成尾流
func (b *Bullet) GenTrails() []*Trail {
	// 已经命中的没有尾流
	if b.HitObjectType != HitObjectTypeNone {
		return nil
	}
	// 刚刚发射的不添加尾流
	if b.ForwardAge <= 10 {
		return nil
	}
	// 不同类型的尾流特性不同
	diffusionRate, multipleSizeAsLife, lifeReductionRate := 0.1, 7.0, 2.0
	if b.Type == BulletTypeTorpedo {
		diffusionRate, multipleSizeAsLife, lifeReductionRate = 0.5, 8.0, 3.0
	}
	size := float64(GetImgWidth(b.Name, b.Type, b.Diameter))
	return []*Trail{
		newTrail(
			b.CurPos, texture.TrailShapeRect,
			size, diffusionRate,
			size*multipleSizeAsLife, lifeReductionRate,
			0, b.Rotation, nil,
		),
	}
}

var bulletMap = map[string]*Bullet{}

// NewBullets 新建弹药
func NewBullets(
	name string,
	curPos, targetPos MapPos,
	shotType BulletShotType,
	speed float64,
	life int,
	shipUid string,
	player faction.Player,
) *Bullet {
	b := deepcopy.Copy(*bulletMap[name]).(Bullet)

	b.Uid = uuid.New().String()
	b.CurPos = curPos
	b.TargetPos = targetPos
	b.ShotType = shotType

	b.Rotation = geometry.CalcAngleBetweenPoints(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	b.Speed = speed
	b.Life = life

	b.BelongShip = shipUid
	b.BelongPlayer = player

	b.CriticalType = CriticalTypeNone
	b.HitObjectType = HitObjectTypeNone
	return &b
}
