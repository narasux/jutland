package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

func primeWakeEmission(ship *BattleShip) {
	ship.lastWakeEmitPos = objPos.NewR(ship.CurPos.RX-1, ship.CurPos.RY)
	ship.wakeTrailInitialized = true
}

// TestObserveWakeRestDetectsInPlaceTurnAndResetsWhileMoving 验证停船原地转向检测会跨 0/360 取最短夹角，
// 并在重新航行后清除停船状态。
// 场景：舰船在 0° 停船，先转 5° 再转 90°；随后以 0.1 格/拍重新航行，再在 350° 停船后转到 10°。
// 预期：首次采样 false；5° false；90° true；航行时调用返回 false；重新停船后先记录新航向并返回 false，
// 跨北向转到 10°（最短夹角 20°）返回 true。
// 覆盖：直接构造 BattleShip 修改 CurSpeed/CurRotation，表驱动断言 ObserveWakeRest(8) 的返回值。
func TestObserveWakeRestDetectsInPlaceTurnAndResetsWhileMoving(t *testing.T) {
	ship := &BattleShip{CurSpeed: 0, CurRotation: 0}
	if ship.ObserveWakeRest(8) {
		t.Fatal("first stationary sample should only record heading")
	}
	ship.CurRotation = 5
	if ship.ObserveWakeRest(8) {
		t.Fatal("5-degree turn should stay below the 8-degree threshold")
	}
	ship.CurRotation = 90
	if !ship.ObserveWakeRest(8) {
		t.Fatal("90-degree in-place turn was not detected")
	}

	ship.CurSpeed = 0.1
	if ship.ObserveWakeRest(8) {
		t.Fatal("moving ship should not report an in-place turn")
	}
	ship.CurSpeed = 0
	ship.CurRotation = 350
	if ship.ObserveWakeRest(8) {
		t.Fatal("first sample after re-stopping should only record heading")
	}
	ship.CurRotation = 10
	if !ship.ObserveWakeRest(8) {
		t.Fatal("20-degree turn across north was not detected")
	}
}

// TestSwordfishTrailsFollowTailAnimationWithoutPerTickStacking 验证剑鱼尾流跟随尾部动画步进，
// 同一动画帧不重复生成。
// 场景：剑鱼动画 8 帧、每 6 tick 换帧；尾流应只在动画步变化时生成左右两簇，而不是每帧堆叠。
// 预期：首帧 2 条小圆尾流、位于舰尾左后方；同一步再次调用返回 nil；AnimationAge=24 切到右后方，
// 且扩散率大于 0。
// 覆盖：直接设置 AnimationAge/LastTrailAnimationStep，连续调用 GenTrails 断言数量/形状/位置/扩散。
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

