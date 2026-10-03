package bullet

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

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

type CriticalType int

const (
	// CriticalTypeNone 没有暴击
	CriticalTypeNone CriticalType = iota
	// CriticalTypeThreeTimes 三倍暴击
	CriticalTypeThreeTimes
	// CriticalTypeTenTimes 十倍暴击
	CriticalTypeTenTimes
)

// 弹道落角参数：0 表示完全平射，1 表示垂直落下。
// 开火时由射程百分比连续算出，同时决定命中判定、危险界与装甲带，
// 避免在某个固定射程上硬切换直射 / 曲射。
const (
	// PlungeRangeStart 平射阶段的上界（射程百分比）：更近的射击一律按平射处理
	PlungeRangeStart = 0.50
	// PlungeRangeRef 落角达到参考值的射程百分比
	PlungeRangeRef = 0.65
	// dangerHeight 目标有效高度（地图格，约 26 m）：跨射带宽度由它除以落角正切得到
	dangerHeight = 0.20
	// dangerTanRef 参考落角的正切（约 35°）
	dangerTanRef = 0.70
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

	// 当前位置
	CurPos objPos.MapPos
	// 目标位置
	TargetPos objPos.MapPos
	// 旋转角度
	Rotation float64
	// 速度
	Speed float64
	// 弹道落角参数 0~1：0 平射（危险界不限制、打水平装甲带），1 垂直落下（跨射带最窄、打垂直装甲带）
	Plunge float64
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
	// 近炸触发半径，火箭弹与对空编程炮弹使用
	ProximityRadius float64
	// 爆炸伤害半径，火箭弹与对空编程炮弹使用
	BlastRadius float64
	// 离上次留下炮弹或鱼雷尾迹又飞过的距离，单位是地图格
	trailCarry float64
}

// HasAirburst 对空近炸破片：火箭弹及编程引信炮弹共用结算。
func (b *Bullet) HasAirburst() bool {
	return b.BlastRadius > 0 && b.TargetObjType == object.TypePlane
}

// PlungeForRange 由射程百分比换算弹道落角参数。
// 近距离弹道低伸，落角接近 0；远距离落角变陡，0.65R 附近接近垂直落下。
func PlungeForRange(rangePercent float64) float64 {
	progress := (rangePercent - PlungeRangeStart) / (PlungeRangeRef - PlungeRangeStart)
	progress = min(1, max(0, progress))
	// smoothstep：两端斜率为 0，落角与装甲带都不会在某个射程上突然变化
	return progress * progress * (3 - 2*progress)
}

// DangerSpace 沿弹道方向的命中容差（地图格）。
// 平射时弹道低伸，炮弹总能从舰体上扫过，返回 +Inf 表示不限制；
// 落角越陡，危险界越窄，打远 / 打近就会脱靶。
func (b *Bullet) DangerSpace() float64 {
	if b.Plunge <= 0 {
		return math.Inf(1)
	}
	return dangerHeight / (b.Plunge * dangerTanRef)
}

// PassedAim 沿弹道方向是否已经越过瞄准点一个危险界的距离。
// 平射弹药永不成立（会一直飞到寿命结束），吊射弹药据此判定落水。
func (b *Bullet) PassedAim() bool {
	radians := b.Rotation * math.Pi / 180
	along := (b.CurPos.RX-b.TargetPos.RX)*math.Sin(radians) -
		(b.CurPos.RY-b.TargetPos.RY)*math.Cos(radians)
	return along > b.DangerSpace()
}

// Forward 弹药前进。
// 所有弹药都沿直线飞行，能否命中由危险界与舰体几何决定；
// 越过瞄准点的吊射弹药由 PassedAim 判定落水。
func (b *Bullet) Forward() {
	nextPos := b.CurPos.Copy()
	nextPos.AddRx(math.Sin(b.Rotation*math.Pi/180) * b.Speed)
	nextPos.SubRy(math.Cos(b.Rotation*math.Pi/180) * b.Speed)
	b.CurPos = nextPos

	// 修改生命 & 前进周期数
	b.Life--
	b.ForwardAge++
}

const (
	// 炮弹和鱼雷大约每飞过半个地图格留一个尾迹点，一拍最多一个。
	trailSampleDistance = 0.5
	rocketTrailInterval = 3
)

// GenTrails 生成尾流
func (b *Bullet) GenTrails() []*trail.Trail {
	// 已经命中的没有尾流
	if b.HitObjType != object.TypeNone {
		return nil
	}
	if b.Type == TypeRocket {
		return b.genRocketTrails()
	}
	// 刚刚发射的不添加尾流，镭射也没有尾流
	if b.ForwardAge <= 10 || b.Type == TypeLaser {
		return nil
	}
	b.trailCarry += b.Speed
	if b.trailCarry < trailSampleDistance {
		return nil
	}
	b.trailCarry -= trailSampleDistance

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

// genRocketTrails 每 3 拍留一个圆点，尾焰和灰烟交替。
func (b *Bullet) genRocketTrails() []*trail.Trail {
	if b.ForwardAge <= 1 || b.ForwardAge%rocketTrailInterval != 0 {
		return nil
	}

	sinVal := math.Sin(b.Rotation * math.Pi / 180)
	cosVal := math.Cos(b.Rotation * math.Pi / 180)
	pos := b.CurPos.Copy()
	if (b.ForwardAge/rocketTrailInterval)%2 == 1 {
		pos.SubRx(sinVal * b.Speed * 0.9)
		pos.AddRy(cosVal * b.Speed * 0.9)
		return []*trail.Trail{
			trail.New(
				pos, textureImg.TrailShapeCircle,
				3.2, 0.18,
				120, 7.5,
				0, 0, colorx.Orange,
			),
		}
	}

	pos.SubRx(sinVal * b.Speed * 1.5)
	pos.AddRy(cosVal * b.Speed * 1.5)
	return []*trail.Trail{
		trail.New(
			pos, textureImg.TrailShapeCircle,
			5.5, 0.10,
			105, 3.0,
			0, 0, colorx.DarkSilver,
		),
	}
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
	targetObjectType object.Type,
	speed float64,
	life int,
	plunge float64,
) *Bullet {
	tpl, ok := Map[name]
	if !ok {
		log.Fatalf("bullet %s no found", name)
	}
	b := *tpl

	b.CurPos = curPos
	b.TargetPos = targetPos
	b.Plunge = plunge
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
