package unit

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestPlaneRocketLauncherAntiTypeGate 验证飞机火箭挂载按 AntiShip/AntiAircraft 标志过滤目标。
// 场景：反舰挂载（AntiShip）与对空挂载（AntiAircraft）是两套独立配置，射击时可能拿到飞机也可能拿到战舰；
// 若 Fire 不先过 IsAvailableAntiType，对空火箭会朝战舰打、反舰火箭会朝飞机打。
// 预期：对空挂载对战舰返回 0 发、对正前方 0.5 格敌机返回 1 发且 TargetObjType=TypePlane；
// 反舰挂载对敌机返回 0 发、对同一位置战舰返回 1 发且 TargetObjType=TypeShip。
// 覆盖：注册 4 发、射程 4、射界 [0,25]/[335,360] 的 PlaneRocketLauncher，
// 对空/反舰两种挂载分别 Fire 敌机与敌舰，断言发数与目标类型。
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
