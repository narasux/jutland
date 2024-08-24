package object

import (
	"fmt"
	"math"
	"math/rand"
	"slices"

	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/faction"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
	"github.com/narasux/jutland/pkg/utils/colorx"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

type WeaponType string

const (
	// 所有
	WeaponTypeAll WeaponType = "all"
	// 主炮
	WeaponTypeMainGun WeaponType = "mainGun"
	// 副炮
	WeaponTypeSecondaryGun WeaponType = "secondaryGun"
	// 鱼雷
	WeaponTypeTorpedo WeaponType = "torpedo"
	// 导弹
	WeaponTypeMissile WeaponType = "missile"
)

const (
	// 顺时针
	RotateFlagClockwise = 1
	// 逆时针
	RotateFlagAnticlockwise = -1
)

type WeaponMetadata struct {
	Name string `json:"name"`
	// 相对位置
	// 0.35 -> 从中心往舰首 35% 舰体长度
	// -0.3 -> 从中心往舰尾 30% 舰体长度
	PosPercent float64 `json:"posPercent"`
	// 左射界
	LeftFiringArc [2]float64 `json:"leftFiringArc"`
	// 右射界
	RightFiringArc [2]float64 `json:"rightFiringArc"`
}

// Weapon 武器系统
type Weapon struct {
	// 主炮元数据
	MainGunsMD []WeaponMetadata `json:"mainGuns"`
	// 副炮元数据
	SecondaryGunsMD []WeaponMetadata `json:"secondaryGuns"`
	// 鱼雷元数据
	TorpedoesMD []WeaponMetadata `json:"torpedoes"`
	// 主炮
	MainGuns []*Gun
	// 副炮
	SecondaryGuns []*Gun
	// 鱼雷
	Torpedoes []*TorpedoLauncher
	// 最大射程（各类武器射程最大值）
	MaxRange float64
	// 拥有的武器情况
	HasMainGun      bool
	HasSecondaryGun bool
	HasTorpedo      bool
	// 武器禁用情况
	MainGunDisabled      bool
	SecondaryGunDisabled bool
	TorpedoDisabled      bool
}

// BattleShip 战舰
type BattleShip struct {
	// 名称
	Name string `json:"name"`
	// 展示用名称
	DisplayName string `json:"displayName"`
	// 类别
	Type string `json:"type"`
	// 类别缩写
	TypeAbbr string `json:"typeAbbr"`

	// 初始生命值
	TotalHP float64 `json:"totalHP"`
	// 水平伤害减免（0.7 -> 仅受到击中的 70% 伤害)
	HorizontalDamageReduction float64 `json:"horizontalDamageReduction"`
	// 垂直伤害减免
	VerticalDamageReduction float64 `json:"verticalDamageReduction"`
	// 最大速度
	MaxSpeed float64 `json:"maxSpeed"`
	// 加速度
	Acceleration float64 `json:"acceleration"`
	// 转向速度（度）
	RotateSpeed float64 `json:"rotateSpeed"`
	// 战舰长度
	Length float64 `json:"length"`
	// 战舰宽度
	Width float64 `json:"width"`
	// 造价
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
	// 旋转角度
	CurRotation float64
	// 当前速度
	CurSpeed float64
	// 分组ID
	GroupID GroupID

	// 所属阵营（玩家）
	BelongPlayer faction.Player
}

// DisableWeapon 禁用武器
func (s *BattleShip) DisableWeapon(t WeaponType) {
	if t == WeaponTypeAll || t == WeaponTypeMainGun {
		for i := 0; i < len(s.Weapon.MainGuns); i++ {
			s.Weapon.MainGuns[i].Disable = true
		}
		s.Weapon.MainGunDisabled = true
	}
	if t == WeaponTypeAll || t == WeaponTypeSecondaryGun {
		for i := 0; i < len(s.Weapon.SecondaryGuns); i++ {
			s.Weapon.SecondaryGuns[i].Disable = true
		}
		s.Weapon.SecondaryGunDisabled = true
	}
	if t == WeaponTypeAll || t == WeaponTypeTorpedo {
		for i := 0; i < len(s.Weapon.Torpedoes); i++ {
			s.Weapon.Torpedoes[i].Disable = true
		}
		s.Weapon.TorpedoDisabled = true
	}
}

// EnableWeapon 启用武器
func (s *BattleShip) EnableWeapon(t WeaponType) {
	if t == WeaponTypeAll || t == WeaponTypeMainGun {
		for i := 0; i < len(s.Weapon.MainGuns); i++ {
			s.Weapon.MainGuns[i].Disable = false
		}
		s.Weapon.MainGunDisabled = false
	}
	if t == WeaponTypeAll || t == WeaponTypeSecondaryGun {
		for i := 0; i < len(s.Weapon.SecondaryGuns); i++ {
			s.Weapon.SecondaryGuns[i].Disable = false
		}
		s.Weapon.SecondaryGunDisabled = false
	}
	if t == WeaponTypeAll || t == WeaponTypeTorpedo {
		for i := 0; i < len(s.Weapon.Torpedoes); i++ {
			s.Weapon.Torpedoes[i].Disable = false
		}
		s.Weapon.TorpedoDisabled = false
	}
}

// Fire 向指定目标发射武器
func (s *BattleShip) Fire(enemy *BattleShip) []*Bullet {
	shotBullets := []*Bullet{}
	// 如果生命值为 0，那还 Fire 个锤子，直接返回
	if s.CurHP <= 0 {
		return shotBullets
	}
	for i := 0; i < len(s.Weapon.MainGuns); i++ {
		shotBullets = slices.Concat(shotBullets, s.Weapon.MainGuns[i].Fire(s, enemy))
	}
	for i := 0; i < len(s.Weapon.SecondaryGuns); i++ {
		shotBullets = slices.Concat(shotBullets, s.Weapon.SecondaryGuns[i].Fire(s, enemy))
	}
	for i := 0; i < len(s.Weapon.Torpedoes); i++ {
		shotBullets = slices.Concat(shotBullets, s.Weapon.Torpedoes[i].Fire(s, enemy))
	}
	return shotBullets
}

// HurtBy 受到伤害
func (s *BattleShip) HurtBy(bullet *Bullet) {
	realDamage := 0.0
	if bullet.ShotType == BulletShotTypeDirect {
		// 平射打击水平装甲带
		realDamage = bullet.Damage * (1 - s.HorizontalDamageReduction)
	} else {
		// 曲射打击垂直装甲带
		realDamage = bullet.Damage * (1 - s.VerticalDamageReduction)
	}

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
	s.CurHP = max(0, s.CurHP-realDamage)
	// 弹药是可以造成重复伤害的，这里需要计算累计值，暴击类型统计，只统计最高倍数
	bullet.RealDamage += realDamage
	bullet.CriticalType = max(criticalType, bullet.CriticalType)
}

// GenTrails 生成尾流
func (s *BattleShip) GenTrails() []*Trail {
	if s.CurSpeed <= 0 {
		return nil
	}
	// 水滴应该是特殊的尾流（蓝色光尾流，负扩散）
	if s.TypeAbbr == "WaterDrop" {
		return []*Trail{
			newTrail(
				s.CurPos, textureImg.TrailShapeRect,
				s.Width*0.5, -2,
				s.Length/6+150*s.CurSpeed, 5,
				0, s.CurRotation, colorx.SkyBlue,
			),
		}
	}

	offset := s.Length / constants.MapBlockSize
	sinVal := math.Sin(s.CurRotation * math.Pi / 180)
	cosVal := math.Cos(s.CurRotation * math.Pi / 180)

	frontPos, backPos := s.CurPos.Copy(), s.CurPos.Copy()
	frontPos.AddRx(sinVal * offset * 0.25)
	frontPos.SubRy(cosVal * offset * 0.25)
	backPos.SubRx(sinVal * offset * 0.2)
	backPos.AddRy(cosVal * offset * 0.2)

	return []*Trail{
		newTrail(
			frontPos, textureImg.TrailShapeCircle,
			s.Width*0.6, 1.2,
			s.Length/6+150*s.CurSpeed, 1,
			0, 0, nil,
		),
		newTrail(
			backPos, textureImg.TrailShapeCircle,
			s.Width, 0.4,
			s.Length/7+155*s.CurSpeed, 1.5,
			0, 0, nil,
		),
	}
}

// CanOnLand 能在陆地上
func (s *BattleShip) CanOnLand() bool {
	return s.TypeAbbr == "WaterDrop"
}

// MoveTo 移动到指定位置
// TODO 路线规划 -> 绕过陆地
func (s *BattleShip) MoveTo(mapCfg *mapcfg.MapCfg, targetPos MapPos) (arrive bool) {
	// 如果生命值为 0，肯定是走不动，直接返回
	if s.CurHP <= 0 {
		return true
	}
	// 差不多到目标位置即可，不要强求准确，否则需要微调，视觉效果不佳
	if s.CurPos.Near(targetPos, 1) {
		s.CurSpeed = 0
		return true
	}
	// 未到达目标位置，逐渐加速
	if s.CurSpeed < s.MaxSpeed {
		s.CurSpeed = min(s.MaxSpeed, s.CurSpeed+s.Acceleration)
	}
	// 到目标位置附近，逐渐减速
	if s.CurPos.Near(targetPos, s.Length/constants.MapBlockSize*1.5) {
		s.CurSpeed = max(s.Acceleration*20, s.CurSpeed-s.Acceleration*10)
	}
	targetRotation := geometry.CalcAngleBetweenPoints(s.CurPos.RX, s.CurPos.RY, targetPos.RX, targetPos.RY)
	// 逐渐转向
	if s.CurRotation != targetRotation {
		// 默认顺时针旋转
		rotateFlag := RotateFlagClockwise
		// 如果逆时针夹角小于顺时针夹角，则需要逆时针旋转
		if math.Mod(targetRotation-s.CurRotation+360, 360) > 180 {
			rotateFlag = RotateFlagAnticlockwise
		}
		s.CurRotation += float64(rotateFlag) * min(math.Abs(targetRotation-s.CurRotation), s.RotateSpeed)
		s.CurRotation = math.Mod(s.CurRotation+360, 360)
		// 如果距离太近，则原地旋转到差不多角度，才开始移动
		if s.CurPos.Near(targetPos, 4) && math.Abs(s.CurRotation-targetRotation) > 1 {
			s.CurSpeed = 0
		}
	}
	nextPos := s.CurPos.Copy()
	// 修改位置
	nextPos.AddRx(math.Sin(s.CurRotation*math.Pi/180) * s.CurSpeed)
	nextPos.SubRy(math.Cos(s.CurRotation*math.Pi/180) * s.CurSpeed)
	// 防止出边界
	nextPos.EnsureBorder(float64(mapCfg.Width-1), float64(mapCfg.Height-1))
	// FIXME 这里先粗暴点，直接停船，假装到目的地，后面再搞定路线规划
	// 特殊船舶是可以在陆地上的（飞起来的那些）
	if mapCfg.Map.IsLand(nextPos.MX, nextPos.MY) && !s.CanOnLand() {
		s.CurSpeed = 0
		return true
	}
	// 移动到新位置
	s.CurPos = nextPos

	return false
}

var shipMap = map[string]*BattleShip{}

// NewShip 新建战舰
func NewShip(
	uidGenerator *ShipUidGenerator, name string, pos MapPos, rotation float64, player faction.Player,
) *BattleShip {
	s := deepcopy.Copy(*shipMap[name]).(BattleShip)
	s.Uid = uidGenerator.Gen(s.TypeAbbr)
	s.CurPos = pos
	s.CurRotation = rotation
	s.BelongPlayer = player
	// 战舰默认不编组
	s.GroupID = GroupIDNone
	return &s
}

// ShipUidGenerator 战舰 Uid 生成器
type ShipUidGenerator struct {
	player  faction.Player
	counter map[string]int
}

// NewShipUidGenerator ...
func NewShipUidGenerator(player faction.Player) *ShipUidGenerator {
	return &ShipUidGenerator{
		player:  player,
		counter: map[string]int{},
	}
}

// Gen 生成战舰 Uid
func (g *ShipUidGenerator) Gen(typeAbbr string) string {
	if _, ok := g.counter[typeAbbr]; !ok {
		g.counter[typeAbbr] = 1
	} else {
		g.counter[typeAbbr]++
	}
	return fmt.Sprintf("%s/%s-%d", g.player, typeAbbr, g.counter[typeAbbr])
}
