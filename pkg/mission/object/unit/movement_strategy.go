package unit

import (
	"math"

	"github.com/narasux/jutland/pkg/config"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// MovementStrategy 飞机移动策略接口
// 不同类型的飞机有不同的移动行为：
// - 战斗机：需要追踪敌机，接近时减速避免飞过头
// - 轰炸机/鱼雷机：以最大速度飞向目标区域
type MovementStrategy interface {
	// MoveTo 移动到指定位置
	// 参数:
	//   - plane: 飞机对象
	//   - mapCfg: 地图配置
	//   - targetPos: 目标位置（提前量位置）
	//   - enemyPos: 敌人当前位置（用于战斗机计算追踪距离）
	//   - targetSpeed: 目标速度（用于战斗机追踪时调整自身速度）
	MoveTo(plane *Plane, mapCfg *mapcfg.MapCfg, targetPos, enemyPos objPos.MapPos, targetSpeed float64)
}

// NewMovementStrategy 根据飞机类型创建对应的移动策略
func NewMovementStrategy(planeType PlaneType) MovementStrategy {
	switch planeType {
	case PlaneTypeFighter:
		return &FighterPursuitStrategy{}
	default:
		return &BasicMovementStrategy{}
	}
}

// BasicMovementStrategy 基础移动策略
// 适用于俯冲/水平轰炸机和鱼雷轰炸机，始终以最大速度飞行
type BasicMovementStrategy struct{}

// MoveTo 实现基础移动策略，始终以最大速度飞行
func (s *BasicMovementStrategy) MoveTo(
	plane *Plane, mapCfg *mapcfg.MapCfg, targetPos, _ objPos.MapPos, _ float64,
) {
	if plane.CurHP <= 0 {
		return
	}
	maxSpeed := plane.MaxSpeed * config.G.SpeedMultiplier
	executePlaneMovement(plane, mapCfg, targetPos, maxSpeed)
}

// FighterPursuitStrategy 战斗机追踪策略
// 根据武器提前点追踪目标，并按距离动态调整速度，避免飞过头。
type FighterPursuitStrategy struct{}

// tailDistance 目标咬尾距离（地图坐标单位）
// 战斗机会尝试维持这个距离跟在敌机后面
const tailDistance = 0.7

// tailApproachDistance 开始减速的距离
// 当距离小于此值时，战斗机开始从全速切换到追踪速度控制
const tailApproachDistance = 1.0

// stallSpeedRatio 失速速度比例
// 飞机速度不得低于最大速度的此比例，否则会失速
const stallSpeedRatio = 0.5

// MoveTo 实现战斗机追踪策略
// 在远距离时全速追击，进入追踪距离后减速贴近。
// targetPos 使用实际武器弹速计算，机头对准该点即可同时满足前置机炮射界。
func (s *FighterPursuitStrategy) MoveTo(
	plane *Plane, mapCfg *mapcfg.MapCfg, targetPos, enemyPos objPos.MapPos, targetSpeed float64,
) {
	if plane.CurHP <= 0 {
		return
	}
	maxSpeed := plane.MaxSpeed * config.G.SpeedMultiplier
	// 使用敌人当前位置计算距离（而非提前量位置），确保减速逻辑正确
	distance := plane.CurPos.Distance(enemyPos)
	speed := adjustSpeedForPlanePursuit(distance, maxSpeed, targetSpeed)
	executePlaneMovement(plane, mapCfg, targetPos, speed)
}

// adjustSpeedForPlanePursuit 根据距离调整战斗机速度
//
// 使用"目标咬尾距离"机制，避免速度震荡：
//   - 距离 > tailApproachDistance：全速追击
//   - tailDistance < 距离 <= tailApproachDistance：按比例从全速过渡到目标速度
//   - 距离 <= tailDistance：比目标速度慢 10%，自然拉开到咬尾距离
//
// 这样战斗机会自然收敛到 tailDistance 附近：太近就减速被拉开，拉开到咬尾距离就匹配速度
//
// 注意：所有分支最终都受失速速度限制（maxSpeed * stallSpeedRatio），
// 防止战斗机以极低速度飞行导致不符合物理规律的失速现象
func adjustSpeedForPlanePursuit(distance, maxSpeed, targetSpeed float64) float64 {
	// 失速速度：飞机速度不得低于此值
	stallSpeed := maxSpeed * stallSpeedRatio

	// 如果目标速度比战斗机快，始终以最大速度追逐
	if targetSpeed > maxSpeed {
		return maxSpeed
	}
	// 远距离：全速追击
	if distance > tailApproachDistance {
		return maxSpeed
	}
	// 已经在咬尾距离以内：比敌机慢 10%，自然被拉开到咬尾距离
	if distance <= tailDistance {
		return stallSpeed
	}
	// 在 tailDistance ~ tailApproachDistance 之间：按比例过渡
	// distance=tailDistance → targetSpeed（匹配敌机速度）
	// distance=tailApproachDistance → maxSpeed（全速追击）
	ratio := (distance - tailDistance) / (tailApproachDistance - tailDistance)
	return max(stallSpeed, targetSpeed+(maxSpeed-targetSpeed)*ratio)
}

// executePlaneMovement 执行飞机移动（公共逻辑）
// 参数 speed 是已经计算好的当前帧速度
func executePlaneMovement(plane *Plane, mapCfg *mapcfg.MapCfg, targetPos objPos.MapPos, speed float64) {
	rotateSpeed := plane.RotateSpeed * config.G.SpeedMultiplier
	plane.CurSpeed = speed

	// 计算目标航向
	targetRotation := plane.CurPos.Angle(targetPos)
	// 使用最短有符号角差，避免 0° / 360° 附近的浮点误差触发大幅反向转弯。
	rotationDelta := math.Mod(targetRotation-plane.CurRotation+540, 360) - 180
	if math.Abs(rotationDelta) > 1e-9 {
		rotationDelta = max(-rotateSpeed, min(rotateSpeed, rotationDelta))
		plane.CurRotation = math.Mod(plane.CurRotation+rotationDelta+360, 360)
	}

	// 更新位置
	nextPos := plane.CurPos.Copy()
	nextPos.AddRx(math.Sin(plane.CurRotation*math.Pi/180) * plane.CurSpeed)
	nextPos.SubRy(math.Cos(plane.CurRotation*math.Pi/180) * plane.CurSpeed)
	if mapCfg != nil {
		nextPos.EnsureBorder(float64(mapCfg.Width-2), float64(mapCfg.Height-2))
	}
	plane.CurPos = nextPos
	plane.RemainRange -= plane.CurSpeed
}
