package instruction

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// newShipAttackTestState 构造一架对舰轰炸机与一艘敌舰的攻击场景。
// 炸弹 / 鱼雷数量写进机型模板，NewPlane 起飞时会深拷贝成独立释放器。
func newShipAttackTestState(
	t *testing.T, planeName string, bombs, torpedoes int,
) (*state.MissionState, *objUnit.Plane) {
	t.Helper()
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	weapon := objUnit.PlaneWeapon{}
	for range bombs {
		weapon.Bombs = append(weapon.Bombs, &objUnit.Releaser{})
	}
	for range torpedoes {
		weapon.Torpedoes = append(weapon.Torpedoes, &objUnit.Releaser{})
	}
	oldTemplate, hadTemplate := objUnit.PlaneMap[planeName]
	objUnit.PlaneMap[planeName] = &objUnit.Plane{
		Name:         planeName,
		Type:         objUnit.PlaneTypeDiveBomber,
		TotalHP:      100,
		CurHP:        100,
		MaxSpeed:     0.12,
		Acceleration: 0.01,
		RotateSpeed:  12,
		RemainRange:  100,
		Weapon:       weapon,
	}
	t.Cleanup(func() {
		if hadTemplate {
			objUnit.PlaneMap[planeName] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, planeName)
		}
	})

	ship := &objUnit.BattleShip{
		Uid:          "enemy-ship",
		Type:         objUnit.ShipTypeCruiser,
		CurHP:        1000,
		TotalHP:      1000,
		CurPos:       objPos.NewR(80, 50),
		BelongPlayer: faction.ComputerAlpha,
	}
	plane := objUnit.NewPlane(planeName, objPos.NewR(50, 50), 0, "", faction.HumanAlpha)
	return &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{
				MapCfg: &mapcfg.MapCfg{Width: 100, Height: 100},
			},
		},
		Arena: state.MissionArenaState{
			Ships:  map[string]*objUnit.BattleShip{ship.Uid: ship},
			Planes: map[string]*objUnit.Plane{plane.Uid: plane},
		},
	}, plane
}

func execPlaneAttack(t *testing.T, i *PlaneAttack, missionState *state.MissionState) {
	t.Helper()
	if err := i.Exec(missionState); err != nil {
		t.Fatal(err)
	}
}

// 炸弹要整舱投完才脱离：投出第一枚后指令必须继续跟踪同一目标，
// 全部投完才结束指令并清空目标（否则轰炸机每次通场只会丢一枚）。
// 额外带一枚鱼雷，保证 MustReturn 不会提前结束指令，只检验脱离条件本身。
func TestPlaneAttackKeepsTrackingUntilBombsAllReleased(t *testing.T) {
	ms, plane := newShipAttackTestState(t, "test-salvo-bomber", 2, 1)

	i := NewPlaneAttack(plane.Uid, object.TypeShip, "enemy-ship")
	execPlaneAttack(t, i, ms)
	if i.Executed() {
		t.Fatal("attack instruction ended before any bomb was released")
	}
	if plane.CurAttackTarget != "enemy-ship" {
		t.Fatalf("target = %q, want enemy-ship", plane.CurAttackTarget)
	}

	plane.Weapon.Bombs[0].Released = true
	execPlaneAttack(t, i, ms)
	if i.Executed() {
		t.Fatal("bomber disengaged after only one bomb of its salvo was released")
	}
	if plane.CurAttackTarget != "enemy-ship" {
		t.Fatalf("target = %q, want to keep tracking until the salvo is empty", plane.CurAttackTarget)
	}

	plane.Weapon.Bombs[1].Released = true
	execPlaneAttack(t, i, ms)
	if !i.Executed() {
		t.Fatal("bomber should disengage once all bombs are released")
	}
	if plane.CurAttackTarget != "" {
		t.Fatalf("target = %q, want cleared after the salvo", plane.CurAttackTarget)
	}
}

// 鱼雷保持一次通场投一雷：投出一雷即脱离；
// 投前炸弹舱已经投完（快照时全部 Released）不会让指令刚接敌就掉头。
func TestPlaneAttackDisengagesAfterSingleTorpedo(t *testing.T) {
	ms, plane := newShipAttackTestState(t, "test-run-torpedo-bomber", 1, 2)
	plane.Weapon.Bombs[0].Released = true

	i := NewPlaneAttack(plane.Uid, object.TypeShip, "enemy-ship")
	execPlaneAttack(t, i, ms)
	if i.Executed() {
		t.Fatal("bomber with an empty bomb bay should keep tracking for a torpedo run")
	}

	plane.Weapon.Torpedoes[0].Released = true
	execPlaneAttack(t, i, ms)
	if !i.Executed() {
		t.Fatal("torpedo bomber should disengage after releasing one torpedo")
	}
	if plane.CurAttackTarget != "" {
		t.Fatalf("target = %q, want cleared after the torpedo release", plane.CurAttackTarget)
	}
}
