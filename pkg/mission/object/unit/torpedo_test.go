package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

func TestTorpedoFanProfiles(t *testing.T) {
	testCases := []struct {
		name     string
		shipType ShipType
		step     float64
		maxSpan  float64
	}{
		{name: "destroyer", shipType: ShipTypeDestroyer, step: 4, maxSpan: 24},
		{name: "frigate", shipType: ShipTypeFrigate, step: 4, maxSpan: 24},
		{name: "torpedo boat", shipType: ShipTypeTorpedoBoat, step: 4, maxSpan: 24},
		{name: "cruiser", shipType: ShipTypeCruiser, step: 3, maxSpan: 18},
		{name: "battleship", shipType: ShipTypeBattleShip, step: 2, maxSpan: 12},
		{name: "carrier", shipType: ShipTypeAircraftCarrier, step: 2, maxSpan: 12},
		{name: "cargo", shipType: ShipTypeCargo, step: 2, maxSpan: 12},
		{name: "repair", shipType: ShipTypeRepair, step: 2, maxSpan: 12},
		{name: "hospital", shipType: ShipTypeHospital, step: 2, maxSpan: 12},
		{name: "default", shipType: ShipTypeDefault, step: 3, maxSpan: 18},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			step, maxSpan := torpedoFanProfile(testCase.shipType)
			if step != testCase.step || maxSpan != testCase.maxSpan {
				t.Fatalf(
					"profile = (%v, %v), want (%v, %v)",
					step, maxSpan, testCase.step, testCase.maxSpan,
				)
			}
		})
	}
}

// 同一座发射器逐根发射时，航迹从瞄准线往外走：0, +1, -1, +2, …。
func TestTorpedoFanFiresFromTheAimLineOutwards(t *testing.T) {
	launcher := &TorpedoLauncher{FanTrack: 0, FanStride: 1, FanTracks: 4}
	want := []float64{0, 4, -4, 8}

	for shot, wantOffset := range want {
		launcher.ShotCountBeforeReload = shot
		if got := launcher.fanAngleOffset(ShipTypeDestroyer, 0, 0); math.Abs(got-wantOffset) > 1e-9 {
			t.Fatalf("shot %d offset = %v, want %v", shot, got, wantOffset)
		}
	}
}

// 每拍齐射（每座出一根）必须有一根压在瞄准线上，且拿到的是最内侧的相邻航迹。
// 回归：旧实现按 (槽位 - (n-1)/2) 排扇形，偶数管数只有 ±半步，正中那根永远空缺。
func TestTorpedoFanFirstVolleyCoversTheAimLine(t *testing.T) {
	for mounts := 1; mounts <= 5; mounts++ {
		for tubes := 1; tubes <= 5; tubes++ {
			first := []float64{}
			for mount := 0; mount < mounts; mount++ {
				launcher := &TorpedoLauncher{
					FanTrack: mount, FanStride: mounts, FanTracks: mounts * tubes,
				}
				first = append(first, launcher.fanAngleOffset(ShipTypeDestroyer, 0, 0))
			}
			hasCenter := false
			for _, offset := range first {
				if math.Abs(offset) < 1e-9 {
					hasCenter = true
				}
			}
			if !hasCenter {
				t.Fatalf("%d 座 x %d 管的首发齐射 %v 没有瞄准线上的那一根", mounts, tubes, first)
			}
		}
	}
}

