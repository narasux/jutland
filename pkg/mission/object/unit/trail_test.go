package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

func TestSwordfishTrailsFollowTailAnimationWithoutPerTickStacking(t *testing.T) {
	ship := &BattleShip{
		TypeAbbr: "Swordfish",
		Length:   180,
		Width:    76,
		MaxSpeed: 1,
		CurSpeed: 1,
		CurPos:   objPos.NewR(10, 10),
		Animation: ShipAnimation{
			TopFrames:  []string{"01", "02", "03", "04", "05", "04", "03", "02"},
			FrameTicks: 6,
		},
		LastTrailAnimationStep: -1,
	}

	left := ship.GenTrails()
	if len(left) != 2 {
		t.Fatalf("first animation step trails=%d, want 2", len(left))
	}
	if left[0].Shape != textureImg.TrailShapeCircle || left[1].Shape != textureImg.TrailShapeCircle {
		t.Fatalf("unexpected swordfish trail shapes: %v, %v", left[0].Shape, left[1].Shape)
	}
	if left[0].CurSize >= 10 || left[1].CurSize >= 10 {
		t.Fatalf("water disturbance is too large: %v/%v", left[0].CurSize, left[1].CurSize)
	}
	if left[0].Pos.RX >= ship.CurPos.RX || left[0].Pos.RY <= ship.CurPos.RY {
		t.Fatalf("left-tail wake position=%s, want behind and left of ship", left[0].Pos.String())
	}
	if trails := ship.GenTrails(); trails != nil {
		t.Fatalf("same animation step generated %d duplicate trails", len(trails))
	}

	ship.AnimationAge = 24
	right := ship.GenTrails()
	if len(right) != 2 {
		t.Fatalf("right animation step trails=%d, want 2", len(right))
	}
	if right[0].Pos.RX <= ship.CurPos.RX || right[0].Pos.RY <= ship.CurPos.RY {
		t.Fatalf("right-tail wake position=%s, want behind and right of ship", right[0].Pos.String())
	}
	if left[0].DiffusionRate <= 0 || right[0].DiffusionRate <= 0 {
		t.Fatalf("water disturbance must expand before fading")
	}
}

func TestSurfaceShipWakeMatchesOriginalWhitePair(t *testing.T) {
	ship := &BattleShip{
		Length:   128,
		Width:    20,
		MaxSpeed: 0.1,
		CurSpeed: 0.05,
		CurPos:   objPos.NewR(10, 10),
	}

	trails := ship.GenTrails()
	if len(trails) != 2 {
		t.Fatalf("default hull trails=%d, want 2", len(trails))
	}
	if trails[0].Color != nil || trails[1].Color != nil {
		t.Fatalf("original foam is default white, got %v %v", trails[0].Color, trails[1].Color)
	}
	if trails[0].CurSize != ship.Width*0.6 || trails[1].CurSize != ship.Width {
		t.Fatalf("unexpected wake size: %v %v", trails[0].CurSize, trails[1].CurSize)
	}
	if trails[0].Pos.RY >= ship.CurPos.RY {
		t.Fatalf("bow wake %s should be forward of ship", trails[0].Pos.String())
	}
	if trails[1].Pos.RY <= ship.CurPos.RY {
		t.Fatalf("stern wake %s should be aft of ship", trails[1].Pos.String())
	}
	// 尾迹总长 = 设计航迹长度（life / 衰减速度 = 航迹需要的帧数）
	travelCells := ship.CurSpeed * trails[0].CurLife / trails[0].LifeReductionRate
	wantCells := ship.Length * bowWakeTrailRatio / constants.MapBlockSize
	if math.Abs(travelCells-wantCells) > 1e-9 {
		t.Fatalf("default bow trail=%v cells, want %v", travelCells, wantCells)
	}
	if trails[0].CurLife != wakeAlpha {
		t.Fatalf("default bow life=%v, want %v", trails[0].CurLife, wakeAlpha)
	}
}

