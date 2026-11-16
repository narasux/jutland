package unit

import (
	"math"
	"testing"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// TestLandingQueueAssignsStableSlotsAndReusesThem 验证着舰队列按申请顺序分配稳定槽位，取消后空位可复用。
// 场景：多架飞机同时返航，航母着舰位有限；先到的飞机应先进入进近，复飞/取消后不能留下永久空洞。
// 预期：首次申请依次得到 0..n；取消中间槽位后，新申请复用该槽位而不是继续递增。
// 覆盖：连续申请 5 个槽位，取消第 2 个，再申请第 6 架，断言其复用槽位 1。
func TestLandingQueueAssignsStableSlotsAndReusesThem(t *testing.T) {
	aircraft := ShipAircraft{}
	for idx, uid := range []string{"plane-a", "plane-b", "plane-c", "plane-d", "plane-e"} {
		slot := aircraft.RequestLanding(uid)
		if slot != idx {
			t.Fatalf("plane %s slot = %d, want %d", uid, slot, idx)
		}
	}

	aircraft.CancelLanding("plane-b")
	if slot := aircraft.RequestLanding("plane-f"); slot != 1 {
		t.Fatalf("reused slot = %d, want slot 1", slot)
	}
}

// TestLandingStagingTargetsAreSeparated 验证同一波次和相邻波次的待场点不会重叠。
// 场景：多机在航母上方盘旋等待；若待场点重合，飞机模型会叠在一起并干扰进近排序。
// 预期：相邻槽位横向错开，跨波次前后错开，最近两点的地图距离保持安全间隔。
// 覆盖：计算 slot 0/1/下一波/最左槽位的待场点，转成航母局部坐标后断言横向比例与距离。
func TestLandingStagingTargetsAreSeparated(t *testing.T) {
	ship := &BattleShip{
		Length:      256,
		CurPos:      objPos.NewR(20, 20),
		CurRotation: 0,
	}
	aircraft := ShipAircraft{}
	first := aircraft.landingStagingTarget(nil, ship, 0)
	second := aircraft.landingStagingTarget(nil, ship, 1)
	nextWave := aircraft.landingStagingTarget(nil, ship, landingApproachLanes)
	firstLocal := planeCarrierLocalOffset(&Plane{CurPos: first}, ship)
	secondLocal := planeCarrierLocalOffset(&Plane{CurPos: second}, ship)
	lastLeft := aircraft.landingStagingTarget(nil, ship, landingApproachLanes-2)
	lastLeftLocal := planeCarrierLocalOffset(&Plane{CurPos: lastLeft}, ship)

	requireClose(t, firstLocal.forward/carrierLengthInMapBlocks(ship), landingStagingBaseForwardRatio)
	requireClose(t, firstLocal.lateral/carrierLengthInMapBlocks(ship), -landingLaneMinLateralRatio)
	requireClose(t, secondLocal.lateral/carrierLengthInMapBlocks(ship), landingLaneMinLateralRatio)
	requireClose(t, lastLeftLocal.lateral/carrierLengthInMapBlocks(ship), -landingLaneMaxLateralRatio)

	if first.Distance(second) < 0.2 {
		t.Fatalf("adjacent staging targets overlap: distance = %v", first.Distance(second))
	}
	if first.Distance(nextWave) < 0.2 {
		t.Fatalf("successive staging waves overlap: distance = %v", first.Distance(nextWave))
	}
}

// TestLandingStagingWavesReachApproachGates 验证首波和后续波的待场飞机都能飞到进近入口。
// 场景：返航飞机被分配到第 0、1、3 波待场位，航母保持直航；所有飞机最终都应能进场而不是卡在盘旋。
// 预期：在 800 帧内 UpdateLandingStaging 返回 true（已到达入口门）。
// 覆盖：对三个代表性槽位启动待场，循环推进并断言 reachedGate。
func TestLandingStagingWavesReachApproachGates(t *testing.T) {
	useDefaultSettings(t)
	for _, slot := range []int{0, landingApproachLanes, landingApproachLanes * 3} {
		ship := &BattleShip{
			Length:      256,
			CurPos:      objPos.NewR(50, 50),
			CurRotation: 0,
		}
		plane := &Plane{
			MaxSpeed:     0.1,
			Acceleration: 0.01,
			RotateSpeed:  12,
			CurHP:        100,
			CurPos:       objPos.NewR(50, 35),
			CurRotation:  180,
			CurSpeed:     0.1,
			RemainRange:  1000,
		}
		plane.StartLandingStaging(nil, ship, slot)
		reachedGate := false
		for range 800 {
			if plane.UpdateLandingStaging(nil, ship) {
				reachedGate = true
				break
			}
		}
		if !reachedGate {
			local := planeCarrierLocalOffset(plane, ship)
			t.Fatalf(
				"staging slot %d never reached its recovery gate: leg=%v local=%+v heading=%.2f speed=%.3f",
				slot,
				plane.landingStagingLeg,
				local,
				plane.CurRotation,
				plane.CurSpeed,
			)
		}
	}
}

// TestLandingStagingFeedsCurveWithoutSharpEntry 验证从待场切入进近圆弧时航向连续，不出现急拐。
// 场景：飞机从航母四周八个方向进入待场，随后切入最终进近；入口切线不匹配时画面会瞬间甩头。
// 预期：切入后首帧的实际运动方向与切入前机头夹角不超过 30°。
// 覆盖：八个起始方位各跑一次待场到入口，比较 StartLandingApproach 前后的运动航向。
func TestLandingStagingFeedsCurveWithoutSharpEntry(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		CurSpeed:    0.04,
	}
	starts := []carrierLocalOffset{
		{forward: -8},
		{forward: -5, lateral: -5},
		{lateral: -8},
		{forward: 5, lateral: -5},
		{forward: 8},
		{forward: 5, lateral: 5},
		{lateral: 8},
		{forward: -5, lateral: 5},
	}
	for slot, start := range starts {
		plane := &Plane{
			MaxSpeed:     0.12,
			Acceleration: 0.01,
			RotateSpeed:  12,
			CurHP:        100,
			CurPos:       carrierRelativePos2D(ship, start.forward, start.lateral),
			CurRotation:  float64(slot * 45),
			CurSpeed:     0.12,
			RemainRange:  1000,
		}
		plane.StartLandingStaging(nil, ship, slot)
		reachedGate := false
		for range 800 {
			moveTestCarrier(ship, 0)
			if plane.UpdateLandingStaging(nil, ship) {
				reachedGate = true
				break
			}
		}
		if !reachedGate {
			t.Fatalf("staging path %d did not reach its approach gate", slot)
		}

		headingBefore := plane.CurRotation
		positionBefore := plane.CurPos.Copy()
		plane.StartLandingApproach(ship)
		moveTestCarrier(ship, 0)
		plane.UpdateLandingApproach(ship)
		movementHeading := positionBefore.Angle(plane.CurPos)
		if delta := angleDifference(movementHeading, headingBefore); delta > 30 {
			t.Fatalf(
				"staging path %d enters landing curve with %.2f degree path jump",
				slot,
				delta,
			)
		}
	}
}