// TestSurfaceShipWakeMatchesOriginalWhitePair 验证普通舰船默认生成前后两条白色尾流，
// 尺寸/寿命/航迹长度符合设计。
// 场景：驱逐/战列等没有自定义 WakeHulls 的舰船，航行时应保留原有“一前一后白沫”的观感。
// 预期：2 条尾流、Color=nil（默认白）、OwnerUid=舰 UID；艏尾尺寸 0.6W/W；艏在舰前、艉在舰后；
// 航迹长度=设计比例 × 航速/最大速度，寿命=wakeAlpha。
// 覆盖：primeWakeEmission 后调用 GenTrails，逐项断言颜色/归属/尺寸/位置/寿命/长度。
func TestSurfaceShipWakeMatchesOriginalWhitePair(t *testing.T) {
	ship := &BattleShip{
		Length:   128,
		Width:    20,
		MaxSpeed: 0.1,
		CurSpeed: 0.05,
		CurPos:   objPos.NewR(10, 10),
	}
	primeWakeEmission(ship)

	trails := ship.GenTrails()
	if len(trails) != 2 {
		t.Fatalf("default hull trails=%d, want 2", len(trails))
	}
	if trails[0].Color != nil || trails[1].Color != nil {
		t.Fatalf("original foam is default white, got %v %v", trails[0].Color, trails[1].Color)
	}
	if trails[0].OwnerUid != ship.Uid || trails[1].OwnerUid != ship.Uid {
		t.Fatalf(
			"wake owners = %q/%q, want ship uid %q",
			trails[0].OwnerUid, trails[1].OwnerUid, ship.Uid,
		)
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
	wantCells := ship.Length * bowWakeTrailRatio / constants.MapBlockSize * ship.CurSpeed / ship.MaxSpeed
	if math.Abs(travelCells-wantCells) > 1e-9 {
		t.Fatalf("default bow trail=%v cells, want %v", travelCells, wantCells)
	}
	if trails[0].CurLife != wakeAlpha {
		t.Fatalf("default bow life=%v, want %v", trails[0].CurLife, wakeAlpha)
	}
}

// TestSurfaceShipWakeIsVisibleBehindHull 验证尾流点能在寿命内走出舰体投影，并且露出舰体后仍保持可见，
// 否则整段尾流会被战舰贴图盖住、玩家看不到。
// 场景：驱逐舰 118×11/36kn、战列舰 175×32/21kn、慢速战列舰 114×22/16kn、超战列舰 251×36/30kn
// 全速航行；舰艏尾流点在 +0.25 舰长、舰艉点在 -0.20 舰长，走出舰体分别需要 0.75 / 0.30 舰长。
// 预期：每舰生成 2 条尾流；世界航迹长度 ≥ 舰长 ×(走出比例 + 0.5 舰长可见尾迹)，
// 且走出舰体那一刻剩余 life（= 透明度）≥ 10。
// 覆盖：四型舰调用 primeWakeEmission + GenTrails，按 clearRatios{0.75, 0.30} 与 wantVisibleRatio=0.5
// 逐条断言航迹长度与透明度。
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
		primeWakeEmission(ship)
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

// TestSlowShipWakeStaysNarrowAndFaint 验证低速航行时尾流寿命与尺寸扩散随航速收紧，
// 不会因为寿命变长而膨成又白又圆的泡沫团。
// 场景：同为 175×32 的战列舰，分别以 21kn 巡航与 4kn 慢速航行；低速时寿命衰减更慢，
// 若不按航速收紧，尾流点会越漂越宽、越漂越白。
// 预期：慢速的艏/艉尾流世界航迹长度都短于巡航；两条尾流 CurLife ≤ wakeAlpha=100；
// 末端尺寸 CurSize + DiffusionRate×CurLife/LifeReductionRate 不超过 舰宽 ×(bowWakeEndWidth=2 /
// sternWakeEndWidth=1.6)。
// 覆盖：newShip 构造两种航速并 GenTrails，逐条比较航迹长度、透明度上限与末端尺寸。
func TestSlowShipWakeStaysNarrowAndFaint(t *testing.T) {
	newShip := func(knots float64) *BattleShip {
		speed := knots / ShipSpeedScale
		ship := &BattleShip{
			Length: 175, Width: 32,
			MaxSpeed: speed, CurSpeed: speed,
			CurPos: objPos.NewR(20, 20),
		}
		primeWakeEmission(ship)
		return ship
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

// TestSurfaceShipWakeLengthIsIndependentOfGlobalSpeed 验证全局速度倍率只改变尾流寿命衰减速度，
// 不改变世界空间长度。
// 场景：玩家在 0.5x/1x/2x 之间切换；若尾流长度随倍率变化，慢速档会留下过短/过长的白带。
// 预期：三档倍率下 CurSpeed*CurLife/(LifeReductionRate*multiplier) 一致。
// 覆盖：遍历三档设置 config.G，构造同型舰生成尾流，比较归一化长度。
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
		primeWakeEmission(ship)
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

// TestSurfaceShipWakeDoesNotStackAtLowSpeed 验证低速航行不会逐帧堆出大量尾流。
// 场景：舰船以 0.003 格/帧爬行，100 帧只走 0.3 格；若每帧都按半舰艏半径采样，会生成上百组白点。
// 预期：100 帧内生成尾流的帧数小于 10。
// 覆盖：每帧移动后调用 GenTrails，统计非空次数。
func TestSurfaceShipWakeDoesNotStackAtLowSpeed(t *testing.T) {
	previous := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = previous })

	ship := &BattleShip{
		Length:   128,
		Width:    20,
		MaxSpeed: 0.03,
		CurSpeed: 0.003,
		CurPos:   objPos.NewR(10, 10),
	}
	if trails := ship.GenTrails(); trails != nil {
		t.Fatalf("initial stationary position generated %d trails, want none", len(trails))
	}

	emitted := 0
	for range 100 {
		ship.CurPos.AddRy(ship.CurSpeed)
		if trails := ship.GenTrails(); len(trails) > 0 {
			emitted++
		}
	}

	// 100 帧只航行 0.3 格，按半个舰艏圆半径采样不应逐帧堆出一百组白点。
	if emitted >= 10 {
		t.Fatalf("low-speed wake emitted %d groups in 100 frames", emitted)
	}
}

// TestConfiguredWakeHullsEmitPerHull 验证三体舰按 WakeHulls 配置给每个船体生成独立尾流。
// 场景：主舰体 + 左右副体，横向偏移 ±93.5；若只按一个船体生成，副体尾流会缺失或叠在中间。
// 预期：6 条尾流（每船体前后各 1）；X 顺序为中心→右→左；主舰尾在飞行甲板之后；副体尺寸=28；
// 副体航迹短于主舰且符合船体长度比例。
// 覆盖：配置 3 个 WakeHull，生成后断言数量/顺序/位置/尺寸/长度。
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
	primeWakeEmission(ship)
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
	wantSideCells *= ship.CurSpeed / ship.MaxSpeed
	sideCells := ship.CurSpeed * sideStern.CurLife / sideStern.LifeReductionRate
	if sideCells >= fullShipBowCells || math.Abs(sideCells-wantSideCells) > 1e-9 {
		t.Fatalf(
			"side wake=%v cells, want hull-span trail %v shorter than full ship %v",
			sideCells, wantSideCells, fullShipBowCells,
		)
	}
}

