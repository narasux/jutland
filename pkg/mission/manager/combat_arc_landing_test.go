package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func newShellImpactManager(ship *objUnit.BattleShip, bullet *objBullet.Bullet) *MissionManager {
	return &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Ships:             map[string]*objUnit.BattleShip{ship.Uid: ship},
				ForwardingBullets: []*objBullet.Bullet{bullet},
			},
		},
	}
}

func TestArcingShellUsesFinalLandingPoint(t *testing.T) {
	target := &objUnit.BattleShip{
		Uid:          "target",
		TotalHP:      100,
		CurHP:        100,
		Length:       12.8,
		Width:        12.8,
		CurPos:       objPos.NewR(10, 10),
		BelongPlayer: faction.ComputerAlpha,
	}
	bullet := &objBullet.Bullet{
		Type:           objBullet.TypeShell,
		Damage:         10,
		ShotType:       objBullet.ShotTypeArcing,
		TargetObjType:  object.TypeShip,
		Shooter:        "shooter",
		ShooterObjType: object.TypeShip,
		BelongPlayer:   faction.HumanAlpha,
		CurPos:         objPos.NewR(10, 10),
		TargetPos:      objPos.NewR(10, 10),
		Speed:          0.5,
		Life:           10,
	}

	newShellImpactManager(target, bullet).updateShotBullets()

	if target.CurHP != 90 {
		t.Fatalf("target HP = %.1f, want 90 after final-point hit", target.CurHP)
	}
	if bullet.HitObjType != object.TypeShip {
		t.Fatalf("hit type = %v, want ship", bullet.HitObjType)
	}
}

func TestArcingShellCanStraddleWithoutFinalPointHit(t *testing.T) {
	target := &objUnit.BattleShip{
		Uid:          "target",
		TotalHP:      100,
		CurHP:        100,
		Length:       51.2,
		Width:        12.8,
		CurPos:       objPos.NewR(10, 10),
		BelongPlayer: faction.ComputerAlpha,
	}
	bullet := &objBullet.Bullet{
		Type:           objBullet.TypeShell,
		Damage:         10,
		ShotType:       objBullet.ShotTypeArcing,
		TargetObjType:  object.TypeShip,
		Shooter:        "shooter",
		ShooterObjType: object.TypeShip,
		BelongPlayer:   faction.HumanAlpha,
		CurPos:         objPos.NewR(10, 9),
		TargetPos:      objPos.NewR(10, 9),
		Speed:          2,
		Life:           10,
	}

	newShellImpactManager(target, bullet).updateShotBullets()

	if target.CurHP != 100 {
		t.Fatalf("target HP = %.1f, want 100 for a straddle miss", target.CurHP)
	}
	if bullet.HitObjType != object.TypeWater {
		t.Fatalf("hit type = %v, want water", bullet.HitObjType)
	}
}

func TestDirectShellStillUsesSweptCollision(t *testing.T) {
	target := &objUnit.BattleShip{
		Uid:          "target",
		TotalHP:      100,
		CurHP:        100,
		Length:       51.2,
		Width:        12.8,
		CurPos:       objPos.NewR(10, 10),
		BelongPlayer: faction.ComputerAlpha,
	}
	bullet := &objBullet.Bullet{
		Type:           objBullet.TypeShell,
		Damage:         10,
		ShotType:       objBullet.ShotTypeDirect,
		TargetObjType:  object.TypeShip,
		Shooter:        "shooter",
		ShooterObjType: object.TypeShip,
		BelongPlayer:   faction.HumanAlpha,
		CurPos:         objPos.NewR(10, 11),
		TargetPos:      objPos.NewR(10, 9),
		Speed:          2,
		Life:           10,
	}

	newShellImpactManager(target, bullet).updateShotBullets()

	if target.CurHP != 90 {
		t.Fatalf("target HP = %.1f, want 90 after swept direct hit", target.CurHP)
	}
	if bullet.HitObjType != object.TypeShip {
		t.Fatalf("hit type = %v, want ship", bullet.HitObjType)
	}
}
