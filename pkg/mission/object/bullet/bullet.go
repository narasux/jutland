package bullet

import (
	"log"
	"math"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mohae/deepcopy"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/object/trail"
	bulletImg "github.com/narasux/jutland/pkg/resources/images/bullet"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// Type 弹药类型
type Type string

const (
	// TypeShell 火炮炮弹
	TypeShell Type = "shell"
	// TypeTorpedo 鱼雷
	TypeTorpedo Type = "torpedo"
	// TypeBomb 炸弹
	TypeBomb Type = "bomb"
	// TypeRocket 火箭弹
	TypeRocket Type = "rocket"
	// TypeLaser 镭射
	TypeLaser Type = "laser"
)

// ShotType 射击方式
type ShotType int

const (
	// ShotTypeDirect 直射
	ShotTypeDirect ShotType = iota
	// ShotTypeArcing 曲射（抛物线射击）
	ShotTypeArcing
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

// 火炮 / 鱼雷弹药
type Bullet struct {
	// 弹药名称
	Name string `json:"name"`
	// 弹药类型
	Type Type `json:"type"`
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
	CurPos objPos.MapPos
	// 目标位置
	TargetPos objPos.MapPos
	// 旋转角度
	Rotation float64
	// 速度
	Speed float64
	// 射击方式
	ShotType ShotType
	// 目标对象类型
	TargetObjType object.Type
	// 前进周期数
	ForwardAge int

	// 所属战舰/战机
	Shooter string
	// 所属对象类型
	ShooterObjType object.Type
	// 所属阵营（玩家）
	BelongPlayer faction.Player

	// 实际造成的伤害
	RealDamage float64
	// 造成暴击类型
	CriticalType CriticalType
	// 击中的对象类型
	HitObjType object.Type
	// 近炸触发半径，仅火箭弹使用
	ProximityRadius float64
	// 爆炸伤害半径，仅火箭弹使用
	BlastRadius float64
}

// Forward 弹药前进
func (b *Bullet) Forward() {
	// 修改位置
	nextPos := b.CurPos.Copy()
	nextPos.AddRx(math.Sin(b.Rotation*math.Pi/180) * b.Speed)
	nextPos.SubRy(math.Cos(b.Rotation*math.Pi/180) * b.Speed)

	// 直射的弹药只要一直塔塔开就好了，曲射的要考虑的就多了去了 :）
	if b.ShotType == ShotTypeDirect {
		b.CurPos = nextPos
	} else if b.ShotType == ShotTypeArcing {
		curDist, nextDist := b.CurPos.Distance(b.TargetPos), nextPos.Distance(b.TargetPos)
		// 离目标地点越来越远，说明下一个位置已经过了，曲射就是已经命中
		b.CurPos = lo.Ternary(nextDist > curDist, b.TargetPos, nextPos)
	} else {
		log.Fatal("unknown bullet shot type: ", b.ShotType)
	}

	// 修改生命 & 前进周期数
	b.Life--
	b.ForwardAge++
}

// GenTrails 生成尾流
func (b *Bullet) GenTrails() []*trail.Trail {
	// 已经命中的没有尾流
	if b.HitObjType != object.TypeNone {
		return nil
	}
	if b.Type == TypeRocket {
		return b.genRocketTrails()
	}
	// 刚刚发射的不添加尾流
	if b.ForwardAge <= 10 {
		return nil
	}
	// 镭射弹药没有尾流
	if b.Type == TypeLaser {
		return nil
	}
	// 不同类型的尾流特性不同
	diffusionRate, multipleSizeAsLife, lifeReductionRate := 0.1, 7.0, 2.0
	if b.Type == TypeTorpedo {
		diffusionRate, multipleSizeAsLife, lifeReductionRate = 0.5, 8.0, 3.0
	}
	size := float64(GetImgWidth(b.Name, b.Type, b.Diameter))
	return []*trail.Trail{
		trail.New(
			b.CurPos, textureImg.TrailShapeRect,
			size, diffusionRate,
			size*multipleSizeAsLife, lifeReductionRate,
			0, b.Rotation, nil,
		),
	}
}

// genRocketTrails 生成火箭专属尾流：短促尾焰叠加连续深灰烟点，和炮弹的线状尾流区分开。
func (b *Bullet) genRocketTrails() []*trail.Trail {
	if b.ForwardAge <= 1 {
		return nil
	}

	sinVal := math.Sin(b.Rotation * math.Pi / 180)
	cosVal := math.Cos(b.Rotation * math.Pi / 180)
	tailPos := func(distance float64) objPos.MapPos {
		pos := b.CurPos.Copy()
		pos.SubRx(sinVal * b.Speed * distance)
		pos.AddRy(cosVal * b.Speed * distance)
		return pos
	}

	trails := []*trail.Trail{
		trail.New(
			tailPos(0.9), textureImg.TrailShapeCircle,
			3.2, 0.18,
			120, 7.5,
			0, 0, colorx.Orange,
		),
	}
	if b.ForwardAge%2 != 0 {
		return trails
	}

	trails = append(trails,
		trail.New(
			tailPos(1.5), textureImg.TrailShapeCircle,
			5.5, 0.10,
			105, 3.0,
			0, 0, colorx.DarkSilver,
		),
		trail.New(
			tailPos(2.4), textureImg.TrailShapeCircle,
			4.6, 0.08,
			82, 2.6,
			0, 0, colorx.Gray,
		),
	)
	return trails
}

// Map 弹药表
var Map = map[string]*Bullet{}

// New 新建弹药
func New(
	name string,
	curPos, targetPos objPos.MapPos,
	shooterUid string,
	shooterObjType object.Type,
	shooterBelongPlayer faction.Player,
	shotType ShotType,
	targetObjectType object.Type,
	speed float64,
	life int,
) *Bullet {
	b := deepcopy.Copy(*Map[name]).(Bullet)

	b.Uid = uuid.New().String()
	b.CurPos = curPos
	b.TargetPos = targetPos
	b.ShotType = shotType
	b.TargetObjType = targetObjectType

	b.Rotation = curPos.Angle(targetPos)
	b.Speed = speed
	b.Life = life

	b.Shooter = shooterUid
	b.ShooterObjType = shooterObjType
	b.BelongPlayer = shooterBelongPlayer

	b.CriticalType = CriticalTypeNone
	b.HitObjType = object.TypeNone
	return &b
}

// GetType 获取弹药类型
func GetType(name string) Type {
	b, ok := Map[name]
	if !ok {
		log.Fatalf("bullet %s no found", name)
	}
	return b.Type
}

// GetImg 获取弹药图片
func GetImg(btType Type, diameter int) *ebiten.Image {
	switch btType {
	case TypeShell:
		return bulletImg.GetShell(diameter)
	case TypeTorpedo:
		return bulletImg.GetTorpedo(diameter)
	case TypeBomb:
		return bulletImg.GetBomb(diameter)
	case TypeRocket:
		return bulletImg.GetRocket(diameter)
	case TypeLaser:
		return bulletImg.GetLaser(diameter)
	}
	return bulletImg.NotFount
}

var BulletImgWidthMap = map[string]int{}

// GetImgWidth 获取弹药图片宽度（虽然可能价值不大，总之先加一点缓存 :）
func GetImgWidth(btName string, btType Type, diameter int) int {
	if width, ok := BulletImgWidthMap[btName]; ok {
		return width
	}
	width := GetImg(btType, diameter).Bounds().Dx()
	BulletImgWidthMap[btName] = width
	return width
}
