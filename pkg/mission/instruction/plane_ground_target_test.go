package instruction

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// 轰炸机锁定的地面目标一旦升空（滑行结束起飞），指令应立即终止：
// 炸弹无法攻击空中目标，持续追踪只会绕着机场空转。
func TestPlaneAttackDisengagesWhenGroundTargetTakesOff(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	const bulletName = "test-disengage-bomb"
	oldBullet, hadBullet := objBullet.Map[bulletName]
	objBullet.Map[bulletName] = &objBullet.Bullet{Type: objBullet.TypeBomb}
	t.Cleanup(func() {
		if hadBullet {
			objBullet.Map[bulletName] = oldBullet
		}
		delete(objBullet.Map, bulletName)
	})

	const bomberName = "SBD-3"
	bomber := &objUnit.Plane{
		Uid:          "bomber-1",
		Name:         bomberName,
		Type:         objUnit.PlaneTypeDiveBomber,
		TotalHP:      100,
		CurHP:        100,
		FlightPhase:  objUnit.PlaneFlightPhaseCruising,
		BelongPlayer: faction.ComputerAlpha,
		Weapon: objUnit.PlaneWeapon{
			Bombs: []*objUnit.Releaser{{BulletName: bulletName, Range: 1.5, BulletSpeed: 0.4}},
		},
	}
	target := &objUnit.Plane{
		Uid:          "ground-target-1",
		TotalHP:      100,
		CurHP:        100,
		FlightPhase:  objUnit.PlaneFlightPhaseTakingOff,
		BelongPlayer: faction.HumanAlpha,
	}
	ms := &state.MissionState{
		Arena: state.MissionArenaState{
			Planes: map[string]*objUnit.Plane{bomber.Uid: bomber, target.Uid: target},
		},
	}

	i := NewPlaneAttack(bomber.Uid, object.TypePlane, target.Uid)
	if err := i.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if !i.Executed() {
		t.Fatal("bomber should disengage once its ground target leaves the ground")
	}
}
