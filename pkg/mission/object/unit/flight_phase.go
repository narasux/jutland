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
	// 低空 0.5 倍乘在常规飞机的 2 倍绘制比例上，视觉结果与舰船同级。
	planeLowAltitudeVisualScale = 0.5
	// landingDeckScaleDistanceRatio 是飞机缩放到最小时已完成的着舰路程比例。
	landingDeckScaleDistanceRatio = 0.8

	// takeoffInitialSpeedRatio 是静止航母上飞机的初始滑跑速度相对最大速度的比例。
	takeoffInitialSpeedRatio = 0.10
	// takeoffLiftoffSpeedRatio 是滑跑离舰时的目标速度相对最大速度的比例；
	// 剩余速度留到爬升段缓慢补满，避免起飞瞬间就逼近全速。
	takeoffLiftoffSpeedRatio = 0.6
	// takeoffAccelerationFrames 是滑跑段 S 曲线加速持续的模拟帧数。
	takeoffAccelerationFrames = 60.0
	// takeoffClimbSpeedStepRate 是爬升段每帧加速相对最大速度的比例，
	// 让离舰后的剩余速度在数十帧内均匀补满。
	takeoffClimbSpeedStepRate = 0.02
	// takeoffClimbLength 是从滑跑起点到转入巡航的总距离（舰长倍数），
	// 包含滑跑与离舰后的直线爬升段，视觉高度在整段距离内渐升到巡航高度。
	takeoffClimbLength = 2.0

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

// takeoffStartPos 返回起飞点滑跑起点的地图坐标。
func takeoffStartPos(ship *BattleShip, point TakeoffPoint) objPos.MapPos {
	// 配置纵向为距舰艏比例，内部以舰中为原点（正方向朝舰艏）
	return carrierRelativePos2D(
		ship,
		carrierLengthInMapBlocks(ship)*(0.5-point.Forward),
		carrierWidthInMapBlocks(ship)*point.Lateral,
	)
}

// takeoffEndPos 返回沿弹射航向滑跑 runLength 个舰长后的升空点。
func takeoffEndPos(start objPos.MapPos, launchRotation, runLength float64) objPos.MapPos {
	radians := launchRotation * math.Pi / 180
	return objPos.NewR(
		start.RX+math.Sin(radians)*runLength,
		start.RY-math.Cos(radians)*runLength,
	)
}

// StartTakeoff 从指定起飞点滑跑起飞，弹射航向 = 舰体航向 + 偏转角。
// 阶段终点延伸到爬升段末端：离舰后继续沿弹射航向直线爬升，
// 视觉高度在整段距离内渐升，避免滑跑一结束就进入满高巡航。
func (p *Plane) StartTakeoff(ship *BattleShip, point TakeoffPoint) {
	startPos := takeoffStartPos(ship, point)
	launchRotation := normalizeAngle(ship.CurRotation + point.LaunchAngle)
	length := carrierLengthInMapBlocks(ship)
	endPos := takeoffEndPos(
		startPos,
		launchRotation,
		length*max(point.RunLength, takeoffClimbLength),
	)
	p.takeoffRunLength = length * point.RunLength

	p.CurPos = startPos
	p.CurRotation = launchRotation
	p.FlightPhase = PlaneFlightPhaseTakingOff
	p.FlightPhaseStartPos = startPos
	p.FlightPhaseEndPos = endPos
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

// UpdateTakeoff 分两段推进：滑跑段用 S 曲线加速到离舰速度，
// 离舰后的爬升段按单帧变速限制缓慢加满，飞完爬升距离后进入巡航。
func (p *Plane) UpdateTakeoff(mapCfg *mapcfg.MapCfg) bool {
	if p.FlightPhase != PlaneFlightPhaseTakingOff {
		return p.IsCruising()
	}
	if p.CurHP <= 0 {
		return false
	}

	multiplier := gameSpeedMultiplier()
	maxSpeed := p.MaxSpeed * multiplier
	startSpeed := min(p.FlightPhaseStartSpeed, maxSpeed)
	if p.FlightPhaseStartPos.Distance(p.CurPos) < p.takeoffRunLength {
		// 滑跑段：S 曲线起步平缓，加速目标为离舰速度而非全速
		p.FlightPhaseElapsed += multiplier
		timeProgress := clamp01(p.FlightPhaseElapsed / takeoffAccelerationFrames)
		p.CurSpeed = startSpeed +
			(maxSpeed*takeoffLiftoffSpeedRatio-startSpeed)*smoothstep(timeProgress)
	} else {
		// 爬升段：离舰后继续沿弹射航向直线爬升，剩余速度均匀补满
		p.CurSpeed = moveSpeedToward(
			p.CurSpeed, maxSpeed, maxSpeed*takeoffClimbSpeedStepRate*multiplier,
		)
	}
	p.forward(mapCfg, p.CurRotation, p.CurSpeed)
	if p.FlightPhaseProgress() >= 1 {
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
		// 着舰段按实际滑跑路程缩放：路程 80% 处降到最低视觉倍率（视为触舰/触水）
		progress = smoothstep(clamp01(progress / landingDeckScaleDistanceRatio))
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