// TestLandingStagingEventuallyCapturesAcrossAircraftPerformanceRange 验证不同性能飞机/航母运动组合
// 都能完成待场捕获。
// 场景：从低速鱼雷机到高速机，航母静止/直航/转弯，玩家从任意方位返航；着舰逻辑不能只对某类机型成立。
// 预期：所有组合在 900 帧内进入进近入口，失败数为 0。
// 覆盖：7 种机速/转向 × 5 种航母运动 × 8 个进入角 × 12 个初始航向的矩阵循环。
func TestLandingStagingEventuallyCapturesAcrossAircraftPerformanceRange(t *testing.T) {
	useDefaultSettings(t)
	performanceCases := []struct {
		maxSpeedKPH float64
		rotateSpeed float64
	}{
		{maxSpeedKPH: 332, rotateSpeed: 6},
		{maxSpeedKPH: 436, rotateSpeed: 6},
		{maxSpeedKPH: 482, rotateSpeed: 7},
		{maxSpeedKPH: 567, rotateSpeed: 8},
		{maxSpeedKPH: 644, rotateSpeed: 14},
		{maxSpeedKPH: 1300, rotateSpeed: 10},
		{maxSpeedKPH: 2500, rotateSpeed: 25},
	}
	carrierCases := []struct {
		speed        float64
		rotationStep float64
	}{
		{speed: 0, rotationStep: 0},
		{speed: 0.055, rotationStep: 0},
		{speed: 0.055, rotationStep: 0.05},
		{speed: 0.055, rotationStep: 0.25},
		{speed: 0.055, rotationStep: 0.8},
	}
	failedCases := 0
	for _, performance := range performanceCases {
		for _, carrierCase := range carrierCases {
			for entryAngle := 0.0; entryAngle < 360; entryAngle += 45 {
				for rotation := 0.0; rotation < 360; rotation += 30 {
					ship := &BattleShip{
						Length:      128,
						CurPos:      objPos.NewR(50, 50),
						CurRotation: 0,
						CurSpeed:    carrierCase.speed,
					}
					entry := (&ShipAircraft{}).landingStagingTarget(nil, ship, 0)
					radians := entryAngle * math.Pi / 180
					plane := &Plane{
						MaxSpeed:     performance.maxSpeedKPH / 5400,
						Acceleration: 0.001,
						RotateSpeed:  performance.rotateSpeed,
						CurHP:        100,
						CurPos: objPos.NewR(
							entry.RX+math.Sin(radians),
							entry.RY-math.Cos(radians),
						),
						CurRotation: rotation,
						CurSpeed:    performance.maxSpeedKPH / 5400,
						RemainRange: 1000,
					}
					plane.StartLandingStaging(nil, ship, 0)
					captured := false
					for range 900 {
						moveTestCarrier(ship, carrierCase.rotationStep)
						if plane.UpdateLandingStaging(nil, ship) {
							captured = true
							break
						}
					}
					if !captured {
						failedCases++
						if failedCases <= 10 {
							t.Logf(
								"failed: plane=%.0f/%.0f carrier=%.3f/%.3f entry=%.0f heading=%.0f distance=%.2f ready=%t",
								performance.maxSpeedKPH,
								performance.rotateSpeed,
								carrierCase.speed,
								carrierCase.rotationStep,
								entryAngle,
								rotation,
								plane.CurPos.Distance(ship.CurPos),
								plane.landingStagingLeg == landingStagingLegGate,
							)
						}
					}
				}
			}
		}
	}
	if failedCases > 0 {
		t.Fatalf("planes failed to capture landing entry in %d performance cases", failedCases)
	}
}

// TestLandingStagingCapturesWhenCarrierSternIsNearMapBorder 验证航母贴地图边缘时待场点被夹取后仍可进场。
// 场景：航母位于左/右/上/下边缘，舰尾待场点会超出地图；若无边界处理，飞机会追一个地图外的点。
// 预期：待场点经 EnsureBorder 夹取后，各槽位/各初始航向仍能在 1200 帧内捕获入口。
// 覆盖：四个边缘位置 × 6 个槽位 × 8 个航向，循环推进 UpdateLandingStaging。
func TestLandingStagingCapturesWhenCarrierSternIsNearMapBorder(t *testing.T) {
	useDefaultSettings(t)
	mapCfg := &mapcfg.MapCfg{Width: 100, Height: 100}
	testCases := []struct {
		name             string
		carrierPos       objPos.MapPos
		carrierRotation  float64
		planeStartOffset carrierLocalOffset
	}{
		{
			name:             "left",
			carrierPos:       objPos.NewR(1, 50),
			carrierRotation:  90,
			planeStartOffset: carrierLocalOffset{forward: 0.75},
		},
		{
			name:             "right",
			carrierPos:       objPos.NewR(97, 50),
			carrierRotation:  270,
			planeStartOffset: carrierLocalOffset{forward: 0.75},
		},
		{
			name:             "top",
			carrierPos:       objPos.NewR(50, 1),
			carrierRotation:  180,
			planeStartOffset: carrierLocalOffset{forward: 0.75},
		},
		{
			name:             "bottom",
			carrierPos:       objPos.NewR(50, 97),
			carrierRotation:  0,
			planeStartOffset: carrierLocalOffset{forward: 0.75},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for _, slot := range []int{0, 7, 15, 16, 31, 47} {
				for heading := 0.0; heading < 360; heading += 45 {
					ship := &BattleShip{
						Length:      256,
						CurPos:      testCase.carrierPos,
						CurRotation: testCase.carrierRotation,
					}
					target := (&ShipAircraft{}).landingStagingTarget(mapCfg, ship, slot)
					start := carrierRelativePos2D(
						&BattleShip{CurPos: target, CurRotation: ship.CurRotation},
						testCase.planeStartOffset.forward,
						testCase.planeStartOffset.lateral,
					)
					start.EnsureBorder(float64(mapCfg.Width-2), float64(mapCfg.Height-2))
					plane := &Plane{
						MaxSpeed:     0.1,
						Acceleration: 0.01,
						RotateSpeed:  12,
						CurHP:        100,
						CurPos:       start,
						CurRotation:  heading,
						CurSpeed:     0.1,
						RemainRange:  1000,
					}

					plane.StartLandingStaging(mapCfg, ship, slot)
					captured := false
					for range 1200 {
						if plane.UpdateLandingStaging(mapCfg, ship) {
							captured = true
							break
						}
					}
					if !captured {
						t.Fatalf(
							"slot %d heading %.0f never captured a reachable landing entry",
							slot,
							heading,
						)
					}
				}
			}
		})
	}
}

