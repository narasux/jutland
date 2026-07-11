package unit

import (
	"math"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

const (
	// landingLeadInCrossTrackGain 将入口直线的横向误差转换为每帧修正速度。
	landingLeadInCrossTrackGain = 0.08
	// landingLeadInMaxCorrectionRatio 限制横向修正不超过圆弧相对速度的 25%。
	landingLeadInMaxCorrectionRatio = 0.25
	// landingCatchupClosureRatio 是远距离追赶移动入口时保留的最小闭合速度比例。
	landingCatchupClosureRatio = 0.15
	// landingStagingSlowdownInnerRatio 是近舰速度包络的内边界，单位为舰长。
	landingStagingSlowdownInnerRatio = 4.0
	// landingStagingSlowdownOuterRatio 是开始从追赶速度减速的外边界，单位为舰长。
	landingStagingSlowdownOuterRatio = 6.0
	// landingDeckMinFrames 是最终直线着舰段允许的最短模拟帧数。
	landingDeckMinFrames = 60.0
	// landingDeckMaxFrames 是最终直线着舰段允许的最长模拟帧数。
	landingDeckMaxFrames = 180.0
)

// landingStagingLeg 是 landing_staging 内部的两段引导，不对外增加飞行阶段。
type landingStagingLeg uint8

const (
	// landingStagingLegLeadIn 表示飞机尚在捕获圆弧入口切线后方的远端引导点。
	landingStagingLegLeadIn landingStagingLeg = iota
	// landingStagingLegGate 表示飞机正沿入口切线向圆弧起点收敛。
	landingStagingLegGate
)

// StartLandingStaging 将返航飞机导向地图范围内可达的舰尾进近通道。
func (p *Plane) StartLandingStaging(_ *mapcfg.MapCfg, ship *BattleShip, slot int) {
	p.CurAttackTarget = ""
	p.FlightPhase = PlaneFlightPhaseLandingStaging
	p.FlightPhaseStartPos = p.CurPos.Copy()
	p.FlightPhaseElapsed = 0
	p.FlightPhaseProgressValue = 0
	p.FlightVisualScaleStart = 1
	p.FlightVisualScaleEnd = 1
	p.LandingSlot = slot
	p.landingStagingLeg = landingStagingLegLeadIn
	p.landingCarrierRotation = ship.CurRotation
	p.landingCarrierTurnRate = 0
	p.landingDeckFrames = 0
	p.updateLandingStagingEndPos(ship)
}

// StartLandingApproach 开始单向定半径圆弧进近动画。
func (p *Plane) StartLandingApproach(ship *BattleShip) {
	length := carrierLengthInMapBlocks(ship)
	local := planeCarrierLocalOffset(p, ship)
	arc, ok := buildLandingApproachArc(
		carrierLocalOffset{forward: local.forward / length, lateral: local.lateral / length},
		length,
		p.MaxSpeed,
	)
	if !ok {
		return
	}
	p.landingArc = arc
	p.landingCarrierRotation = ship.CurRotation
	p.landingDeckFrames = p.landingDeckDurationFrames(ship, arc.relativeSpeed)
	p.FlightPhase = PlaneFlightPhaseLandingApproach
	p.FlightPhaseStartPos = p.CurPos.Copy()
	p.FlightPhaseEndPos = carrierLandingFinalStartPos(ship)
	p.FlightPhaseElapsed = 0
	p.FlightPhaseProgressValue = 0
	p.FlightVisualScaleStart = 1
	p.FlightVisualScaleEnd = 1
}

// StartLandingDeck 初始化最终直线进近与甲板回收阶段。
func (p *Plane) StartLandingDeck(ship *BattleShip) {
	if p.landingDeckFrames <= 0 {
		relativeSpeed := p.landingArc.relativeSpeed
		if relativeSpeed <= 0 {
			relativeSpeed = p.landingRelativeSpeed(ship)
		}
		p.landingDeckFrames = p.landingDeckDurationFrames(ship, relativeSpeed)
	}
	currentScale := p.VisualScaleMultiplier()
	p.FlightPhase = PlaneFlightPhaseLandingDeck
	p.FlightPhaseStartPos = p.CurPos.Copy()
	p.FlightPhaseEndPos = carrierLandingDeckEndPos(ship)
	p.FlightPhaseElapsed = 0
	p.FlightPhaseProgressValue = 0
	p.FlightVisualScaleStart = currentScale
	p.FlightVisualScaleEnd = planeLowAltitudeVisualScale
}

// UpdateLandingStaging 推进舰尾入口阶段，返回是否进入最终进近捕获范围。
func (p *Plane) UpdateLandingStaging(_ *mapcfg.MapCfg, ship *BattleShip) bool {
	if p.CurHP <= 0 {
		return false
	}
	p.FlightPhaseElapsed += gameSpeedMultiplier()
	p.updateLandingCarrierTurnRate(ship)
	length := carrierLengthInMapBlocks(ship)
	gate := landingGateLocalOffset(p.LandingSlot)
	plannedArc, ok := buildLandingApproachArc(gate, length, p.MaxSpeed)
	if !ok {
		return false
	}
	leadIn := landingLeadInLocalOffset(plannedArc)
	gatePos := carrierRelativePos2D(ship, length*gate.forward, length*gate.lateral)
	leadInPos := carrierRelativePos2D(ship, length*leadIn.forward, length*leadIn.lateral)
	entrySpeed := landingArcEntryTargetSpeed(plannedArc, ship, p.landingCarrierTurnRate)

	// 先捕获切线后方的 lead-in，再沿切线进入 gate；两段都绑定航母当前姿态。
	previousLeg := p.landingStagingLeg
	previousTarget := p.FlightPhaseEndPos
	if p.landingStagingLeg == landingStagingLegLeadIn &&
		p.CurPos.Near(leadInPos, max(length*landingLeadInCaptureRadiusRatio, 0.5)) {
		p.landingStagingLeg = landingStagingLegGate
	}
	if p.landingStagingLeg == landingStagingLegGate {
		p.FlightPhaseEndPos = gatePos
	} else {
		p.FlightPhaseEndPos = leadInPos
	}

	// lead-in 会随航母平移和转向。复用上一帧目标可估算其世界速度，避免低速机追不上移动入口。
	targetMotion := 0.0
	if previousLeg == p.landingStagingLeg {
		targetMotion = previousTarget.Distance(leadInPos)
		if p.landingStagingLeg == landingStagingLegGate {
			targetMotion = previousTarget.Distance(gatePos)
		}
	}
	if p.landingStagingLeg == landingStagingLegGate {
		p.executeLandingGateMovement(ship, plannedArc)
	} else {
		carrierDistance := p.CurPos.Distance(ship.CurPos)
		targetSpeed := p.landingStagingTargetSpeed(
			ship,
			length,
			carrierDistance,
			entrySpeed,
			targetMotion,
		)
		executeLandingMovement(
			p,
			p.FlightPhaseEndPos,
			p.approachSpeed(targetSpeed),
		)
	}
	if p.landingStagingLeg != landingStagingLegGate {
		return false
	}
	if landingApproachEntryReady(p, ship, gate) {
		return true
	}

	local := planeCarrierLocalOffset(p, ship)
	current := carrierLocalOffset{forward: local.forward / length, lateral: local.lateral / length}
	tangent := landingArcTangent(plannedArc, 0)
	passedGate := (current.forward-gate.forward)*tangent.forward+
		(current.lateral-gate.lateral)*tangent.lateral > landingGateMissDistanceRatio
	// 错过入口或穿越中线时返回远端重新建立切线，禁止从舰侧强行接入圆弧。
	if passedGate || current.lateral*gate.lateral <= 0 {
		p.landingStagingLeg = landingStagingLegLeadIn
		p.FlightPhaseEndPos = leadInPos
	}
	return false
}

// updateLandingStagingEndPos 初始化 landing_staging 的远端切线引导目标。
func (p *Plane) updateLandingStagingEndPos(ship *BattleShip) {
	length := carrierLengthInMapBlocks(ship)
	gate := landingGateLocalOffset(p.LandingSlot)
	arc, ok := buildLandingApproachArc(gate, length, p.MaxSpeed)
	if !ok {
		p.FlightPhaseEndPos = ship.Aircraft.landingStagingTarget(nil, ship, p.LandingSlot)
		return
	}
	target := landingLeadInLocalOffset(arc)
	p.FlightPhaseEndPos = carrierRelativePos2D(ship, length*target.forward, length*target.lateral)
}

// landingStagingTargetSpeed 根据距航母的径向距离，在远端追赶速度和入口速度之间平滑插值。
func (p *Plane) landingStagingTargetSpeed(
	ship *BattleShip,
	length, distance, entrySpeed float64,
	targetMotion float64,
) float64 {
	maxSpeed := p.MaxSpeed * gameSpeedMultiplier()
	// 远处允许略微超过常规最大速度，确保低速舰载机也能追上高速母舰。
	catchupSpeed := max(
		maxSpeed,
		ship.CurSpeed+maxSpeed*landingCatchupClosureRatio,
		targetMotion+maxSpeed*landingCatchupClosureRatio,
		entrySpeed,
	)
	innerRadius := length * landingStagingSlowdownInnerRatio
	outerRadius := max(length*landingStagingSlowdownOuterRatio, innerRadius+0.5)
	slowdownProgress := clamp01((distance - innerRadius) / (outerRadius - innerRadius))
	return lerp(entrySpeed, catchupSpeed, smoothstep(slowdownProgress))
}

// landingDeckDurationFrames 根据圆弧出口相对速度计算最终直线段时长，保持阶段速度连续。
func (p *Plane) landingDeckDurationFrames(ship *BattleShip, relativeSpeed float64) float64 {
	length := carrierLengthInMapBlocks(ship)
	multiplier := max(gameSpeedMultiplier(), 0.001)
	relativeSpeed /= multiplier
	if relativeSpeed <= 0.001 {
		return landingDeckMaxFrames
	}
	// ease-out 二次曲线的初始导数是 2*distance/frames，据此匹配圆弧出口速度。
	distance := -carrierLandingFinalStartRatio * length
	frames := 2 * distance / relativeSpeed
	return max(landingDeckMinFrames, min(landingDeckMaxFrames, frames))
}

// landingRelativeSpeed 返回飞机相对航母的速度大小，用于缺少圆弧数据时的时长回退计算。
func (p *Plane) landingRelativeSpeed(ship *BattleShip) float64 {
	relativeHeading := (p.CurRotation - ship.CurRotation) * math.Pi / 180
	return math.Hypot(
		math.Cos(relativeHeading)*p.CurSpeed-ship.CurSpeed,
		math.Sin(relativeHeading)*p.CurSpeed,
	)
}

// updateLandingCarrierTurnRate 记录航母单个模拟帧内的实际航向变化。
func (p *Plane) updateLandingCarrierTurnRate(ship *BattleShip) {
	// 使用最短角差跨越 0/360 度，结果以“每模拟帧弧度”保存供 omega×r 使用。
	delta := math.Mod(ship.CurRotation-p.landingCarrierRotation+540, 360) - 180
	p.landingCarrierTurnRate = delta * math.Pi / 180
	p.landingCarrierRotation = ship.CurRotation
}

// UpdateLandingApproach 将飞机沿单向定半径圆弧移动到最终直线进近起点。
func (p *Plane) UpdateLandingApproach(ship *BattleShip) bool {
	if p.CurHP <= 0 {
		return false
	}
	p.updateLandingCarrierTurnRate(ship)
	p.FlightPhaseElapsed += gameSpeedMultiplier()
	timeProgress := clamp01(p.FlightPhaseElapsed / p.landingArc.frames)
	p.FlightPhaseProgressValue = timeProgress

	arcPos := landingArcPoint(p.landingArc, timeProgress)
	p.advanceLandingAnimation(
		ship,
		arcPos.forward,
		arcPos.lateral,
		landingArcWorldRotation(
			p.landingArc,
			ship,
			timeProgress,
			p.landingCarrierTurnRate,
		),
	)
	return timeProgress >= 1
}

// UpdateLandingDeck 沿 1.5 倍舰长的中线 ease-out 到舰中，返回是否完成回收。
func (p *Plane) UpdateLandingDeck(ship *BattleShip) bool {
	if p.CurHP <= 0 {
		return false
	}
	p.updateLandingCarrierTurnRate(ship)
	p.FlightPhaseElapsed += gameSpeedMultiplier()
	timeProgress := clamp01(p.FlightPhaseElapsed / p.landingDeckFrames)
	positionProgress := easeOutQuadratic(timeProgress)
	forwardRatio := lerp(carrierLandingFinalStartRatio, 0, positionProgress)
	// ease-out 的导数随进度线性降到 0，使飞机在甲板中心与航母速度完全一致。
	relativeForwardSpeed := 2 * -carrierLandingFinalStartRatio *
		carrierLengthInMapBlocks(ship) * (1 - timeProgress) /
		p.landingDeckFrames * gameSpeedMultiplier()
	p.FlightPhaseProgressValue = timeProgress
	p.FlightPhaseEndPos = carrierLandingDeckEndPos(ship)
	p.advanceLandingAnimation(
		ship,
		forwardRatio,
		0,
		landingDeckWorldRotation(
			ship,
			forwardRatio,
			relativeForwardSpeed,
			p.landingCarrierTurnRate,
		),
	)
	return timeProgress >= 1
}

// landingDeckWorldRotation 合成航母运动与直线闭合速度，返回甲板阶段的实际世界航向。
func landingDeckWorldRotation(
	ship *BattleShip,
	forwardRatio, relativeForwardSpeed, turnRate float64,
) float64 {
	return normalizeAngle(
		ship.CurRotation + math.Atan2(
			turnRate*forwardRatio*carrierLengthInMapBlocks(ship),
			ship.CurSpeed+relativeForwardSpeed,
		)*180/math.Pi,
	)
}

// executeLandingGateMovement 使用入口切线速度场推进飞机，并有限修正横向偏差。
func (p *Plane) executeLandingGateMovement(ship *BattleShip, arc landingApproachArc) {
	length := carrierLengthInMapBlocks(ship)
	local := planeCarrierLocalOffset(p, ship)
	current := carrierLocalOffset{forward: local.forward / length, lateral: local.lateral / length}
	tangent := landingArcTangent(arc, 0)
	delta := carrierLocalOffset{
		forward: current.forward - arc.start.forward,
		lateral: current.lateral - arc.start.lateral,
	}
	along := delta.forward*tangent.forward + delta.lateral*tangent.lateral
	closest := carrierLocalOffset{
		forward: arc.start.forward + tangent.forward*along,
		lateral: arc.start.lateral + tangent.lateral*along,
	}
	// 速度场始终沿圆弧入口切线前进，只加入受限的横向误差修正，因此不会左右摆动。
	correction := carrierLocalOffset{
		forward: (closest.forward - current.forward) * length * landingLeadInCrossTrackGain,
		lateral: (closest.lateral - current.lateral) * length * landingLeadInCrossTrackGain,
	}
	correctionMagnitude := math.Hypot(correction.forward, correction.lateral)
	maxCorrection := arc.relativeSpeed * landingLeadInMaxCorrectionRatio
	if correctionMagnitude > maxCorrection && correctionMagnitude > 0 {
		scale := maxCorrection / correctionMagnitude
		correction.forward *= scale
		correction.lateral *= scale
	}
	// 与圆弧阶段相同，最终世界速度由航母平移、飞机相对速度和航母旋转速度组成。
	velocity := carrierLocalOffset{
		forward: ship.CurSpeed + tangent.forward*arc.relativeSpeed + correction.forward -
			p.landingCarrierTurnRate*current.lateral*length,
		lateral: tangent.lateral*arc.relativeSpeed + correction.lateral +
			p.landingCarrierTurnRate*current.forward*length,
	}
	targetRotation := normalizeAngle(
		ship.CurRotation + math.Atan2(velocity.lateral, velocity.forward)*180/math.Pi,
	)
	executeLandingMovementOnHeading(
		p,
		targetRotation,
		p.approachSpeed(math.Hypot(velocity.forward, velocity.lateral)),
	)
}

// 着舰引导允许短暂飞出地图边界，确保舰尾朝外时仍能建立合法后方航线。
func executeLandingMovement(p *Plane, targetPos objPos.MapPos, speed float64) {
	executeLandingMovementOnHeading(p, p.CurPos.Angle(targetPos), speed)
}

// executeLandingMovementOnHeading 按飞机转向能力逼近目标航向，并执行不受地图边界夹取的位移。
func executeLandingMovementOnHeading(p *Plane, targetRotation, speed float64) {
	p.CurSpeed = speed
	p.CurRotation = rotateAngleToward(
		p.CurRotation,
		targetRotation,
		p.RotateSpeed*gameSpeedMultiplier(),
	)
	radians := p.CurRotation * math.Pi / 180
	p.CurPos.AssignRxy(
		p.CurPos.RX+math.Sin(radians)*speed,
		p.CurPos.RY-math.Cos(radians)*speed,
	)
	p.RemainRange -= speed
}

// advanceLandingAnimation 将动画采样点绑定到航母当前姿态，并同步飞机速度和航向。
func (p *Plane) advanceLandingAnimation(
	ship *BattleShip,
	forwardRatio, lateralRatio, targetRotation float64,
) {
	// 动画点每帧从航母局部坐标重新映射，航母移动或转向时轨迹仍与甲板保持绑定。
	length := carrierLengthInMapBlocks(ship)
	nextPos := carrierRelativePos2D(ship, length*forwardRatio, length*lateralRatio)
	distance := p.CurPos.Distance(nextPos)
	p.CurPos = nextPos
	p.CurSpeed = distance
	p.RemainRange -= distance
	p.CurRotation = rotateAngleToward(
		p.CurRotation,
		targetRotation,
		p.RotateSpeed*gameSpeedMultiplier(),
	)
}
