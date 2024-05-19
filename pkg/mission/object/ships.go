package object

import (
	"math"
	"slices"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/resources/images/ship"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

type ShipName string

const (
	ShipDefault ShipName = "默认战舰"
)

type ShipType string

const (
	// 航空母舰
	ShipTypeCarrier ShipType = "carrier"
	// 战列舰
	ShipTypeBattleship ShipType = "battleship"
	// 巡洋舰
	ShipTypeCruiser ShipType = "cruiser"
	// 驱逐舰
	ShipTypeDestroyer ShipType = "destroyer"
	// 护卫舰
	ShipTypeFrigate ShipType = "frigate"
	// 潜艇
	ShipTypeSubmarine ShipType = "submarine"
)

type WeaponType string

const (
	// 所有
	WeaponTypeAll WeaponType = "all"
	// 火炮
	WeaponTypeGun WeaponType = "gun"
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

// Weapon 武器系统
type Weapon struct {
	// 火炮
	Guns []*Gun
	// 鱼雷
	Torpedoes []*Torpedo
	// 最大射程（各类武器射程最大值）
	MaxRange float64
	// 武器禁用情况
	GunDisabled     bool
	TorpedoDisabled bool
}

// BattleShip 战舰
type BattleShip struct {
	// 名称
	Name ShipName
	// 类别
	Type ShipType

	// 初始生命值
	TotalHP float64
	// 伤害减免（0.7 -> 仅受到击中的 70% 伤害)
	DamageReduction float64
	// 最大速度
	MaxSpeed float64
	// 转向速度（度）
	RotateSpeed float64
	// 战舰长度
	Length float64
	// 战舰宽度
	Width float64
	// 武器
	Weapon Weapon

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
	if t == WeaponTypeAll || t == WeaponTypeGun {
		for i := 0; i < len(s.Weapon.Guns); i++ {
			s.Weapon.Guns[i].Disable = true
		}
		s.Weapon.GunDisabled = true
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
	if t == WeaponTypeAll || t == WeaponTypeGun {
		for i := 0; i < len(s.Weapon.Guns); i++ {
			s.Weapon.Guns[i].Disable = false
		}
		s.Weapon.GunDisabled = false
	}
	if t == WeaponTypeAll || t == WeaponTypeTorpedo {
		for i := 0; i < len(s.Weapon.Torpedoes); i++ {
			s.Weapon.Torpedoes[i].Disable = false
		}
		s.Weapon.TorpedoDisabled = false
	}
}

// InMaxRange 是否在最大射程内
func (s *BattleShip) InMaxRange(targetPos MapPos) bool {
	return geometry.CalcDistance(s.CurPos.RX, s.CurPos.RY, targetPos.RX, targetPos.RY) <= s.Weapon.MaxRange
}

// Fire 向指定目标发射武器
func (s *BattleShip) Fire(enemy *BattleShip) []*Bullet {
	shotBullets := []*Bullet{}
	// 如果生命值为 0，那还 Fire 个锤子，直接返回
	if s.CurHP <= 0 {
		return shotBullets
	}
	for i := 0; i < len(s.Weapon.Guns); i++ {
		shotBullets = slices.Concat(shotBullets, s.Weapon.Guns[i].Fire(s, enemy))
	}
	for i := 0; i < len(s.Weapon.Torpedoes); i++ {
		shotBullets = slices.Concat(shotBullets, s.Weapon.Torpedoes[i].Fire(s, enemy))
	}
	return shotBullets
}

// Hurt 收到伤害
func (s *BattleShip) Hurt(damage float64) {
	s.CurHP = max(0, s.CurHP-damage*s.DamageReduction)
}

// MoveTo 移动到指定位置
// TODO 路线规划 -> 绕过陆地
func (s *BattleShip) MoveTo(targetPos MapPos, borderX, borderY int) (arrive bool) {
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
		s.CurSpeed = min(s.MaxSpeed, s.CurSpeed+s.MaxSpeed/5)
	}
	// 到目标位置附近，逐渐减速
	if s.CurPos.Near(targetPos, 2) {
		s.CurSpeed = max(s.MaxSpeed/5, s.CurSpeed-s.MaxSpeed/5)
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
		if s.CurPos.Near(targetPos, 3) && math.Abs(s.CurRotation-targetRotation) > 1 {
			s.CurSpeed = 0
		}
	}
	// 修改位置
	s.CurPos.AddRx(math.Sin(s.CurRotation*math.Pi/180) * s.CurSpeed)
	s.CurPos.SubRy(math.Cos(s.CurRotation*math.Pi/180) * s.CurSpeed)

	// 防止出边界
	s.CurPos.EnsureBorder(float64(borderX-1), float64(borderY-1))

	return false
}

// NewShip 新建战舰
func NewShip(name ShipName, pos MapPos, rotation float64, player faction.Player) *BattleShip {
	s := deepcopy.Copy(*ships[name]).(BattleShip)
	s.Uid = uuid.New().String()
	s.CurPos = pos
	s.CurRotation = rotation
	s.BelongPlayer = player
	// 战舰默认不编组
	s.GroupID = GroupIDNone
	return &s
}

// ShipTrail 战舰尾流
type ShipTrail struct {
	Pos  MapPos
	Size float64
	Life int
}

// NewShipTrail ...
func NewShipTrail(pos MapPos, size float64, life int) *ShipTrail {
	return &ShipTrail{Pos: pos, Size: size, Life: life}
}

var shipDefault = &BattleShip{
	Name:            ShipDefault,
	Type:            ShipTypeCruiser,
	TotalHP:         1000,
	DamageReduction: 0.5,
	MaxSpeed:        0.1,
	RotateSpeed:     2,
	Length:          220,
	Width:           22,
	Weapon: Weapon{
		Guns: []*Gun{
			newGun(GunMK45, 0.3),
			newGun(GunMK45, -0.3),
		},
		// TODO 鱼雷先欠一下，后面再加
		Torpedoes:       []*Torpedo{},
		MaxRange:        20,
		GunDisabled:     false,
		TorpedoDisabled: false,
	},
	CurHP:       1000,
	CurPos:      MapPos{MX: 0, MY: 0},
	CurRotation: 0,
	CurSpeed:    0,
}

var ships = map[ShipName]*BattleShip{
	ShipDefault: shipDefault,
}

var shipImg = map[ShipName]*ebiten.Image{
	ShipDefault: ship.ShipDefaultZeroImg,
}

// GetShipImg 获取战舰图片
func GetShipImg(name ShipName) *ebiten.Image {
	return shipImg[name]
}
