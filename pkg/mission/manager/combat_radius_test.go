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

// 注册一架带航程的测试俯冲轰炸机模板，供作战半径用例复用。
func registerRadiusTestBomber(t *testing.T, planeName, bulletName string, planeRange float64) {
	t.Helper()
	oldBullet, hadBullet := objBullet.Map[bulletName]
	objBullet.Map[bulletName] = &objBullet.Bullet{Type: objBullet.TypeBomb, Damage: 100}
	oldTemplate, hadTemplate := objUnit.PlaneMap[planeName]
	objUnit.PlaneMap[planeName] = &objUnit.Plane{
		Name: planeName, Type: objUnit.PlaneTypeDiveBomber,
		Range: planeRange, RemainRange: planeRange,
		Weapon: objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{BulletName: bulletName}}},
	}
	t.Cleanup(func() {
		if hadBullet {
			objBullet.Map[bulletName] = oldBullet
		} else {
			delete(objBullet.Map, bulletName)
		}
		if hadTemplate {
			objUnit.PlaneMap[planeName] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, planeName)
		}
	})
}

// 在空飞机重新分配目标时，剩余航程不够「飞抵并返航」的敌舰必须被跳过，
// 即使它单程就在航程之内。
func TestNextTargetUIDForPlaneSkipsTargetBeyondCombatRadius(t *testing.T) {
	const planeName = "radius-test-bomber"
	const bulletName = "radius-test-bomb"
	registerRadiusTestBomber(t, planeName, bulletName, 200)

	// 基地与飞机都在 (50,50)：near 距离 10（往返 20），far 距离 70（往返 140）。
	base := &objUnit.BattleShip{
		Uid: "base", CurHP: 1000, TotalHP: 1000,
		CurPos: objPos.NewR(50, 50), BelongPlayer: faction.HumanAlpha,
	}
	near := &objUnit.BattleShip{
		Uid: "near", CurHP: 1000, TotalHP: 1000,
		CurPos: objPos.NewR(60, 50), BelongPlayer: faction.ComputerAlpha,
	}
	far := &objUnit.BattleShip{
		Uid: "far", CurHP: 1000, TotalHP: 1000,
		CurPos: objPos.NewR(120, 50), BelongPlayer: faction.ComputerAlpha,
	}
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Ships: map[string]*objUnit.BattleShip{base.Uid: base, near.Uid: near, far.Uid: far},
			},
		},
		targetingPlan: targeting.Plan{
			BaseQueues: map[string]map[object.Type][]targeting.TargetRef{
				// 游标从 far 开始：旧逻辑会直接选中它
				"base": {object.TypeShip: {{UID: "far"}, {UID: "near"}}},
			},
		},
		targetingCursors: map[string]map[object.Type]int{},
	}
	plane := &objUnit.Plane{
		Uid: "bomber", Name: planeName, Type: objUnit.PlaneTypeDiveBomber,
		BelongShip: "base", BelongPlayer: faction.HumanAlpha,
		CurPos: objPos.NewR(50, 50), RemainRange: 100,
	}

	// 剩余 100：far 单程 70 够、往返 140 不够 → 只能选 near
	uid, ok := manager.nextTargetUIDForPlane(plane, map[string]int{})
	if !ok || uid != "near" {
		t.Fatalf("target = %q ok=%v, want near (outbound + return within remaining range)", uid, ok)
	}

	// 队里只剩 far 时，剩余航程不足以往返 → 不接受任何目标，交由返航流程处理
	manager.targetingPlan.BaseQueues["base"][object.TypeShip] = []targeting.TargetRef{{UID: "far"}}
	manager.setTargetCursor("base", object.TypeShip, 0)
	if uid, ok := manager.nextTargetUIDForPlane(plane, map[string]int{}); ok {
		t.Fatalf("target = %q, want none when every target is beyond combat radius", uid)
	}
}

// 起飞选目标时同样按作战半径校验：机型航程必须覆盖基地到目标的往返距离，
// 否则机场/航母保持待命，等敌舰进入半径再放飞。
func TestTakeOffFromBaseSkipsTargetBeyondCombatRadius(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	const planeName = "radius-takeoff-bomber"
	const bulletName = "radius-takeoff-bomb"
	registerRadiusTestBomber(t, planeName, bulletName, 100)

	airfield := objBuilding.NewAirfield(
		objPos.New(50, 50), 0, 6, 0.8, faction.HumanAlpha, 0,
		[]objUnit.PlaneGroup{{Name: planeName, MaxCount: 2}},
	)
	airfield.StockPlane(planeName)
	// 距离 60：单程在 100 航程内，往返 120 超出作战半径
	enemy := &objUnit.BattleShip{
		Uid: "far", CurHP: 1000, TotalHP: 1000,
		CurPos: objPos.New(110, 50), BelongPlayer: faction.ComputerAlpha,
	}
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Airfields: map[string]*objBuilding.Airfield{airfield.Uid: airfield},
				Planes:    map[string]*objUnit.Plane{},
				Ships:     map[string]*objUnit.BattleShip{enemy.Uid: enemy},
			},
		},
		targetingPlan: targeting.Plan{
			BaseQueues: map[string]map[object.Type][]targeting.TargetRef{
				airfield.Uid: {object.TypeShip: {{UID: "far"}}},
			},
		},
		targetingCursors: map[string]map[object.Type]int{},
	}

	if plane, _, _, ok := manager.takeOffFromBase(airfield, map[string]int{}); ok {
		t.Fatalf("launched %v against a target beyond its combat radius", plane)
	}

	// 敌舰进入作战半径（距离 40，往返 80 ≤ 100）后才允许起飞
	enemy.CurPos = objPos.New(90, 50)
	plane, targetType, targetUID, ok := manager.takeOffFromBase(airfield, map[string]int{})
	if !ok || plane == nil {
		t.Fatal("airfield should launch once the target is inside its combat radius")
	}
	if targetType != object.TypeShip || targetUID != "far" {
		t.Fatalf("target = %q (%v), want far ship", targetUID, targetType)
	}
}
