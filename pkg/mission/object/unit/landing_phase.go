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
	// 该出口速度同时是最终直线刹车段的入口速度，刹车时长按 2L/v 随它成反比：
	// 取 0.60（原 0.30 的两倍）把最后的进近滑跑提速一倍，缩短飞机滞留降落阶段的时间。
	landingApproachTargetSpeedRatio = 0.60
	// landingMinRelativeSpeedRatio 是进近减速计划的最低相对速度保底比例，
	// 防止低速机在减速积分中停滞。
	landingMinRelativeSpeedRatio = 0.05
	// landingTurnRateSmoothingRate 是机头合成所用航母转向速率的每帧低通增益
	// （随游戏倍速放大）。玩家满舵时舰体角速度是阶跃信号，滤波把阶跃摊成
	// 数十帧的平滑偏航，避免进近中的飞机瞬间甩头。
	landingTurnRateSmoothingRate = 0.08
)

// LandingState 飞机着舰引导、进近减速与最终刹车状态。
type LandingState struct {
	// Slot 当前飞机在所属航母舰尾入口中的分散槽位
	Slot int
	// onWater 本次降落是否为舷侧着水回收（水上飞机）
	onWater bool
	// stagingLeg 当前正在飞向切线引导点或圆弧入口
	stagingLeg landingStagingLeg
	// carrierRotation 上一模拟帧的航母航向
	carrierRotation float64
	// carrierTurnRate 航母当前每模拟帧的转向弧度
	carrierTurnRate float64
	// displayTurnRate 降落机头合成使用的低通转向速率
	displayTurnRate float64
	// scaleCompleteRatio 直线进近段降到最低视觉倍率时的路程比例
	scaleCompleteRatio float64
	// arc 最终进近使用的航母局部坐标圆弧
	arc landingApproachArc
	// 圆弧进近段的等减速计划状态（相对航母速度，单位为地图坐标每模拟帧）
	arcSpeed       float64
	arcExitSpeed   float64
	arcTotalLength float64
	arcDistance    float64
	arcDecel       float64
	// 最终直线进近段的等减速刹车计划状态
	runSpeed       float64
	runTotalLength float64
	runDistance    float64
	runDecel       float64
	// 最终直线进近方向的航母局部单位向量
	runTangent carrierLocalOffset
}

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
	p.Flight.Phase = PlaneFlightPhaseLandingStaging
	p.Flight.StartPos = p.CurPos.Copy()
	p.Flight.Elapsed = 0
	p.Flight.Progress = 0
	p.Flight.VisualScaleStart = 1
	p.Flight.VisualScaleEnd = 1
	p.Landing.Slot = slot
	// 着水回收的舰船（水上飞机母舰），飞机最终降落在舰侧水面而非甲板
	p.Landing.onWater = base.BaseAircraft().landingOnWater()
	p.Landing.stagingLeg = landingStagingLegLeadIn
	p.Landing.carrierRotation = base.BaseRotation()
	p.Landing.carrierTurnRate = 0
	p.Landing.displayTurnRate = 0
	p.resetLandingRunPlan()
	p.updateLandingStagingEndPos(base)
}

// StartLandingApproach 开始单向定半径圆弧进近，并建立空中减速计划。
func (p *Plane) StartLandingApproach(base AircraftBase) {
	landing := base.BaseAircraft().landingConfigForSlot(p.Landing.Slot)
	length := phaseUnitInMapBlocks(base)
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
	p.Landing.arc = arc
	p.Landing.carrierRotation = base.BaseRotation()
	p.resetLandingRunPlan()
	// 以当前实际相对速度为初速建立圆弧段等减速计划，保证与入口引导速度连续
	entry := p.landingRelativeSpeed(base)
	p.startLandingArcPlan(base, entry)
	p.Flight.Phase = PlaneFlightPhaseLandingApproach
	p.Flight.StartPos = p.CurPos.Copy()
	p.Flight.EndPos = landingFinalStartPos(base, landing)
	p.Flight.Elapsed = 0
	p.Flight.Progress = 0
	p.Flight.VisualScaleStart = 1
	p.Flight.VisualScaleEnd = 1
}

