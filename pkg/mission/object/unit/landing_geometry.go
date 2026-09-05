package unit

import (
	"math"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

const (
	// landingStagingBaseForwardRatio 是第一批飞机圆弧入口的纵向位置，单位为舰长。
	// 圆弧终点（最终进近段起点）位于着舰点后方 approachLength 个舰长处，
	// 入口必须保持在终点后方，因此基线取 -5.5 保留约 2 个舰长的圆弧纵深。
	landingStagingBaseForwardRatio = -5.5
	// landingStagingWaveStepRatio 是每满一组入口后向舰尾追加的纵向间距，单位为舰长。
	landingStagingWaveStepRatio = 0.12
	// landingApproachLanes 是每批左右交错的并行入口数量。
	landingApproachLanes = 16
	// landingLaneMinLateralRatio 是最内侧入口距航母中线的舰长倍率。
	landingLaneMinLateralRatio = 0.8
	// landingLaneMaxLateralRatio 是最外侧入口距航母中线的舰长倍率。
	landingLaneMaxLateralRatio = 1.4
	// landingLeadInLengthRatio 是圆弧入口切线向后的直线引导长度，单位为舰长。
	landingLeadInLengthRatio = 1.5
	// landingLeadInCaptureRadiusRatio 是从远端引导点切换到入口直线段的捕获半径。
	landingLeadInCaptureRadiusRatio = 0.30
	// landingLeadInMaxLeadRatio 是 lead-in 前瞻预测点的最大距离，单位为舰长。
	// 限制预测跨度，避免目标位移估算在急转弯时把瞄准点甩得过远。
	landingLeadInMaxLeadRatio = 3.0
	// landingLeadInMotionCaptureFrames 是捕获半径随 lead-in 运动速度放大的折算帧数。
	landingLeadInMotionCaptureFrames = 12.0
	// landingGateErrorCorrectionShare 是横向误差较大时每帧至少修正的比例。
	landingGateErrorCorrectionShare = 0.3
	// landingGateForwardToleranceRatio 是进入圆弧前允许的纵向位置误差，单位为舰长。
	landingGateForwardToleranceRatio = 0.25
	// landingGateLateralToleranceRatio 是进入圆弧前允许的横向位置误差，单位为舰长。
	landingGateLateralToleranceRatio = 0.20
	// landingGateMissDistanceRatio 是飞机越过入口后触发重新引导的切向距离。
	landingGateMissDistanceRatio = 0.25
	// landingGateHeadingTolerance 是进入圆弧前允许的航向误差，单位为度。
	landingGateHeadingTolerance = 8.0
	// landingGateSpeedTolerance 是进入圆弧前允许的速度相对误差。
	landingGateSpeedTolerance = 0.10
	// landingApproachMinFrames 是圆弧进近允许的最短模拟帧数。
	landingApproachMinFrames = 90.0
	// landingApproachMaxFrames 是圆弧进近允许的最长模拟帧数。
	landingApproachMaxFrames = 180.0
	// landingApproachSpeedRatio 是推导圆弧动画时长所用的参考速度比例。
	landingApproachSpeedRatio = 0.40
	// seaLandingSplashdownAftOfSternRatio 是水上飞机触水点在舰尾后方的舰长倍数。
	seaLandingSplashdownAftOfSternRatio = 0.8
)

// carrierLocalOffset 是以航母为原点的局部坐标。
// forward 朝舰艏为正，lateral 朝舰体右舷为正。圆弧结构中的坐标均以舰长为单位，
// 只有转换为 MapPos 或计算实际速度时才乘以 carrierLengthInMapBlocks。
type carrierLocalOffset struct {
	forward float64
	lateral float64
}

// landingApproachArc 保存绑定航母局部坐标的定半径进近圆弧。
type landingApproachArc struct {
	// center、radius 和 start 均以舰长为单位。
	center carrierLocalOffset
	radius float64
	// startAngle 和 sweepAngle 使用弧度；sweepAngle 的符号表示转弯方向。
	startAngle float64
	sweepAngle float64
	// frames 是参考动画时长，relativeSpeed 是参考圆弧切向速度，
	// 二者只用于入口引导的目标速度推导；实际进近按距离积分减速推进。
	frames        float64
	relativeSpeed float64
	start         carrierLocalOffset
}

// carrierRelativePos 返回基地中线上指定前后偏移量对应的地图坐标。
func carrierRelativePos(base AircraftBase, offset float64) objPos.MapPos {
	return carrierRelativePos2D(base, offset, 0)
}

// carrierRelativePos2D 将基地局部坐标旋转、平移到地图坐标。
func carrierRelativePos2D(base AircraftBase, forward, lateral float64) objPos.MapPos {
	radians := base.BaseRotation() * math.Pi / 180
	sinVal, cosVal := math.Sin(radians), math.Cos(radians)
	pos := base.BasePos()
	return objPos.NewR(
		pos.RX+sinVal*forward+cosVal*lateral,
		pos.RY-cosVal*forward+sinVal*lateral,
	)
}

// planeCarrierLocalOffset 是 carrierRelativePos2D 的逆变换。
func planeCarrierLocalOffset(p *Plane, base AircraftBase) carrierLocalOffset {
	radians := base.BaseRotation() * math.Pi / 180
	sinVal, cosVal := math.Sin(radians), math.Cos(radians)
	pos := base.BasePos()
	dx, dy := p.CurPos.RX-pos.RX, p.CurPos.RY-pos.RY
	return carrierLocalOffset{
		forward: dx*sinVal - dy*cosVal,
		lateral: dx*cosVal + dy*sinVal,
	}
}

// landingTouchdownOffset 返回着舰回收点的基地局部地图坐标。
func landingTouchdownOffset(base AircraftBase, landing LandingConfig) carrierLocalOffset {
	return takeoffLandingPos(base, landing)
}

// landingScaleCompleteRatio 返回直线进近段降到最低视觉倍率（触舰/触水）时的路程比例。
// 甲板回收仍用固定 80%；着水回收把触水点放在舰尾后方 0.8 个舰长。
func landingScaleCompleteRatio(base AircraftBase, landing LandingConfig) float64 {
	if landing.Mode != LandingModeSea {
		return landingDeckScaleDistanceRatio
	}
	start := landingFinalStartOffset(base, landing)
	endForward := 0.5 - landing.Forward
	splashForward := -0.5 - seaLandingSplashdownAftOfSternRatio
	run := endForward - start.forward
	if math.Abs(run) < 1e-6 {
		return landingDeckScaleDistanceRatio
	}
	return max(0.05, min(0.95, (splashForward-start.forward)/run))
}

// landingFinalStartOffset 返回最终直线进近段起点的舰长单位局部坐标。
// 起点 = 着舰点沿反进近方向后退 approachLength 个舰长，圆弧在此与直线段衔接。
func landingFinalStartOffset(base AircraftBase, landing LandingConfig) carrierLocalOffset {
	length := carrierLengthInMapBlocks(base)
	touchdown := takeoffLandingPos(base, landing)
	tangent := landingApproachTangent(landing)
	run := length * landing.ApproachLength
	return carrierLocalOffset{
		forward: (touchdown.forward - tangent.forward*run) / length,
		lateral: (touchdown.lateral - tangent.lateral*run) / length,
	}
}

// landingFinalStartPos 返回最终直线进近段起点的地图坐标。
func landingFinalStartPos(base AircraftBase, landing LandingConfig) objPos.MapPos {
	end := landingFinalStartOffset(base, landing)
	length := carrierLengthInMapBlocks(base)
	return carrierRelativePos2D(base, end.forward*length, end.lateral*length)
}

// carrierLandingDeckEndPos 返回最终着舰回收点的地图坐标。
func carrierLandingDeckEndPos(base AircraftBase, landing LandingConfig) objPos.MapPos {
	touchdown := landingTouchdownOffset(base, landing)
	return carrierRelativePos2D(base, touchdown.forward, touchdown.lateral)
}

// landingLaneOffsetRatio 将稳定槽位映射到左右交替的 16 条进近通道。
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

// landingStagingTarget 返回指定稳定槽位对应的圆弧入口地图坐标。
func (sa *ShipAircraft) landingStagingTarget(
	_ *mapcfg.MapCfg,
	base AircraftBase,
	slot int,
) objPos.MapPos {
	length := carrierLengthInMapBlocks(base)
	gate := landingGateLocalOffset(slot)
	return carrierRelativePos2D(base, length*gate.forward, length*gate.lateral)
}

// landingGateLocalOffset 返回槽位对应的航母局部入口；超过 16 架后按批次向舰尾错开。
func landingGateLocalOffset(slot int) carrierLocalOffset {
	wave := slot / landingApproachLanes
	return carrierLocalOffset{
		forward: landingStagingBaseForwardRatio - float64(wave)*landingStagingWaveStepRatio,
		lateral: landingLaneOffsetRatio(slot),
	}
}

// buildLandingApproachArc 构造一条从实际入口汇入最终直线进近段的定半径圆弧。
// 圆弧终点为最终进近段起点，终点切线沿进近方向（approachAngle 决定），
// 因此圆心位于终点法向上；再令圆心到起点和终点的距离相等可得 R=|w|²/(2·w·n)。
// 该构造保证整段只向一侧转弯，横向偏差单调收敛，不会形成反曲。
func buildLandingApproachArc(
	start carrierLocalOffset,
	base AircraftBase,
	maxSpeed float64,
	landing LandingConfig,
) (landingApproachArc, bool) {
	length := carrierLengthInMapBlocks(base)
	end := landingFinalStartOffset(base, landing)
	tangent := landingApproachTangent(landing)

	// 法向量取切线左侧；起点在右侧时翻转到右侧，保证 w·n > 0
	normal := carrierLocalOffset{forward: -tangent.lateral, lateral: tangent.forward}
	wf := start.forward - end.forward
	wl := start.lateral - end.lateral
	// 起点必须位于终点后方（沿反进近方向），否则无法前向汇入
	if wf*tangent.forward+wl*tangent.lateral >= -0.001 {
		return landingApproachArc{}, false
	}
	dotNormal := wf*normal.forward + wl*normal.lateral
	side := 1.0
	if dotNormal < 0 {
		side = -1
		normal.forward, normal.lateral = -normal.forward, -normal.lateral
		dotNormal = -dotNormal
	}
	if dotNormal <= 0.001 {
		return landingApproachArc{}, false
	}

	radius := (wf*wf + wl*wl) / (2 * dotNormal)
	center := carrierLocalOffset{
		forward: end.forward + normal.forward*radius,
		lateral: end.lateral + normal.lateral*radius,
	}
	startAngle := math.Atan2(start.lateral-center.lateral, start.forward-center.forward)
	endAngle := math.Atan2(-normal.lateral, -normal.forward)
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

// landingArcPoint 按归一化进度采样圆弧上的航母局部坐标。
func landingArcPoint(arc landingApproachArc, progress float64) carrierLocalOffset {
	angle := arc.startAngle + arc.sweepAngle*clamp01(progress)
	return carrierLocalOffset{
		forward: arc.center.forward + arc.radius*math.Cos(angle),
		lateral: arc.center.lateral + arc.radius*math.Sin(angle),
	}
}

// landingArcTangent 返回单位切向量；sweepAngle 的符号同时决定左右航线的转弯方向。
func landingArcTangent(arc landingApproachArc, progress float64) carrierLocalOffset {
	angle := arc.startAngle + arc.sweepAngle*clamp01(progress)
	forward := -math.Sin(angle) * arc.sweepAngle
	lateral := math.Cos(angle) * arc.sweepAngle
	magnitude := math.Hypot(forward, lateral)
	return carrierLocalOffset{forward: forward / magnitude, lateral: lateral / magnitude}
}

// landingLeadInLocalOffset 返回圆弧入口沿反切线方向后退 1.5 个舰长的引导点。
func landingLeadInLocalOffset(arc landingApproachArc) carrierLocalOffset {
	tangent := landingArcTangent(arc, 0)
	return carrierLocalOffset{
		forward: arc.start.forward - tangent.forward*landingLeadInLengthRatio,
		lateral: arc.start.lateral - tangent.lateral*landingLeadInLengthRatio,
	}
}

// landingArcEntryTargetSpeed 返回飞机进入圆弧首帧所需的世界速度大小。
func landingArcEntryTargetSpeed(
	arc landingApproachArc,
	base AircraftBase,
	turnRate float64,
) float64 {
	velocity := landingArcWorldVelocity(arc, base, 0, turnRate)
	return math.Hypot(velocity.forward, velocity.lateral)
}

// landingArcWorldVelocity 将相对基地的圆弧速度转换为实际地图速度：
// 基地平移速度 + 圆弧切向速度 + 基地旋转产生的 omega×r 切向速度。
func landingArcWorldVelocity(
	arc landingApproachArc,
	base AircraftBase,
	progress, turnRate float64,
) carrierLocalOffset {
	tangent := landingArcTangent(arc, progress)
	point := landingArcPoint(arc, progress)
	length := carrierLengthInMapBlocks(base)
	return carrierLocalOffset{
		forward: base.BaseSpeed() + tangent.forward*arc.relativeSpeed -
			turnRate*point.lateral*length,
		lateral: tangent.lateral*arc.relativeSpeed + turnRate*point.forward*length,
	}
}

// landingArcWorldRotation 将圆弧上的世界速度向量转换为游戏航向角。
func landingArcWorldRotation(
	arc landingApproachArc,
	base AircraftBase,
	progress, turnRate float64,
) float64 {
	velocity := landingArcWorldVelocity(arc, base, progress, turnRate)
	return normalizeAngle(
		base.BaseRotation() + math.Atan2(velocity.lateral, velocity.forward)*180/math.Pi,
	)
}

// landingApproachEntryReady 只允许位置、航向和速度都落入入口容差的飞机进入固定圆弧。
// 不满足条件的飞机继续执行远端切线引导，避免从舰侧强行接入造成锐角转弯。
func landingApproachEntryReady(p *Plane, base AircraftBase, gate carrierLocalOffset) bool {
	length := carrierLengthInMapBlocks(base)
	local := planeCarrierLocalOffset(p, base)
	start := carrierLocalOffset{
		forward: local.forward / length,
		lateral: local.lateral / length,
	}
	if start.lateral*gate.lateral <= 0 ||
		math.Abs(start.forward-gate.forward) > landingGateForwardToleranceRatio ||
		math.Abs(start.lateral-gate.lateral) > landingGateLateralToleranceRatio {
		return false
	}
	arc, ok := buildLandingApproachArc(
		start, base, p.MaxSpeed, base.BaseAircraft().landingConfigForSlot(p.LandingSlot),
	)
	if !ok {
		return false
	}
	entrySpeed := landingArcEntryTargetSpeed(arc, base, p.landingCarrierTurnRate)
	// 航向容差与圆弧段机头合成使用同一低通转向速率，保证进入圆弧前后
	// 机头目标航向连续；速度容差仍用原始速率，反映真实的相对闭合速度。
	targetRotation := landingArcWorldRotation(arc, base, 0, p.landingDisplayTurnRate)
	if angleDifferenceDegrees(p.CurRotation, targetRotation) > landingGateHeadingTolerance {
		return false
	}
	speedTolerance := max(entrySpeed*landingGateSpeedTolerance, p.phaseSpeedStep(entrySpeed))
	return math.Abs(p.CurSpeed-entrySpeed) <= speedTolerance
}
