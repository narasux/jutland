package unit

import (
	"math"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// PlaneFlightPhase 表示飞机的起降 / 巡航阶段。
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
	carrierTakeoffStartOffsetRatio = 0
	carrierTakeoffLaneRatio        = 2.0
	carrierLandingFinalStartRatio  = -1.5

	landingStagingBaseForwardRatio   = -3.5
	landingStagingWaveStepRatio      = 0.12
	landingApproachLanes             = 16
	landingLaneMinLateralRatio       = 0.8
	landingLaneMaxLateralRatio       = 1.4
	landingLeadInLengthRatio         = 1.5
	landingLeadInCaptureRadiusRatio  = 0.30
	landingLeadInCrossTrackGain      = 0.08
	landingLeadInMaxCorrectionRatio  = 0.25
	landingGateForwardToleranceRatio = 0.25
	landingGateLateralToleranceRatio = 0.20
	landingGateMissDistanceRatio     = 0.25
	landingGateHeadingTolerance      = 8.0
	landingGateSpeedTolerance        = 0.10
	landingCatchupClosureRatio       = 0.15
	landingStagingSlowdownInnerRatio = 4.0
	landingStagingSlowdownOuterRatio = 6.0
	landingApproachMinFrames         = 90.0
	landingApproachMaxFrames         = 120.0
	landingApproachSpeedRatio        = 0.40
	landingDeckMinFrames             = 60.0
	landingDeckMaxFrames             = 180.0

	planeLowAltitudeVisualScale = 0.5

	takeoffInitialSpeedRatio  = 0.15
	takeoffAccelerationFrames = 45.0

	phaseSpeedStepFallbackRate = 0.04
	phaseSpeedStepMaxRate      = 0.08
)

type carrierLocalOffset struct {
	forward float64
	lateral float64
}

type landingStagingLeg uint8

const (
	landingStagingLegLeadIn landingStagingLeg = iota
	landingStagingLegGate
)

type landingApproachArc struct {
	center        carrierLocalOffset
	radius        float64
	startAngle    float64
	sweepAngle    float64
	frames        float64
	relativeSpeed float64
	start         carrierLocalOffset
}

func carrierLengthInMapBlocks(ship *BattleShip) float64 {
	return max(ship.Length/constants.MapBlockSize, 0.1)
}

func gameSpeedMultiplier() float64 {
	if config.G == nil {
		return 1
	}
	return config.G.SpeedMultiplier
}

func carrierRelativePos(ship *BattleShip, offset float64) objPos.MapPos {
	return carrierRelativePos2D(ship, offset, 0)
}

func carrierRelativePos2D(ship *BattleShip, forward, lateral float64) objPos.MapPos {
	radians := ship.CurRotation * math.Pi / 180
	sinVal, cosVal := math.Sin(radians), math.Cos(radians)
	return objPos.NewR(
		ship.CurPos.RX+sinVal*forward+cosVal*lateral,
		ship.CurPos.RY-cosVal*forward+sinVal*lateral,
	)
}

func planeCarrierLocalOffset(p *Plane, ship *BattleShip) carrierLocalOffset {
	radians := ship.CurRotation * math.Pi / 180
	sinVal, cosVal := math.Sin(radians), math.Cos(radians)
	dx, dy := p.CurPos.RX-ship.CurPos.RX, p.CurPos.RY-ship.CurPos.RY
	return carrierLocalOffset{
		forward: dx*sinVal - dy*cosVal,
		lateral: dx*cosVal + dy*sinVal,
	}
}

func carrierTakeoffStartPos(ship *BattleShip) objPos.MapPos {
	return carrierRelativePos(ship, carrierLengthInMapBlocks(ship)*carrierTakeoffStartOffsetRatio)
}

func carrierTakeoffEndPos(ship *BattleShip) objPos.MapPos {
	return carrierRelativePos(
		ship,
		carrierLengthInMapBlocks(ship)*(carrierTakeoffStartOffsetRatio+carrierTakeoffLaneRatio),
	)
}

func carrierLandingFinalStartPos(ship *BattleShip) objPos.MapPos {
	return carrierRelativePos(ship, carrierLengthInMapBlocks(ship)*carrierLandingFinalStartRatio)
}

func carrierLandingDeckEndPos(ship *BattleShip) objPos.MapPos {
	return ship.CurPos.Copy()
}

func landingLaneOffsetRatio(slot int) float64 {
	lane := slot % landingApproachLanes
	side := -1.0
	if lane%2 == 1 {
		side = 1
	}
	sideLane := lane / 2
	sideLaneCount := landingApproachLanes / 2
	progress := float64(sideLane) / float64(sideLaneCount-1)
	return side * lerp(landingLaneMinLateralRatio, landingLaneMaxLateralRatio, progress)
}

