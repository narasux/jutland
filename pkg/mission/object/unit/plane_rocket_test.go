package unit

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// 飞机火箭挂载按反类型标志过滤目标：对空挂载只能打空中目标，
// 对海挂载只能打战舰（对空/对海挂载是两种独立配置）。
func TestPlaneRocketLauncherAntiTypeGate(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	const bulletName = "test-plane-rocket-bullet"
	oldBullet, hadBullet := objBullet.Map[bulletName]
	objBullet.Map[bulletName] = &objBullet.Bullet{Type: objBullet.TypeRocket, Damage: 22, Diameter: 127}
	t.Cleanup(func() {
		if hadBullet {
			objBullet.Map[bulletName] = oldBullet
		} else {
			delete(objBullet.Map, bulletName)
		}
	})

	shooter := &Plane{
		Uid: "shooter", CurHP: 100,
		CurPos: objPos.NewR(10, 10), CurRotation: 0,
		Length: 10, Width: 13, BelongPlayer: faction.HumanAlpha,
	}
	// 目标位于机头正前方 0.5 格，处于射界 [0, 25] 与射程内
	enemyPlane := &Plane{
		Uid: "enemy-plane", CurHP: 40,
		CurPos: objPos.NewR(10, 9.5), CurRotation: 0, CurSpeed: 0,
		Length: 10, Width: 13, BelongPlayer: faction.ComputerAlpha,
	}
	enemyShip := &BattleShip{
		Uid: "enemy-ship", CurHP: 100,
		CurPos: objPos.NewR(10, 9.5), BelongPlayer: faction.ComputerAlpha,
	}

	newLauncher := func(antiShip, antiAircraft bool) *PlaneRocketLauncher {
		return &PlaneRocketLauncher{
			Name: "test-plane-rocket", BulletName: bulletName,
			RocketCount: 4, ShotInterval: 0.1, Range: 4,
			BulletSpread: 10, BulletSpeed: 1.0, PosPercent: 0.5,
			AntiShip: antiShip, AntiAircraft: antiAircraft,
			ProximityRadius: 0.22, BlastRadius: 0.35,
			RightFiringArc: FiringArc{Start: 0, End: 25},
			LeftFiringArc:  FiringArc{Start: 335, End: 360},
		}
	}

	// 对空挂载：只能打空中目标
	aaLauncher := newLauncher(false, true)
	if bullets := aaLauncher.Fire(shooter, enemyShip); len(bullets) != 0 {
		t.Fatalf("AA rocket mount should not fire at a ship, got %d bullets", len(bullets))
	}
	aaBullets := aaLauncher.Fire(shooter, enemyPlane)
	if len(aaBullets) != 1 {
		t.Fatalf("AA rocket mount should fire at an airborne target, got %d bullets", len(aaBullets))
	}
	if aaBullets[0].TargetObjType != object.TypePlane {
		t.Fatalf("AA rocket target type = %v, want plane", aaBullets[0].TargetObjType)
	}

	// 对海挂载：只能打战舰
	shipLauncher := newLauncher(true, false)
	if bullets := shipLauncher.Fire(shooter, enemyPlane); len(bullets) != 0 {
		t.Fatalf("anti-ship rocket mount should not fire at an airborne plane, got %d bullets", len(bullets))
	}
	shipBullets := shipLauncher.Fire(shooter, enemyShip)
	if len(shipBullets) != 1 {
		t.Fatalf("anti-ship rocket mount should fire at a ship, got %d bullets", len(shipBullets))
	}
	if shipBullets[0].TargetObjType != object.TypeShip {
		t.Fatalf("anti-ship rocket target type = %v, want ship", shipBullets[0].TargetObjType)
	}
}
