package unit

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objRef "github.com/narasux/jutland/pkg/mission/object/reference"
	objTrail "github.com/narasux/jutland/pkg/mission/object/trail"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

// ShipType 战舰类型
type ShipType string

const (
	// ShipTypeDefault 默认
	ShipTypeDefault ShipType = "default"
	// ShipTypeAircraftCarrier 航空母舰
	ShipTypeAircraftCarrier ShipType = "aircraft_carrier"
	// ShipTypeBattleShip 战列舰
	ShipTypeBattleShip ShipType = "battleship"
	// ShipTypeCruiser 巡洋舰
	ShipTypeCruiser ShipType = "cruiser"
	// ShipTypeDestroyer 驱逐舰
	ShipTypeDestroyer ShipType = "destroyer"
	// ShipTypeFrigate 护卫舰
	ShipTypeFrigate ShipType = "frigate"
	// ShipTypeTorpedoBoat 鱼雷艇
	ShipTypeTorpedoBoat ShipType = "torpedo_boat"
	// ShipTypeCargo 货轮
	ShipTypeCargo ShipType = "cargo"
	// ShipTypeHospital 医疗船
	ShipTypeHospital ShipType = "hospital"
)

// ToDisplay 舰船类型展示用名称
func (t ShipType) ToDisplay() string {
	switch t {
	case ShipTypeAircraftCarrier:
		return i18n.Text(i18n.MsgShipTypeCarrier)
	case ShipTypeBattleShip:
		return i18n.Text(i18n.MsgShipTypeBattleship)
	case ShipTypeCruiser:
		return i18n.Text(i18n.MsgShipTypeCruiser)
	case ShipTypeDestroyer:
		return i18n.Text(i18n.MsgShipTypeDestroyer)
	case ShipTypeFrigate:
		return i18n.Text(i18n.MsgShipTypeFrigate)
	case ShipTypeHospital:
		return i18n.Text(i18n.MsgShipTypeHospital)
	case ShipTypeCargo:
		return i18n.Text(i18n.MsgShipTypeCargo)
	case ShipTypeTorpedoBoat:
		return i18n.Text(i18n.MsgShipTypeTorpedoBoat)
	case ShipTypeDefault:
		return i18n.Text(i18n.MsgShipTypeDefault)
	}
	return i18n.Text(i18n.MsgShipTypeDefault)
}

// HospitalShipEffectRange 医疗船效果范围（地图三格）
const HospitalShipEffectRange = 3

// BattleShip 战舰
type BattleShip struct {
	// 名称
	Name string `json:"name"`
	// 国籍
	Nation Nation `json:"nation"`
	// 类别
	Type ShipType `json:"type"`
	// 类别缩写
	TypeAbbr string `json:"typeAbbr"`
	// 年份
	Year int `json:"year"`

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
	// 舰体费（不含舰载机）
	FundsCost int64 `json:"fundsCost"`
	// 耗时
	TimeCost int64 `json:"timeCost"`
	// 吨位
	Tonnage float64 `json:"tonnage"`
	// 武器
	Weapon ShipWeapon `json:"weapon"`
	// 舰载机联队
	Aircraft ShipAircraft `json:"aircraft"`
	// 舰船动画
	Animation ShipAnimation `json:"animation"`
	// 战力评估（配置与武器初始化完成后计算）
	CombatPower CombatPowerInfo `json:"-"`

	// 唯一标识
	Uid string
	// 当前生命值
	CurHP float64
	// 当前位置
	CurPos objPos.MapPos
	// 旋转角度
	CurRotation float64
	// 当前速度
	CurSpeed float64
	// 动画累计模拟帧
	AnimationAge float64
	// 上一次生成专用尾流时的采样步，用于避免每帧重复堆叠
	LastTrailAnimationStep int
	// 分组ID
	GroupID object.GroupID
	// 攻击目标（敌舰 Uid）
	AttackTarget string

	// 所属阵营（玩家）
	BelongPlayer faction.Player
	// 上次治疗的 Unix 毫秒时间戳，用于计算固定间隔（仅医疗船使用）
	LastHealAt int64
}

var _ Hurtable = (*BattleShip)(nil)

var _ Attacker = (*BattleShip)(nil)

// ID 唯一标识
func (s *BattleShip) ID() string {
	return s.Uid
}

// Detail 详细信息
func (s *BattleShip) Detail() string {
	return fmt.Sprintf(
		"Ship %s(%s): Pos: %s, Rotation: %.2f, Speed: %.2f/%.2f, HP: %.2f/%.2f",
		s.Name, s.Uid, s.CurPos.String(), s.CurRotation, s.CurSpeed, s.MaxSpeed, s.CurHP, s.TotalHP,
	)
}

// Player 所属玩家
func (s *BattleShip) Player() faction.Player {
	return s.BelongPlayer
}

