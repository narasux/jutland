package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/targeting"
)

const (
	preferTestPlaneName  = "test-prefer-bomber"
	preferTestBulletName = "test-prefer-bomb"
)

// registerPreferTestBomber 注册带一枚炸弹的测试轰炸机模板，载弹面板伤害 500。
func registerPreferTestBomber(t *testing.T) {
	t.Helper()
	oldBullet, hadBullet := objBullet.Map[preferTestBulletName]
	objBullet.Map[preferTestBulletName] = &objBullet.Bullet{Type: objBullet.TypeBomb, Damage: 500}
	oldTemplate, hadTemplate := objUnit.PlaneMap[preferTestPlaneName]
	objUnit.PlaneMap[preferTestPlaneName] = &objUnit.Plane{
		Name:   preferTestPlaneName,
		Type:   objUnit.PlaneTypeDiveBomber,
		Weapon: objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{BulletName: preferTestBulletName}}},
	}
	t.Cleanup(func() {
		if hadBullet {
			objBullet.Map[preferTestBulletName] = oldBullet
		} else {
			delete(objBullet.Map, preferTestBulletName)
		}
		if hadTemplate {
			objUnit.PlaneMap[preferTestPlaneName] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, preferTestPlaneName)
		}
	})
}

// newPreferTestManager 创建带两艘敌舰（残血 wounded / 满血 healthy）目标队列的管理器。
// 队列顺序为 [wounded, healthy]，游标由各测试自行设置。
func newPreferTestManager(t *testing.T) *MissionManager {
	t.Helper()
	wounded := &objUnit.BattleShip{
		Uid: "wounded", CurHP: 100, TotalHP: 3000,
		CurPos: objPos.NewR(60, 50), BelongPlayer: faction.ComputerAlpha,
	}
	healthy := &objUnit.BattleShip{
		Uid: "healthy", CurHP: 9000, TotalHP: 9000,
		CurPos: objPos.NewR(70, 50), BelongPlayer: faction.ComputerAlpha,
	}
	return &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Ships: map[string]*objUnit.BattleShip{
					wounded.Uid: wounded,
					healthy.Uid: healthy,
				},
			},
		},
		targetingPlan: targeting.Plan{
			BaseQueues: map[string]map[object.Type][]targeting.TargetRef{
				"base": {object.TypeShip: {{UID: "wounded"}, {UID: "healthy"}}},
			},
		},
		targetingCursors: map[string]map[object.Type]int{},
	}
}

// 在空轰炸机重新分配目标时，剩余载弹能一波带走的残血敌舰优先于游标轮转。
func TestNextTargetUIDForPlanePrefersOneSalvoKillableShip(t *testing.T) {
	registerPreferTestBomber(t)
	manager := newPreferTestManager(t)

	plane := &objUnit.Plane{
		Uid:          "bomber",
		Name:         preferTestPlaneName,
		Type:         objUnit.PlaneTypeDiveBomber,
		BelongShip:   "base",
		BelongPlayer: faction.HumanAlpha,
		CurPos:       objPos.NewR(50, 50),
		RemainRange:  1000,
		Weapon:       objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{BulletName: preferTestBulletName}}},
	}

	// 游标停在 healthy 上，但 wounded 只剩 100 血、载弹 500 → 优先 wounded
	manager.setTargetCursor("base", object.TypeShip, 1)
	uid, ok := manager.nextTargetUIDForPlane(plane, map[string]int{})
	if !ok || uid != "wounded" {
		t.Fatalf("target = %q ok=%v, want wounded (killable in one salvo)", uid, ok)
	}

	// 没有能一波带走的目标时，回退到游标轮转顺序（游标已被推进到 healthy）
	manager.state.Arena.Ships["wounded"].CurHP = 9000
	uid, ok = manager.nextTargetUIDForPlane(plane, map[string]int{})
	if !ok || uid != "healthy" {
		t.Fatalf("target = %q ok=%v, want healthy (cursor order fallback)", uid, ok)
	}

	// 没有剩余载弹的飞机不做偏好，纯按游标轮转
	manager.setTargetCursor("base", object.TypeShip, 0)
	plain := &objUnit.Plane{
		Uid:          "fighter",
		Name:         preferTestPlaneName,
		Type:         objUnit.PlaneTypeFighter,
		BelongShip:   "base",
		BelongPlayer: faction.HumanAlpha,
		CurPos:       objPos.NewR(50, 50),
		RemainRange:  1000,
	}
	uid, ok = manager.nextTargetUIDForPlane(plain, map[string]int{})
	if !ok || uid != "wounded" {
		t.Fatalf("target = %q ok=%v, want wounded (no payload, cursor order)", uid, ok)
	}
}

// 起飞分配目标时同样优先「一波能带走」的敌舰（按基地可出机型的最大载弹估算）。
func TestTakeOffFromBasePrefersOneSalvoKillableShip(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	registerPreferTestBomber(t)
	af := objBuilding.NewAirfield(
		objPos.New(50, 50), 0, 6, 0.8, faction.HumanAlpha, 0,
		[]objUnit.PlaneGroup{{Name: preferTestPlaneName, MaxCount: 1}},
	)
	af.StockPlane(preferTestPlaneName)
	wounded := &objUnit.BattleShip{
		Uid: "wounded", CurHP: 100, TotalHP: 3000,
		CurPos: objPos.NewR(60, 50), BelongPlayer: faction.ComputerAlpha,
	}
	healthy := &objUnit.BattleShip{
		Uid: "healthy", CurHP: 9000, TotalHP: 9000,
		CurPos: objPos.NewR(70, 50), BelongPlayer: faction.ComputerAlpha,
	}
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Airfields: map[string]*objBuilding.Airfield{af.Uid: af},
				Planes:    map[string]*objUnit.Plane{},
				Ships: map[string]*objUnit.BattleShip{
					wounded.Uid: wounded,
					healthy.Uid: healthy,
				},
			},
		},
		targetingPlan: targeting.Plan{
			BaseQueues: map[string]map[object.Type][]targeting.TargetRef{
				af.Uid: {object.TypeShip: {{UID: "healthy"}, {UID: "wounded"}}},
			},
		},
		targetingCursors: map[string]map[object.Type]int{},
	}

	plane, targetType, targetUID, ok := manager.takeOffFromBase(af, map[string]int{})
	if !ok || plane == nil {
		t.Fatal("airfield should launch a bomber against the enemy ships")
	}
	if targetType != object.TypeShip || targetUID != "wounded" {
		t.Fatalf("target = %q (%v), want wounded ship (killable in one salvo)", targetUID, targetType)
	}
}