// TestSurfaceShipWakeIsVisibleBehindHull 锁定回归：尾流点必须能在寿命内走出舰体投影，
// 否则尾流会整段被战舰贴图盖住，玩家在游戏中看不到任何尾流。
func TestSurfaceShipWakeIsVisibleBehindHull(t *testing.T) {
	cases := []struct {
		name          string
		length, width float64
		knots         float64
	}{
		{name: "destroyer", length: 118, width: 11, knots: 36},
		{name: "battleship", length: 175, width: 32, knots: 21},
		{name: "slow battleship", length: 114, width: 22, knots: 16},
		{name: "super battleship", length: 251, width: 36, knots: 30},
	}
	// 舰艏点在 +0.25 舰长、舰艉点在 -0.20 舰长，走出舰体分别需要 0.75 / 0.30 舰长
	clearRatios := []float64{0.75, 0.30}
	// 露出舰体后还希望保留至少半个舰长的可见尾迹
	const wantVisibleRatio = 0.5

	for _, tc := range cases {
		speed := tc.knots / ShipSpeedScale
		ship := &BattleShip{
			Length:   tc.length,
			Width:    tc.width,
			MaxSpeed: speed,
			CurSpeed: speed,
			CurPos:   objPos.NewR(20, 20),
		}
		trails := ship.GenTrails()
		if len(trails) != 2 {
			t.Fatalf("%s: trails=%d, want 2", tc.name, len(trails))
		}

		for idx, trail := range trails {
			travelCells := ship.CurSpeed * trail.CurLife / trail.LifeReductionRate
			wantCells := tc.length * (clearRatios[idx] + wantVisibleRatio) / constants.MapBlockSize
			if travelCells < wantCells {
				t.Fatalf(
					"%s: trail %d travel=%.2f cells, want >= %.2f", tc.name, idx, travelCells, wantCells,
				)
			}
			// 露出舰体那一刻透明度（= 剩余 life）必须仍然可见，否则尾流等于没画。
			clearFrames := tc.length * clearRatios[idx] / constants.MapBlockSize / ship.CurSpeed
			if visibleLife := trail.CurLife - trail.LifeReductionRate*clearFrames; visibleLife < 10 {
				t.Fatalf("%s: trail %d alpha when leaving hull=%.1f, want >= 10", tc.name, idx, visibleLife)
			}
		}
	}
}

// TestSlowShipWakeStaysNarrowAndFaint 锁定低速观感：尾流点的寿命 / 尺寸扩散都随航速下降，
// 不会因为低速时寿命变长而膨成又白又圆的泡沫团。
func TestSlowShipWakeStaysNarrowAndFaint(t *testing.T) {
	newShip := func(knots float64) *BattleShip {
		return &BattleShip{Length: 175, Width: 32, CurSpeed: knots / ShipSpeedScale, CurPos: objPos.NewR(20, 20)}
	}
	fast, slow := newShip(21), newShip(4)
	fastTrails, slowTrails := fast.GenTrails(), slow.GenTrails()
	endSize := func(size, diffusion, life, reduction float64) float64 {
		return size + diffusion*life/reduction
	}

	endRatios := []float64{bowWakeEndWidth, sternWakeEndWidth}
	for idx, name := range []string{"bow", "stern"} {
		fastTrail, slowTrail := fastTrails[idx], slowTrails[idx]
		fastCells := fast.CurSpeed * fastTrail.CurLife / fastTrail.LifeReductionRate
		slowCells := slow.CurSpeed * slowTrail.CurLife / slowTrail.LifeReductionRate
		if slowCells >= fastCells {
			t.Fatalf("%s: slow wake trail=%v cells, want shorter than cruise %v", name, slowCells, fastCells)
		}
		// 寿命兼透明度，任何航速下都不能比完全展开的尾流更白。
		if fastTrail.CurLife > wakeAlpha || slowTrail.CurLife > wakeAlpha {
			t.Fatalf("%s: wake alpha=%v/%v, want <= %v", name, fastTrail.CurLife, slowTrail.CurLife, wakeAlpha)
		}
		// 末端尺寸始终不超过设计宽度，避免低速时膨胀成大圆泡。
		fastEnd := endSize(fastTrail.CurSize, fastTrail.DiffusionRate, fastTrail.CurLife, fastTrail.LifeReductionRate)
		slowEnd := endSize(slowTrail.CurSize, slowTrail.DiffusionRate, slowTrail.CurLife, slowTrail.LifeReductionRate)
		if maxEnd := slow.Width * endRatios[idx]; fastEnd > maxEnd || slowEnd > maxEnd {
			t.Fatalf("%s: wake end size=%v/%v, want <= %v", name, fastEnd, slowEnd, maxEnd)
		}
	}
}