// startLandingArcPlan 按运动学 v²=v0²-2aL 建立圆弧段等减速计划。
func (p *Plane) startLandingArcPlan(base AircraftBase, entry float64) {
	multiplier := max(gameSpeedMultiplier(), 0.001)
	entry = max(entry, p.MaxSpeed*multiplier*landingMinRelativeSpeedRatio)
	exit := min(entry, p.MaxSpeed*multiplier*landingApproachTargetSpeedRatio)
	total := max(p.Landing.arc.radius*math.Abs(p.Landing.arc.sweepAngle)*
		phaseUnitInMapBlocks(base), 0.001)
	// 单帧减速量：a = (v0²-vt²)/(2L)，恰好在线段末端降到出口速度
	decel := (entry*entry - exit*exit) / (2 * total)

	p.Landing.arcSpeed = entry
	p.Landing.arcExitSpeed = exit
	p.Landing.arcTotalLength = total
	p.Landing.arcDistance = 0
	p.Landing.arcDecel = decel
}

// StartLandingDeck 初始化最终直线进近与着舰回收阶段。
func (p *Plane) StartLandingDeck(base AircraftBase) {
	landing := base.BaseAircraft().landingConfigForSlot(p.Landing.Slot)
	length := phaseUnitInMapBlocks(base)
	// 圆弧出口速度作为刹车段初速，两段速度天然连续
	entry := p.Landing.arcSpeed
	if entry <= 0 {
		entry = p.landingRelativeSpeed(base)
	}
	entry = max(entry, p.MaxSpeed*gameSpeedMultiplier()*landingMinRelativeSpeedRatio)
	total := max(length*landing.ApproachLength, 0.001)

	p.Landing.onWater = base.BaseAircraft().landingOnWater()
	p.Landing.scaleCompleteRatio = landingScaleCompleteRatio(base, landing)
	p.Landing.runTangent = landingApproachTangent(landing)
	p.Landing.runSpeed = entry
	p.Landing.runTotalLength = total
	p.Landing.runDistance = 0
	p.Landing.runDecel = entry * entry / (2 * total)

	currentScale := p.VisualScaleMultiplier()
	p.Flight.Phase = PlaneFlightPhaseLandingDeck
	p.Flight.StartPos = p.CurPos.Copy()
	p.Flight.EndPos = carrierLandingDeckEndPos(base, landing)
	p.Flight.Elapsed = 0
	p.Flight.Progress = 0
	p.Flight.VisualScaleStart = currentScale
	p.Flight.VisualScaleEnd = planeLowAltitudeVisualScale
}

// resetLandingRunPlan 清空最终直线段与圆弧段的减速计划状态。
func (p *Plane) resetLandingRunPlan() {
	p.Landing.arcSpeed = 0
	p.Landing.arcExitSpeed = 0
	p.Landing.arcTotalLength = 0
	p.Landing.arcDistance = 0
	p.Landing.arcDecel = 0
	p.Landing.runSpeed = 0
	p.Landing.runTotalLength = 0
	p.Landing.runDistance = 0
	p.Landing.runDecel = 0
	p.Landing.runTangent = carrierLocalOffset{}
}