// TestLandingStagingMissedGateReturnsToLeadIn 验证飞机冲过入口门后回到 lead-in，而不是直接切入圆弧。
// 场景：飞机速度过快或转弯半径过大，掠过入口点时位置已越过门限；若仍进入圆弧，会从错误方向横穿甲板。
// 预期：UpdateLandingStaging 返回 false，landingStagingLeg 重置为 lead-in。
// 覆盖：构造位于门后 0.4 个切线单位、航向沿切线的飞机，调用一次 Update 并断言阶段。
func TestLandingStagingMissedGateReturnsToLeadIn(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		CurSpeed:    0.04,
	}
	length := carrierLengthInMapBlocks(ship)
	gate := landingGateLocalOffset(0)
	arc, ok := buildLandingApproachArc(gate, ship, 0.12, ship.Aircraft.landingConfig())
	if !ok {
		t.Fatal("failed to build test landing arc")
	}
	tangent := landingArcTangent(arc, 0)
	missed := carrierLocalOffset{
		forward: gate.forward + tangent.forward*0.4,
		lateral: gate.lateral + tangent.lateral*0.4,
	}
	plane := &Plane{
		MaxSpeed:     0.12,
		Acceleration: 0.005,
		RotateSpeed:  12,
		CurHP:        100,
		CurPos:       carrierRelativePos2D(ship, length*missed.forward, length*missed.lateral),
		CurRotation: normalizeAngle(
			ship.CurRotation + math.Atan2(tangent.lateral, tangent.forward)*180/math.Pi,
		),
		CurSpeed:    landingArcEntryTargetSpeed(arc, ship, 0),
		RemainRange: 1000,
	}
	plane.StartLandingStaging(nil, ship, 0)
	plane.landingStagingLeg = landingStagingLegGate
	if plane.UpdateLandingStaging(nil, ship) {
		t.Fatal("plane entered landing arc after missing the gate")
	}
	if plane.landingStagingLeg != landingStagingLegLeadIn {
		t.Fatalf("staging leg = %v, want lead-in reset", plane.landingStagingLeg)
	}
}

// TestLandingGateSpeedConvergesWithoutJump 验证进场前速度按每帧步长上限收敛，不会瞬间跳变。
// 场景：飞机以巡航速度接近入口，需要在进入圆弧前减速到入口目标速度；直接赋值会让速度曲线出现断点。
// 预期：每帧速度变化不超过 phaseSpeedStep 给出的上限。
// 覆盖：从 lead-in 开始连续 30 帧 UpdateLandingStaging，逐帧比较速度差。
func TestLandingGateSpeedConvergesWithoutJump(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		CurSpeed:    0.04,
	}
	length := carrierLengthInMapBlocks(ship)
	gate := landingGateLocalOffset(0)
	arc, ok := buildLandingApproachArc(gate, ship, 0.12, ship.Aircraft.landingConfig())
	if !ok {
		t.Fatal("failed to build test landing arc")
	}
	leadIn := landingLeadInLocalOffset(arc)
	tangent := landingArcTangent(arc, 0)
	plane := &Plane{
		MaxSpeed:     0.12,
		Acceleration: 0.005,
		RotateSpeed:  12,
		CurHP:        100,
		CurPos:       carrierRelativePos2D(ship, length*leadIn.forward, length*leadIn.lateral),
		CurRotation: normalizeAngle(
			ship.CurRotation + math.Atan2(tangent.lateral, tangent.forward)*180/math.Pi,
		),
		CurSpeed:    0.04,
		RemainRange: 1000,
	}
	plane.StartLandingStaging(nil, ship, 0)
	previousSpeed := plane.CurSpeed
	for range 30 {
		stepLimit := plane.phaseSpeedStep(landingArcEntryTargetSpeed(arc, ship, 0))
		plane.UpdateLandingStaging(nil, ship)
		if math.Abs(plane.CurSpeed-previousSpeed) > stepLimit+1e-9 {
			t.Fatalf("landing speed jumped: %v -> %v", previousSpeed, plane.CurSpeed)
		}
		previousSpeed = plane.CurSpeed
	}
}

// TestLandingStagingSpeedUsesCarrierDistanceEnvelope 验证待场速度随距入口距离渐变，而不是固定值。
// 场景：飞机离入口很远时应保持巡航速度，接近内圈时逐步减速；否则会在远处提前爬行或到门口急刹。
// 预期：内圈边界速度等于 entrySpeed，外圈边界速度等于 MaxSpeed，且近处速度小于远处。
// 覆盖：直接调用 landingStagingTargetSpeed 传入内/外圈距离，断言两档速度与单调性。
func TestLandingStagingSpeedUsesCarrierDistanceEnvelope(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{Length: 128, CurSpeed: 0.04}
	plane := &Plane{MaxSpeed: 0.12}
	length := carrierLengthInMapBlocks(ship)
	gate := landingGateLocalOffset(0)
	gateDistance := length * math.Hypot(gate.forward, gate.lateral)
	entrySpeed := 0.08
	nearSpeed := plane.landingStagingTargetSpeed(
		ship,
		length,
		gateDistance+length*landingStagingSlowdownInnerOffset,
		gateDistance,
		entrySpeed,
		0,
	)
	farSpeed := plane.landingStagingTargetSpeed(
		ship,
		length,
		gateDistance+length*(landingStagingSlowdownInnerOffset+landingStagingSlowdownOuterSpan),
		gateDistance,
		entrySpeed,
		0,
	)

	requireClose(t, nearSpeed, entrySpeed)
	requireClose(t, farSpeed, plane.MaxSpeed)
	if nearSpeed >= farSpeed {
		t.Fatalf("landing speed envelope did not decelerate: near=%v far=%v", nearSpeed, farSpeed)
	}
}

