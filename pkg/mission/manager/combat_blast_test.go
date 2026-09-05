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

// 炸弹/曲射炮弹落点爆炸应波及半径内的所有地面飞机（跑道/甲板滑跑中），
// 空中目标与友方地面目标（友军伤害关闭时）不受波及。
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
		objBullet.ShotTypeArcing, object.TypePlane, 0.4, 10,
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
