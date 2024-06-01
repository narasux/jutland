package object

import (
	"math"
	"slices"

	"github.com/google/uuid"
	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/mission/faction"
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
	// 战舰长度
	Length float64 `json:"length"`
	// 战舰宽度
	Width float64 `json:"width"`
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

// Hurt 收到伤害
func (s *BattleShip) Hurt(damage float64) {
	// TODO 加入暴击伤害的机制？比如一发大口径直接起飞（此处 @ 胡德）
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
		s.CurSpeed = min(s.MaxSpeed, s.CurSpeed+s.Acceleration)
	}
	// 到目标位置附近，逐渐减速
	if s.CurPos.Near(targetPos, 4) {
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

var shipMap = map[string]*BattleShip{}

// NewShip 新建战舰
func NewShip(name string, pos MapPos, rotation float64, player faction.Player) *BattleShip {
	s := deepcopy.Copy(*shipMap[name]).(BattleShip)
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