// 每一轮齐射（每座出一根）都必须有一根压在瞄准线上，且同一轮里各管航迹不重复。
// 回归：旧实现按整组航迹顺次外扩，第二轮起全落在上一轮外侧，瞄准线上只剩第一轮
// 那一根，正头对进的目标会从后面几轮的缺口里穿过去。
func TestTorpedoFanEveryVolleyCoversTheAimLine(t *testing.T) {
	for mounts := 1; mounts <= 5; mounts++ {
		for tubes := 1; tubes <= 5; tubes++ {
			if mounts == 1 {
				// 单座组只有逐根外扩，瞄准线由第一根负责。
				continue
			}
			for shot := 0; shot < tubes; shot++ {
				offsets := []float64{}
				seen := map[float64]bool{}
				for mount := 0; mount < mounts; mount++ {
					launcher := &TorpedoLauncher{
						FanTrack: mount, FanStride: mounts, FanTracks: mounts * tubes,
						ShotCountBeforeReload: shot,
					}
					offset := launcher.fanAngleOffset(ShipTypeDestroyer, 0, 0)
					if seen[offset] {
						t.Fatalf("%d 座 x %d 管第 %d 轮航迹重复: %v", mounts, tubes, shot, offsets)
					}
					seen[offset] = true
					offsets = append(offsets, offset)
				}
				hasCenter := false
				for _, offset := range offsets {
					if math.Abs(offset) < 1e-9 {
						hasCenter = true
					}
				}
				if !hasCenter {
					t.Fatalf("%d 座 x %d 管第 %d 轮齐射 %v 没有瞄准线上的那一根",
						mounts, tubes, shot, offsets)
				}
			}
		}
	}
}

// 整组航迹跨度不超过该目标类型的扇面预算，且扇面随轮次向外放宽。
func TestTorpedoFanTracksStayWithinSpan(t *testing.T) {
	const mounts, tubes = 2, 4
	offsets := []float64{}
	volleyEdge := []float64{}
	for shot := 0; shot < tubes; shot++ {
		edge := 0.0
		for mount := 0; mount < mounts; mount++ {
			launcher := &TorpedoLauncher{
				FanTrack: mount, FanStride: mounts, FanTracks: mounts * tubes,
				ShotCountBeforeReload: shot,
			}
			offset := launcher.fanAngleOffset(ShipTypeDestroyer, 0, 0)
			offsets = append(offsets, offset)
			edge = math.Max(edge, math.Abs(offset))
		}
		volleyEdge = append(volleyEdge, edge)
	}

	low, high := offsets[0], offsets[0]
	for _, offset := range offsets {
		low, high = math.Min(low, offset), math.Max(high, offset)
	}
	if high-low > torpedoFanMaxSpanFast+1e-9 {
		t.Fatalf("destroyer fan span = %v, exceeds %v", high-low, torpedoFanMaxSpanFast)
	}
	if high-low <= 0 {
		t.Fatalf("fan span = %v, want a real spread", high-low)
	}
	// 每一轮的外缘只会比上一轮更宽（偶数座数时左右交替，可能连续两轮同宽）。
	for i := 1; i < len(volleyEdge); i++ {
		if volleyEdge[i] < volleyEdge[i-1] {
			t.Fatalf("扇面随轮次收窄: %v", volleyEdge)
		}
	}
}

// 目标处的航迹间距按半个目标长度折算成角度，距离越远扇面越窄；
// 没有目标尺寸 / 距离信息时退回目标类型的基础角。
func TestTorpedoFanSpacingShrinksWithDistance(t *testing.T) {
	// 单座 3 管：第 2 根鱼雷走外侧第 1 号航迹，便于单独观察间距规则。
	launcher := &TorpedoLauncher{FanTrack: 0, FanStride: 1, FanTracks: 3, ShotCountBeforeReload: 1}

	near := launcher.fanAngleOffset(ShipTypeDestroyer, 0, 0)
	if math.Abs(near-torpedoFanBaseAngleFast) > 1e-9 {
		t.Fatalf("缺少目标信息时应退回基础角 %v, got %v", torpedoFanBaseAngleFast, near)
	}

	const distance, targetLength = 20.0, 0.8
	far := launcher.fanAngleOffset(ShipTypeDestroyer, distance, targetLength)
	want := math.Atan2(targetLength*torpedoSpreadRatio, distance) * 180 / math.Pi
	if math.Abs(far-want) > 1e-9 {
		t.Fatalf("%v 格外间距 = %v, want %v", distance, far, want)
	}
	if far >= near {
		t.Fatalf("远距离扇面应该更窄: %v >= %v", far, near)
	}
}

