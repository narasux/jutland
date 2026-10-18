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

	// torpedoSpreadRatio 相邻两条航迹在目标处的横向间距，按目标长度取值。
	// 取 1/2：横穿的目标一定会被两条相邻航迹夹住，直行的目标由中路那根负责。
	torpedoSpreadRatio = 0.5
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

	// FanTrack 本发射器在所属扇形组里的序号（组内第几座发射器）
	FanTrack int `json:"-"`
	// FanStride 所属扇形组内的发射器数量。同一组的发射器交错分航迹：
	// 第 m 座的第 t 根鱼雷走航迹 t*FanStride + m，使每拍齐射拿到相邻航迹。
	FanStride int `json:"-"`
	// FanTracks 所属扇形组的鱼雷管总数，决定扇面步长与最大跨度。
	FanTracks int `json:"-"`
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

	// 考虑提前量
	// 鱼雷慢、飞行时间长，目标的航速在这段时间里往往还在变化：刚接到命令的舰船
	// 正在起步加速，AI 每次换航路也会重排一次速度。只取发射瞬间的航速会把落点
	// 算近，整条扇面甩在目标身后——直行的目标反而永远打不到。这里按"目标继续以
	// 当前加速度加速、最多到巡航速度"预测，并从发射管位置量距离。
	targetPos, ok := torpedoLeadPoint(curPos, enemy, bulletSpeed, lc.Range/bulletSpeed)
	if !ok {
		// 目标不是战舰（正常不会走到这里）：退回通用的等速提前量算法。
		_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
			curPos.RX, curPos.RY, bulletSpeed,
			eState.CurPos.RX, eState.CurPos.RY, eState.CurSpeed, eState.CurRotation,
		)
		targetPos = objPos.NewR(targetRx, targetRY)
	}

	if !lc.InShotRange(sState.CurRotation, curPos, targetPos) {
		return
	}

	targetType := ShipTypeDefault
	targetLengthCells := 0.0
	if target, ok := enemy.(*BattleShip); ok {
		targetType = target.Type
		targetLengthCells = target.Length / constants.MapBlockSize
	}
	// 发射点到预计命中点的距离：扇面的横向间距就是在这一段距离上折算成角度的。
	aimDistance := curPos.Distance(targetPos)
	fanAngleOffset := lc.fanAngleOffset(targetType, aimDistance, targetLengthCells)

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

// torpedoLeadPoint 计算鱼雷的提前量落点：目标保持当前航向、从当前航速继续加速
// （最多到巡航速度）时，鱼雷从 origin 出发、以 bulletSpeed 飞行，两者相遇的位置。
// maxTicks 是鱼雷的寿命拍数；目标比鱼雷快、追不上时退回目标当前位置。
func torpedoLeadPoint(
	origin objPos.MapPos, enemy Hurtable, bulletSpeed, maxTicks float64,
) (objPos.MapPos, bool) {
	target, ok := enemy.(*BattleShip)
	if !ok || bulletSpeed <= 0 {
		return objPos.MapPos{}, false
	}

	ticks := 0.0
	for i := 0; i < 64; i++ {
		next := origin.Distance(target.LeadPosAt(ticks)) / bulletSpeed
		if maxTicks > 0 && next >= maxTicks {
			return target.CurPos, true
		}
		if math.Abs(next-ticks) < 1e-4 {
			ticks = next
			break
		}
		ticks = next
	}
	return target.LeadPosAt(ticks), true
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

// fanAngleOffset 返回这一根鱼雷相对瞄准线的角度偏移。
//
// 组内航迹从中路往外排（倍数 0, +1, -1, +2, -2, …），所以不论管数奇偶，每拍齐射
// 里总有一根压在瞄准线上——编号 0 的航迹就是瞄准线本身。旧实现按
// (slot - (n-1)/2) 排，偶数组只有 ±半步，正对目标的航迹永远空缺。
//
// 步长取三者较小：目标类型的基础角、扇面最大跨度预算、以及“目标处的航迹间距为
// 半个目标长度”对应的角度。目标越远角度越小，扇面不会在远距离散成一片。
func (lc *TorpedoLauncher) fanAngleOffset(
	targetType ShipType, aimDistance, targetLengthCells float64,
) float64 {
	if lc.FanTracks <= 1 {
		return 0
	}

	baseStep, maxSpan := torpedoFanProfile(targetType)
	outermost := maxFanTrackMultiplier(lc.FanTracks)
	if outermost <= 0 {
		return 0
	}
	step := min(baseStep, maxSpan/(2*float64(outermost)))
	if aimDistance > 0 && targetLengthCells > 0 {
		interval := targetLengthCells * torpedoSpreadRatio
		step = min(step, math.Atan2(interval, aimDistance)*180/math.Pi)
	}

	track := lc.FanTrack + lc.ShotCountBeforeReload*lc.FanStride
	return fanTrackMultiplier(track) * step
}

// fanTrackMultiplier 航迹序号 → 以瞄准线为 0 的倍数：0, +1, -1, +2, -2, …
func fanTrackMultiplier(track int) float64 {
	if track <= 0 {
		return 0
	}
	magnitude := float64((track + 1) / 2)
	if track%2 == 1 {
		return magnitude
	}
	return -magnitude
}

// maxFanTrackMultiplier 组内最外缘航迹的倍数绝对值。
func maxFanTrackMultiplier(tracks int) int {
	if tracks <= 1 {
		return 0
	}
	return tracks / 2
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

// NewTorpedoLauncher 从模板创建独立发射器实例，并设置安装位置、左右射界和所属扇形组。
// fanTrack / fanStride / fanTracks 由初始化阶段按舷别分组后给出：同一舷的发射器共用
// 一张以瞄准线为中心的扇面，中轴、左舷、右舷各自成组。
func NewTorpedoLauncher(
	name string, posPercent float64,
	leftFireArc, rightFireArc FiringArc,
	fanTrack, fanStride, fanTracks int,
) *TorpedoLauncher {
	launcher, ok := TorpedoLauncherMap[name]
	if !ok {
		log.Fatalf("torpedo launcher %s no found", name)
	}
	lc := deepcopy.Copy(*launcher).(TorpedoLauncher)
	lc.PosPercent = posPercent
	lc.LeftFiringArc = leftFireArc
	lc.RightFiringArc = rightFireArc
	lc.FanTrack = max(fanTrack, 0)
	lc.FanStride = max(fanStride, 1)
	lc.FanTracks = max(fanTracks, 1)
	return &lc
}
