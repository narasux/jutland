package unit

import (
	"math"

	"github.com/narasux/jutland/pkg/common/constants"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objTrail "github.com/narasux/jutland/pkg/mission/object/trail"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
	"github.com/narasux/jutland/pkg/utils/colorx"
)

const (
	// landingLeadInCrossTrackGain 将入口直线的横向误差转换为每帧修正速度。
	landingLeadInCrossTrackGain = 0.08
	// landingLeadInMaxCorrectionRatio 限制横向修正不超过圆弧相对速度的 75%。
	// 修正过弱时，偏航捕获的飞机会在越过入口前来不及回到切线，触发反复重引导。
	landingLeadInMaxCorrectionRatio = 0.75
	// landingCatchupClosureRatio 是远距离追赶移动入口时保留的最小闭合速度比例。
	landingCatchupClosureRatio = 0.15
	// landingStagingSlowdownInnerOffset 是近舰减速包线内边界超出 gate 的距离，单位为舰长。
	landingStagingSlowdownInnerOffset = 0.5
	// landingStagingSlowdownOuterSpan 是减速包线从内边界再外推的距离，单位为舰长。
	landingStagingSlowdownOuterSpan = 2.0
	// landingApproachTargetSpeedRatio 是圆弧进近段的目标出口速度相对最大速度的比例。
	// 空中先平滑减速到该速度，触舰后再由刹车段减到 0，避免以最高速着舰。
	landingApproachTargetSpeedRatio = 0.30
	// landingMinRelativeSpeedRatio 是进近减速计划的最低相对速度保底比例，
	// 防止低速机在减速积分中停滞。
	landingMinRelativeSpeedRatio = 0.05
	// landingTurnRateSmoothingRate 是机头合成所用航母转向速率的每帧低通增益
	// （随游戏倍速放大）。玩家满舵时舰体角速度是阶跃信号，滤波把阶跃摊成
	// 数十帧的平滑偏航，避免进近中的飞机瞬间甩头。
	landingTurnRateSmoothingRate = 0.08
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
func (p *Plane) StartLandingStaging(_ *mapcfg.MapCfg, base AircraftBase, slot int) {
	p.CurAttackTarget = ""
	p.FlightPhase = PlaneFlightPhaseLandingStaging
	p.FlightPhaseStartPos = p.CurPos.Copy()
	p.FlightPhaseElapsed = 0
	p.FlightPhaseProgressValue = 0
	p.FlightVisualScaleStart = 1
	p.FlightVisualScaleEnd = 1
	p.LandingSlot = slot
	// 着水回收的舰船（水上飞机母舰），飞机最终降落在舰侧水面而非甲板
	p.landingOnWater = base.BaseAircraft().landingOnWater()
	p.landingStagingLeg = landingStagingLegLeadIn
	p.landingCarrierRotation = base.BaseRotation()
	p.landingCarrierTurnRate = 0
	p.landingDisplayTurnRate = 0
	p.resetLandingRunPlan()
	p.updateLandingStagingEndPos(base)
}

// StartLandingApproach 开始单向定半径圆弧进近，并建立空中减速计划。
func (p *Plane) StartLandingApproach(base AircraftBase) {
	landing := base.BaseAircraft().landingConfigForSlot(p.LandingSlot)
	length := carrierLengthInMapBlocks(base)
	local := planeCarrierLocalOffset(p, base)
	arc, ok := buildLandingApproachArc(
		carrierLocalOffset{forward: local.forward / length, lateral: local.lateral / length},
		base,
		p.MaxSpeed,
		landing,
	)
	if !ok {
		return
	}
	p.landingArc = arc
	p.landingCarrierRotation = base.BaseRotation()
	p.resetLandingRunPlan()
	// 以当前实际相对速度为初速建立圆弧段等减速计划，保证与入口引导速度连续
	entry := p.landingRelativeSpeed(base)
	p.startLandingArcPlan(base, entry)
	p.FlightPhase = PlaneFlightPhaseLandingApproach
	p.FlightPhaseStartPos = p.CurPos.Copy()
	p.FlightPhaseEndPos = landingFinalStartPos(base, landing)
	p.FlightPhaseElapsed = 0
	p.FlightPhaseProgressValue = 0
	p.FlightVisualScaleStart = 1
	p.FlightVisualScaleEnd = 1
}

// startLandingArcPlan 按运动学 v²=v0²-2aL 建立圆弧段等减速计划。
func (p *Plane) startLandingArcPlan(base AircraftBase, entry float64) {
	multiplier := max(gameSpeedMultiplier(), 0.001)
	entry = max(entry, p.MaxSpeed*multiplier*landingMinRelativeSpeedRatio)
	exit := min(entry, p.MaxSpeed*multiplier*landingApproachTargetSpeedRatio)
	total := max(p.landingArc.radius*math.Abs(p.landingArc.sweepAngle)*
		carrierLengthInMapBlocks(base), 0.001)
	// 单帧减速量：a = (v0²-vt²)/(2L)，恰好在线段末端降到出口速度
	decel := (entry*entry - exit*exit) / (2 * total)

	p.landingArcSpeed = entry
	p.landingArcExitSpeed = exit
	p.landingArcTotalLength = total
	p.landingArcDistance = 0
	p.landingArcDecel = decel
}

// StartLandingDeck 初始化最终直线进近与着舰回收阶段。
func (p *Plane) StartLandingDeck(base AircraftBase) {
	landing := base.BaseAircraft().landingConfigForSlot(p.LandingSlot)
	length := carrierLengthInMapBlocks(base)
	// 圆弧出口速度作为刹车段初速，两段速度天然连续
	entry := p.landingArcSpeed
	if entry <= 0 {
		entry = p.landingRelativeSpeed(base)
	}
	entry = max(entry, p.MaxSpeed*gameSpeedMultiplier()*landingMinRelativeSpeedRatio)
	total := max(length*landing.ApproachLength, 0.001)

	p.landingOnWater = base.BaseAircraft().landingOnWater()
	p.landingScaleCompleteRatio = landingScaleCompleteRatio(base, landing)
	p.landingRunTangent = landingApproachTangent(landing)
	p.landingRunSpeed = entry
	p.landingRunTotalLength = total
	p.landingRunDistance = 0
	p.landingRunDecel = entry * entry / (2 * total)

	currentScale := p.VisualScaleMultiplier()
	p.FlightPhase = PlaneFlightPhaseLandingDeck
	p.FlightPhaseStartPos = p.CurPos.Copy()
	p.FlightPhaseEndPos = carrierLandingDeckEndPos(base, landing)
	p.FlightPhaseElapsed = 0
	p.FlightPhaseProgressValue = 0
	p.FlightVisualScaleStart = currentScale
	p.FlightVisualScaleEnd = planeLowAltitudeVisualScale
}

// resetLandingRunPlan 清空最终直线段与圆弧段的减速计划状态。
func (p *Plane) resetLandingRunPlan() {
	p.landingArcSpeed = 0
	p.landingArcExitSpeed = 0
	p.landingArcTotalLength = 0
	p.landingArcDistance = 0
	p.landingArcDecel = 0
	p.landingRunSpeed = 0
	p.landingRunTotalLength = 0
	p.landingRunDistance = 0
	p.landingRunDecel = 0
	p.landingRunTangent = carrierLocalOffset{}
}

// UpdateLandingStaging 推进舰尾入口阶段，返回是否进入最终进近捕获范围。
func (p *Plane) UpdateLandingStaging(_ *mapcfg.MapCfg, base AircraftBase) bool {
	if p.CurHP <= 0 {
		return false
	}
	p.FlightPhaseElapsed += gameSpeedMultiplier()
	p.updateLandingCarrierTurnRate(base)
	length := carrierLengthInMapBlocks(base)
	landing := base.BaseAircraft().landingConfigForSlot(p.LandingSlot)
	gate := landingGateLocalOffset(p.LandingSlot)
	plannedArc, ok := buildLandingApproachArc(gate, base, p.MaxSpeed, landing)
	if !ok {
		return false
	}
	leadIn := landingLeadInLocalOffset(plannedArc)
	gatePos := carrierRelativePos2D(base, length*gate.forward, length*gate.lateral)
	leadInPos := carrierRelativePos2D(base, length*leadIn.forward, length*leadIn.lateral)
	entrySpeed := landingArcEntryTargetSpeed(plannedArc, base, p.landingCarrierTurnRate)

	// 先捕获切线后方的 lead-in，再沿切线进入 gate；两段都绑定航母当前姿态。
	// 捕获半径随 lead-in 的世界运动速度放大：快速机动的入口下方机会形成
	// 尾随平衡，飞机悬停距离与目标运动速度成正比，固定半径会永远差一点。
	previousLeg := p.landingStagingLeg
	previousTarget := p.FlightPhaseEndPos
	leadInMotion := 0.0
	if previousLeg == p.landingStagingLeg {
		leadInMotion = previousTarget.Distance(leadInPos)
	}
	captureRadius := max(
		length*landingLeadInCaptureRadiusRatio,
		0.5,
		leadInMotion*landingLeadInMotionCaptureFrames,
	)
	if p.landingStagingLeg == landingStagingLegLeadIn &&
		p.CurPos.Near(leadInPos, captureRadius) {
		p.landingStagingLeg = landingStagingLegGate
	}
	if p.landingStagingLeg == landingStagingLegGate {
		p.FlightPhaseEndPos = gatePos
	} else {
		p.FlightPhaseEndPos = leadInPos
	}

	// lead-in 会随航母平移和转向。复用上一帧目标可估算其世界速度，避免低速机追不上移动入口。
	targetMotion := 0.0
	aimPos := p.FlightPhaseEndPos
	if previousLeg == p.landingStagingLeg {
		targetMotion = previousTarget.Distance(leadInPos)
		aimPos = leadInPos
		if p.landingStagingLeg == landingStagingLegGate {
			targetMotion = previousTarget.Distance(gatePos)
		} else {
			// lead-in 随航母平移与转向，世界系追踪容易与旋转入口形成尾随平衡。
			// 按当前间距估算抵达帧数进行比例前瞻，瞄准目标的预测位置切弦收敛。
			gap := p.CurPos.Distance(leadInPos)
			motionForward := leadInPos.RX - previousTarget.RX
			motionLateral := leadInPos.RY - previousTarget.RY
			pursuitSpeed := p.approachSpeed(p.landingStagingTargetSpeed(
				base,
				length,
				p.CurPos.Distance(base.BasePos()),
				gatePos.Distance(base.BasePos()),
				entrySpeed,
				targetMotion,
			))
			leadScale := gap / max(pursuitSpeed, 1e-6)
			leadForward := motionForward * leadScale
			leadLateral := motionLateral * leadScale
			// 预测点距离不超过上限，避免急转弯时把瞄准点甩得过远
			leadMagnitude := math.Hypot(leadForward, leadLateral)
			maxLead := length * landingLeadInMaxLeadRatio
			if leadMagnitude > maxLead && leadMagnitude > 0 {
				leadForward *= maxLead / leadMagnitude
				leadLateral *= maxLead / leadMagnitude
			}
			aimPos = objPos.NewR(leadInPos.RX+leadForward, leadInPos.RY+leadLateral)
		}
	}
	if p.landingStagingLeg == landingStagingLegGate {
		p.executeLandingGateMovement(base, plannedArc)
	} else {
		carrierDistance := p.CurPos.Distance(base.BasePos())
		targetSpeed := p.landingStagingTargetSpeed(
			base,
			length,
			carrierDistance,
			gatePos.Distance(base.BasePos()),
			entrySpeed,
			targetMotion,
		)
		executeLandingMovement(
			p,
			aimPos,
			p.approachSpeed(targetSpeed),
		)
	}
	if p.landingStagingLeg != landingStagingLegGate {
		return false
	}
	if landingApproachEntryReady(p, base, gate) {
		return true
	}

	local := planeCarrierLocalOffset(p, base)
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
func (p *Plane) updateLandingStagingEndPos(base AircraftBase) {
	length := carrierLengthInMapBlocks(base)
	gate := landingGateLocalOffset(p.LandingSlot)
	arc, ok := buildLandingApproachArc(gate, base, p.MaxSpeed, base.BaseAircraft().landingConfigForSlot(p.LandingSlot))
	if !ok {
		p.FlightPhaseEndPos = base.BaseAircraft().landingStagingTarget(nil, base, p.LandingSlot)
		return
	}
	target := landingLeadInLocalOffset(arc)
	p.FlightPhaseEndPos = carrierRelativePos2D(base, length*target.forward, length*target.lateral)
}

// landingStagingTargetSpeed 根据距 gate 入口的径向距离，在远端追赶速度和入口速度之间平滑插值。
func (p *Plane) landingStagingTargetSpeed(
	base AircraftBase,
	length, distance, gateDistance, entrySpeed float64,
	targetMotion float64,
) float64 {
	maxSpeed := p.MaxSpeed * gameSpeedMultiplier()
	// 远处允许略微超过常规最大速度，确保低速舰载机也能追上高速母舰。
	catchupSpeed := max(
		maxSpeed,
		base.BaseSpeed()+maxSpeed*landingCatchupClosureRatio,
		targetMotion+maxSpeed*landingCatchupClosureRatio,
		entrySpeed,
	)
	// 包线以 gate 入口为参考：飞机到达 gate 前 0.5 舰长时已降到入口速度，
	// 与 gate 相对舰艉的配置距离解耦（历史上以舰中心为参考在 gate 后移后失配）。
	innerRadius := gateDistance + length*landingStagingSlowdownInnerOffset
	outerRadius := max(innerRadius+length*landingStagingSlowdownOuterSpan, innerRadius+0.5)
	slowdownProgress := clamp01((distance - innerRadius) / (outerRadius - innerRadius))
	return lerp(entrySpeed, catchupSpeed, smoothstep(slowdownProgress))
}

// landingRelativeSpeed 返回飞机相对航母的速度大小，用于缺少圆弧数据时的时长回退计算。
func (p *Plane) landingRelativeSpeed(base AircraftBase) float64 {
	relativeHeading := (p.CurRotation - base.BaseRotation()) * math.Pi / 180
	return math.Hypot(
		math.Cos(relativeHeading)*p.CurSpeed-base.BaseSpeed(),
		math.Sin(relativeHeading)*p.CurSpeed,
	)
}

// updateLandingCarrierTurnRate 记录航母单个模拟帧内的实际航向变化，
// 并维护机头合成使用的低通转向速率。
func (p *Plane) updateLandingCarrierTurnRate(base AircraftBase) {
	// 使用最短角差跨越 0/360 度，结果以“每模拟帧弧度”保存供 omega×r 使用。
	delta := math.Mod(base.BaseRotation()-p.landingCarrierRotation+540, 360) - 180
	turnRate := delta * math.Pi / 180
	p.landingCarrierRotation = base.BaseRotation()
	// 原始转向速率保留给 staging 自由飞行的速度合成；机头航向只用于表现，
	// 使用低通后的速率，使转向起止时机头的偏航变化连续平滑。
	p.landingCarrierTurnRate = turnRate
	p.landingDisplayTurnRate += (turnRate - p.landingDisplayTurnRate) *
		min(1, landingTurnRateSmoothingRate*gameSpeedMultiplier())
}

// UpdateLandingApproach 沿定半径圆弧按等减速推进到最终直线进近起点。
func (p *Plane) UpdateLandingApproach(base AircraftBase) bool {
	if p.CurHP <= 0 {
		return false
	}
	p.updateLandingCarrierTurnRate(base)
	p.FlightPhaseElapsed += gameSpeedMultiplier()

	// 按距离积分推进：每帧先前进再按等减速运动学反算下一帧速度，
	// v² = v² - 2aL 保证离散累计距离与速度计划一致，末端平滑收敛不出跳变
	step := min(p.landingArcSpeed, max(p.landingArcTotalLength-p.landingArcDistance, 0))
	p.landingArcDistance += step
	timeProgress := clamp01(p.landingArcDistance / max(p.landingArcTotalLength, 0.001))
	p.FlightPhaseProgressValue = timeProgress
	p.landingArcSpeed = max(
		p.landingArcExitSpeed,
		math.Sqrt(max(0, p.landingArcSpeed*p.landingArcSpeed-2*p.landingArcDecel*step)),
	)

	arcPos := landingArcPoint(p.landingArc, timeProgress)
	// 机头航向使用低通转向速率合成：位置已硬绑定随舰旋转的圆弧，
	// 若跟随含原始 omega×r 的合成航向，转向起止时机会瞬间甩动数十度。
	p.advanceLandingAnimation(
		base,
		arcPos.forward,
		arcPos.lateral,
		landingArcWorldRotation(
			p.landingArc,
			base,
			timeProgress,
			p.landingDisplayTurnRate,
		),
	)
	return timeProgress >= 1
}

// UpdateLandingDeck 沿最终直线进近段等减速滑跑到着舰点，返回是否完成回收。
func (p *Plane) UpdateLandingDeck(base AircraftBase) bool {
	if p.CurHP <= 0 {
		return false
	}
	p.updateLandingCarrierTurnRate(base)
	p.FlightPhaseElapsed += gameSpeedMultiplier()

	// 等减速刹车：每帧按剩余距离反算速度（v² = v² - 2aL），最后一步恰好停在着舰点
	step := min(p.landingRunSpeed, max(p.landingRunTotalLength-p.landingRunDistance, 0))
	p.landingRunDistance += step
	timeProgress := clamp01(p.landingRunDistance / max(p.landingRunTotalLength, 0.001))
	p.landingRunSpeed = math.Sqrt(max(0, p.landingRunSpeed*p.landingRunSpeed-2*p.landingRunDecel*step))
	p.FlightPhaseProgressValue = timeProgress

	// 起点 = 最终进近段起点（舰长单位局部坐标），沿进近方向前进已滑跑距离
	landing := base.BaseAircraft().landingConfigForSlot(p.LandingSlot)
	end := landingFinalStartOffset(base, landing)
	length := carrierLengthInMapBlocks(base)
	forwardRatio := end.forward + p.landingRunTangent.forward*(p.landingRunDistance/length)
	lateralRatio := end.lateral + p.landingRunTangent.lateral*(p.landingRunDistance/length)
	p.FlightPhaseEndPos = carrierLandingDeckEndPos(base, landing)
	// 与圆弧段一致，机头航向使用低通转向速率合成，转向时机头贴合甲板进近线
	p.advanceLandingAnimation(
		base,
		forwardRatio,
		lateralRatio,
		landingRunWorldRotation(
			base,
			p.LandingSlot,
			forwardRatio,
			lateralRatio,
			p.landingRunSpeed,
			p.landingDisplayTurnRate,
		),
	)
	return timeProgress >= 1
}

// landingRunWorldRotation 合成航母运动与直线刹车速度，返回甲板阶段的实际世界航向。
func landingRunWorldRotation(
	base AircraftBase,
	slot int,
	forwardRatio, lateralRatio, relativeSpeed, turnRate float64,
) float64 {
	// 相对速度沿进近方向，再叠加航母旋转产生的 omega×r 速度后转为世界航向
	tangent := base.BaseAircraft().landingConfigForSlot(slot)
	tangentVec := landingApproachTangent(tangent)
	length := carrierLengthInMapBlocks(base)
	velocity := carrierLocalOffset{
		forward: base.BaseSpeed() + tangentVec.forward*relativeSpeed -
			turnRate*lateralRatio*length,
		lateral: tangentVec.lateral*relativeSpeed + turnRate*forwardRatio*length,
	}
	return normalizeAngle(
		base.BaseRotation() + math.Atan2(velocity.lateral, velocity.forward)*180/math.Pi,
	)
}

// executeLandingGateMovement 使用入口切线速度场推进飞机，并有限修正横向偏差。
func (p *Plane) executeLandingGateMovement(base AircraftBase, arc landingApproachArc) {
	length := carrierLengthInMapBlocks(base)
	local := planeCarrierLocalOffset(p, base)
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
	// 修正下限随误差自适应：横向误差较大时保证每帧至少消去一定比例，
	// 否则飞机会在横向收敛完成前越过入口，陷入反复重引导；
	// 小误差时仍由比例增益阻尼，避免过冲振荡。
	maxCorrection := max(
		arc.relativeSpeed*landingLeadInMaxCorrectionRatio,
		correctionMagnitude*landingGateErrorCorrectionShare,
	)
	if correctionMagnitude > maxCorrection && correctionMagnitude > 0 {
		scale := maxCorrection / correctionMagnitude
		correction.forward *= scale
		correction.lateral *= scale
	}
	// 与圆弧阶段相同，最终世界速度由航母平移、飞机相对速度和航母旋转速度组成。
	velocity := carrierLocalOffset{
		forward: base.BaseSpeed() + tangent.forward*arc.relativeSpeed + correction.forward -
			p.landingCarrierTurnRate*current.lateral*length,
		lateral: tangent.lateral*arc.relativeSpeed + correction.lateral +
			p.landingCarrierTurnRate*current.forward*length,
	}
	targetRotation := normalizeAngle(
		base.BaseRotation() + math.Atan2(velocity.lateral, velocity.forward)*180/math.Pi,
	)
	executeLandingMovementOnHeading(
		p,
		targetRotation,
		p.approachSpeed(math.Hypot(velocity.forward, velocity.lateral)),
	)
}

// executeLandingMovement 沿指向目标点的方向推进飞机（世界坐标系）。
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

// GenWaterWakeTrails 为着水滑跑中的水上飞机生成水面尾流（泡沫 + 水痕）。
// 仅在舷侧着水降落的触水段（低空缩放完成之后）产生，甲板降落不产生水尾流。
func (p *Plane) GenWaterWakeTrails() []*objTrail.Trail {
	if !p.landingOnWater || p.FlightPhase != PlaneFlightPhaseLandingDeck {
		return nil
	}
	// 低空缩放完成后视为已触水；触水前保持空中无尾流
	if p.FlightPhaseProgressValue < p.landingScaleCompleteAt() {
		return nil
	}
	// 节流：每 3 个模拟帧生成一次，避免尾流过密
	if int(p.FlightPhaseElapsed)%3 != 0 {
		return nil
	}

	// 尾流从飞机尾部后方产生（与击落飞机的尾部偏移算法一致）
	tailPos := p.CurPos.Copy()
	sinVal := math.Sin(p.CurRotation * math.Pi / 180)
	cosVal := math.Cos(p.CurRotation * math.Pi / 180)
	tailOffset := p.Length / constants.MapBlockSize * 0.3
	tailPos.SubRx(sinVal * tailOffset)
	tailPos.AddRy(cosVal * tailOffset)

	return []*objTrail.Trail{
		// 白色泡沫：小尺寸快消散
		objTrail.New(
			tailPos, textureImg.TrailShapeCircle,
			p.Width*0.5, 0.5,
			30, 2.5,
			0, 0, colorx.White,
		),
		// 天蓝色水痕：略大、扩散慢、停留稍久
		objTrail.New(
			tailPos, textureImg.TrailShapeCircle,
			p.Width*0.8, 0.3,
			45, 1.8,
			0, 0, colorx.SkyBlue,
		),
	}
}

// advanceLandingAnimation 将动画采样点绑定到航母当前姿态，并同步飞机速度和航向。
func (p *Plane) advanceLandingAnimation(
	base AircraftBase,
	forwardRatio, lateralRatio, targetRotation float64,
) {
	// 动画点每帧从航母局部坐标重新映射，航母移动或转向时轨迹仍与甲板保持绑定。
	length := carrierLengthInMapBlocks(base)
	nextPos := carrierRelativePos2D(base, length*forwardRatio, length*lateralRatio)
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