func TestTorpedoFireAppliesSequentialFanAndKeepsReload(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })
	// 装填 / 发射间隔按拍计算，固定 SimTick 免得依赖其他用例留下的时钟。
	oldTick := SimTick()
	SetSimTick(100)
	t.Cleanup(func() { SetSimTick(oldTick) })

	const bulletName = "test-fan-torpedo"
	oldBullet, hadBullet := objBullet.Map[bulletName]
	objBullet.Map[bulletName] = &objBullet.Bullet{
		Type: objBullet.TypeTorpedo,
	}
	t.Cleanup(func() {
		if hadBullet {
			objBullet.Map[bulletName] = oldBullet
		} else {
			delete(objBullet.Map, bulletName)
		}
	})

	shooter := &BattleShip{
		Uid:          "shooter",
		CurHP:        100,
		Length:       128,
		Width:        12.8,
		CurPos:       objPos.NewR(10, 10),
		BelongPlayer: faction.HumanAlpha,
	}
	target := &BattleShip{
		Uid:          "target",
		Type:         ShipTypeDestroyer,
		CurHP:        100,
		CurPos:       objPos.NewR(10, 4),
		BelongPlayer: faction.ComputerAlpha,
	}
	launcher := &TorpedoLauncher{
		BulletName:     bulletName,
		BulletCount:    4,
		ReloadTime:     1,
		Range:          20,
		BulletSpeed:    1,
		RightFiringArc: FiringArc{Start: 0, End: 180},
		LeftFiringArc:  FiringArc{Start: 180, End: 360},
		FanTrack:       0,
		FanStride:      1,
		FanTracks:      4,
	}

	baseAngle := shooter.CurPos.Angle(target.CurPos)
	wantOffsets := []float64{0, 4, -4, 8}
	for shot, wantOffset := range wantOffsets {
		bullets := launcher.Fire(shooter, target)
		if len(bullets) != 1 {
			t.Fatalf("shot %d produced %d bullets, want 1", shot, len(bullets))
		}
		gotAngle := shooter.CurPos.Angle(bullets[0].TargetPos)
		wantAngle := math.Mod(baseAngle+wantOffset+360, 360)
		if math.Abs(gotAngle-wantAngle) > 1e-6 {
			t.Fatalf("shot %d angle = %.6f, want %.6f", shot, gotAngle, wantAngle)
		}
	}

	if bullets := launcher.Fire(shooter, target); len(bullets) != 0 {
		t.Fatalf("launcher fired during reload: %d bullets", len(bullets))
	}
}

// 停泊的舰船（没有航速）不该被"它自己会加速"骗出提前量；起步中的舰船则必须
// 按加速过程提前，最多算到巡航速度。
func TestTorpedoLeadFollowsAcceleration(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	parked := &BattleShip{
		CurPos: objPos.NewR(10, 10), CurRotation: 0,
		MaxSpeed: 0.03, Acceleration: 0.001, CurSpeed: 0,
	}
	if got := parked.LeadPosAt(200); got.Distance(parked.CurPos) > 1e-9 {
		t.Fatalf("停泊目标被提前了 %.4f 格", got.Distance(parked.CurPos))
	}

	starting := &BattleShip{
		CurPos: objPos.NewR(10, 10), CurRotation: 0,
		MaxSpeed: 0.03, Acceleration: 0.001, CurSpeed: 0.001,
	}
	lead := starting.LeadPosAt(600)
	travelled := starting.CurPos.Distance(lead)
	rampTicks := (0.03 - 0.001) / 0.001
	want := 0.001*rampTicks + 0.5*0.001*rampTicks*rampTicks + 0.03*(600-rampTicks)
	if math.Abs(travelled-want) > 1e-6 {
		t.Fatalf("起步目标位移 %.4f, want %.4f", travelled, want)
	}
	if travelled <= 0.001*600 {
		t.Fatalf("起步目标的位移 %.4f 没有算进加速（匀速只有 %.4f）", travelled, 0.001*600)
	}

	// 巡航速度随全局倍率放大，位移上限也必须跟着放大。
	config.G.SpeedMultiplier = 2
	if got := starting.CruiseSpeed(); math.Abs(got-0.06) > 1e-9 {
		t.Fatalf("倍率 2 时巡航速度 = %v, want 0.06", got)
	}
}

