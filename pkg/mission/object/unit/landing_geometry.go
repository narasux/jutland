package unit

import (
	"math"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

const (
	// carrierLandingFinalStartRatio 是最终直线着舰段起点，位于航母中心后方 1.5 个舰长。
	carrierLandingFinalStartRatio = -1.5

	// landingStagingBaseForwardRatio 是第一批飞机圆弧入口的纵向位置，单位为舰长。
	landingStagingBaseForwardRatio = -3.5
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
	landingApproachMaxFrames = 120.0
	// landingApproachSpeedRatio 是推导圆弧动画时长所用的参考速度比例。
	landingApproachSpeedRatio = 0.40
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
	// frames 是动画时长，relativeSpeed 是飞机相对航母的圆弧切向速度。
	frames        float64
	relativeSpeed float64
	start         carrierLocalOffset
}

// carrierRelativePos 返回航母中线上指定前后偏移量对应的地图坐标。
func carrierRelativePos(ship *BattleShip, offset float64) objPos.MapPos {
	return carrierRelativePos2D(ship, offset, 0)
}

// carrierRelativePos2D 将航母局部坐标旋转、平移到地图坐标。
func carrierRelativePos2D(ship *BattleShip, forward, lateral float64) objPos.MapPos {
	radians := ship.CurRotation * math.Pi / 180
	sinVal, cosVal := math.Sin(radians), math.Cos(radians)
	return objPos.NewR(
		ship.CurPos.RX+sinVal*forward+cosVal*lateral,
		ship.CurPos.RY-cosVal*forward+sinVal*lateral,
	)
}

// planeCarrierLocalOffset 是 carrierRelativePos2D 的逆变换。
func planeCarrierLocalOffset(p *Plane, ship *BattleShip) carrierLocalOffset {
	radians := ship.CurRotation * math.Pi / 180
	sinVal, cosVal := math.Sin(radians), math.Cos(radians)
	dx, dy := p.CurPos.RX-ship.CurPos.RX, p.CurPos.RY-ship.CurPos.RY
	return carrierLocalOffset{
		forward: dx*sinVal - dy*cosVal,
		lateral: dx*cosVal + dy*sinVal,
	}
}

// carrierLandingFinalStartPos 返回舰尾后 1.5 个舰长处的最终直线着舰起点。
func carrierLandingFinalStartPos(ship *BattleShip) objPos.MapPos {
	return carrierRelativePos(ship, carrierLengthInMapBlocks(ship)*carrierLandingFinalStartRatio)
}

// carrierLandingDeckEndPos 返回最终回收点；当前设计固定为甲板中心。
func carrierLandingDeckEndPos(ship *BattleShip) objPos.MapPos {
	return ship.CurPos.Copy()
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
	ship *BattleShip,
	slot int,
) objPos.MapPos {
	length := carrierLengthInMapBlocks(ship)
	gate := landingGateLocalOffset(slot)
	return carrierRelativePos2D(ship, length*gate.forward, length*gate.lateral)
}

// landingGateLocalOffset 返回槽位对应的航母局部入口；超过 16 架后按批次向舰尾错开。
func landingGateLocalOffset(slot int) carrierLocalOffset {
	wave := slot / landingApproachLanes
	return carrierLocalOffset{
		forward: landingStagingBaseForwardRatio - float64(wave)*landingStagingWaveStepRatio,
		lateral: landingLaneOffsetRatio(slot),
	}
}

// buildLandingApproachArc 构造一条从实际入口汇入舰尾中线的定半径圆弧。
// 终点切线必须沿 forward 方向，因此圆心的 forward 坐标等于终点；再令圆心到
// 起点和终点的距离相等，可得 R=(dx²+y²)/(2|y|)。该构造保证整段只向前转弯，
// 横向偏差单调收敛到 0，不会形成 Bézier 曲线可能出现的反曲。
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
	ship *BattleShip,
	turnRate float64,
) float64 {
	velocity := landingArcWorldVelocity(arc, ship, 0, turnRate)
	return math.Hypot(velocity.forward, velocity.lateral)
}

// landingArcWorldVelocity 将相对航母的圆弧速度转换为实际地图速度：
// 航母平移速度 + 圆弧切向速度 + 航母旋转产生的 omega×r 切向速度。
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

// landingArcWorldRotation 将圆弧上的世界速度向量转换为游戏航向角。
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

// landingApproachEntryReady 只允许位置、航向和速度都落入入口容差的飞机进入固定圆弧。
// 不满足条件的飞机继续执行远端切线引导，避免从舰侧强行接入造成锐角转弯。
func landingApproachEntryReady(p *Plane, ship *BattleShip, gate carrierLocalOffset) bool {
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