// TestLandingApproachRejectsInvalidEntryPosition 验证飞机在航母侧方/前方时不能直接进入进近圆弧。
// 场景：玩家手动点选着舰或 AI 复飞后位置异常，飞机可能位于舰艏一侧；若接受，会从错误方向切上甲板。
// 预期：landingApproachEntryReady 返回 false。
// 覆盖：构造航母侧前方 0.75 舰长处的飞机，调用入口检查并断言拒绝。
func TestLandingApproachRejectsInvalidEntryPosition(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{Length: 128, CurPos: objPos.NewR(50, 50), CurSpeed: 0.04}
	length := carrierLengthInMapBlocks(ship)
	gate := landingGateLocalOffset(0)
	arc, ok := buildLandingApproachArc(gate, ship, 0.12, ship.Aircraft.landingConfig())
	if !ok {
		t.Fatal("failed to build test landing arc")
	}
	plane := &Plane{
		MaxSpeed:     0.12,
		Acceleration: 0.005,
		RotateSpeed:  12,
		CurHP:        100,
		CurPos:       carrierRelativePos2D(ship, length*0.75, 0),
		CurSpeed:     landingArcEntryTargetSpeed(arc, ship, 0),
		RemainRange:  1000,
	}
	if landingApproachEntryReady(plane, ship, gate) {
		t.Fatal("plane beside or ahead of carrier was accepted into landing arc")
	}
}

// TestLandingStagingUpdateDoesNotAllocate 验证待场更新在热路径上零分配。
// 场景：每帧每架返航飞机都会调用 UpdateLandingStaging；内部若创建切片/字符串，长时间空战会持续触发 GC。
// 预期：1000 次 Update 的平均分配次数为 0。
// 覆盖：用 testing.AllocsPerRun 包裹待场更新，断言 allocations 为 0。
func TestLandingStagingUpdateDoesNotAllocate(t *testing.T) {
	useDefaultSettings(t)
	mapCfg := &mapcfg.MapCfg{Width: 100, Height: 100}
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		CurSpeed:    0.04,
	}
	plane := &Plane{
		MaxSpeed:     0.1,
		Acceleration: 0.005,
		RotateSpeed:  10,
		CurHP:        100,
		CurPos:       objPos.NewR(48, 48),
		CurRotation:  180,
		CurSpeed:     0.1,
		RemainRange:  1000,
	}
	plane.StartLandingStaging(mapCfg, ship, 0)

	if allocations := testing.AllocsPerRun(1000, func() {
		plane.UpdateLandingStaging(mapCfg, ship)
	}); allocations != 0 {
		t.Fatalf("landing staging allocations per update = %v, want 0", allocations)
	}
}

// TestLandingArcEndsTangentToDeck 验证进近圆弧终点与甲板中线相切，而不是斜着撞上甲板。
// 场景：飞机从左侧进入最终进近，圆弧末段必须对准舰艏方向；否则着舰前会突然修正航向。
// 预期：终点横向偏移为 0、切线横向分量为 0、切线朝舰艏为正。
// 覆盖：启动进近后取 arc 参数 1.0 的点和切线，与 landingFinalStartOffset 比较。
func TestLandingArcEndsTangentToDeck(t *testing.T) {
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	plane := &Plane{
		MaxSpeed:    0.12,
		CurPos:      carrierRelativePos2D(ship, -3.5, 1.1),
		CurRotation: 330,
		CurSpeed:    0.08,
	}
	plane.StartLandingApproach(ship)

	end := landingArcPoint(plane.landingArc, 1)
	tangent := landingArcTangent(plane.landingArc, 1)
	requireClose(t, end.forward, landingFinalStartOffset(ship, ship.Aircraft.landingConfig()).forward)
	requireClose(t, end.lateral, 0)
	requireClose(t, tangent.lateral, 0)
	if tangent.forward <= 0 {
		t.Fatalf("landing arc end tangent points away from deck: %v", tangent.forward)
	}
}

// TestLandingArcsAreMirroredAndMonotonic 验证左右进近圆弧镜像对称，且前进/收敛/转弯方向单调。
// 场景：飞机从航母左右两舷进入，玩家期望两侧路径只是镜像，不应一侧绕远或反向转弯。
// 预期：同一进度下 forward 相等、lateral 互为相反数；forward 不后退、|lateral| 不增大、半径恒定、
// 转弯方向不反转。
// 覆盖：左右各建一条 arc，按 0..1 采样 121 个点逐项断言。
func TestLandingArcsAreMirroredAndMonotonic(t *testing.T) {
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	left := &Plane{
		MaxSpeed:    0.12,
		CurPos:      carrierRelativePos2D(ship, -5.5, -1.1),
		CurRotation: 30,
		CurSpeed:    0.08,
	}
	right := &Plane{
		MaxSpeed:    0.12,
		CurPos:      carrierRelativePos2D(ship, -5.5, 1.1),
		CurRotation: 330,
		CurSpeed:    0.08,
	}
	left.StartLandingApproach(ship)
	right.StartLandingApproach(ship)

	previousForward := right.landingArc.start.forward
	previousLateral := math.Abs(right.landingArc.start.lateral)
	previousHeading := math.Atan2(
		landingArcTangent(right.landingArc, 0).lateral,
		landingArcTangent(right.landingArc, 0).forward,
	)
	for frame := 0; frame <= 120; frame++ {
		progress := float64(frame) / 120
		leftPos := landingArcPoint(left.landingArc, progress)
		rightPos := landingArcPoint(right.landingArc, progress)
		if math.Abs(leftPos.forward-rightPos.forward) > 1e-9 ||
			math.Abs(leftPos.lateral+rightPos.lateral) > 1e-9 {
			t.Fatalf(
				"landing arcs are not mirrored at %.3f: left=%+v right=%+v",
				progress,
				leftPos,
				rightPos,
			)
		}
		if rightPos.forward+1e-9 < previousForward {
			t.Fatalf("landing arc moved backward: %v -> %v", previousForward, rightPos.forward)
		}
		if math.Abs(rightPos.lateral) > previousLateral+1e-9 || rightPos.lateral < -1e-9 {
			t.Fatalf("landing arc lateral error did not converge: %v -> %v", previousLateral, rightPos.lateral)
		}
		radius := math.Hypot(
			rightPos.forward-right.landingArc.center.forward,
			rightPos.lateral-right.landingArc.center.lateral,
		)
		if math.Abs(radius-right.landingArc.radius) > 1e-9 {
			t.Fatalf("landing arc radius changed: got %v want %v", radius, right.landingArc.radius)
		}

		tangent := landingArcTangent(right.landingArc, progress)
		heading := math.Atan2(tangent.lateral, tangent.forward)
		if frame > 0 && normalizeRadians(heading-previousHeading) < -1e-9 {
			t.Fatalf("landing arc reversed turn direction at frame %d", frame)
		}
		previousForward = rightPos.forward
		previousLateral = math.Abs(rightPos.lateral)
		previousHeading = heading
	}
}

