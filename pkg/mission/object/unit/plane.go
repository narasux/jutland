package unit

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objRef "github.com/narasux/jutland/pkg/mission/object/reference"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// PlaneType 飞机类型
type PlaneType string

const (
	// PlaneTypeFighter 战斗机
	PlaneTypeFighter PlaneType = "fighter"
	// PlaneTypeDiveBomber 俯冲轰炸机
	PlaneTypeDiveBomber PlaneType = "dive_bomber"
	// PlaneTypeLevelBomber 水平轰炸机
	PlaneTypeLevelBomber PlaneType = "level_bomber"
	// PlaneTypeTorpedoBomber 鱼雷轰炸机
	PlaneTypeTorpedoBomber PlaneType = "torpedo_bomber"
)

// ToDisplay 飞机类型展示用名称。
func (t PlaneType) ToDisplay() string {
	switch t {
	case PlaneTypeFighter:
		return i18n.Text(i18n.MsgPlaneTypeFighter)
	case PlaneTypeDiveBomber:
		return i18n.Text(i18n.MsgPlaneTypeDiveBomber)
	case PlaneTypeLevelBomber:
		return i18n.Text(i18n.MsgPlaneTypeLevelBomber)
	case PlaneTypeTorpedoBomber:
		return i18n.Text(i18n.MsgPlaneTypeTorpedoBomber)
	default:
		return i18n.Text(i18n.MsgUnknown)
	}
}

// AttacksShips 对舰攻击机种：俯冲/水平轰炸机与鱼雷机。
func (t PlaneType) AttacksShips() bool {
	switch t {
	case PlaneTypeDiveBomber, PlaneTypeLevelBomber, PlaneTypeTorpedoBomber:
		return true
	default:
		return false
	}
}

// Plane 战机
type Plane struct {
	// 名称
	Name string `json:"name"`
	// 国籍
	Nation Nation `json:"nation"`
	// 类别
	Type PlaneType `json:"type"`
	// 类别缩写
	TypeAbbr string `json:"typeAbbr"`

	// 服役年份
	Year int `json:"year"`
	// 初始生命值
	TotalHP float64 `json:"totalHP"`
	// 伤害减免（0.7 -> 仅受到击中的 30% 伤害)
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
	// 造价
	FundsCost int64 `json:"fundsCost"`
	// 耗时
	TimeCost int64 `json:"timeCost"`
	// 吨位
	Tonnage float64 `json:"tonnage"`
	// 武器
	Weapon PlaneWeapon `json:"weapon"`
	// 战力评估（配置与武器初始化完成后计算）
	CombatPower CombatPowerInfo `json:"-"`

	// 唯一标识
	Uid string
	// 当前生命值
	CurHP float64
	// 当前位置
	CurPos objPos.MapPos
	// 当前高度 TODO 是否引入高度概念？
	CurHeight float64
	// 旋转角度
	CurRotation float64
	// 当前速度
	CurSpeed float64
	// 剩余航程
	RemainRange float64
	// 当前攻击目标 (uid)
	CurAttackTarget string
	// 飞行阶段（起飞 / 巡航 / 降落）
	FlightPhase PlaneFlightPhase
	// 当前飞行阶段起点
	FlightPhaseStartPos objPos.MapPos
	// 当前飞行阶段终点
	FlightPhaseEndPos objPos.MapPos
	// 当前阶段已经经过的模拟帧数
	FlightPhaseElapsed float64
	// 当前阶段开始时的速度
	FlightPhaseStartSpeed float64
	// 无法直接通过固定起止点计算时使用的阶段进度
	FlightPhaseProgressValue float64
	// 当前飞行阶段起始视觉倍率（相对常规飞机 2 倍绘制）
	FlightVisualScaleStart float64
	// 当前飞行阶段结束视觉倍率（相对常规飞机 2 倍绘制）
	FlightVisualScaleEnd float64
	// 当前飞机在所属航母舰尾入口中的分散槽位
	LandingSlot int
	// 最终进近使用的航母局部坐标圆弧，仅在局内运行时初始化。
	landingArc landingApproachArc
	// 当前正在飞向切线引导点或圆弧入口。
	landingStagingLeg landingStagingLeg
	// 上一模拟帧的航母航向，用于计算移动着舰航线的切向速度。
	landingCarrierRotation float64
	// 航母当前每模拟帧的转向弧度。
	landingCarrierTurnRate float64
	// 本次降落是否为舷侧着水回收（水上飞机）。
	landingOnWater bool
	// 直线进近段降到最低视觉倍率（触舰/触水）时的路程比例。
	landingScaleCompleteRatio float64
	// 起飞滑跑段长度（地图坐标）；滑跑结束后进入直线爬升段缓慢加满速度。
	takeoffRunLength float64
	// 弹射点配置副本：滑跑段绑定甲板推进，每帧按舰体当前姿态重算滑跑线。
	takeoffPoint TakeoffPoint
	// 起飞阶段计划总距离（滑跑 + 爬升，地图坐标）与已飞行距离。
	takeoffTotalLength float64
	takeoffDistance    float64
	// 离舰瞬间合成的世界速度航向，爬升段机头以转向速率平滑过渡到该航向。
	takeoffClimbHeading float64
	// 降落机头合成使用的航母转向速率低通值。玩家满舵时舰体角速度是阶跃信号，
	// 直接合成机头航向会让进近中的飞机在两三帧内甩动数十度；
	// 滤波后阶跃被摊成数十帧的平滑偏航，机头始终贴合随舰旋转的进近航线。
	landingDisplayTurnRate float64

	// 圆弧进近段的等减速计划状态（相对航母速度，单位为地图坐标每模拟帧）。
	landingArcSpeed       float64
	landingArcExitSpeed   float64
	landingArcTotalLength float64
	landingArcDistance    float64
	landingArcDecel       float64
	// 最终直线进近段的等减速刹车计划状态。
	landingRunSpeed       float64
	landingRunTotalLength float64
	landingRunDistance    float64
	landingRunDecel       float64
	// 最终直线进近方向的航母局部单位向量。
	landingRunTangent carrierLocalOffset

	// 所属阵营（玩家）
	BelongPlayer faction.Player
	// 所属基地（uid）：可以是航母，也可以是陆地机场
	BelongShip string

	// 移动策略（根据飞机类型自动设置）
	movementStrategy MovementStrategy
}

