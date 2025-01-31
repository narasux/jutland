package object

import (
	"log"

	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/mission/faction"
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

// Plane 战舰
type Plane struct {
	// 名称
	Name string `json:"name"`
	// 展示用名称
	DisplayName string `json:"displayName"`
	// 类别
	Type ShipType `json:"type"`
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
	Weapon Weapon `json:"weapon"`

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
	// 攻击目标（敌舰/敌机 Uid）
	AttackTarget string

	// 所属阵营（玩家）
	BelongPlayer faction.Player
}

var planeMap = map[string]*Plane{}

func newPlane(name string, curPos MapPos, rotation float64) *Plane {
	plane, ok := planeMap[name]
	if !ok {
		log.Fatalf("plane %s no found", name)
	}
	p := deepcopy.Copy(*plane).(Plane)
	p.CurPos = curPos
	p.CurRotation = rotation
	// FIXME 战机的其他属性初始化
	return &p
}
