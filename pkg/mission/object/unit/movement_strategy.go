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
// 适用于俯冲轰炸机和鱼雷轰炸机，始终以最大速度飞行
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
// 根据与目标的距离动态调整速度，避免飞过头
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

// deflectionOffset 偏转攻击的横向偏移量（地图坐标单位）
// 战斗机在追踪距离内会向侧面偏移此距离，实现偏转射击而非正后方尾追
// 这样战斗机的机头可以指向敌机，使前置机枪进入射界
const deflectionOffset = 0.3

// MoveTo 实现战斗机追踪策略
// 在远距离时全速追击，进入追踪距离后采用偏转攻击：
// 不直接从正后方跟随，而是从侧面切入，让前置机枪能对准敌机
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

	// 偏转攻击：在追踪距离内，给目标位置加横向偏移
	// 战斗机不再从正后方跟随（导致前置机枪无法射击），而是从侧面切入
	// 偏移方向根据 UID 哈希固定为左或右，避免战斗机频繁切换方向
	if distance <= tailApproachDistance {
		targetPos = calcDeflectionTarget(plane, targetPos, enemyPos, distance)
	}

	executePlaneMovement(plane, mapCfg, targetPos, speed)
}

// calcDeflectionTarget 计算偏转攻击的目标位置
// 在原始目标位置（提前量位置）的基础上，加一个垂直于"战斗机→敌机"方向的横向偏移
// 使战斗机从侧面接近，而非正后方尾追
//
// 参数 distance 是调用方已计算好的战斗机到敌机距离，复用以避免重复计算 math.Sqrt
// targetPos 为值传递，可以直接修改而无需 Copy
func calcDeflectionTarget(plane *Plane, targetPos, enemyPos objPos.MapPos, distance float64) objPos.MapPos {
	if distance < 0.001 {
		return targetPos
	}

	// 计算战斗机到敌机的归一化方向向量（复用已有 distance，避免重复 math.Sqrt）
	dx := (enemyPos.RX - plane.CurPos.RX) / distance
	dy := (enemyPos.RY - plane.CurPos.RY) / distance

	// 垂直方向（顺时针旋转90°）：(dx, dy) -> (dy, -dx)
	// 根据 UID 哈希决定偏转方向（左/右），保证同一架飞机始终往同一方向偏
	perpX, perpY := dy, -dx
	if plane.Uid[0]%2 == 0 {
		perpX, perpY = -perpX, -perpY
	}

	// 直接在值类型副本上操作 RX/RY，最后一次性计算 MX/MY
	// 避免 Copy() + 多次 AddRx/AddRy 中重复调用 math.Floor
	targetPos.RX += perpX * deflectionOffset
	targetPos.RY += perpY * deflectionOffset
	targetPos.MX = int(math.Floor(targetPos.RX))
	targetPos.MY = int(math.Floor(targetPos.RY))
	return targetPos
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

	// 逐渐转向
	if plane.CurRotation != targetRotation {
		rotateFlag := RotateFlagClockwise
		if math.Mod(targetRotation-plane.CurRotation+360, 360) > 180 {
			rotateFlag = RotateFlagAnticlockwise
		}
		plane.CurRotation += float64(rotateFlag) * min(math.Abs(targetRotation-plane.CurRotation), rotateSpeed)
		plane.CurRotation = math.Mod(plane.CurRotation+360, 360)
	}

	// 更新位置
	nextPos := plane.CurPos.Copy()
	nextPos.AddRx(math.Sin(plane.CurRotation*math.Pi/180) * plane.CurSpeed)
	nextPos.SubRy(math.Cos(plane.CurRotation*math.Pi/180) * plane.CurSpeed)
	nextPos.EnsureBorder(float64(mapCfg.Width-2), float64(mapCfg.Height-2))
	plane.CurPos = nextPos
	plane.RemainRange -= plane.CurSpeed
}