// TestSurfaceShipWakeLifetimeUsesMaxSpeed 验证尾流寿命/扩散按 MaxSpeed 计算，而不是当前航速。
// 场景：同一艘船慢速与全速航行；若按当前速度算，慢速时尾流寿命会被拉长、扩散变快，看起来像泡沫团。
// 预期：慢速/全速生成的对应尾流 LifeReductionRate、DiffusionRate 相同。
// 覆盖：同型舰分别以 0.01/0.1 生成尾流，逐条比较两个衰减参数。
func TestSurfaceShipWakeLifetimeUsesMaxSpeed(t *testing.T) {
	newShip := func(speed float64) *BattleShip {
		ship := &BattleShip{
			Length:   128,
			Width:    20,
			MaxSpeed: 0.1,
			CurSpeed: speed,
			CurPos:   objPos.NewR(10, 10),
		}
		primeWakeEmission(ship)
		return ship
	}

	slowTrails := newShip(0.01).GenTrails()
	fastTrails := newShip(0.1).GenTrails()
	for idx := range slowTrails {
		if slowTrails[idx].LifeReductionRate != fastTrails[idx].LifeReductionRate {
			t.Fatalf(
				"trail %d slow fade rate=%v, fast=%v; lifetime must use max speed",
				idx, slowTrails[idx].LifeReductionRate, fastTrails[idx].LifeReductionRate,
			)
		}
		if slowTrails[idx].DiffusionRate != fastTrails[idx].DiffusionRate {
			t.Fatalf(
				"trail %d slow diffusion=%v, fast=%v; diffusion must use max speed",
				idx, slowTrails[idx].DiffusionRate, fastTrails[idx].DiffusionRate,
			)
		}
	}
}

// TestStoppedSwordfishResetsTrailAnimationStep 验证停船时动画计时与尾流步进复位。
// 场景：剑鱼从航行到停船；若 AnimationAge 继续累加，下次起步会从中途帧开始，尾流也会跳步。
// 预期：AdvanceAnimation 后 AnimationAge=0、LastTrailAnimationStep=-1。
// 覆盖：构造 CurSpeed=0、已有 AnimationAge/LastTrailAnimationStep 的舰船，调用 AdvanceAnimation(1) 断言。
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

// TestWakeTrailTuningScalesShipWake 验证设置页的尾流长度/宽度/浓度会作用到舰船尾流。
// 场景：默认值生成一组 128×20、0.1 格/帧的尾流后，把 config.G.WakeTrail 改成
// Length=2、Width=1.5、Opacity=0.5；玩家期望尾迹拖长一倍、变宽一半、变淡一半。
// 预期：新尾流的 world 航迹长度是基准的 2 倍，艏/艉初始尺寸是基准的 1.5 倍，
// AlphaScale=0.5；失败说明设置只写进配置、没有传给 GenTrails。
// 覆盖：两次 primeWakeEmission + GenTrails，比较 CurLife/LifeReductionRate/CurSize/AlphaScale。
func TestWakeTrailTuningScalesShipWake(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })

	newShip := func() *BattleShip {
		ship := &BattleShip{
			Length:   128,
			Width:    20,
			MaxSpeed: 0.1,
			CurSpeed: 0.1,
			CurPos:   objPos.NewR(10, 10),
		}
		primeWakeEmission(ship)
		return ship
	}

	config.G = config.NewDefaultGameSettings()
	baseShip := newShip()
	base := baseShip.GenTrails()
	if len(base) != 2 {
		t.Fatalf("default trails=%d, want 2", len(base))
	}

	config.G.WakeTrail = config.WakeTrailSettings{Length: 2, Width: 1.5, Opacity: 0.5}
	tunedShip := newShip()
	tuned := tunedShip.GenTrails()
	if len(tuned) != 2 {
		t.Fatalf("tuned trails=%d, want 2", len(tuned))
	}

	baseLength := baseShip.CurSpeed * base[0].CurLife / base[0].LifeReductionRate
	tunedLength := tunedShip.CurSpeed * tuned[0].CurLife / tuned[0].LifeReductionRate
	if math.Abs(tunedLength-2*baseLength) > 1e-9 {
		t.Fatalf("tuned wake length=%v cells, want %v", tunedLength, 2*baseLength)
	}
	for idx := range base {
		if math.Abs(tuned[idx].CurSize-1.5*base[idx].CurSize) > 1e-9 {
			t.Fatalf("trail %d tuned size=%v, want %v", idx, tuned[idx].CurSize, 1.5*base[idx].CurSize)
		}
		if tuned[idx].AlphaScale != 0.5 {
			t.Fatalf("trail %d alpha scale=%v, want 0.5", idx, tuned[idx].AlphaScale)
		}
	}
}