func TestSurfaceShipWakeLengthIsIndependentOfGlobalSpeed(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })

	var wantLength float64
	for idx, multiplier := range []float64{
		config.SpeedSlowMultiplier,
		config.SpeedStandardMultiplier,
		config.SpeedFastMultiplier,
	} {
		config.G = &config.GameSettings{SpeedMultiplier: multiplier}
		ship := &BattleShip{
			Length:   128,
			Width:    20,
			MaxSpeed: 0.1,
			CurSpeed: 0.1 * multiplier,
			CurPos:   objPos.NewR(10, 10),
		}
		trails := ship.GenTrails()
		gotLength := ship.CurSpeed * trails[0].CurLife / (trails[0].LifeReductionRate * multiplier)
		if idx == 0 {
			wantLength = gotLength
			continue
		}
		if math.Abs(gotLength-wantLength) > 1e-9 {
			t.Fatalf("wake length at multiplier %v = %v, want %v", multiplier, gotLength, wantLength)
		}
	}
}

func TestConfiguredWakeHullsEmitPerHull(t *testing.T) {
	ship := &BattleShip{
		Length:   493,
		Width:    54,
		MaxSpeed: 0.1,
		CurSpeed: 0.05,
		CurPos:   objPos.NewR(20, 20),
		WakeHulls: []WakeHull{
			{Lateral: 0, Width: 54, Front: 0.25, Back: -0.45},
			{Lateral: 93.5, Width: 28, Front: 0.08, Back: -0.46},
			{Lateral: -93.5, Width: 28, Front: 0.08, Back: -0.46},
		},
	}
	trails := ship.GenTrails()
	if len(trails) != 6 {
		t.Fatalf("trimaran trails=%d, want 6", len(trails))
	}

	var xs []float64
	for i := 0; i < len(trails); i += 2 {
		xs = append(xs, trails[i].Pos.RX)
	}
	if xs[1] <= xs[0] || xs[2] >= xs[0] {
		t.Fatalf("hull wake X=%v, want center then starboard then port", xs)
	}

	mainStern := trails[1]
	sideStern := trails[3]
	sideWakeLength := ship.Length * (0.08 - (-0.46))
	if mainStern.Pos.RY < ship.CurPos.RY+ship.Length/constants.MapBlockSize*0.40 {
		t.Fatalf("main stern wake %s is still under the flight deck", mainStern.Pos.String())
	}
	if sideStern.CurSize != 28 {
		t.Fatalf("side stern size=%v, want widened 28", sideStern.CurSize)
	}
	// 侧船体航迹比主船体短；初始透明度一致（长度差异由寿命衰减速度体现）。
	fullShipBowCells := ship.CurSpeed * trails[0].CurLife / trails[0].LifeReductionRate
	wantSideCells := sideWakeLength * sternWakeTrailRatio / constants.MapBlockSize
	sideCells := ship.CurSpeed * sideStern.CurLife / sideStern.LifeReductionRate
	if sideCells >= fullShipBowCells || math.Abs(sideCells-wantSideCells) > 1e-9 {
		t.Fatalf(
			"side wake=%v cells, want hull-span trail %v shorter than full ship %v",
			sideCells, wantSideCells, fullShipBowCells,
		)
	}
}

func TestStoppedSwordfishResetsTrailAnimationStep(t *testing.T) {
	ship := &BattleShip{
		CurSpeed: 0,
		Animation: ShipAnimation{
			TopFrames:  []string{"01"},
			FrameTicks: 6,
		},
		AnimationAge:           12,
		LastTrailAnimationStep: 2,
	}
	ship.AdvanceAnimation(1)
	if ship.AnimationAge != 0 || ship.LastTrailAnimationStep != -1 {
		t.Fatalf("stopped animation state=%v/%d, want 0/-1", ship.AnimationAge, ship.LastTrailAnimationStep)
	}
}