func (sa *ShipAircraft) landingStagingTarget(
	_ *mapcfg.MapCfg,
	ship *BattleShip,
	slot int,
) objPos.MapPos {
	length := carrierLengthInMapBlocks(ship)
	gate := landingGateLocalOffset(slot)
	return carrierRelativePos2D(
		ship,
		length*gate.forward,
		length*gate.lateral,
	)
}

func landingGateLocalOffset(slot int) carrierLocalOffset {
	wave := slot / landingApproachLanes
	return carrierLocalOffset{
		forward: landingStagingBaseForwardRatio - float64(wave)*landingStagingWaveStepRatio,
		lateral: landingLaneOffsetRatio(slot),
	}
}

func buildLandingApproachArc(
	start carrierLocalOffset,
	length, maxSpeed float64,
) (landingApproachArc, bool) {
	end := carrierLocalOffset{forward: carrierLandingFinalStartRatio}
	side := 1.0
	if start.lateral < 0 {
		side = -1
	}
	lateral := math.Abs(start.lateral)
	forwardDistance := end.forward - start.forward
	if lateral <= 0.001 || forwardDistance <= 0.001 {
		return landingApproachArc{}, false
	}

	radius := (forwardDistance*forwardDistance + lateral*lateral) / (2 * lateral)
	center := carrierLocalOffset{forward: end.forward, lateral: side * radius}
	startAngle := math.Atan2(start.lateral-center.lateral, start.forward-center.forward)
	endAngle := -side * math.Pi / 2
	sweepAngle := normalizeRadians(endAngle - startAngle)
	if sweepAngle*side <= 0 || math.Abs(sweepAngle) >= math.Pi {
		return landingApproachArc{}, false
	}

	arcLength := radius * math.Abs(sweepAngle) * length
	referenceSpeed := max(maxSpeed*landingApproachSpeedRatio, 0.001)
	frames := max(
		landingApproachMinFrames,
		min(landingApproachMaxFrames, arcLength/referenceSpeed),
	)
	return landingApproachArc{
		center:        center,
		radius:        radius,
		startAngle:    startAngle,
		sweepAngle:    sweepAngle,
		frames:        frames,
		relativeSpeed: arcLength / frames * gameSpeedMultiplier(),
		start:         start,
	}, true
}

func landingArcPoint(arc landingApproachArc, progress float64) carrierLocalOffset {
	angle := arc.startAngle + arc.sweepAngle*clamp01(progress)
	return carrierLocalOffset{
		forward: arc.center.forward + arc.radius*math.Cos(angle),
		lateral: arc.center.lateral + arc.radius*math.Sin(angle),
	}
}

func landingArcTangent(arc landingApproachArc, progress float64) carrierLocalOffset {
	angle := arc.startAngle + arc.sweepAngle*clamp01(progress)
	forward := -math.Sin(angle) * arc.sweepAngle
	lateral := math.Cos(angle) * arc.sweepAngle
	magnitude := math.Hypot(forward, lateral)
	return carrierLocalOffset{forward: forward / magnitude, lateral: lateral / magnitude}
}

func landingLeadInLocalOffset(arc landingApproachArc) carrierLocalOffset {
	tangent := landingArcTangent(arc, 0)
	return carrierLocalOffset{
		forward: arc.start.forward - tangent.forward*landingLeadInLengthRatio,
		lateral: arc.start.lateral - tangent.lateral*landingLeadInLengthRatio,
	}
}

func landingArcEntryTargetSpeed(
	arc landingApproachArc,
	ship *BattleShip,
	turnRate float64,
) float64 {
	velocity := landingArcWorldVelocity(arc, ship, 0, turnRate)
	return math.Hypot(velocity.forward, velocity.lateral)
}

func landingArcWorldVelocity(
	arc landingApproachArc,
	ship *BattleShip,
	progress, turnRate float64,
) carrierLocalOffset {
	tangent := landingArcTangent(arc, progress)
	point := landingArcPoint(arc, progress)
	length := carrierLengthInMapBlocks(ship)
	return carrierLocalOffset{
		forward: ship.CurSpeed + tangent.forward*arc.relativeSpeed -
			turnRate*point.lateral*length,
		lateral: tangent.lateral*arc.relativeSpeed + turnRate*point.forward*length,
	}
}

func landingArcWorldRotation(
	arc landingApproachArc,
	ship *BattleShip,
	progress, turnRate float64,
) float64 {
	velocity := landingArcWorldVelocity(arc, ship, progress, turnRate)
	return normalizeAngle(
		ship.CurRotation + math.Atan2(velocity.lateral, velocity.forward)*180/math.Pi,
	)
}