// UpdateLandingStaging 推进舰尾入口阶段，返回是否进入最终进近捕获范围。
func (p *Plane) UpdateLandingStaging(_ *mapcfg.MapCfg, base AircraftBase) bool {
	if p.CurHP <= 0 {
		return false
	}
	p.Flight.Elapsed += gameSpeedMultiplier()
	p.updateLandingCarrierTurnRate(base)
	length := phaseUnitInMapBlocks(base)
	landing := base.BaseAircraft().landingConfigForSlot(p.Landing.Slot)
	gate := landingGateLocalOffset(p.Landing.Slot)
	plannedArc, ok := buildLandingApproachArc(gate, base, p.MaxSpeed, landing)
	if !ok {
		return false
	}
	leadIn := landingLeadInLocalOffset(plannedArc)
	gatePos := carrierRelativePos2D(base, length*gate.forward, length*gate.lateral)
	leadInPos := carrierRelativePos2D(base, length*leadIn.forward, length*leadIn.lateral)
	entrySpeed := landingArcEntryTargetSpeed(plannedArc, base, p.Landing.carrierTurnRate)

	// 先捕获切线后方的 lead-in，再沿切线进入 gate；两段都绑定航母当前姿态。
	// 捕获半径随 lead-in 的世界运动速度放大：快速机动的入口下方机会形成
	// 尾随平衡，飞机悬停距离与目标运动速度成正比，固定半径会永远差一点。
	previousLeg := p.Landing.stagingLeg
	previousTarget := p.Flight.EndPos
	leadInMotion := 0.0
	if previousLeg == p.Landing.stagingLeg {
		leadInMotion = previousTarget.Distance(leadInPos)
	}
	captureRadius := max(
		length*landingLeadInCaptureRadiusRatio,
		0.5,
		leadInMotion*landingLeadInMotionCaptureFrames,
	)
	if p.Landing.stagingLeg == landingStagingLegLeadIn &&
		p.CurPos.Near(leadInPos, captureRadius) {
		p.Landing.stagingLeg = landingStagingLegGate
	}
	if p.Landing.stagingLeg == landingStagingLegGate {
		p.Flight.EndPos = gatePos
	} else {
		p.Flight.EndPos = leadInPos
	}

	// lead-in 会随航母平移和转向。复用上一帧目标可估算其世界速度，避免低速机追不上移动入口。
	targetMotion := 0.0
	aimPos := p.Flight.EndPos
	if previousLeg == p.Landing.stagingLeg {
		targetMotion = previousTarget.Distance(leadInPos)
		aimPos = leadInPos
		if p.Landing.stagingLeg == landingStagingLegGate {
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
	if p.Landing.stagingLeg == landingStagingLegGate {
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
	if p.Landing.stagingLeg != landingStagingLegGate {
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
		p.Landing.stagingLeg = landingStagingLegLeadIn
		p.Flight.EndPos = leadInPos
	}
	return false
}

// updateLandingStagingEndPos 初始化 landing_staging 的远端切线引导目标。
func (p *Plane) updateLandingStagingEndPos(base AircraftBase) {
	length := phaseUnitInMapBlocks(base)
	gate := landingGateLocalOffset(p.Landing.Slot)
	arc, ok := buildLandingApproachArc(
		gate,
		base,
		p.MaxSpeed,
		base.BaseAircraft().landingConfigForSlot(p.Landing.Slot),
	)
	if !ok {
		p.Flight.EndPos = base.BaseAircraft().landingStagingTarget(nil, base, p.Landing.Slot)
		return
	}
	target := landingLeadInLocalOffset(arc)
	p.Flight.EndPos = carrierRelativePos2D(base, length*target.forward, length*target.lateral)
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
	delta := math.Mod(base.BaseRotation()-p.Landing.carrierRotation+540, 360) - 180
	turnRate := delta * math.Pi / 180
	p.Landing.carrierRotation = base.BaseRotation()
	// 原始转向速率保留给 staging 自由飞行的速度合成；机头航向只用于表现，
	// 使用低通后的速率，使转向起止时机头的偏航变化连续平滑。
	p.Landing.carrierTurnRate = turnRate
	p.Landing.displayTurnRate += (turnRate - p.Landing.displayTurnRate) *
		min(1, landingTurnRateSmoothingRate*gameSpeedMultiplier())
}

// UpdateLandingApproach 沿定半径圆弧按等减速推进到最终直线进近起点。
func (p *Plane) UpdateLandingApproach(base AircraftBase) bool {
	if p.CurHP <= 0 {
		return false
	}
	p.updateLandingCarrierTurnRate(base)
	p.Flight.Elapsed += gameSpeedMultiplier()

	// 按距离积分推进：每帧先前进再按等减速运动学反算下一帧速度，
	// v² = v² - 2aL 保证离散累计距离与速度计划一致，末端平滑收敛不出跳变
	step := min(p.Landing.arcSpeed, max(p.Landing.arcTotalLength-p.Landing.arcDistance, 0))
	p.Landing.arcDistance += step
	timeProgress := clamp01(p.Landing.arcDistance / max(p.Landing.arcTotalLength, 0.001))
	p.Flight.Progress = timeProgress
	p.Landing.arcSpeed = max(
		p.Landing.arcExitSpeed,
		math.Sqrt(max(0, p.Landing.arcSpeed*p.Landing.arcSpeed-2*p.Landing.arcDecel*step)),
	)

	arcPos := landingArcPoint(p.Landing.arc, timeProgress)
	// 机头航向使用低通转向速率合成：位置已硬绑定随舰旋转的圆弧，
	// 若跟随含原始 omega×r 的合成航向，转向起止时机会瞬间甩动数十度。
	p.advanceLandingAnimation(
		base,
		arcPos.forward,
		arcPos.lateral,
		landingArcWorldRotation(
			p.Landing.arc,
			base,
			timeProgress,
			p.Landing.displayTurnRate,
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
	p.Flight.Elapsed += gameSpeedMultiplier()

	// 等减速刹车：每帧按剩余距离反算速度（v² = v² - 2aL），最后一步恰好停在着舰点
	step := min(p.Landing.runSpeed, max(p.Landing.runTotalLength-p.Landing.runDistance, 0))
	p.Landing.runDistance += step
	timeProgress := clamp01(p.Landing.runDistance / max(p.Landing.runTotalLength, 0.001))
	p.Landing.runSpeed = math.Sqrt(max(0, p.Landing.runSpeed*p.Landing.runSpeed-2*p.Landing.runDecel*step))
	p.Flight.Progress = timeProgress

	// 起点 = 最终进近段起点（舰长单位局部坐标），沿进近方向前进已滑跑距离
	landing := base.BaseAircraft().landingConfigForSlot(p.Landing.Slot)
	end := landingFinalStartOffset(base, landing)
	length := phaseUnitInMapBlocks(base)
	forwardRatio := end.forward + p.Landing.runTangent.forward*(p.Landing.runDistance/length)
	lateralRatio := end.lateral + p.Landing.runTangent.lateral*(p.Landing.runDistance/length)
	p.Flight.EndPos = carrierLandingDeckEndPos(base, landing)
	// 与圆弧段一致，机头航向使用低通转向速率合成，转向时机头贴合甲板进近线
	p.advanceLandingAnimation(
		base,
		forwardRatio,
		lateralRatio,
		landingRunWorldRotation(
			base,
			p.Landing.Slot,
			forwardRatio,
			lateralRatio,
			p.Landing.runSpeed,
			p.Landing.displayTurnRate,
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
	length := phaseUnitInMapBlocks(base)
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
	length := phaseUnitInMapBlocks(base)
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
			p.Landing.carrierTurnRate*current.lateral*length,
		lateral: tangent.lateral*arc.relativeSpeed + correction.lateral +
			p.Landing.carrierTurnRate*current.forward*length,
	}
	targetRotation := normalizeAngle(
		base.BaseRotation() + math.Atan2(velocity.lateral, velocity.forward)*180/math.Pi,
	)
	// 横向修正项最多能把速度顶到 1.75 倍切向速度。宽转弯的重机被它加速后转弯半径
	// 同步变大，反而永远追不上入口；但航母转向时入口点本身带有 omega×r 的运动，
	// 飞机必须跟得上，所以上限以不含修正项的运动学速度为基准给一点余量。
	kinematic := carrierLocalOffset{
		forward: base.BaseSpeed() + tangent.forward*arc.relativeSpeed -
			p.Landing.carrierTurnRate*current.lateral*length,
		lateral: tangent.lateral*arc.relativeSpeed +
			p.Landing.carrierTurnRate*current.forward*length,
	}
	targetSpeed := min(
		math.Hypot(velocity.forward, velocity.lateral),
		math.Hypot(kinematic.forward, kinematic.lateral)*(1+landingGateCorrectionSpeedAllowance),
	)
	executeLandingMovementOnHeading(
		p,
		targetRotation,
		p.approachSpeed(targetSpeed),
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
	if !p.Landing.onWater || p.Flight.Phase != PlaneFlightPhaseLandingDeck {
		return nil
	}
	// 低空缩放完成后视为已触水；触水前保持空中无尾流
	if p.Flight.Progress < p.landingScaleCompleteAt() {
		return nil
	}
	// 节流：每 3 个模拟帧生成一次，避免尾流过密
	if int(p.Flight.Elapsed)%3 != 0 {
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
	length := phaseUnitInMapBlocks(base)
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