// ObjType 对象类型
func (s *BattleShip) ObjType() object.Type {
	return object.TypeShip
}

// MovementState 机动状态（速度，方向，位置等信息）
func (s *BattleShip) MovementState() UnitMovementState {
	return UnitMovementState{
		CurPos:      s.CurPos.Copy(),
		CurRotation: s.CurRotation,
		CurSpeed:    s.CurSpeed,
	}
}

// GeometricSize 几何尺寸（长、宽等信息）
func (s *BattleShip) GeometricSize() UnitGeometricSize {
	return UnitGeometricSize{Length: s.Length, Width: s.Width}
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
	if t == WeaponTypeAll || t == WeaponTypeAntiAircraftGun {
		for i := 0; i < len(s.Weapon.AntiAircraftGuns); i++ {
			s.Weapon.AntiAircraftGuns[i].Disable = true
		}
		s.Weapon.AntiAircraftGunDisabled = true
	}
	if t == WeaponTypeAll || t == WeaponTypeTorpedo {
		for i := 0; i < len(s.Weapon.Torpedoes); i++ {
			s.Weapon.Torpedoes[i].Disable = true
		}
		s.Weapon.TorpedoDisabled = true
	}
	if t == WeaponTypeAll || t == WeaponTypeRocket {
		for i := 0; i < len(s.Weapon.Rockets); i++ {
			s.Weapon.Rockets[i].Disable = true
		}
		s.Weapon.RocketDisabled = true
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
	if t == WeaponTypeAll || t == WeaponTypeAntiAircraftGun {
		for i := 0; i < len(s.Weapon.AntiAircraftGuns); i++ {
			s.Weapon.AntiAircraftGuns[i].Disable = false
		}
		s.Weapon.AntiAircraftGunDisabled = false
	}
	if t == WeaponTypeAll || t == WeaponTypeTorpedo {
		for i := 0; i < len(s.Weapon.Torpedoes); i++ {
			s.Weapon.Torpedoes[i].Disable = false
		}
		s.Weapon.TorpedoDisabled = false
	}
	if t == WeaponTypeAll || t == WeaponTypeRocket {
		for i := 0; i < len(s.Weapon.Rockets); i++ {
			s.Weapon.Rockets[i].Disable = false
		}
		s.Weapon.RocketDisabled = false
	}
}

// Attack 攻击指定目标
func (s *BattleShip) Attack(shipUid string) {
	s.AttackTarget = shipUid
}

// Fire 向指定目标发射武器
func (s *BattleShip) Fire(enemy Hurtable) (shotBullets []*objBullet.Bullet) {
	// 如果生命值为 0，那还 Fire 个锤子，直接返回
	if s.CurHP <= 0 {
		return
	}
	for _, gun := range s.Weapon.MainGuns {
		shotBullets = append(shotBullets, gun.Fire(s, enemy)...)
	}
	for _, gun := range s.Weapon.SecondaryGuns {
		shotBullets = append(shotBullets, gun.Fire(s, enemy)...)
	}
	for _, gun := range s.Weapon.AntiAircraftGuns {
		shotBullets = append(shotBullets, gun.Fire(s, enemy)...)
	}
	for _, tp := range s.Weapon.Torpedoes {
		shotBullets = append(shotBullets, tp.Fire(s, enemy)...)
	}
	for _, rocket := range s.Weapon.Rockets {
		shotBullets = append(shotBullets, rocket.Fire(s, enemy)...)
	}
	return shotBullets
}

// HurtBy 受到伤害
func (s *BattleShip) HurtBy(bullet *objBullet.Bullet) {
	realDamage := 0.0
	if bullet.ShotType == objBullet.ShotTypeDirect {
		// 平射打击水平装甲带
		realDamage = bullet.Damage * (1 - s.HorizontalDamageReduction)
	} else {
		// 曲射打击垂直装甲带
		realDamage = bullet.Damage * (1 - s.VerticalDamageReduction)
	}

	// 暴击伤害的机制，一发大口径可能直接起飞，支持多段暴击
	criticalType := objBullet.CriticalTypeNone
	randVal := rand.Float64()
	if randVal < bullet.CriticalRate/10 {
		realDamage *= 10
		criticalType = objBullet.CriticalTypeTenTimes
	} else if randVal < bullet.CriticalRate {
		realDamage *= 3
		criticalType = objBullet.CriticalTypeThreeTimes
	}

	// 计算生命值 & 累计伤害
	s.CurHP = max(0, s.CurHP-realDamage)
	// 弹药是可以造成重复伤害的，这里需要计算累计值，暴击类型统计，只统计最高倍数
	bullet.RealDamage += realDamage
	bullet.CriticalType = max(criticalType, bullet.CriticalType)
}

// GenTrails 生成尾流
func (s *BattleShip) GenTrails() []*objTrail.Trail {
	if s.CurSpeed <= 0 {
		return nil
	}
	// 水滴应该是特殊的尾流（蓝色光尾流，负扩散）
	if s.TypeAbbr == "WaterDrop" {
		return []*objTrail.Trail{
			objTrail.New(
				s.CurPos, textureImg.TrailShapeRect,
				(0.4+(s.CurSpeed/s.MaxSpeed))*s.Width*0.5, -2,
				s.Length/6+150*s.CurSpeed, 5,
				0, s.CurRotation, colorx.SkyBlue,
			),
		}
	} else if s.TypeAbbr == "Swordfish" {
		return s.genSwordfishTrails()
	} else if s.TypeAbbr == "Molamola" {
		// 翻车鱼暂时不提供尾流
		return []*objTrail.Trail{}
	}

	offset := s.Length / constants.MapBlockSize
	sinVal := math.Sin(s.CurRotation * math.Pi / 180)
	cosVal := math.Cos(s.CurRotation * math.Pi / 180)

	frontPos, backPos := s.CurPos.Copy(), s.CurPos.Copy()
	frontPos.AddRx(sinVal * offset * 0.25)
	frontPos.SubRy(cosVal * offset * 0.25)
	backPos.SubRx(sinVal * offset * 0.2)
	backPos.AddRy(cosVal * offset * 0.2)

	return []*objTrail.Trail{
		objTrail.New(
			frontPos, textureImg.TrailShapeCircle,
			s.Width*0.6, 1.1,
			s.Length/8+555*s.CurSpeed, 1,
			0, 0, nil,
		),
		objTrail.New(
			backPos, textureImg.TrailShapeCircle,
			s.Width, 0.6,
			s.Length/9+380*s.CurSpeed, 1.5,
			0, 0, nil,
		),
	}
}

// genSwordfishTrails 模拟鲔形游动在尾鳍后方形成的水体扰动与涡流。
// 剑鱼前部躯干相对稳定，主要由后段身体和新月形尾鳍左右摆动推进，
// 因此尾流只从当前动画帧对应的尾尖位置产生，而不覆盖整个鱼身。
func (s *BattleShip) genSwordfishTrails() []*objTrail.Trail {
	frameCount := len(s.Animation.TopFrames)
	frameTicks := s.Animation.FrameTicks
	if frameCount == 0 || frameTicks <= 0 {
		return nil
	}

	// 每个动画帧采样三次，使水迹连续但仍保留尾摆形成的蛇形路径。
	trailStepDuration := max(1.0, frameTicks/3)
	trailStep := int(s.AnimationAge / trailStepDuration)
	if trailStep == s.LastTrailAnimationStep {
		return nil
	}
	s.LastTrailAnimationStep = trailStep

	cycleTicks := frameTicks * float64(frameCount)
	// 动画从左摆极限开始，半个周期后到达右摆极限。
	phase := -math.Cos(2 * math.Pi * s.AnimationAge / cycleTicks)

	speedRatio := 0.0
	if s.MaxSpeed > 0 {
		speedRatio = max(0, min(1, s.CurSpeed/s.MaxSpeed))
	}
	sinVal := math.Sin(s.CurRotation * math.Pi / 180)
	cosVal := math.Cos(s.CurRotation * math.Pi / 180)

	tailPos := s.CurPos.Copy()
	rearOffset := s.Length / constants.MapBlockSize * 0.48
	tailPos.SubRx(sinVal * rearOffset)
	tailPos.AddRy(cosVal * rearOffset)
	lateralOffset := phase * s.Width / constants.MapBlockSize * 0.48
	tailPos.AddRx(cosVal * lateralOffset)
	tailPos.AddRy(sinVal * lateralOffset)

	foamPos := tailPos.Copy()
	foamRearOffset := s.Length / constants.MapBlockSize * 0.025
	foamPos.SubRx(sinVal * foamRearOffset)
	foamPos.AddRy(cosVal * foamRearOffset)
	foamLateralOffset := -phase * s.Width / constants.MapBlockSize * 0.06
	foamPos.AddRx(cosVal * foamLateralOffset)
	foamPos.AddRy(sinVal * foamLateralOffset)

	return []*objTrail.Trail{
		// 低透明度水涡从尾尖扩散，沿尾摆轨迹形成断续的蛇形水迹。
		objTrail.New(
			tailPos, textureImg.TrailShapeCircle,
			3.5+speedRatio*2.5, 0.45+speedRatio*0.2,
			24+speedRatio*24, 4,
			0, 0, colorx.SkyBlue,
		),
		// 少量白色泡沫与主涡错开，表现尾鳍拍水而不是发光推进器。
		objTrail.New(
			foamPos, textureImg.TrailShapeCircle,
			1.5+speedRatio*2, 0.25,
			18+speedRatio*18, 4,
			0, 0, colorx.White,
		),
	}
}

// CanOnLand 能在陆地上
func (s *BattleShip) CanOnLand() bool {
	return s.TypeAbbr == "WaterDrop"
}

// MoveTo 移动到指定位置
func (s *BattleShip) MoveTo(mapCfg *mapcfg.MapCfg, targetPos objPos.MapPos, nearGoal bool) (arrive bool) {
	// 如果生命值为 0，肯定是走不动，直接返回
	if s.CurHP <= 0 {
		return true
	}
	// 差不多到目标位置即可，不要强求准确，否则需要微调，视觉效果不佳
	if s.CurPos.Near(targetPos, 0.6) {
		s.CurSpeed = 0
		return true
	}

	// 应用全局速度倍率
	multiplier := config.G.SpeedMultiplier
	maxSpeed := s.MaxSpeed * multiplier
	acceleration := s.Acceleration * multiplier
	rotateSpeed := s.RotateSpeed * multiplier

	// 未到达目标位置，逐渐加速
	if s.CurSpeed < maxSpeed {
		s.CurSpeed = min(maxSpeed, s.CurSpeed+acceleration)
	}
	// 到目标位置附近，逐渐减速
	if nearGoal && s.CurPos.Near(targetPos, s.Length/constants.MapBlockSize*1.5) {
		s.CurSpeed = max(acceleration*20, s.CurSpeed-acceleration*10)
	}
	targetRotation := s.CurPos.Angle(targetPos)
	// 逐渐转向
	if s.CurRotation != targetRotation {
		// 默认顺时针旋转
		rotateFlag := RotateFlagClockwise
		// 如果逆时针夹角小于顺时针夹角，则需要逆时针旋转
		if math.Mod(targetRotation-s.CurRotation+360, 360) > 180 {
			rotateFlag = RotateFlagAnticlockwise
		}
		s.CurRotation += float64(rotateFlag) * min(math.Abs(targetRotation-s.CurRotation), rotateSpeed)
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
	nextPos.EnsureBorder(float64(mapCfg.Width-2), float64(mapCfg.Height-2))
	// 特殊船舶是可以在陆地上的（飞起来的那些）
	if nearGoal && mapCfg.Map.IsLand(nextPos.MX, nextPos.MY) && !s.CanOnLand() {
		s.CurSpeed = 0
		return true
	}
	// 移动到新位置
	s.CurPos = nextPos

	return false
}

// ShipMap 保存按配置名称索引的舰船模板。
var ShipMap = map[string]*BattleShip{}

// AllShipNames 保存可用舰船模板名称，顺序由配置初始化过程确定。
var AllShipNames = []string{}

// NewShip 新建战舰
func NewShip(
	uidGenerator *ShipUidGenerator, name string, pos objPos.MapPos, rotation float64, player faction.Player,
) *BattleShip {
	// FIXME-P1 小黄鸭太 Bug 了，需要默认禁用一些武器
	s := deepcopy.Copy(*ShipMap[name]).(BattleShip)
	s.Uid = uidGenerator.Gen(s.TypeAbbr)
	s.CurPos = pos
	s.CurRotation = rotation
	s.LastTrailAnimationStep = -1
	s.BelongPlayer = player
	// 战舰默认不编组
	s.GroupID = object.GroupIDNone
	return &s
}

// GetShipDisplayName 获取战舰展示用名称
func GetShipDisplayName(name string) string {
	if ref := objRef.GetReference(name); ref != nil {
		return ref.DisplayName
	}
	return name
}

// GetShipCost 获取战舰成本（舰体费 + 舰载机费）。
func GetShipCost(name string) (fundsCost int64, timeCost int64) {
	ship, ok := ShipMap[name]
	if !ok {
		return 0, 0
	}
	aircraftCost := int64(0)
	for _, group := range ship.Aircraft.Groups {
		if plane, ok := PlaneMap[group.Name]; ok {
			aircraftCost += group.MaxCount * plane.FundsCost
		}
	}
	return ship.FundsCost + aircraftCost, ship.TimeCost
}

// GetShipHullCost 获取战舰舰体成本（不含舰载机）。
func GetShipHullCost(name string) (fundsCost int64, timeCost int64) {
	ship, ok := ShipMap[name]
	if !ok {
		return 0, 0
	}
	return ship.FundsCost, ship.TimeCost
}

// ShipUidGenerator 按玩家和舰种缩写生成局内稳定递增的舰船 UID。
type ShipUidGenerator struct {
	player  faction.Player
	counter map[string]int
}

// NewShipUidGenerator 创建指定玩家独立使用的舰船 UID 生成器。
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
