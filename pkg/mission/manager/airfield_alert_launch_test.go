package manager

import (
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// registerAlertTestBomber 注册测试用俯冲轰炸机模板（目标类型为舰船）。
func registerAlertTestBomber(t *testing.T) {
	t.Helper()
	const name = "test-alert-bomber"
	oldTemplate, hadTemplate := objUnit.PlaneMap[name]
	objUnit.PlaneMap[name] = &objUnit.Plane{
		Name:   name,
		Type:   objUnit.PlaneTypeDiveBomber,
		Weapon: objUnit.PlaneWeapon{Bombs: []*objUnit.Releaser{{}}},
	}
	t.Cleanup(func() {
		if hadTemplate {
			objUnit.PlaneMap[name] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, name)
		}
	})
}

// newAlertTestManager 创建单机场（库存 1 架测试轰炸机）的任务管理器。
func newAlertTestManager(t *testing.T) (*MissionManager, *objBuilding.Airfield) {
	t.Helper()
	registerAlertTestBomber(t)

	af := objBuilding.NewAirfield(
		objPos.New(50, 50), 0, 6, 0.8, faction.HumanAlpha, 0,
		[]objUnit.PlaneGroup{{Name: "test-alert-bomber", MaxCount: 2}},
	)
	af.StockPlane("test-alert-bomber")
	ms := &state.MissionState{
		Arena: state.MissionArenaState{
			Airfields: map[string]*objBuilding.Airfield{af.Uid: af},
			Planes:    map[string]*objUnit.Plane{},
			Ships:     map[string]*objUnit.BattleShip{},
		},
	}
	return &MissionManager{state: ms, instructionSet: NewInstructionSet()}, af
}

// TestAirfieldDisabledNoAlertLaunch 停用机场不自动起飞：敌舰在场也不响应；
// 恢复启用后立即正常警戒起飞。
func TestAirfieldDisabledNoAlertLaunch(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	manager, af := newAlertTestManager(t)
	af.Disabled = true
	manager.state.Arena.Ships["enemy-ship"] = &objUnit.BattleShip{
		Uid:          "enemy-ship",
		CurHP:        100,
		CurPos:       objPos.New(90, 90),
		BelongPlayer: faction.ComputerAlpha,
	}
	manager.updateAirfieldAlertLaunch()
	if len(manager.state.Arena.Planes) != 0 {
		t.Fatalf("disabled airfield should stay parked, planes = %d", len(manager.state.Arena.Planes))
	}

	af.Disabled = false
	manager.updateAirfieldAlertLaunch()
	if len(manager.state.Arena.Planes) != 1 {
		t.Fatalf("re-enabled airfield should launch, planes = %d", len(manager.state.Arena.Planes))
	}
}

// TestAirfieldAlertLaunchWithoutRadius 机场不设警戒范围：场上没有任何敌机 /
// 敌舰时保持待命；远在全场另一端的敌舰出现即自动起飞攻击。
func TestAirfieldAlertLaunchWithoutRadius(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	// 无敌人：不起飞
	manager, af := newAlertTestManager(t)
	manager.updateAirfieldAlertLaunch()
	if len(manager.state.Arena.Planes) != 0 {
		t.Fatalf("no enemy should keep planes parked, got %d airborne", len(manager.state.Arena.Planes))
	}
	if af.Aircraft.Groups[0].CurCount != 1 {
		t.Fatalf("stock = %d, want 1 while parked", af.Aircraft.Groups[0].CurCount)
	}

	// 远处出现敌舰：不限距离，立即起飞并下达攻击指令
	enemyShip := &objUnit.BattleShip{
		Uid:          "enemy-ship",
		CurHP:        100,
		CurPos:       objPos.New(90, 90),
		BelongPlayer: faction.ComputerAlpha,
	}
	manager.state.Arena.Ships[enemyShip.Uid] = enemyShip
	manager.updateAirfieldAlertLaunch()

	if len(manager.state.Arena.Planes) != 1 {
		t.Fatalf("enemy ship should trigger takeoff, got %d planes", len(manager.state.Arena.Planes))
	}
	if af.Aircraft.Groups[0].CurCount != 0 {
		t.Fatalf("stock = %d, want 0 after takeoff", af.Aircraft.Groups[0].CurCount)
	}
	var attack instr.Instruction
	for _, item := range manager.instructionSet.Items() {
		if strings.Contains(item.Uid(), instr.NamePlaneAttack) {
			attack = item
			break
		}
	}
	if attack == nil {
		t.Fatal("takeoff should come with a plane attack instruction")
	}
}

