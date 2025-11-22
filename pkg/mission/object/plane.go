package object

import (
	"log"
	"math"
	"math/rand"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// PlaneType 飞机类型
type PlaneType string

const (
	// PlaneTypeFighter 战斗机
	PlaneTypeFighter PlaneType = "fighter"
	// PlaneTypeDiveBomber 俯冲轰炸机
	PlaneTypeDiveBomber PlaneType = "dive_bomber"
	// PlaneTypeTorpedoBomber 鱼雷轰炸机
	PlaneTypeTorpedoBomber PlaneType = "torpedo_bomber"
)

// Plane 战机
type Plane struct {
	// 名称
	Name string `json:"name"`
	// 展示用名称
	DisplayName string `json:"displayName"`
	// 类别
	Type PlaneType `json:"type"`
	// 类别缩写
	TypeAbbr string `json:"typeAbbr"`
	// 描述
	Description []string `json:"description"`

	// 初始生命值
	TotalHP float64 `json:"totalHP"`
	// 伤害减免（0.7 -> 仅受到击中的 70% 伤害)
	DamageReduction float64 `json:"damageReduction"`
	// 最大速度
	MaxSpeed float64 `json:"maxSpeed"`
	// 加速度
	Acceleration float64 `json:"acceleration"`
	// 转向速度（度）
	RotateSpeed float64 `json:"rotateSpeed"`
	// 总航程
	Range float64 `json:"range"`
	// 战机长度
	Length float64 `json:"length"`
	// 战机宽度
	Width float64 `json:"width"`
	// 造价 TODO 航空母舰可以无限补充飞机，但是得花时间 & 钱？
	FundsCost int64 `json:"fundsCost"`
	// 耗时
	TimeCost int64 `json:"timeCost"`
	// 吨位
	Tonnage float64 `json:"tonnage"`
	// 武器
	Weapon PlaneWeapon `json:"weapon"`

	// 唯一标识
	Uid string
	// 当前生命值
	CurHP float64
	// 当前位置
	CurPos MapPos
	// 当前高度 TODO 是否引入高度概念？
	CurHeight float64
	// 旋转角度
	CurRotation float64
	// 当前速度
	CurSpeed float64
	// 剩余航程
	RemainRange float64

	// 所属阵营（玩家）
	BelongPlayer faction.Player
	// 所属战舰（uid）
	BelongShip string
}

var _ Hurtable = (*Plane)(nil)

var _ Attacker = (*Plane)(nil)

// ID 唯一标识
func (p *Plane) ID() string {
	return p.Uid
}

// Player 所属玩家
func (p *Plane) Player() faction.Player {
	return p.BelongPlayer
}

// ObjType 对象类型
func (p *Plane) ObjType() ObjectType {
	return ObjectTypePlane
}

// MovementState 机动状态
func (p *Plane) MovementState() UnitMovementState {
	return UnitMovementState{
		CurPos:      p.CurPos.Copy(),
		CurRotation: p.CurRotation,
		CurSpeed:    p.CurSpeed,
	}
}

// GeometricSize 几何尺寸
func (p *Plane) GeometricSize() UnitGeometricSize {
	return UnitGeometricSize{Length: p.Length, Width: p.Width}
}

// Fire 向指定目标发射武器
func (p *Plane) Fire(enemy Hurtable) (shotBullets []*Bullet) {
	// 如果生命值为 0，那还 Fire 个锤子，直接返回
	if p.CurHP <= 0 {
		return
	}
	// 机炮不用记录射击时间
	for i := 0; i < len(p.Weapon.Guns); i++ {
		shotBullets = slices.Concat(shotBullets, p.Weapon.Guns[i].Fire(p, enemy))
	}
	// 释放器类武器，有最小的释放间隔限制，且目前只能攻击战舰（后面有导弹，火箭弹再说）
	if enemy.ObjType() == ObjectTypeShip {
		timeNow := time.Now().UnixMilli()
		if timeNow > p.Weapon.LatestReleaseAt+p.Weapon.ReleaseInterval*1e3 {
			for _, releasers := range [2][]*Releaser{
				p.Weapon.Bombs, p.Weapon.Torpedoes,
			} {
				for i := 0; i < len(releasers); i++ {
					if bullets := releasers[i].Fire(p, enemy); len(bullets) > 0 {
						shotBullets = slices.Concat(shotBullets, bullets)
						p.Weapon.LatestReleaseAt = timeNow
						break
					}
				}
			}
		}
	}
	return shotBullets
}

// HurtBy 受到伤害
func (p *Plane) HurtBy(bullet *Bullet) {
	// 计算真实伤害，飞机比较脆，所以伤害要再额外乘以 3
	realDamage := bullet.Damage * (1 - p.DamageReduction) * 3

	// 暴击伤害的机制，一发大口径可能直接起飞，支持多段暴击
	criticalType := CriticalTypeNone
	randVal := rand.Float64()
	if randVal < bullet.CriticalRate/10 {
		realDamage *= 10
		criticalType = CriticalTypeTenTimes
	} else if randVal < bullet.CriticalRate {
		realDamage *= 3
		criticalType = CriticalTypeThreeTimes
	}

	// 计算生命值 & 累计伤害
	p.CurHP = max(0, p.CurHP-realDamage)
	// 弹药是可以造成重复伤害的，这里需要计算累计值，暴击类型统计，只统计最高倍数
	bullet.RealDamage += realDamage
	bullet.CriticalType = max(criticalType, bullet.CriticalType)
}

// MoveTo 移动到指定位置
func (p *Plane) MoveTo(mapCfg *mapcfg.MapCfg, targetPos MapPos) {
	// 如果生命值为 0，肯定是走不动，直接返回
	if p.CurHP <= 0 {
		return
	}
	// 飞机只要移动，就是最大速度（简化逻辑）
	p.CurSpeed = p.MaxSpeed

	targetRotation := p.CurPos.Angle(targetPos)
	// 逐渐转向
	if p.CurRotation != targetRotation {
		// 默认顺时针旋转
		rotateFlag := RotateFlagClockwise
		// 如果逆时针夹角小于顺时针夹角，则需要逆时针旋转
		if math.Mod(targetRotation-p.CurRotation+360, 360) > 180 {
			rotateFlag = RotateFlagAnticlockwise
		}
		p.CurRotation += float64(rotateFlag) * min(math.Abs(targetRotation-p.CurRotation), p.RotateSpeed)
		p.CurRotation = math.Mod(p.CurRotation+360, 360)
	}
	nextPos := p.CurPos.Copy()
	// 修改位置
	nextPos.AddRx(math.Sin(p.CurRotation*math.Pi/180) * p.CurSpeed)
	nextPos.SubRy(math.Cos(p.CurRotation*math.Pi/180) * p.CurSpeed)
	// 防止出边界
	nextPos.EnsureBorder(float64(mapCfg.Width-2), float64(mapCfg.Height-2))
	// 移动到新位置
	p.CurPos = nextPos
	p.RemainRange -= p.CurSpeed
}

var planeMap = map[string]*Plane{}

// NewPlane 生成飞机
func NewPlane(
	name string,
	curPos MapPos,
	rotation float64,
	shipUid string,
	player faction.Player,
) *Plane {
	plane, ok := planeMap[name]
	if !ok {
		log.Fatalf("plane %s no found", name)
	}
	p := deepcopy.Copy(*plane).(Plane)

	p.Uid = uuid.New().String()
	p.CurPos = curPos
	p.CurRotation = rotation
	p.BelongPlayer = player
	p.BelongShip = shipUid
	return &p
}

// getPlaneTargetObjType 获取飞机攻击目标类型
func getPlaneTargetObjType(name string) ObjectType {
	plane, ok := planeMap[name]
	if !ok {
		log.Fatalf("plane %s no found", name)
	}
	// 根据飞机类型获取目标类型
	switch plane.Type {
	case PlaneTypeFighter:
		return ObjectTypePlane
	case PlaneTypeDiveBomber, PlaneTypeTorpedoBomber:
		return ObjectTypeShip
	default:
		return ObjectTypeNone
	}
}