// 制导弹药发射时必须记录目标身份与初始追踪点，寿命按绕路系数放宽；
// 普通弹药保持原寿命与空目标，行为完全不变。
func TestTorpedoFireRecordsHomingTargetOnlyForGuidedAmmo(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	const guidedName, plainName = "test-guided-torpedo", "test-plain-torpedo"
	oldGuided, hadGuided := objBullet.Map[guidedName]
	oldPlain, hadPlain := objBullet.Map[plainName]
	objBullet.Map[guidedName] = &objBullet.Bullet{
		Type:     objBullet.TypeTorpedo,
		Guidance: &objBullet.Guidance{Mode: objBullet.GuidanceModePursuit, TurnRate: 180},
	}
	objBullet.Map[plainName] = &objBullet.Bullet{Type: objBullet.TypeTorpedo}
	t.Cleanup(func() {
		if hadGuided {
			objBullet.Map[guidedName] = oldGuided
		} else {
			delete(objBullet.Map, guidedName)
		}
		if hadPlain {
			objBullet.Map[plainName] = oldPlain
		} else {
			delete(objBullet.Map, plainName)
		}
	})

	shooter := &BattleShip{
		Uid:          "shooter",
		CurHP:        100,
		Length:       128,
		Width:        12.8,
		CurPos:       objPos.NewR(10, 10),
		BelongPlayer: faction.HumanAlpha,
	}
	target := &BattleShip{
		Uid:          "target",
		Type:         ShipTypeDestroyer,
		CurHP:        100,
		CurPos:       objPos.NewR(10, 4),
		BelongPlayer: faction.ComputerAlpha,
	}
	newLauncher := func(name string) *TorpedoLauncher {
		return &TorpedoLauncher{
			BulletName:     name,
			BulletCount:    1,
			Range:          20,
			BulletSpeed:    1,
			RightFiringArc: FiringArc{Start: 0, End: 180},
			LeftFiringArc:  FiringArc{Start: 180, End: 360},
			FanStride:      1,
			FanTracks:      1,
		}
	}

	guided := newLauncher(guidedName).Fire(shooter, target)
	if len(guided) != 1 {
		t.Fatalf("guided launcher fired %d bullets, want 1", len(guided))
	}
	if !guided[0].IsGuided() || guided[0].TargetShipUid != target.Uid {
		t.Fatalf("guided bullet target = %q, want %q", guided[0].TargetShipUid, target.Uid)
	}
	if guided[0].GuidePos != guided[0].TargetPos {
		t.Fatalf("guide pos = %v, want launch aim point %v", guided[0].GuidePos, guided[0].TargetPos)
	}
	if wantLife := int(20/1.0*homingLifeRatio) + 5; guided[0].Life != wantLife {
		t.Fatalf("guided life = %d, want %d", guided[0].Life, wantLife)
	}

	plain := newLauncher(plainName).Fire(shooter, target)
	if len(plain) != 1 {
		t.Fatalf("plain launcher fired %d bullets, want 1", len(plain))
	}
	if plain[0].IsGuided() || plain[0].TargetShipUid != "" {
		t.Fatalf("plain bullet should stay unguided, got uid %q", plain[0].TargetShipUid)
	}
	if wantLife := int(20/1.0) + 5; plain[0].Life != wantLife {
		t.Fatalf("plain life = %d, want %d", plain[0].Life, wantLife)
	}
}
