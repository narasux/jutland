package unit

import (
	"math"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// PlaneFlightPhase 表示飞机在局内的起降 / 巡航阶段。
// 阶段只描述飞机当前允许执行的行为；具体着舰几何由 landing_geometry.go 负责。
type PlaneFlightPhase string

const (
	// PlaneFlightPhaseTakingOff 起飞直线爬升阶段。
	PlaneFlightPhaseTakingOff PlaneFlightPhase = "taking_off"
	// PlaneFlightPhaseCruising 巡航 / 交战阶段。
	PlaneFlightPhaseCruising PlaneFlightPhase = "cruising"
	// PlaneFlightPhaseLandingStaging 飞向舰尾直线待进近通道的阶段。
	PlaneFlightPhaseLandingStaging PlaneFlightPhase = "landing_staging"
	// PlaneFlightPhaseLandingApproach 沿定半径圆弧汇入舰尾中线的阶段。
	PlaneFlightPhaseLandingApproach PlaneFlightPhase = "landing_approach"
	// PlaneFlightPhaseLandingDeck 最终直线进近与甲板回收阶段。
	PlaneFlightPhaseLandingDeck PlaneFlightPhase = "landing_deck"
)

const (
	// carrierTakeoffStartOffsetRatio 表示滑跑起点相对航母中心的舰长倍率，0 即甲板中部。
	carrierTakeoffStartOffsetRatio = 0
	// carrierTakeoffLaneRatio 表示飞机保持航母航向直飞的最短滑跑距离，单位为舰长。
	carrierTakeoffLaneRatio = 2.0

	// 低空 0.5 倍乘在常规飞机的 2 倍绘制比例上，视觉结果与舰船同级。
	planeLowAltitudeVisualScale = 0.5
	// landingDeckScaleDistanceRatio 是飞机缩放到最小时已完成的着舰路程比例。
	landingDeckScaleDistanceRatio = 0.8

	// takeoffInitialSpeedRatio 是静止航母上飞机的初始滑跑速度相对最大速度的比例。
	takeoffInitialSpeedRatio = 0.15
	// takeoffAccelerationFrames 是起飞 S 曲线加速持续的模拟帧数。
	takeoffAccelerationFrames = 45.0

	// phaseSpeedStepFallbackRate 是机型未配置加速度时的单帧保底变速比例。
	phaseSpeedStepFallbackRate = 0.04
	// phaseSpeedStepMaxRate 限制单帧速度变化，防止阶段切换时速度突跳。
	phaseSpeedStepMaxRate = 0.08
)

// carrierLengthInMapBlocks 将资源像素长度换算为地图坐标长度，并避免零长度参与比例计算。
func carrierLengthInMapBlocks(ship *BattleShip) float64 {
	return max(ship.Length/constants.MapBlockSize, 0.1)
}

// gameSpeedMultiplier 同时缩放位移和阶段时间，保证游戏倍速不会改变轨迹形状。
func gameSpeedMultiplier() float64 {
	if config.G == nil {
		return 1
	}
	return config.G.SpeedMultiplier
}

// carrierTakeoffStartPos 返回甲板滑跑起点的地图坐标。
func carrierTakeoffStartPos(ship *BattleShip) objPos.MapPos {
	return carrierRelativePos(ship, carrierLengthInMapBlocks(ship)*carrierTakeoffStartOffsetRatio)
}

// carrierTakeoffEndPos 返回最短直线滑跑距离的终点；达到后仍需等待加速阶段完成。
func carrierTakeoffEndPos(ship *BattleShip) objPos.MapPos {
	return carrierRelativePos(
		ship,
		carrierLengthInMapBlocks(ship)*(carrierTakeoffStartOffsetRatio+carrierTakeoffLaneRatio),
	)
}

// StartTakeoff 从甲板中部开始滑跑，初始航向与航母一致。
func (p *Plane) StartTakeoff(ship *BattleShip) {
	startPos := carrierTakeoffStartPos(ship)
	p.CurPos = startPos
	p.CurRotation = ship.CurRotation
	p.FlightPhase = PlaneFlightPhaseTakingOff
	p.FlightPhaseStartPos = startPos
	p.FlightPhaseEndPos = carrierTakeoffEndPos(ship)
	p.FlightPhaseElapsed = 0
	p.FlightPhaseProgressValue = 0
	p.FlightVisualScaleStart = planeLowAltitudeVisualScale
	p.FlightVisualScaleEnd = 1
	p.CurSpeed = max(ship.CurSpeed, p.MaxSpeed*gameSpeedMultiplier()*takeoffInitialSpeedRatio)
	p.FlightPhaseStartSpeed = p.CurSpeed
}

// IsCruising 返回飞机是否处于可正常接敌 / 开火的巡航阶段。
func (p *Plane) IsCruising() bool {
	return p.FlightPhase == "" || p.FlightPhase == PlaneFlightPhaseCruising
}

// FinishTakeoff 切换到巡航阶段。
func (p *Plane) FinishTakeoff() {
	p.FlightPhase = PlaneFlightPhaseCruising
	p.FlightPhaseProgressValue = 1
	p.FlightVisualScaleStart = 1
	p.FlightVisualScaleEnd = 1
}

// UpdateTakeoff 使用 smoothstep 在固定模拟帧内加速；距离和加速过程都完成后才进入巡航。
func (p *Plane) UpdateTakeoff(mapCfg *mapcfg.MapCfg) bool {
	if p.FlightPhase != PlaneFlightPhaseTakingOff {
		return p.IsCruising()
	}
	if p.CurHP <= 0 {
		return false
	}

	multiplier := gameSpeedMultiplier()
	p.FlightPhaseElapsed += multiplier
	timeProgress := clamp01(p.FlightPhaseElapsed / takeoffAccelerationFrames)
	speedProgress := smoothstep(timeProgress)
	maxSpeed := p.MaxSpeed * multiplier
	startSpeed := min(p.FlightPhaseStartSpeed, maxSpeed)
	p.CurSpeed = startSpeed + (maxSpeed-startSpeed)*speedProgress
	p.forward(mapCfg, p.CurRotation, p.CurSpeed)
	if p.FlightPhaseProgress() >= 1 && timeProgress >= 1 {
		p.FinishTakeoff()
		return true
	}
	return false
}

// VisualScaleMultiplier 返回当前起降阶段相对常规飞机绘制比例的倍率。
func (p *Plane) VisualScaleMultiplier() float64 {
	if p.IsCruising() {
		return 1
	}
	if p.FlightVisualScaleStart <= 0 && p.FlightVisualScaleEnd <= 0 {
		return 1
	}
	progress := p.FlightPhaseProgress()
	if p.FlightPhase == PlaneFlightPhaseLandingDeck {
		distanceProgress := easeOutQuadratic(progress)
		progress = smoothstep(clamp01(distanceProgress / landingDeckScaleDistanceRatio))
	}
	return p.FlightVisualScaleStart + (p.FlightVisualScaleEnd-p.FlightVisualScaleStart)*progress
}

// FlightPhaseProgress 返回当前阶段进度，范围 [0, 1]。
// 起飞按实际滑跑距离计算；圆弧和着舰动画由各阶段直接维护时间进度。
func (p *Plane) FlightPhaseProgress() float64 {
	if p.FlightPhase == PlaneFlightPhaseTakingOff {
		total := p.FlightPhaseStartPos.Distance(p.FlightPhaseEndPos)
		if total <= 0.001 {
			return 1
		}
		return clamp01(p.FlightPhaseStartPos.Distance(p.CurPos) / total)
	}
	return clamp01(p.FlightPhaseProgressValue)
}

// approachSpeed 将当前速度向目标速度推进一步，并复用统一的单帧变速限制。
func (p *Plane) approachSpeed(targetSpeed float64) float64 {
	return moveSpeedToward(p.CurSpeed, targetSpeed, p.phaseSpeedStep(targetSpeed))
}

// phaseSpeedStep 限制单帧速度变化；未配置加速度时使用与最大速度成比例的保底值。
func (p *Plane) phaseSpeedStep(targetSpeed float64) float64 {
	maxSpeed := max(p.MaxSpeed*gameSpeedMultiplier(), targetSpeed)
	if maxSpeed <= 0 {
		return 0
	}
	step := p.Acceleration * gameSpeedMultiplier()
	if step <= 0 {
		step = maxSpeed * phaseSpeedStepFallbackRate
	}
	return min(step, maxSpeed*phaseSpeedStepMaxRate)
}

// moveSpeedToward 在不越过目标值的前提下，将速度增加或减少 step。
func moveSpeedToward(curSpeed, targetSpeed, step float64) float64 {
	if step <= 0 {
		return targetSpeed
	}
	if curSpeed < targetSpeed {
		return min(targetSpeed, curSpeed+step)
	}
	return max(targetSpeed, curSpeed-step)
}

// forward 按游戏角度约定推进飞机，并在提供地图配置时限制到有效边界。
func (p *Plane) forward(mapCfg *mapcfg.MapCfg, rotation, speed float64) {
	nextPos := p.CurPos.Copy()
	nextPos.AddRx(math.Sin(rotation*math.Pi/180) * speed)
	nextPos.SubRy(math.Cos(rotation*math.Pi/180) * speed)
	if mapCfg != nil {
		nextPos.EnsureBorder(float64(mapCfg.Width-2), float64(mapCfg.Height-2))
	}
	p.CurPos = nextPos
	p.CurSpeed = speed
	p.RemainRange -= speed
}

// smoothstep 返回端点一阶导数为 0 的 S 曲线插值进度。
func smoothstep(progress float64) float64 {
	progress = clamp01(progress)
	return progress * progress * (3 - 2*progress)
}

// easeOutQuadratic 返回逐渐减速到终点的二次插值进度。
func easeOutQuadratic(progress float64) float64 {
	progress = clamp01(progress)
	return 1 - (1-progress)*(1-progress)
}

// lerp 在 start 和 end 之间执行线性插值。
func lerp(start, end, progress float64) float64 {
	return start + (end-start)*progress
}

// clamp01 将插值进度限制在 [0, 1]。
func clamp01(value float64) float64 {
	return max(0, min(1, value))
}

// normalizeRadians 将弧度角归一化到 [-pi, pi)。
func normalizeRadians(angle float64) float64 {
	return math.Mod(angle+3*math.Pi, 2*math.Pi) - math.Pi
}

// angleDifferenceDegrees 返回两个角度之间的最小绝对夹角，单位为度。
func angleDifferenceDegrees(left, right float64) float64 {
	return math.Abs(math.Mod(left-right+540, 360) - 180)
}

// rotateAngleToward 沿最短方向转向目标角度，并限制本次最大转角。
func rotateAngleToward(current, target, maxDelta float64) float64 {
	if maxDelta <= 0 {
		return normalizeAngle(target)
	}
	delta := math.Mod(target-current+540, 360) - 180
	return normalizeAngle(current + max(-maxDelta, min(maxDelta, delta)))
}

// normalizeAngle 将角度归一化到 [0, 360)。
func normalizeAngle(angle float64) float64 {
	return math.Mod(angle+360, 360)
}