// TestLandingArcGeometryAcrossValidEntries 验证合法入口范围内都能生成参数合理的圆弧。
// 场景：飞机在舰尾后方不同前后/左右偏移处进入进近，配置不能只对某一个入口点有效。
// 预期：arc 构建成功，frames 在 [min,max] 内，sweepAngle 与 lateral 同号且 |sweep| < π。
// 覆盖：3 个 forward × 4 个 lateral 的入口矩阵，逐个检查 arc 参数。
func TestLandingArcGeometryAcrossValidEntries(t *testing.T) {
	ship := &BattleShip{Length: 256, CurPos: objPos.NewR(50, 50)}
	for _, forward := range []float64{-5.8, -5.5, -5.2} {
		for _, lateral := range []float64{-1.5, -0.7, 0.7, 1.5} {
			arc, ok := buildLandingApproachArc(
				carrierLocalOffset{forward: forward, lateral: lateral},
				ship,
				0.12,
				ship.Aircraft.landingConfig(),
			)
			if !ok {
				t.Fatalf("valid entry did not produce arc: forward=%v lateral=%v", forward, lateral)
			}
			if arc.frames < landingApproachMinFrames || arc.frames > landingApproachMaxFrames {
				t.Fatalf("landing arc frames = %v", arc.frames)
			}
			if arc.sweepAngle*lateral <= 0 || math.Abs(arc.sweepAngle) >= math.Pi {
				t.Fatalf("invalid one-way sweep: %+v", arc)
			}
		}
	}
}

// TestLandingApproachStartsAtCurrentSpeed 验证进入圆弧首帧不会把速度瞬移到目标值。
// 场景：不同最大速度的飞机以入口速度开始进近；首帧若直接赋值，速度曲线会出现断点。
// 预期：首帧速度相对入口目标速度偏差不超过 10%。
// 覆盖：三种 MaxSpeed 各建一条 arc，StartLandingApproach 后 Update 一帧并比较速度。
func TestLandingApproachStartsAtCurrentSpeed(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	startLocal := carrierLocalOffset{forward: -3.5, lateral: 1.1}
	start := carrierRelativePos2D(ship, startLocal.forward, startLocal.lateral)
	for _, maxSpeed := range []float64{0.06, 0.12, 0.24} {
		arc, ok := buildLandingApproachArc(startLocal, ship, maxSpeed, ship.Aircraft.landingConfig())
		if !ok {
			t.Fatal("failed to build test landing arc")
		}
		tangent := landingArcTangent(arc, 0)
		entrySpeed := landingArcEntryTargetSpeed(arc, ship, 0)
		plane := &Plane{
			MaxSpeed:     maxSpeed,
			Acceleration: 0.005,
			RotateSpeed:  12,
			CurHP:        100,
			CurPos:       start,
			CurRotation: normalizeAngle(
				ship.CurRotation + math.Atan2(tangent.lateral, tangent.forward)*180/math.Pi,
			),
			CurSpeed:    entrySpeed,
			RemainRange: 100,
		}
		plane.StartLandingApproach(ship)
		plane.UpdateLandingApproach(ship)
		if speedDelta := math.Abs(plane.CurSpeed-entrySpeed) / entrySpeed; speedDelta > 0.1 {
			t.Fatalf(
				"landing approach speed jumped at %.3f: first frame %.3f",
				entrySpeed,
				plane.CurSpeed,
			)
		}
	}
}

// TestLandingApproachDeckTransitionIsContinuous 验证从圆弧切换到甲板滑跑时位置/速度/航向连续。
// 场景：飞机完成进近、进入甲板减速段；若阶段切换重置位置或速度，画面上会瞬移/急停。
// 预期：StartLandingDeck 当帧位置不变；下一帧速度相对圆弧末速偏差不超过 10%，航向偏差不超过 1°。
// 覆盖：跑完 800 帧进近，记录末速/航向/位置，切换甲板后逐项断言。
func TestLandingApproachDeckTransitionIsContinuous(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	plane := &Plane{
		MaxSpeed:     0.12,
		Acceleration: 0.005,
		RotateSpeed:  12,
		CurHP:        100,
		CurPos:       carrierRelativePos2D(ship, -3.5, 1.1),
		CurRotation:  330,
		CurSpeed:     0.08,
		RemainRange:  100,
	}
	plane.StartLandingApproach(ship)
	if plane.landingArcTotalLength <= 0 || plane.landingArcExitSpeed <= 0 {
		t.Fatalf("landing approach plan invalid: total=%v exit=%v",
			plane.landingArcTotalLength, plane.landingArcExitSpeed)
	}
	for range 800 {
		if plane.UpdateLandingApproach(ship) {
			break
		}
	}
	approachSpeed := plane.landingArcSpeed
	approachRotation := plane.CurRotation
	deckStart := plane.CurPos.Copy()

	plane.StartLandingDeck(ship)
	if plane.CurPos.Distance(deckStart) > 1e-9 {
		t.Fatalf("starting deck phase moved plane before the next frame")
	}
	plane.UpdateLandingDeck(ship)
	// 圆弧末端存在被剩余距离钳制的显示速度尾步，这里校验运动学计划速度的连续性
	if speedDelta := math.Abs(plane.landingRunSpeed-approachSpeed) / max(approachSpeed, 0.001); speedDelta > 0.1 {
		t.Fatalf(
			"landing speed jumped at deck transition: approach=%v deck=%v",
			approachSpeed,
			plane.landingRunSpeed,
		)
	}
	if rotationDelta := angleDifference(plane.CurRotation, approachRotation); rotationDelta > 1 {
		t.Fatalf(
			"landing heading jumped at deck transition: approach=%v deck=%v",
			approachRotation,
			plane.CurRotation,
		)
	}
}