func landingApproachEntryReady(
	p *Plane,
	ship *BattleShip,
	gate carrierLocalOffset,
) bool {
	length := carrierLengthInMapBlocks(ship)
	local := planeCarrierLocalOffset(p, ship)
	start := carrierLocalOffset{
		forward: local.forward / length,
		lateral: local.lateral / length,
	}
	if start.lateral*gate.lateral <= 0 ||
		math.Abs(start.forward-gate.forward) > landingGateForwardToleranceRatio ||
		math.Abs(start.lateral-gate.lateral) > landingGateLateralToleranceRatio {
		return false
	}
	arc, ok := buildLandingApproachArc(start, length, p.MaxSpeed)
	if !ok {
		return false
	}
	entrySpeed := landingArcEntryTargetSpeed(arc, ship, p.landingCarrierTurnRate)
	targetRotation := landingArcWorldRotation(arc, ship, 0, p.landingCarrierTurnRate)
	if angleDifferenceDegrees(p.CurRotation, targetRotation) > landingGateHeadingTolerance {
		return false
	}
	speedTolerance := max(entrySpeed*landingGateSpeedTolerance, p.phaseSpeedStep(entrySpeed))
	return math.Abs(p.CurSpeed-entrySpeed) <= speedTolerance
}

// StartTakeoff 初始化航母起飞阶段。
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

// UpdateTakeoff 推进起飞阶段，返回是否已经进入巡航。
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
	if passedGate || current.lateral*gate.lateral <= 0 {
		p.landingStagingLeg = landingStagingLegLeadIn
		p.FlightPhaseEndPos = leadInPos
	}
	return false
}

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

func (p *Plane) landingDeckDurationFrames(ship *BattleShip, relativeSpeed float64) float64 {
	length := carrierLengthInMapBlocks(ship)
	multiplier := max(gameSpeedMultiplier(), 0.001)
	relativeSpeed /= multiplier
	if relativeSpeed <= 0.001 {
		return landingDeckMaxFrames
	}
	distance := -carrierLandingFinalStartRatio * length
	frames := 2 * distance / relativeSpeed
	return max(landingDeckMinFrames, min(landingDeckMaxFrames, frames))
}

func (p *Plane) landingRelativeSpeed(ship *BattleShip) float64 {
	relativeHeading := (p.CurRotation - ship.CurRotation) * math.Pi / 180
	return math.Hypot(
		math.Cos(relativeHeading)*p.CurSpeed-ship.CurSpeed,
		math.Sin(relativeHeading)*p.CurSpeed,
	)
}

func (p *Plane) updateLandingCarrierTurnRate(ship *BattleShip) {
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

func (p *Plane) advanceLandingAnimation(
	ship *BattleShip,
	forwardRatio, lateralRatio, targetRotation float64,
) {
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
		progress = smoothstep(progress)
	}
	return p.FlightVisualScaleStart + (p.FlightVisualScaleEnd-p.FlightVisualScaleStart)*progress
}

// FlightPhaseProgress 返回当前阶段进度，范围 [0, 1]。
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

func (p *Plane) approachSpeed(targetSpeed float64) float64 {
	return moveSpeedToward(p.CurSpeed, targetSpeed, p.phaseSpeedStep(targetSpeed))
}

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

func moveSpeedToward(curSpeed, targetSpeed, step float64) float64 {
	if step <= 0 {
		return targetSpeed
	}
	if curSpeed < targetSpeed {
		return min(targetSpeed, curSpeed+step)
	}
	return max(targetSpeed, curSpeed-step)
}

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

func smoothstep(progress float64) float64 {
	progress = clamp01(progress)
	return progress * progress * (3 - 2*progress)
}

func easeOutQuadratic(progress float64) float64 {
	progress = clamp01(progress)
	return 1 - (1-progress)*(1-progress)
}

func lerp(start, end, progress float64) float64 {
	return start + (end-start)*progress
}

func clamp01(value float64) float64 {
	return max(0, min(1, value))
}

func normalizeRadians(angle float64) float64 {
	return math.Mod(angle+3*math.Pi, 2*math.Pi) - math.Pi
}

func angleDifferenceDegrees(left, right float64) float64 {
	return math.Abs(math.Mod(left-right+540, 360) - 180)
}

func rotateAngleToward(current, target, maxDelta float64) float64 {
	if maxDelta <= 0 {
		return normalizeAngle(target)
	}
	delta := math.Mod(target-current+540, 360) - 180
	return normalizeAngle(current + max(-maxDelta, min(maxDelta, delta)))
}

func normalizeAngle(angle float64) float64 {
	return math.Mod(angle+360, 360)
}
