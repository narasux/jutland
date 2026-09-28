package manager

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func TestDirectShellHitsCornerBeyondHalfLength(t *testing.T) {
	const (
		speed    = 1.0
		rotation = 135.0
		length   = 256.0
		width    = 256.0
	)
	// updateShotBullets 会先 Forward 再判定。起点放在舷侧角点内侧，
	// 沿对角线飞出后，到舰心的距离大于半长加弹速，但仍小于对角线半径加弹速。
	start := objPos.NewR(10.99, 10.99)
	sinR := math.Sin(rotation * math.Pi / 180)
	cosR := math.Cos(rotation * math.Pi / 180)
	endX := start.RX + sinR*speed
	endY := start.RY - cosR*speed

	dx, dy := endX-10, endY-10
	dist2 := dx*dx + dy*dy
	halfLengthReach := 0.5*length/constants.MapBlockSize + speed
	diagReach := targetHitRadius(length, width) + speed
	if dist2 <= halfLengthReach*halfLengthReach {
		t.Fatal("fixture no longer exceeds half-length reach")
	}
	if dist2 > diagReach*diagReach {
		t.Fatal("fixture is outside the diagonal reach")
	}

	target := &objUnit.BattleShip{
		Uid:          "target",
		TotalHP:      100,
		CurHP:        100,
		Length:       length,
		Width:        width,
		CurPos:       objPos.NewR(10, 10),
		BelongPlayer: faction.ComputerAlpha,
	}
	bullet := &objBullet.Bullet{
		Type:          objBullet.TypeShell,
		Damage:        10,
		ShotType:      objBullet.ShotTypeDirect,
		TargetObjType: object.TypeShip,
		Shooter:       "shooter",
		BelongPlayer:  faction.HumanAlpha,
		CurPos:        start,
		Rotation:      rotation,
		Speed:         speed,
		Life:          10,
	}

	newShellImpactManager(target, bullet).updateShotBullets()

	if target.CurHP != 90 {
		t.Fatalf("target HP = %.1f, want 90 for a corner hit beyond half length", target.CurHP)
	}
	if bullet.HitObjType != object.TypeShip {
		t.Fatalf("hit type = %v, want ship", bullet.HitObjType)
	}
}

func TestDirectShellIgnoresShipOutsideItsBuckets(t *testing.T) {
	const (
		speed    = 1.0
		rotation = 135.0
		length   = 256.0
		width    = 256.0
	)
	start := objPos.NewR(10.99, 10.99)
	target := &objUnit.BattleShip{
		Uid:          "target",
		TotalHP:      100,
		CurHP:        100,
		Length:       length,
		Width:        width,
		CurPos:       objPos.NewR(10, 10),
		BelongPlayer: faction.ComputerAlpha,
	}
	far := &objUnit.BattleShip{
		Uid:          "far",
		TotalHP:      100,
		CurHP:        100,
		Length:       length,
		Width:        width,
		CurPos:       objPos.NewR(80, 10),
		BelongPlayer: faction.ComputerAlpha,
	}
	bullet := &objBullet.Bullet{
		Type:          objBullet.TypeShell,
		Damage:        10,
		ShotType:      objBullet.ShotTypeDirect,
		TargetObjType: object.TypeShip,
		Shooter:       "shooter",
		BelongPlayer:  faction.HumanAlpha,
		CurPos:        start,
		Rotation:      rotation,
		Speed:         speed,
		Life:          10,
	}
	manager := &MissionManager{state: &state.MissionState{
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				target.Uid: target,
				far.Uid:    far,
			},
			ForwardingBullets: []*objBullet.Bullet{bullet},
		},
	}}
	manager.updateShotBullets()

	if target.CurHP != 90 {
		t.Fatalf("near HP = %.1f, want 90", target.CurHP)
	}
	if far.CurHP != 100 {
		t.Fatalf("far HP = %.1f, want 100", far.CurHP)
	}
}

func TestDirectShellSkipsDistantShip(t *testing.T) {
	target := &objUnit.BattleShip{
		Uid:          "target",
		TotalHP:      100,
		CurHP:        100,
		Length:       256,
		Width:        256,
		CurPos:       objPos.NewR(10, 10),
		BelongPlayer: faction.ComputerAlpha,
	}
	bullet := &objBullet.Bullet{
		Type:          objBullet.TypeShell,
		Damage:        10,
		ShotType:      objBullet.ShotTypeDirect,
		TargetObjType: object.TypeShip,
		Shooter:       "shooter",
		BelongPlayer:  faction.HumanAlpha,
		CurPos:        objPos.NewR(50, 50),
		Speed:         1,
		Life:          10,
	}

	manager := newShellImpactManager(target, bullet)
	manager.updateShotBullets()

	if target.CurHP != 100 {
		t.Fatalf("target HP = %.1f, want 100 for a distant miss", target.CurHP)
	}
	if bullet.HitObjType != object.TypeNone {
		t.Fatalf("hit type = %v, want none", bullet.HitObjType)
	}
	if len(manager.state.Arena.ForwardingBullets) != 1 {
		t.Fatalf("forwarding bullets = %d, want 1", len(manager.state.Arena.ForwardingBullets))
	}
}