var _ Hurtable = (*Plane)(nil)

var _ Attacker = (*Plane)(nil)

// ID 唯一标识
func (p *Plane) ID() string {
	return p.Uid
}

// Detail 详细信息
func (p *Plane) Detail() string {
	return fmt.Sprintf(
		"Plane %s(%s): Pos: %s, Rotation: %.2f, Speed: %.2f/%.2f, HP: %.2f/%.2f, AttackTarget: %s",
		p.Name, p.Uid, p.CurPos.String(), p.CurRotation, p.CurSpeed, p.MaxSpeed, p.CurHP, p.TotalHP, p.CurAttackTarget,
	)
}

// Player 所属玩家
func (p *Plane) Player() faction.Player {
	return p.BelongPlayer
}

// ObjType 对象类型
func (p *Plane) ObjType() object.Type {
	return object.TypePlane
}

// AttackObjType 攻击对象类型
func (p *Plane) AttackObjType() object.Type {
	return GetPlaneTargetObjType(p.Name)
}

// MovementState 机动状态（速度，方向，位置等信息）
func (p *Plane) MovementState() UnitMovementState {
	return UnitMovementState{
		CurPos:      p.CurPos.Copy(),
		CurRotation: p.CurRotation,
		CurSpeed:    p.CurSpeed,
	}
}

// GeometricSize 几何尺寸（长、宽等信息）
func (p *Plane) GeometricSize() UnitGeometricSize {
	return UnitGeometricSize{Length: p.Length, Width: p.Width}
}

// Fire 向指定目标发射武器
func (p *Plane) Fire(enemy Hurtable) (shotBullets []*objBullet.Bullet) {
	// 如果生命值为 0，那还 Fire 个锤子，直接返回
	if p.CurHP <= 0 {
		return
	}
	// 机炮不用记录射击时间
	for i := 0; i < len(p.Weapon.Guns); i++ {
		shotBullets = append(shotBullets, p.Weapon.Guns[i].Fire(p, enemy)...)
	}
	for i := 0; i < len(p.Weapon.Rockets); i++ {
		shotBullets = append(shotBullets, p.Weapon.Rockets[i].Fire(p, enemy)...)
	}
	// 释放器类武器，有最小的释放间隔限制。
	// 炸弹可攻击战舰与地面停放目标；鱼雷只能攻击战舰（无法在陆地使用）。
	canRelease := enemy.ObjType() == object.TypeShip || isGroundHurtable(enemy)
	if canRelease {
		timeNow := time.Now().UnixMilli()
		if float64(timeNow-p.Weapon.LatestReleaseAt) > p.Weapon.ReleaseInterval*1e3 {
			for _, releasers := range p.releaseGroups(enemy) {
				for i := 0; i < len(releasers); i++ {
					if bullets := releasers[i].Fire(p, enemy); len(bullets) > 0 {
						shotBullets = append(shotBullets, bullets...)
						p.Weapon.LatestReleaseAt = timeNow
						break
					}
				}
			}
		}
	}
	return shotBullets
}

// releaseGroups 返回当前目标可用的释放器分组：对舰目标为炸弹 + 鱼雷，
// 对地面停放目标仅炸弹（鱼雷入水即毁，不能攻击陆地目标）。
func (p *Plane) releaseGroups(enemy Hurtable) [2][]*Releaser {
	if enemy.ObjType() == object.TypeShip {
		return [2][]*Releaser{p.Weapon.Bombs, p.Weapon.Torpedoes}
	}
	return [2][]*Releaser{p.Weapon.Bombs, nil}
}

// isGroundHurtable 判断目标是否为地面飞机（停放或滑行中，可被炸弹攻击的地面目标）。
func isGroundHurtable(enemy Hurtable) bool {
	plane, ok := enemy.(*Plane)
	return ok && plane.IsOnGround()
}

