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

// TestPlaneAttackKeepsTrackingUntilBombsAllReleased 验证炸弹整舱投完才脱离目标。
// 场景：test-salvo-bomber 机型模板带 2 枚炸弹 + 1 枚鱼雷、RemainRange=100；
// 轰炸机从 (50,50) 攻击 (80,50) 的 1000/1000 血敌巡洋舰 enemy-ship。第 1 次 Exec
// 后手动把 Bombs[0].Released 置 true，再 Exec；最后置 Bombs[1].Released 再 Exec。
// 预期：第一次不结束且 CurAttackTarget="enemy-ship"；只投 1 枚仍继续跟踪同一目标；
// 两枚都投完才 Executed 且清空目标。多带的鱼雷保证 MustReturn 不会提前结束，
// 从而只检验炸弹齐投的脱离条件；否则每次通场只丢一枚。
// 覆盖：newShipAttackTestState(t, "test-salvo-bomber", 2, 1) + 三次
// execPlaneAttack，逐步断言 Executed 与 CurAttackTarget。
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

// TestPlaneAttackDisengagesAfterSingleTorpedo 验证鱼雷机一次通场投一雷即脱离。
// 场景：test-run-torpedo-bomber 模板带 1 枚炸弹 + 2 枚鱼雷；创建指令前就把
// Bombs[0].Released 置 true（模拟炸弹舱已空），敌巡洋舰在 (80,50)。
// 预期：首拍不结束（快照里炸弹已投完，不触发炸弹脱离，继续为鱼雷通场跟踪）；
// 投出 Torpedoes[0] 后立即 Executed 并清空 CurAttackTarget，剩余鱼雷不阻塞脱离。
// 覆盖：newShipAttackTestState(t, "test-run-torpedo-bomber", 1, 2) + 两次
// execPlaneAttack，断言 Executed 与目标清空。
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