// TestLandingDeckPlanMatchesArcExitSpeedAcrossAircraftSpeeds 验证甲板滑跑初速与圆弧出口速度一致。
// 场景：不同最大速度的飞机完成进近后进入甲板；若两段各自算速度，接缝处会跳变。
// 预期：landingArcExitSpeed 在 (0, relativeSpeed] 内；
// StartLandingDeck 后 landingRunSpeed 等于 landingArcSpeed，
// 滑跑总长等于舰长 × ApproachLength。
// 覆盖：三种 MaxSpeed 各构建 arc/plan，切换甲板后逐字段断言。
func TestLandingDeckPlanMatchesArcExitSpeedAcrossAircraftSpeeds(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{Length: 128, CurPos: objPos.NewR(50, 50)}
	length := carrierLengthInMapBlocks(ship)
	start := carrierLocalOffset{forward: -3.5, lateral: 1.1}
	for _, maxSpeed := range []float64{0.06, 0.12, 0.24} {
		arc, ok := buildLandingApproachArc(start, ship, maxSpeed, ship.Aircraft.landingConfig())
		if !ok {
			t.Fatal("failed to build test landing arc")
		}
		plane := &Plane{MaxSpeed: maxSpeed, CurHP: 100}
		plane.landingArc = arc
		plane.startLandingArcPlan(ship, arc.relativeSpeed)
		if plane.landingArcExitSpeed <= 0 ||
			plane.landingArcExitSpeed > arc.relativeSpeed+1e-9 {
			t.Fatalf(
				"arc exit speed = %v, want within (0, %.4f]",
				plane.landingArcExitSpeed,
				arc.relativeSpeed,
			)
		}
		plane.StartLandingDeck(ship)
		if plane.landingRunSpeed != plane.landingArcSpeed {
			t.Fatalf(
				"deck entry speed %v != arc speed %v for max speed %.3f",
				plane.landingRunSpeed,
				plane.landingArcSpeed,
				maxSpeed,
			)
		}
		expectedTotal := length * ship.Aircraft.landingConfig().ApproachLength
		if plane.landingRunTotalLength != expectedTotal {
			t.Fatalf("deck run length = %v, want %v", plane.landingRunTotalLength, expectedTotal)
		}
	}
}

// TestLandingDeckUsesLongMonotonicDecelerationAndScale 验证甲板段速度/视觉缩放单调递减并最终停在甲板中心。
// 场景：飞机从进近末端滑跑到停机位，需要持续减速且模型逐渐缩小，不能中途加速或后退。
// 预期：每帧速度不增、VisualScaleMultiplier 不增、forward 不后退；结束时 local=(0,0)、缩放为低空倍率。
// 覆盖：跑完进近后进入甲板，循环 UpdateLandingDeck 逐帧断言，直到 finished。
func TestLandingDeckUsesLongMonotonicDecelerationAndScale(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	plane := &Plane{
		MaxSpeed:     0.12,
		Acceleration: 0.005,
		RotateSpeed:  12,
		CurHP:        100,
		CurPos:       carrierRelativePos2D(ship, -3.5, 1.1),
		CurRotation:  330,
		CurSpeed:     0.08,
		RemainRange:  100,
	}
	plane.StartLandingApproach(ship)
	for range 800 {
		if plane.UpdateLandingApproach(ship) {
			break
		}
	}
	local := planeCarrierLocalOffset(plane, ship)
	expectedEnd := landingFinalStartOffset(ship, ship.Aircraft.landingConfig())
	requireClose(t, local.forward, carrierLengthInMapBlocks(ship)*expectedEnd.forward)
	requireClose(t, local.lateral, 0)

	plane.StartLandingDeck(ship)
	previousSpeed := math.Inf(1)
	previousScale := plane.VisualScaleMultiplier()
	previousForward := local.forward
	finished := false
	for range 800 {
		finished = plane.UpdateLandingDeck(ship)
		local = planeCarrierLocalOffset(plane, ship)
		if plane.CurSpeed > previousSpeed+1e-9 {
			t.Fatalf("landing speed increased: %v -> %v", previousSpeed, plane.CurSpeed)
		}
		scale := plane.VisualScaleMultiplier()
		if scale > previousScale+1e-9 {
			t.Fatalf("landing scale increased: %v -> %v", previousScale, scale)
		}
		if local.forward+1e-9 < previousForward {
			t.Fatalf("landing moved backward: %v -> %v", previousForward, local.forward)
		}
		previousSpeed = plane.CurSpeed
		previousScale = scale
		previousForward = local.forward
		if finished {
			break
		}
	}
	if !finished {
		t.Fatal("landing deck did not finish within dynamic duration")
	}
	requireClose(t, local.forward, 0)
	requireClose(t, local.lateral, 0)
	requireClose(t, plane.VisualScaleMultiplier(), planeLowAltitudeVisualScale)
}

// TestLandingDeckBrakingSegmentStaysShort 验证甲板刹车段不会因出口速度倍率被调低而磨蹭。
// 场景：航母 Length=128、飞机 MaxSpeed=0.12，圆弧出口速度取 MaxSpeed×landingApproachTargetSpeedRatio(0.60)；
// 最终进近刹车段约 3 个基准长，时长按 2L/v 与出口速度成反比。
// 预期：圆弧进近在 800 帧内结束，随后甲板滑跑在 120 帧内完成（0.60 倍速理论约 83 帧；
// 若倍率被调回 0.30 会涨到约 167 帧）。
// 覆盖：从舰尾相对位置 (-3.5, 1.1)、航向 330、速度 0.08 启动进近，跑完进近后
// 循环 UpdateLandingDeck，断言完成帧数 ≤ 120。
func TestLandingDeckBrakingSegmentStaysShort(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	plane := &Plane{
		MaxSpeed:     0.12,
		Acceleration: 0.005,
		RotateSpeed:  12,
		CurHP:        100,
		CurPos:       carrierRelativePos2D(ship, -3.5, 1.1),
		CurRotation:  330,
		CurSpeed:     0.08,
		RemainRange:  100,
	}
	plane.StartLandingApproach(ship)
	for range 800 {
		if plane.UpdateLandingApproach(ship) {
			break
		}
	}
	plane.StartLandingDeck(ship)
	frames := 0
	for ; frames < 800; frames++ {
		if plane.UpdateLandingDeck(ship) {
			break
		}
	}
	if frames >= 800 {
		t.Fatal("landing deck did not finish")
	}
	// 0.60 倍速下理论时长约 83 帧；0.30 倍速会涨到约 167 帧。
	if frames > 120 {
		t.Fatalf("最终进近刹车段用了 %d 帧，落地速度倍率可能被调回低速", frames)
	}
}