// TorpedoPathCrossesLand 返回当前可投放的航空鱼雷航迹是否经过陆地。
func (p *Plane) TorpedoPathCrossesLand(enemy Hurtable, terrain *mapcfg.MapData) bool {
	if p.CurHP <= 0 || enemy.ObjType() != object.TypeShip {
		return false
	}
	for _, torpedo := range p.Weapon.Torpedoes {
		if torpedo.pathCrossesLand(p, enemy, terrain) {
			return true
		}
	}
	return false
}

// HurtBy 受到伤害
func (p *Plane) HurtBy(bullet *objBullet.Bullet) {
	// 计算真实伤害，飞机比较脆，所以伤害要再额外乘以 3
	realDamage := bullet.Damage * (1 - p.DamageReduction) * 3

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
	p.CurHP = max(0, p.CurHP-realDamage)
	// 弹药是可以造成重复伤害的，这里需要计算累计值，暴击类型统计，只统计最高倍数
	bullet.RealDamage += realDamage
	bullet.CriticalType = max(criticalType, bullet.CriticalType)
}

// MoveTo 移动到指定位置
// 参数:
//   - mapCfg: 地图配置
//   - targetPos: 目标位置（提前量位置）
//   - enemyPos: 敌人当前位置（用于战斗机计算追踪距离）
//   - targetSpeed: 目标速度（用于战斗机追踪时调整自身速度，默认为0表示不调整）
func (p *Plane) MoveTo(mapCfg *mapcfg.MapCfg, targetPos, enemyPos objPos.MapPos, targetSpeed float64) {
	// 委托给移动策略处理
	p.movementStrategy.MoveTo(p, mapCfg, targetPos, enemyPos, targetSpeed)
}

// MustReturn 必须返航
func (p *Plane) MustReturn() bool {
	// 如果剩余航程 <= 0，必须返航
	if p.RemainRange <= 0 {
		return true
	}
	// 轰炸机 / 鱼雷机，只要没有进攻武器了，就返航（我滴任务完成啦！）
	if p.Type.AttacksShips() {
		for _, r := range p.Weapon.Bombs {
			if !r.Released {
				return false
			}
		}
		for _, r := range p.Weapon.Torpedoes {
			if !r.Released {
				return false
			}
		}
		for _, r := range p.Weapon.Rockets {
			if !r.Exhausted() {
				return false
			}
		}
		return true
	}
	// 战斗机只能是到没燃油才返航
	return false
}

// PlaneMap 保存按配置名称索引的飞机模板。
var PlaneMap = map[string]*Plane{}

// AllPlaneNames 保存可用飞机模板名称，顺序由配置初始化过程确定。
var AllPlaneNames = []string{}

// NewPlane 生成飞机
func NewPlane(
	name string,
	curPos objPos.MapPos,
	rotation float64,
	shipUid string,
	player faction.Player,
) *Plane {
	plane, ok := PlaneMap[name]
	if !ok {
		log.Fatalf("plane %s no found", name)
	}
	p := deepcopy.Copy(*plane).(Plane)

	p.Uid = uuid.New().String()
	p.CurSpeed = p.MaxSpeed
	p.CurPos = curPos
	p.CurRotation = rotation
	p.BelongPlayer = player
	p.BelongShip = shipUid
	p.FlightPhase = PlaneFlightPhaseCruising
	p.FlightVisualScaleStart = 1
	p.FlightVisualScaleEnd = 1
	p.LandingSlot = -1

	// 根据飞机类型初始化移动策略
	p.movementStrategy = NewMovementStrategy(p.Type)

	return &p
}

// GetPlaneTargetObjType 获取飞机攻击目标类型
// FIXME 目前这样很暴力，会导致战斗机只能打飞机，轰炸机只能打舰船
func GetPlaneTargetObjType(name string) object.Type {
	plane, ok := PlaneMap[name]
	if !ok {
		log.Fatalf("plane %s no found", name)
	}
	// 侦察机等无武装飞机不应被自动派出参与攻击。
	// FIXME 未来会有其他用途
	if len(plane.Weapon.Guns) == 0 && len(plane.Weapon.Bombs) == 0 &&
		len(plane.Weapon.Torpedoes) == 0 && len(plane.Weapon.Rockets) == 0 {
		return object.TypeNone
	}
	// 根据飞机类型获取目标类型
	switch plane.Type {
	case PlaneTypeFighter:
		return object.TypePlane
	case PlaneTypeDiveBomber, PlaneTypeLevelBomber, PlaneTypeTorpedoBomber:
		return object.TypeShip
	default:
		return object.TypeNone
	}
}

// GetPlaneDisplayName 获取战机展示用名称
func GetPlaneDisplayName(name string) string {
	if ref := objRef.GetReference(name); ref != nil {
		return ref.DisplayName
	}
	return name
}

// GetPlaneCost 获取飞机成本
func GetPlaneCost(name string) (fundsCost int64, timeCost int64) {
	plane, ok := PlaneMap[name]
	if !ok {
		return 0, 0
	}
	return plane.FundsCost, plane.TimeCost
}
