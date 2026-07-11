package unit

import (
	"math"
	"testing"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

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
	arc, ok := buildLandingApproachArc(gate, length, 0.12)
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
	arc, ok := buildLandingApproachArc(gate, length, 0.12)
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

func TestLandingStagingSpeedUsesCarrierDistanceEnvelope(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{Length: 128, CurSpeed: 0.04}
	plane := &Plane{MaxSpeed: 0.12}
	length := carrierLengthInMapBlocks(ship)
	entrySpeed := 0.08
	nearSpeed := plane.landingStagingTargetSpeed(
		ship,
		length,
		length*landingStagingSlowdownInnerRatio,
		entrySpeed,
		0,
	)
	farSpeed := plane.landingStagingTargetSpeed(
		ship,
		length,
		length*landingStagingSlowdownOuterRatio,
		entrySpeed,
		0,
	)

	requireClose(t, nearSpeed, entrySpeed)
	requireClose(t, farSpeed, plane.MaxSpeed)
	if nearSpeed >= farSpeed {
		t.Fatalf("landing speed envelope did not decelerate: near=%v far=%v", nearSpeed, farSpeed)
	}
}

func TestLandingApproachRejectsInvalidEntryPosition(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{Length: 128, CurPos: objPos.NewR(50, 50), CurSpeed: 0.04}
	length := carrierLengthInMapBlocks(ship)
	gate := landingGateLocalOffset(0)
	arc, ok := buildLandingApproachArc(gate, length, 0.12)
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
	requireClose(t, end.forward, carrierLandingFinalStartRatio)
	requireClose(t, end.lateral, 0)
	requireClose(t, tangent.lateral, 0)
	if tangent.forward <= 0 {
		t.Fatalf("landing arc end tangent points away from deck: %v", tangent.forward)
	}
}

func TestLandingArcsAreMirroredAndMonotonic(t *testing.T) {
	ship := &BattleShip{
		Length:      128,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
	}
	left := &Plane{
		MaxSpeed:    0.12,
		CurPos:      carrierRelativePos2D(ship, -3.5, -1.1),
		CurRotation: 30,
		CurSpeed:    0.08,
	}
	right := &Plane{
		MaxSpeed:    0.12,
		CurPos:      carrierRelativePos2D(ship, -3.5, 1.1),
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

func TestLandingArcGeometryAcrossValidEntries(t *testing.T) {
	for _, forward := range []float64{-3.8, -3.5, -3.2} {
		for _, lateral := range []float64{-1.5, -0.7, 0.7, 1.5} {
			arc, ok := buildLandingApproachArc(
				carrierLocalOffset{forward: forward, lateral: lateral},
				2,
				0.12,
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
		arc, ok := buildLandingApproachArc(startLocal, carrierLengthInMapBlocks(ship), maxSpeed)
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
	if plane.landingDeckFrames < landingDeckMinFrames ||
		plane.landingDeckFrames > landingDeckMaxFrames {
		t.Fatalf("landing deck frames = %v", plane.landingDeckFrames)
	}
	for range int(math.Ceil(plane.landingArc.frames)) + 1 {
		if plane.UpdateLandingApproach(ship) {
			break
		}
	}
	approachSpeed := plane.CurSpeed
	approachRotation := plane.CurRotation
	deckStart := plane.CurPos.Copy()

	plane.StartLandingDeck(ship)
	if plane.CurPos.Distance(deckStart) > 1e-9 {
		t.Fatalf("starting deck phase moved plane before the next frame")
	}
	plane.UpdateLandingDeck(ship)
	if speedDelta := math.Abs(plane.CurSpeed-approachSpeed) / max(approachSpeed, 0.001); speedDelta > 0.1 {
		t.Fatalf(
			"landing speed jumped at deck transition: approach=%v deck=%v",
			approachSpeed,
			plane.CurSpeed,
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

func TestLandingDeckDurationMatchesArcExitSpeedAcrossAircraftSpeeds(t *testing.T) {
	useDefaultSettings(t)
	ship := &BattleShip{Length: 128, CurPos: objPos.NewR(50, 50)}
	length := carrierLengthInMapBlocks(ship)
	start := carrierLocalOffset{forward: -3.5, lateral: 1.1}
	for _, maxSpeed := range []float64{0.06, 0.12, 0.24} {
		arc, ok := buildLandingApproachArc(start, length, maxSpeed)
		if !ok {
			t.Fatal("failed to build test landing arc")
		}
		plane := &Plane{MaxSpeed: maxSpeed}
		frames := plane.landingDeckDurationFrames(ship, arc.relativeSpeed)
		deckEntrySpeed := 2 * -carrierLandingFinalStartRatio * length / frames
		if delta := math.Abs(deckEntrySpeed-arc.relativeSpeed) / arc.relativeSpeed; delta > 0.1 {
			t.Fatalf(
				"landing deck entry speed mismatch for max speed %.3f: arc=%.4f deck=%.4f frames=%.1f",
				maxSpeed,
				arc.relativeSpeed,
				deckEntrySpeed,
				frames,
			)
		}
	}
}

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
	for range int(math.Ceil(plane.landingArc.frames)) + 1 {
		if plane.UpdateLandingApproach(ship) {
			break
		}
	}
	local := planeCarrierLocalOffset(plane, ship)
	requireClose(t, local.forward, carrierLengthInMapBlocks(ship)*carrierLandingFinalStartRatio)
	requireClose(t, local.lateral, 0)

	plane.StartLandingDeck(ship)
	previousSpeed := math.Inf(1)
	previousScale := plane.VisualScaleMultiplier()
	previousForward := local.forward
	finished := false
	for range int(landingDeckMaxFrames) + 2 {
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
	for range int(math.Ceil(plane.landingArc.frames)) + 2 {
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
	requireClose(t, local.forward, carrierLengthInMapBlocks(ship)*carrierLandingFinalStartRatio)
	requireClose(t, local.lateral, 0)

	plane.StartLandingDeck(ship)
	deckFinished := false
	for range int(plane.landingDeckFrames) + 2 {
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

func moveTestCarrier(ship *BattleShip, rotationStep float64) {
	ship.CurRotation = normalizeAngle(ship.CurRotation + rotationStep)
	radians := ship.CurRotation * math.Pi / 180
	ship.CurPos.AssignRxy(
		ship.CurPos.RX+math.Sin(radians)*ship.CurSpeed,
		ship.CurPos.RY-math.Cos(radians)*ship.CurSpeed,
	)
}

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
	plane.FlightPhaseProgressValue = 0.5
	requireClose(t, plane.VisualScaleMultiplier(), 0.75)
	plane.FlightPhaseProgressValue = 1
	requireClose(t, plane.VisualScaleMultiplier(), planeLowAltitudeVisualScale)
}
