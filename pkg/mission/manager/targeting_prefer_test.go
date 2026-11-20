package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
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

// TestNextTargetUIDForPlanePrefersOneSalvoKillableShip 验证在空轰炸机重选目标时优先剩余载弹能一波击沉的残血舰。
// 场景：测试轰炸机挂 1 枚 500 伤害炸弹、RemainRange=1000；wounded 100/3000、healthy 9000/9000，队列 [wounded, healthy]，游标先停在 healthy。
// 预期：游标在 healthy 也先选 wounded；wounded 补满 9000 后回退游标顺序选 healthy；载弹不足以一波带走的飞机纯按游标选 wounded。
// 覆盖：oneSalvoKillablePrefer(payloadDamage) 与 findTargetAtCursorWhere 的 prefer/fallback 分支，配合 setTargetCursor。
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

	// 载弹伤害不足以一波带走的飞机不做偏好，纯按游标轮转
	manager.setTargetCursor("base", object.TypeShip, 0)
	plain := &objUnit.Plane{
		Uid:          "fighter",
		Name:         preferTestPlaneName,
		Type:         objUnit.PlaneTypeFighter,
		BelongShip:   "base",
		BelongPlayer: faction.HumanAlpha,
		CurPos:       objPos.NewR(50, 50),
		RemainRange:  1000,
		Weapon:       objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{}}},
	}
	uid, ok = manager.nextTargetUIDForPlane(plain, map[string]int{})
	if !ok || uid != "wounded" {
		t.Fatalf("target = %q ok=%v, want wounded (not one-salvo-killable, cursor order)", uid, ok)
	}
}

// TestUpdatePlaneAttackOrReturnReturnsPlaneWithoutPayload 验证只剩不可投放弹药的飞机直接返航而不是继续绕圈。
// 场景：飞机只挂 1 枚反潜深弹、普通炸弹已投完，目标队列里只有两艘水面敌舰；若仍分配水面目标，
// PlaneAttack 每拍都投不出任何东西，飞机会在舰上方永久盘旋。
// 预期：updatePlaneAttackOrReturn 不添加 PlaneAttack，而是添加 PlaneReturn 指令。
// 覆盖：newPreferTestManager 的 [wounded, healthy] 队列 + 仅深弹飞机，直接跑一帧后检查指令集。
func TestUpdatePlaneAttackOrReturnReturnsPlaneWithoutPayload(t *testing.T) {
	registerPreferTestBomber(t)
	manager := newPreferTestManager(t)
	manager.instructionSet = NewInstructionSet()

	plane := &objUnit.Plane{
		Uid:          "asw-bomber",
		Name:         preferTestPlaneName,
		Type:         objUnit.PlaneTypeDiveBomber,
		BelongShip:   "base",
		BelongPlayer: faction.HumanAlpha,
		CurPos:       objPos.NewR(50, 50),
		RemainRange:  1000,
		Weapon:       objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{AntiSubmarine: true}}},
	}
	manager.state.Arena.Planes = map[string]*objUnit.Plane{plane.Uid: plane}

	manager.updatePlaneAttackOrReturn()

	if manager.instructionSet.Exists(instr.GenInstrUid(instr.NamePlaneAttack, plane.Uid)) {
		t.Fatal("plane without usable payload must not receive a surface attack instruction")
	}
	if !manager.instructionSet.Exists(instr.GenInstrUid(instr.NamePlaneReturn, plane.Uid)) {
		t.Fatal("plane without usable payload should be sent home instead of circling the target")
	}
}

// TestTakeOffFromBasePrefersOneSalvoKillableShip 验证机场起飞分配目标时同样优先一波可带走的残血舰。
// 场景：机场 (50,50) 库存 1 架 500 伤害测试轰炸机；wounded 100/3000 在 (60,50)、healthy 9000/9000 在 (70,50)，队列 [healthy, wounded]。
// 预期：takeOffFromBase 返回飞机且目标为 wounded，而不是游标在前的 healthy；伤害按基地可出机型的最大单机载弹估算。
// 覆盖：maxGroupPayloadDamage(aircraft, TypeShip) 配合 oneSalvoKillablePrefer 注入 peekTargetUIDWhere，使用真实 Airfield/StockPlane。
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

// TestNextTargetUIDForPlaneSkipsSurfaceTargetWithoutPayload 验证只剩反潜弹的飞机不会再接水面舰任务。
// 场景：飞机只挂 1 枚 AntiSubmarine 深弹（普通炸弹已投完），目标队列里只有两艘水面敌舰；
// 若仍把水面舰分配给飞机，PlaneAttack 无法投放任何弹药，飞机会在舰上方永久盘旋。
// 预期：nextTargetUIDForPlane 返回 ok=false，交由 updatePlaneAttackOrReturn 触发返航。
// 覆盖：复用 newPreferTestManager 的 [wounded, healthy] 队列，构造仅深弹机并断言无目标；
// 同一测试里用仅普通炸弹机做对照，确认过滤只针对不可投放的挂载。
func TestNextTargetUIDForPlaneSkipsSurfaceTargetWithoutPayload(t *testing.T) {
	registerPreferTestBomber(t)
	manager := newPreferTestManager(t)

	aswPlane := &objUnit.Plane{
		Uid:          "asw-bomber",
		Name:         preferTestPlaneName,
		Type:         objUnit.PlaneTypeDiveBomber,
		BelongShip:   "base",
		BelongPlayer: faction.HumanAlpha,
		CurPos:       objPos.NewR(50, 50),
		RemainRange:  1000,
		Weapon:       objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{AntiSubmarine: true}}},
	}
	if uid, ok := manager.nextTargetUIDForPlane(aswPlane, map[string]int{}); ok {
		t.Fatalf("target = %q ok=true, want no surface target for anti-sub-only loadout", uid)
	}

	bombPlane := &objUnit.Plane{
		Uid:          "bomb-bomber",
		Name:         preferTestPlaneName,
		Type:         objUnit.PlaneTypeDiveBomber,
		BelongShip:   "base",
		BelongPlayer: faction.HumanAlpha,
		CurPos:       objPos.NewR(50, 50),
		RemainRange:  1000,
		Weapon:       objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{}}},
	}
	if uid, ok := manager.nextTargetUIDForPlane(bombPlane, map[string]int{}); !ok || uid == "" {
		t.Fatal("bomber with a droppable bomb should still receive a surface target")
	}
}
