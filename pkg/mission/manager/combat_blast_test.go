package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestBombBlastDamagesNearbyGroundPlanes 验证炸弹落点 1 格半径内波及所有地面飞机，空中与友军不受影响。
// 场景：30 伤害炸弹在 (10,10) 引爆，附近有 0.3 格内 TakingOff 的敌机、1.5 格外敌机、0.28 格但 Cruising 的敌机，以及 0.3 格内 ComputerAlpha 友机；FriendlyFire=false。
// 预期：damageGroundPlanesNear 返回 true；近处地面敌机掉血，远处、空中与同阵营地面机保持 100 HP。
// 覆盖：bombBlastRadius=1.0 的距离过滤、IsOnGround 与 FriendlyFire 判定，断言四架飞机的 CurHP。
func TestBombBlastDamagesNearbyGroundPlanes(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	const bulletName = "test-blast-bomb"
	oldBullet, hadBullet := objBullet.Map[bulletName]
	objBullet.Map[bulletName] = &objBullet.Bullet{
		Type: objBullet.TypeBomb, Damage: 30,
	}
	t.Cleanup(func() {
		if hadBullet {
			objBullet.Map[bulletName] = oldBullet
		}
		delete(objBullet.Map, bulletName)
	})

	newGroundPlane := func(uid string, x, y float64, player faction.Player) *objUnit.Plane {
		return &objUnit.Plane{
			Uid: uid, TotalHP: 100, CurHP: 100,
			CurPos:       objPos.NewR(x, y),
			FlightPhase:  objUnit.PlaneFlightPhaseTakingOff,
			BelongPlayer: player,
		}
	}
	// 落点 (10,10)：敌机紧邻、敌机在波及边缘外、敌机在空中、友机在半径内
	enemyNear := newGroundPlane("enemy-near", 10.3, 10, faction.HumanAlpha)
	enemyFar := newGroundPlane("enemy-far", 11.5, 10, faction.HumanAlpha)
	enemyAir := newGroundPlane("enemy-air", 10.2, 10.2, faction.HumanAlpha)
	enemyAir.FlightPhase = objUnit.PlaneFlightPhaseCruising
	friendlyNear := newGroundPlane("friendly-near", 9.7, 10, faction.ComputerAlpha)

	missionState := &state.MissionState{
		Arena: state.MissionArenaState{
			Planes: map[string]*objUnit.Plane{
				enemyNear.Uid: enemyNear, enemyFar.Uid: enemyFar,
				enemyAir.Uid: enemyAir, friendlyNear.Uid: friendlyNear,
			},
		},
	}
	missionState.UI.GameOpts.FriendlyFire = false
	manager := &MissionManager{state: missionState}

	bt := objBullet.New(
		bulletName, objPos.NewR(9, 10), objPos.NewR(10, 10),
		"shooter-1", object.TypePlane, faction.ComputerAlpha,
		object.TypePlane, 0.4, 10, 1,
	)
	bt.CurPos = objPos.NewR(10, 10)

	if !manager.damageGroundPlanesNear(bt, "") {
		t.Fatal("blast should damage at least one ground plane")
	}
	if enemyNear.CurHP >= enemyNear.TotalHP {
		t.Fatal("ground plane within blast radius should take damage")
	}
	if enemyFar.CurHP != enemyFar.TotalHP {
		t.Fatal("ground plane outside blast radius should not take damage")
	}
	if enemyAir.CurHP != enemyAir.TotalHP {
		t.Fatal("airborne plane should not take blast damage")
	}
	if friendlyNear.CurHP != friendlyNear.TotalHP {
		t.Fatal("friendly ground plane should not take damage with friendly fire off")
	}
}
