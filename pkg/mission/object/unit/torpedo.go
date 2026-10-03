package unit

import (
	"log"
	"math"

	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// torpedoFireRangeRatio 预留 20% 最大射程作为敌舰规避缓冲，避免极限距离发射。
const torpedoFireRangeRatio = 0.8

const (
	// 鱼雷扇形按目标类型使用不同的基础角度和整舰最大跨度。
	torpedoFanBaseAngleFast    = 4.0
	torpedoFanBaseAngleCruiser = 3.0
	torpedoFanBaseAngleCapital = 2.0

	torpedoFanMaxSpanFast    = 24.0
	torpedoFanMaxSpanCruiser = 18.0
	torpedoFanMaxSpanCapital = 12.0
)

// TorpedoLauncher 表示舰船鱼雷发射器的配置和局内装填状态。
type TorpedoLauncher struct {
	// 发射器名称
	Name string `json:"name"`
	// 鱼雷类型
	BulletName string `json:"bulletName"`
	// 鱼雷数量
	BulletCount int `json:"bulletCount"`
	// 发射间隔（单位: s）
	ShotInterval float64 `json:"shotInterval"`
	// 装填时间（单位: s）
	ReloadTime float64 `json:"reloadTime"`
	// 射程
	Range float64 `json:"range"`
	// 鱼雷速度
	BulletSpeed float64 `json:"bulletSpeed"`
	// 造价
	FundsCost int64 `json:"fundsCost"`
	// 相对位置
	// 0.35 -> 从中心往舰首 35% 舰体长度
	// -0.3 -> 从中心往舰尾 30% 舰体长度
	PosPercent float64
	// 左射界 (180, 360]
	LeftFiringArc FiringArc
	// 右射界 (0, 180]
	RightFiringArc FiringArc

	// 动态参数
	// 当前鱼雷是否可用（如战损 / 禁用）
	Disable         bool
	ReloadStartTick int64
	// 开始装填时间（时间戳）
	ReloadStartAt int64
	// 最近发射的游戏拍
	LatestFireTick int64
	// 最近发射时间（时间戳）
	LatestFireAt int64
	// 本次装填鱼雷已发射数量
	ShotCountBeforeReload int

	// 该发射器第一根鱼雷在全舰鱼雷管序列中的槽位
	FanSlot int `json:"-"`
	// 所属舰船的鱼雷管总槽位数
	FanSlotCount int `json:"-"`
}

var _ AttackWeapon = (*TorpedoLauncher)(nil)

// Reloaded 是否在重新装填 / 发射间隔
func (lc *TorpedoLauncher) Reloaded() bool {
	if !ticksReady(lc.ReloadStartTick, lc.ReloadTime) {
		return false
	}
	if !ticksReady(lc.LatestFireTick, lc.ShotInterval) {
		return false
	}
	return lc.ShotCountBeforeReload < lc.BulletCount
}

// InShotRange 是否在射程 & 射界内
func (lc *TorpedoLauncher) InShotRange(shipCurRotation float64, curPos, targetPos objPos.MapPos) bool {
	// 不在射程内，不可发射
	if curPos.Distance(targetPos) > lc.Range*torpedoFireRangeRatio {
		return false
	}
	// 不在射界范围内，不可发射
	rotation := math.Mod(curPos.Angle(targetPos)-shipCurRotation+360, 360)
	if !lc.LeftFiringArc.Contains(rotation) && !lc.RightFiringArc.Contains(rotation) {
		return false
	}
	return true
}

// Fire 发射
func (lc *TorpedoLauncher) Fire(shooter Attacker, enemy Hurtable) (bullets []*objBullet.Bullet) {
	// 未启用 / 装填中 / 对象不是战舰，不可发射
	if lc.Disable || !lc.Reloaded() || enemy.ObjType() != object.TypeShip {
		return
	}

	sState, eState := shooter.MovementState(), enemy.MovementState()

	curPos := sState.CurPos.Copy()
	// 炮塔距离战舰中心的距离
	gunOffset := lc.PosPercent * shooter.GeometricSize().Length / constants.MapBlockSize / 2
	curPos.AddRx(math.Sin(sState.CurRotation*math.Pi/180) * gunOffset)
	curPos.SubRy(math.Cos(sState.CurRotation*math.Pi/180) * gunOffset)

	// 应用全局速度倍率
	bulletSpeed := lc.BulletSpeed * config.G.SpeedMultiplier

	// 考虑提前量（依赖敌舰速度，角度）
	_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
		sState.CurPos.RX, sState.CurPos.RY, bulletSpeed,
		eState.CurPos.RX, eState.CurPos.RY, eState.CurSpeed, eState.CurRotation,
	)
	targetPos := objPos.NewR(targetRx, targetRY)

	if !lc.InShotRange(sState.CurRotation, curPos, targetPos) {
		return
	}

	targetType := ShipTypeDefault
	if target, ok := enemy.(*BattleShip); ok {
		targetType = target.Type
	}
	fanAngleOffset := lc.fanAngleOffset(targetType)

	// 鱼雷不是齐射的，是一个一个来的
	lc.ShotCountBeforeReload++

	lc.LatestFireTick = SimTick()
	if lc.LatestFireTick == 0 {
		lc.LatestFireTick = 1
	}
	// 弹药打完了，重新装填
	if lc.ShotCountBeforeReload >= lc.BulletCount {
		lc.ShotCountBeforeReload = 0
		lc.ReloadStartTick = lc.LatestFireTick
	}

	// 鱼雷的生命值就是最大射程（+5 预留）
	life := int(lc.Range/bulletSpeed) + 5

	targetPos = rotateTargetPos(curPos, targetPos, fanAngleOffset)

	// 注：鱼雷只有直射的情况，哪来的曲射？
	return []*objBullet.Bullet{objBullet.New(
		lc.BulletName, curPos, targetPos,
		shooter.ID(), shooter.ObjType(), shooter.Player(),
		enemy.ObjType(), bulletSpeed, life, 0,
	)}
}

