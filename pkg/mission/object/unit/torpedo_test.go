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

func TestTorpedoFanUsesSymmetricSequentialSlots(t *testing.T) {
	launcher := &TorpedoLauncher{FanSlot: 0, FanSlotCount: 4}
	want := []float64{-6, -2, 2, 6}

	for shot, wantOffset := range want {
		launcher.ShotCountBeforeReload = shot
		if got := launcher.fanAngleOffset(ShipTypeDestroyer); math.Abs(got-wantOffset) > 1e-9 {
			t.Fatalf("shot %d offset = %v, want %v", shot, got, wantOffset)
		}
	}
}

func TestTorpedoFanSlotsStayUniqueAcrossLaunchersAndRespectMaxSpan(t *testing.T) {
	const slotCount = 8
	offsets := map[float64]bool{}
	for slot := range slotCount {
		launcher := &TorpedoLauncher{FanSlot: slot, FanSlotCount: slotCount}
		offset := launcher.fanAngleOffset(ShipTypeDestroyer)
		if offsets[offset] {
			t.Fatalf("duplicate fan offset %v at slot %d", offset, slot)
		}
		offsets[offset] = true
	}

	first := (&TorpedoLauncher{FanSlot: 0, FanSlotCount: slotCount}).fanAngleOffset(ShipTypeDestroyer)
	last := (&TorpedoLauncher{
		FanSlot: slotCount - 1, FanSlotCount: slotCount,
	}).fanAngleOffset(ShipTypeDestroyer)
	if got := last - first; got > torpedoFanMaxSpanFast+1e-9 {
		t.Fatalf("destroyer fan span = %v, exceeds %v", got, torpedoFanMaxSpanFast)
	}
}

func TestTorpedoFireAppliesSequentialFanAndKeepsReload(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

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
		FanSlot:        0,
		FanSlotCount:   4,
	}

	baseAngle := shooter.CurPos.Angle(target.CurPos)
	wantOffsets := []float64{-6, -2, 2, 6}
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