// TestAirfieldAlertLaunchDecoupledTargets 对空 / 对舰两路择敌互不阻塞：
// 地面停放 / 滑跑中的敌机不触发警戒；空中敌机在空时轰炸机照常出击敌舰，
// 只有既无舰可打、又无机可拦时才保持待命。
func TestAirfieldAlertLaunchDecoupledTargets(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	enemyShip := &objUnit.BattleShip{
		Uid:          "enemy-ship",
		CurHP:        100,
		CurPos:       objPos.New(90, 90),
		BelongPlayer: faction.ComputerAlpha,
	}
	airEnemy := func() *objUnit.Plane {
		return &objUnit.Plane{
			Uid:          "air-enemy",
			CurHP:        10,
			CurPos:       objPos.New(52, 52),
			FlightPhase:  objUnit.PlaneFlightPhaseCruising,
			BelongPlayer: faction.ComputerAlpha,
		}
	}

	// 地面敌机不触发警戒（滑跑中的敌机 + 敌舰：应转而打舰）
	manager, af := newAlertTestManager(t)
	manager.state.Arena.Planes["ground-enemy"] = &objUnit.Plane{
		Uid:          "ground-enemy",
		CurHP:        10,
		CurPos:       objPos.New(52, 52),
		FlightPhase:  objUnit.PlaneFlightPhaseTakingOff,
		BelongPlayer: faction.ComputerAlpha,
	}
	manager.state.Arena.Ships[enemyShip.Uid] = enemyShip
	manager.updateAirfieldAlertLaunch()
	if len(manager.state.Arena.Planes) != 2 { // 1 架滑跑敌机 + 1 架我方起飞
		t.Fatalf("ground enemy should not block launch, planes = %d", len(manager.state.Arena.Planes))
	}

	// 空中敌机 + 敌舰同时存在：轰炸机不受敌机阻塞，照常出击敌舰
	manager, af = newAlertTestManager(t)
	manager.state.Arena.Ships[enemyShip.Uid] = enemyShip
	manager.state.Arena.Planes[airEnemy().Uid] = airEnemy()
	manager.updateAirfieldAlertLaunch()
	if af.Aircraft.Groups[0].CurCount != 0 {
		t.Fatalf("air enemy must not pin bombers down, stock = %d", af.Aircraft.Groups[0].CurCount)
	}
	if len(manager.state.Arena.Planes) != 2 { // 1 架空中敌机 + 1 架我方轰炸机
		t.Fatalf(
			"bombers should still strike ships with air enemy around, planes = %d",
			len(manager.state.Arena.Planes),
		)
	}

	// 只有空中敌机（无敌舰）：本场只有轰炸机，无机可拦，保持待命
	manager, af = newAlertTestManager(t)
	manager.state.Arena.Planes[airEnemy().Uid] = airEnemy()
	manager.updateAirfieldAlertLaunch()
	if af.Aircraft.Groups[0].CurCount != 1 {
		t.Fatalf("bomber airfield with no ship target should stay parked, stock = %d", af.Aircraft.Groups[0].CurCount)
	}
	if len(manager.state.Arena.Planes) != 1 {
		t.Fatalf("bomber should not launch against air target, planes = %d", len(manager.state.Arena.Planes))
	}
}

// TestAirfieldAlertLaunchSplitByTargetType 战斗机 + 轰炸机混编机场：
// 敌机与敌舰同时出现时，战斗机拦截敌机、轰炸机打击敌舰，一波同时升空。
func TestAirfieldAlertLaunchSplitByTargetType(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	registerAlertTestBomber(t)
	const fighterName = "test-alert-fighter"
	oldTemplate, hadTemplate := objUnit.PlaneMap[fighterName]
	objUnit.PlaneMap[fighterName] = &objUnit.Plane{
		Name:   fighterName,
		Type:   objUnit.PlaneTypeFighter,
		Weapon: objUnit.PlaneWeapon{Guns: []*objUnit.Gun{{}}},
	}
	t.Cleanup(func() {
		if hadTemplate {
			objUnit.PlaneMap[fighterName] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, fighterName)
		}
	})

	af := objBuilding.NewAirfield(
		objPos.New(50, 50), 0, 6, 0.8, faction.HumanAlpha, 0,
		[]objUnit.PlaneGroup{
			{Name: fighterName, MaxCount: 2},
			{Name: "test-alert-bomber", MaxCount: 2},
		},
	)
	af.StockPlane(fighterName)
	af.StockPlane("test-alert-bomber")
	ms := &state.MissionState{
		Arena: state.MissionArenaState{
			Airfields: map[string]*objBuilding.Airfield{af.Uid: af},
			Planes:    map[string]*objUnit.Plane{},
			Ships:     map[string]*objUnit.BattleShip{},
		},
	}
	manager := &MissionManager{state: ms, instructionSet: NewInstructionSet()}

	enemyShip := &objUnit.BattleShip{
		Uid:          "enemy-ship",
		CurHP:        100,
		CurPos:       objPos.New(90, 90),
		BelongPlayer: faction.ComputerAlpha,
	}
	manager.state.Arena.Ships[enemyShip.Uid] = enemyShip
	manager.state.Arena.Planes["air-enemy"] = &objUnit.Plane{
		Uid:          "air-enemy",
		CurHP:        10,
		CurPos:       objPos.New(52, 52),
		FlightPhase:  objUnit.PlaneFlightPhaseCruising,
		BelongPlayer: faction.ComputerAlpha,
	}
	manager.updateAirfieldAlertLaunch()

	fighters, bombers := 0, 0
	for _, plane := range manager.state.Arena.Planes {
		switch plane.Name {
		case fighterName:
			fighters++
		case "test-alert-bomber":
			bombers++
		}
	}
	if fighters != 1 || bombers != 1 {
		t.Fatalf("expected 1 fighter + 1 bomber launched, got %d fighters + %d bombers", fighters, bombers)
	}
	if af.Aircraft.Groups[0].CurCount != 0 || af.Aircraft.Groups[1].CurCount != 0 {
		t.Fatalf("both groups should be airborne, stock = %d/%d",
			af.Aircraft.Groups[0].CurCount, af.Aircraft.Groups[1].CurCount)
	}
	// 指令按目标类型分流：战斗机攻击空中敌机，轰炸机攻击敌舰
	var attacks []string
	for _, item := range manager.instructionSet.Items() {
		if strings.Contains(item.Uid(), instr.NamePlaneAttack) {
			attacks = append(attacks, item.String())
		}
	}
	hasTarget := func(uid string) bool {
		for _, text := range attacks {
			if strings.Contains(text, uid) {
				return true
			}
		}
		return false
	}
	if !hasTarget("air-enemy") || !hasTarget("enemy-ship") {
		t.Fatalf("attack instructions should target both air enemy and ship, got %v", attacks)
	}
}