// torpedoFanProfile 返回目标类型对应的鱼雷扇形基础步长和最大总跨度。
func torpedoFanProfile(targetType ShipType) (float64, float64) {
	switch targetType {
	case ShipTypeDestroyer, ShipTypeFrigate, ShipTypeTorpedoBoat:
		return torpedoFanBaseAngleFast, torpedoFanMaxSpanFast
	case ShipTypeBattleShip, ShipTypeAircraftCarrier,
		ShipTypeCargo, ShipTypeRepair, ShipTypeHospital:
		return torpedoFanBaseAngleCapital, torpedoFanMaxSpanCapital
	case ShipTypeCruiser, ShipTypeDefault:
		return torpedoFanBaseAngleCruiser, torpedoFanMaxSpanCruiser
	default:
		return torpedoFanBaseAngleCruiser, torpedoFanMaxSpanCruiser
	}
}

// fanAngleOffset 返回当前鱼雷管相对全舰扇形中心的角度偏移。
func (lc *TorpedoLauncher) fanAngleOffset(targetType ShipType) float64 {
	if lc.FanSlotCount <= 1 {
		return 0
	}

	slot := min(max(lc.FanSlot+lc.ShotCountBeforeReload, 0), lc.FanSlotCount-1)
	baseStep, maxSpan := torpedoFanProfile(targetType)
	step := min(baseStep, maxSpan/float64(lc.FanSlotCount-1))
	return (float64(slot) - float64(lc.FanSlotCount-1)/2) * step
}

// rotateTargetPos 以 origin 为中心旋转目标方向，保持原预计命中距离不变。
func rotateTargetPos(origin, target objPos.MapPos, degrees float64) objPos.MapPos {
	if degrees == 0 {
		return target
	}

	distance := origin.Distance(target)
	angle := math.Mod(origin.Angle(target)+degrees+360, 360)
	radians := angle * math.Pi / 180
	return objPos.NewR(
		origin.RX+math.Sin(radians)*distance,
		origin.RY-math.Cos(radians)*distance,
	)
}

// TorpedoLauncherMap 保存按配置名称索引的鱼雷发射器模板。
var TorpedoLauncherMap = map[string]*TorpedoLauncher{}

// NewTorpedoLauncher 从模板创建独立发射器实例，并设置安装位置和左右射界。
func NewTorpedoLauncher(
	name string, posPercent float64,
	leftFireArc, rightFireArc FiringArc,
	fanSlot, fanSlotCount int,
) *TorpedoLauncher {
	launcher, ok := TorpedoLauncherMap[name]
	if !ok {
		log.Fatalf("torpedo launcher %s no found", name)
	}
	lc := deepcopy.Copy(*launcher).(TorpedoLauncher)
	lc.PosPercent = posPercent
	lc.LeftFiringArc = leftFireArc
	lc.RightFiringArc = rightFireArc
	lc.FanSlot = max(fanSlot, 0)
	lc.FanSlotCount = max(fanSlotCount, 1)
	return &lc
}