// TestLandingAnimationTracksMovingTurningCarrier 验证航母移动/转向时飞机仍能按相对坐标完成进近和甲板滑跑。
// 场景：航母以 0.03 速度前进并每帧 0.25° 转向，飞机进近期间甲板一直在动；着舰逻辑必须跟踪相对位置。
// 预期：FlightPhaseProgress 不倒退、单帧转角不超过 RotateSpeed；进近/甲板都能在有限帧内完成并落到相对 (0,
// 0)。
// 覆盖：每帧先 moveTestCarrier 再 Update，记录进度/航向，最后检查航母局部坐标。
func TestLandingAnimationTracksMovingTurningCarrier(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		CurSpeed:    0.03,
	}
	plane := &Plane{
		MaxSpeed:     0.12,
		Acceleration: 0.005,
		RotateSpeed:  5,
		CurHP:        100,
		CurPos:       carrierRelativePos2D(ship, -3.5, 1.1),
		CurRotation:  330,
		CurSpeed:     0.08,
		RemainRange:  100,
	}
	plane.StartLandingApproach(ship)
	previousProgress := plane.FlightPhaseProgress()
	previousRotation := plane.CurRotation
	approachFinished := false
	for range 800 {
		moveTestCarrier(ship, 0.25)
		approachFinished = plane.UpdateLandingApproach(ship)
		if progress := plane.FlightPhaseProgress(); progress < previousProgress {
			t.Fatalf("landing progress decreased: %v -> %v", previousProgress, progress)
		} else {
			previousProgress = progress
		}
		if step := angleDifference(plane.CurRotation, previousRotation); step > plane.RotateSpeed+1e-9 {
			t.Fatalf("turn step = %v, exceeds rotate speed %v", step, plane.RotateSpeed)
		}
		previousRotation = plane.CurRotation
		if approachFinished {
			break
		}
	}
	if !approachFinished {
		t.Fatalf("landing approach did not finish in fixed duration")
	}
	local := planeCarrierLocalOffset(plane, ship)
	expectedEnd := landingFinalStartOffset(ship, ship.Aircraft.landingConfig())
	requireClose(t, local.forward, carrierLengthInMapBlocks(ship)*expectedEnd.forward)
	requireClose(t, local.lateral, 0)

	plane.StartLandingDeck(ship)
	deckFinished := false
	for range 800 {
		moveTestCarrier(ship, 0.25)
		deckFinished = plane.UpdateLandingDeck(ship)
		if deckFinished {
			break
		}
	}
	if !deckFinished {
		t.Fatalf("landing deck animation did not finish in fixed duration")
	}
	local = planeCarrierLocalOffset(plane, ship)
	requireClose(t, local.forward, 0)
	requireClose(t, local.lateral, 0)
}

// TestLandingStagingCanCatchCarrierAtPlaneMaxSpeed 验证飞机最大速度等于航母速度时仍能通过待场加速追上。
// 场景：飞机和航母同速，飞机在舰尾后方 3 格；若待场目标速度不超过航母，飞机永远追不上。
// 预期：20 帧后 plane.CurSpeed 大于 ship.CurSpeed。
// 覆盖：同速 0.12 起步，循环 20 帧 UpdateLandingStaging 并比较速度。
func TestLandingStagingCanCatchCarrierAtPlaneMaxSpeed(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		CurSpeed:    0.12,
	}
	plane := &Plane{
		MaxSpeed:     0.12,
		CurSpeed:     0.12,
		Acceleration: 0.01,
		RotateSpeed:  12,
		CurHP:        100,
		CurPos:       carrierRelativePos2D(ship, -3, 0),
		RemainRange:  100,
	}
	plane.StartLandingStaging(nil, ship, 0)
	for range 20 {
		moveTestCarrier(ship, 0)
		plane.UpdateLandingStaging(nil, ship)
	}
	if plane.CurSpeed <= ship.CurSpeed {
		t.Fatalf("landing catch-up speed = %v, carrier speed = %v", plane.CurSpeed, ship.CurSpeed)
	}
}

// TestLandingHeadingDoesNotSnapWhenCarrierStartsTurning 验证航母突然满舵时飞机航向被低通滤波平滑，不甩头。
// 场景：航母从直航突然以 0.8°/帧转向，目标航向中的 ω×r 项阶跃；直接赋值会让飞机单帧转数十度。
// 预期：转向开始后 60 帧内单帧转角不超过 8°；随后仍能在 800 帧内完成进近并落到甲板中线。
// 覆盖：先直航 30 帧，再满舵 60 帧记录最大单帧转角，最后继续推进直到 finished。
func TestLandingHeadingDoesNotSnapWhenCarrierStartsTurning(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		CurSpeed:    0.03,
	}
	plane := &Plane{
		MaxSpeed:     0.06,
		Acceleration: 0.005,
		RotateSpeed:  12,
		CurHP:        100,
		CurPos:       carrierRelativePos2D(ship, -3.5, 1.1),
		CurRotation:  330,
		CurSpeed:     0.05,
		RemainRange:  100,
	}
	plane.StartLandingApproach(ship)
	for range 30 {
		plane.UpdateLandingApproach(ship)
	}

	// 航母从直航突然满舵：机头目标航向中的 omega×r 项是阶跃信号，
	// 低通滤波必须把机头偏转摊到数十帧，单帧转角不允许出现数十度的甩头。
	maxStep := 0.0
	for range 60 {
		moveTestCarrier(ship, 0.8)
		before := plane.CurRotation
		plane.UpdateLandingApproach(ship)
		maxStep = max(maxStep, angleDifference(plane.CurRotation, before))
	}
	if maxStep > 8 {
		t.Fatalf(
			"landing heading snapped while carrier started turning: max step = %.2f degrees/frame",
			maxStep,
		)
	}

	// 滤波不能拖慢阶段推进：转向中仍应在有限时间内完成进近
	finished := false
	for range 800 {
		moveTestCarrier(ship, 0.8)
		if plane.UpdateLandingApproach(ship) {
			finished = true
			break
		}
	}
	if !finished {
		t.Fatal("landing approach did not finish while carrier was turning")
	}
	local := planeCarrierLocalOffset(plane, ship)
	expectedEnd := landingFinalStartOffset(ship, ship.Aircraft.landingConfig())
	requireClose(t, local.forward, carrierLengthInMapBlocks(ship)*expectedEnd.forward)
	requireClose(t, local.lateral, 0)
}

func moveTestCarrier(ship *BattleShip, rotationStep float64) {
	ship.CurRotation = normalizeAngle(ship.CurRotation + rotationStep)
	radians := ship.CurRotation * math.Pi / 180
	ship.CurPos.AssignRxy(
		ship.CurPos.RX+math.Sin(radians)*ship.CurSpeed,
		ship.CurPos.RY-math.Cos(radians)*ship.CurSpeed,
	)
}

// TestSeaLandingSplashdownIsBehindStern 验证水上飞机着舰溅落点在舰尾之后，而不是压在甲板上。
// 场景：水上飞机/水上飞机母舰采用 LandingModeSea；最终溅落点若按甲板回收点计算，会显示在舰体内部。
// 预期：splashForward 等于 -0.5 减 seaLandingSplashdownAftOfSternRatio；进度到该比例时视觉缩放为低空倍率。
// 覆盖：构造 Sea 模式甲板，计算 splashForward 与 ratio，再设置进度断言 VisualScaleMultiplier。
func TestSeaLandingSplashdownIsBehindStern(t *testing.T) {
	ship := &BattleShip{Length: 128, Width: 24, CurPos: objPos.NewR(50, 50)}
	sa := &ShipAircraft{}
	sa.ResolveDeck(&CarrierDeck{
		Name: "seaplane-tender",
		Landing: LandingConfig{
			Mode:           LandingModeSea,
			Forward:        0.5,
			Lateral:        1.1,
			ApproachLength: 3,
		},
	})
	ship.Aircraft = *sa
	landing := ship.Aircraft.landingConfigForSlot(0)

	ratio := landingScaleCompleteRatio(ship, landing)
	start := landingFinalStartOffset(ship, landing)
	splashForward := start.forward + ratio*(0.5-landing.Forward-start.forward)
	sternForward := -0.5
	requireClose(t, splashForward, sternForward-seaLandingSplashdownAftOfSternRatio)

	plane := &Plane{CurPos: objPos.NewR(50, 52), FlightPhase: PlaneFlightPhaseCruising}
	plane.StartLandingDeck(ship)
	requireClose(t, plane.landingScaleCompleteRatio, ratio)
	plane.FlightPhaseProgressValue = ratio
	requireClose(t, plane.VisualScaleMultiplier(), planeLowAltitudeVisualScale)
}

// TestLandingVisualScaleInterpolation 验证着舰过程中视觉缩放的分段线性插值。
// 场景：飞机从进近到甲板滑跑，模型需要从 1x 逐渐缩到低空倍率；进近段保持原大小，甲板段按路程比例缩小。
// 预期：进近阶段始终 1x；甲板段 40% 路程为 0.75x，达到 landingDeckScaleDistanceRatio 后保持低空倍率。
// 覆盖：手动设置 FlightPhaseProgressValue 的 0/0.4/阈值/0.9/1.0，逐点断言 VisualScaleMultiplier。
func TestLandingVisualScaleInterpolation(t *testing.T) {
	ship := &BattleShip{Length: 128, CurPos: objPos.NewR(50, 50)}
	plane := &Plane{CurPos: objPos.NewR(50, 52), FlightPhase: PlaneFlightPhaseCruising}

	plane.StartLandingApproach(ship)
	requireClose(t, plane.VisualScaleMultiplier(), 1)
	plane.FlightPhaseProgressValue = 0.5
	requireClose(t, plane.VisualScaleMultiplier(), 1)
	plane.FlightPhaseProgressValue = 1
	requireClose(t, plane.VisualScaleMultiplier(), 1)

	plane.StartLandingDeck(ship)
	requireClose(t, plane.VisualScaleMultiplier(), 1)

	// 着舰段进度为线性路程比例：80% 路程处降到最低视觉倍率
	plane.FlightPhaseProgressValue = 0.4
	requireClose(t, plane.VisualScaleMultiplier(), 0.75)

	plane.FlightPhaseProgressValue = landingDeckScaleDistanceRatio
	requireClose(t, plane.VisualScaleMultiplier(), planeLowAltitudeVisualScale)
	plane.FlightPhaseProgressValue = 0.9
	requireClose(t, plane.VisualScaleMultiplier(), planeLowAltitudeVisualScale)
	plane.FlightPhaseProgressValue = 1
	requireClose(t, plane.VisualScaleMultiplier(), planeLowAltitudeVisualScale)
}

// TestLandingGateToleranceScaleGrowsWithTurnRadius 验证入口容差倍数随机型转弯半径单调放大，并在上限处封顶。
// 场景：战斗机 RotateSpeed=6°/帧、MaxSpeed=0.05，重机只有 1.5°/帧、MaxSpeed=0.043，
// 转弯半径 r=v/ω 分别约 0.48 / 1.64 格（0.12 / 0.41 个基准长度）；若用舰载机级别的固定容差，重机会在待场与入口之间无限重引导。
// 预期：战斗机 scale ≥ 1 且重机 scale 严格更大；RotateSpeed=0（直线飞行）退回 1；
// 极低转向能力（RotateSpeed=1、MaxSpeed=0.1）封顶在 landingGateToleranceMaxScale=8。
// 覆盖：四种 Plane 调用 landingGateToleranceScale(p, 4.0)，比较倍数并断言下限/单调/退化/封顶。
func TestLandingGateToleranceScaleGrowsWithTurnRadius(t *testing.T) {
	useDefaultSettings(t)
	const length = 4.0
	fighter := &Plane{MaxSpeed: 0.05, RotateSpeed: 6, CurSpeed: 0.05}
	heavy := &Plane{MaxSpeed: 0.043, RotateSpeed: 1.5, CurSpeed: 0.043}
	straightFlying := &Plane{MaxSpeed: 0.1, RotateSpeed: 0, CurSpeed: 0.1}
	extreme := &Plane{MaxSpeed: 0.1, RotateSpeed: 1, CurSpeed: 0.1}

	fighterScale := landingGateToleranceScale(fighter, length)
	heavyScale := landingGateToleranceScale(heavy, length)
	if fighterScale < 1 || heavyScale <= fighterScale {
		t.Fatalf("容差倍数应随转弯半径单调放大: fighter=%.2f heavy=%.2f", fighterScale, heavyScale)
	}
	if got := landingGateToleranceScale(straightFlying, length); got != 1 {
		t.Fatalf("转向速度为 0 时应退回基准容差, got %.2f", got)
	}
	if got := landingGateToleranceScale(extreme, length); got != landingGateToleranceMaxScale {
		t.Fatalf("容差倍数应封顶在 %.1f, got %.2f", landingGateToleranceMaxScale, got)
	}
}
