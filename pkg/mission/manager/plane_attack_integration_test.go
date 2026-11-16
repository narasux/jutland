package manager

import (
	"testing"

	audioPlayer "github.com/narasux/jutland/pkg/audio/player"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/object"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// TestBomberReleasesBeforeReturning 验证警戒起飞的轰炸机必须先投弹再进入返航/着舰流程。
// 场景：机场在 (20,150) 只库存 1 架，敌巡洋舰在同一纬度 100 或 115 处（后者往返 190，
// 用于覆盖接近作战半径的长距离出击），地图 300x300；覆盖 B-17G、B6N2、B6N2 远距、D4Y3 四个子用例。
// 预期：起飞后 Arena.Planes 恰好 1 架且带有 PlaneAttack 指令、初始没有任何 Released 弹药；
// 随后 5000 帧内至少 1 件炸弹/鱼雷/火箭被释放，若先进入 PlaneFlightPhaseLandingStaging 判失败，
// 超时未释放也判失败。失败意味着飞机空手返航或被提前标记已投弹。
// 覆盖：refreshAlertTargetPlan + updateAirfieldAlertLaunch 完成同步起飞，逐帧执行指令、
// updatePlaneAttackOrReturn 与 updatePlaneWeaponFire，并检查 Bombs/Torpedoes/Rockets 的释放状态。
func TestBomberReleasesBeforeReturning(t *testing.T) {
	testCases := []struct {
		name      string
		planeName string
		enemyX    float64
	}{
		{name: "B-17G", planeName: "B-17G", enemyX: 100},
		{name: "B6N2", planeName: "B6N2", enemyX: 100},
		// 往返 190 仍在 B6N2 的 218 航程内，覆盖作战半径附近的长距离出击
		{name: "B6N2-long-range", planeName: "B6N2", enemyX: 115},
		{name: "D4Y3", planeName: "D4Y3", enemyX: 100},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testBomberReleasesBeforeReturning(t, testCase.planeName, testCase.enemyX)
		})
	}
}

func testBomberReleasesBeforeReturning(t *testing.T, planeName string, enemyX float64) {
	t.Helper()
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	airfield := objBuilding.NewAirfield(
		objPos.New(20, 150), 0, 6, 0.8, faction.HumanAlpha, 0,
		[]objUnit.PlaneGroup{{Name: planeName, MaxCount: 1}},
	)
	airfield.StockPlane(planeName)
	enemy := &objUnit.BattleShip{
		Uid:          "enemy",
		Type:         objUnit.ShipTypeCruiser,
		CurHP:        1000,
		TotalHP:      1000,
		CurPos:       objPos.NewR(enemyX, 150),
		BelongPlayer: faction.ComputerAlpha,
	}
	missionState := &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{
				MapCfg: &mapcfg.MapCfg{Width: 300, Height: 300},
			},
		},
		Arena: state.MissionArenaState{
			Airfields: map[string]*objBuilding.Airfield{airfield.Uid: airfield},
			Planes:    map[string]*objUnit.Plane{},
			Ships:     map[string]*objUnit.BattleShip{enemy.Uid: enemy},
		},
	}
	manager := &MissionManager{
		state:            missionState,
		instructionSet:   NewInstructionSet(),
		weaponFirePlayer: audioPlayer.NewWeaponFire(),
		targetingDirty:   true,
		targetingCursors: map[string]map[object.Type]int{},
	}
	refreshAlertTargetPlan(manager)
	manager.updateAirfieldAlertLaunch()
	if len(missionState.Arena.Planes) != 1 {
		t.Fatalf("bomber count = %d, want 1", len(missionState.Arena.Planes))
	}

	var bomber *objUnit.Plane
	for _, plane := range missionState.Arena.Planes {
		bomber = plane
	}
	if attack := manager.instructionSet.Items()[instr.GenInstrUid(instr.NamePlaneAttack, bomber.Uid)]; attack == nil {
		t.Fatal("bomber launched without an attack instruction")
	}
	for _, bomb := range bomber.Weapon.Bombs {
		if bomb.Released {
			t.Fatal("bomber inherited a released bomb before its first attack frame")
		}
	}

	released := false
	for frame := 0; frame < 5000; frame++ {
		manager.instructionSet.ExecAll(missionState)
		manager.updatePlaneAttackOrReturn()
		manager.updatePlaneWeaponFire()
		for _, bomb := range bomber.Weapon.Bombs {
			released = released || bomb.Released
		}
		for _, torpedo := range bomber.Weapon.Torpedoes {
			released = released || torpedo.Released
		}
		for _, rocket := range bomber.Weapon.Rockets {
			released = released || rocket.Exhausted()
		}
		if released {
			break
		}
		if bomber.FlightPhase == objUnit.PlaneFlightPhaseLandingStaging {
			t.Fatalf(
				"bomber returned before releasing: frame=%d pos=%s hp=%.1f remainRange=%.1f",
				frame,
				bomber.CurPos.String(),
				bomber.CurHP,
				bomber.RemainRange,
			)
		}
	}
	if !released {
		t.Fatalf(
			"bomber did not release within test window: pos=%s target=%s",
			bomber.CurPos.String(),
			bomber.CurAttackTarget,
		)
	}
}

// TestAirfieldDoesNotLaunchBomberBeyondCombatRadius 验证机场不会为单程可达但超出作战半径的目标放飞轰炸机。
// 场景：机场 (20,150) 库存 1 架 B6N2（航程 436 格），敌巡洋舰在 (250,150)，距离 230 格、往返 460 格。
// 预期：refreshAlertTargetPlan + updateAirfieldAlertLaunch 后 Arena.Planes 仍为 0，机场保持待命。
// 覆盖：真实 B6N2 模板与 300x300 地图，走 takeOffFromBase 的 CanTakeOffWithinRange(roundTrip) 过滤后断言起飞数。
func TestAirfieldDoesNotLaunchBomberBeyondCombatRadius(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	airfield := objBuilding.NewAirfield(
		objPos.New(20, 150), 0, 6, 0.8, faction.HumanAlpha, 0,
		[]objUnit.PlaneGroup{{Name: "B6N2", MaxCount: 1}},
	)
	airfield.StockPlane("B6N2")
	enemy := &objUnit.BattleShip{
		Uid:          "far-enemy",
		Type:         objUnit.ShipTypeCruiser,
		CurHP:        1000,
		TotalHP:      1000,
		CurPos:       objPos.New(250, 150),
		BelongPlayer: faction.ComputerAlpha,
	}
	missionState := &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{
				MapCfg: &mapcfg.MapCfg{Width: 300, Height: 300},
			},
		},
		Arena: state.MissionArenaState{
			Airfields: map[string]*objBuilding.Airfield{airfield.Uid: airfield},
			Planes:    map[string]*objUnit.Plane{},
			Ships:     map[string]*objUnit.BattleShip{enemy.Uid: enemy},
		},
	}
	manager := &MissionManager{
		state:            missionState,
		instructionSet:   NewInstructionSet(),
		weaponFirePlayer: audioPlayer.NewWeaponFire(),
		targetingDirty:   true,
		targetingCursors: map[string]map[object.Type]int{},
	}
	refreshAlertTargetPlan(manager)
	manager.updateAirfieldAlertLaunch()
	if len(missionState.Arena.Planes) != 0 {
		t.Fatalf(
			"airfield launched %d bomber(s) against a target beyond its combat radius",
			len(missionState.Arena.Planes),
		)
	}
}

// TestPearlHarborAirfieldHasReachableStrikeTargets 验证珍珠港 1941 中每个机场的对舰编组都有可达目标。
// 场景：用 New("PearlHarbor1941", faction.SideP1) 载入真实任务配置，refreshAlertTargetPlan 同步构建
// 目标计划，然后遍历所有机场的编组，只看 TargetType 为 TypeShip 且 CurCount>0 的打击编组。
// 预期：targetingPlan.BaseQueues[机场UID][TypeShip] 非空；为空说明该编组航程覆盖不到任何敌舰，
// 警戒起飞会一直待命，任务开局就有一批飞机永远不出动。
// 覆盖：真实配置数据校验，逐个编组用队列长度断言，不构造任何测试专用模板。
func TestPearlHarborAirfieldHasReachableStrikeTargets(t *testing.T) {
	manager := New("PearlHarbor1941", faction.SideP1)
	refreshAlertTargetPlan(manager)

	for _, airfield := range manager.state.Arena.Airfields {
		for _, group := range airfield.Aircraft.Groups {
			if group.TargetType != object.TypeShip || group.CurCount <= 0 {
				continue
			}
			queue := manager.targetingPlan.BaseQueues[airfield.Uid][object.TypeShip]
			if len(queue) == 0 {
				t.Fatalf("Pearl Harbor airfield has no reachable ship target for %s", group.Name)
			}
		}
	}
}

// TestPearlHarborEveryStrikeBaseHasReachableTargets 验证珍珠港 1941 中每艘载机舰的打击编组同样有可达目标。
// 场景：New("PearlHarbor1941", faction.SideP1) 建好任务后刷新目标计划，遍历 Arena.Ships 中
// Aircraft.HasPlane 的舰船，逐个检查 TargetType 为 TypeShip 且 CurCount>0 的编组。
// 预期：targetingPlan.BaseQueues[舰船UID][TypeShip] 非空；为空意味着该载机舰的舰载机
// 起飞后无目标可打，会在甲板上空转，开局航空兵力被浪费。
// 覆盖：真实任务配置下的基地队列非空断言，与机场用例互补，覆盖航母类载机基地。
func TestPearlHarborEveryStrikeBaseHasReachableTargets(t *testing.T) {
	manager := New("PearlHarbor1941", faction.SideP1)
	refreshAlertTargetPlan(manager)

	for uid, ship := range manager.state.Arena.Ships {
		if !ship.Aircraft.HasPlane {
			continue
		}
		for _, group := range ship.Aircraft.Groups {
			if group.TargetType != object.TypeShip || group.CurCount <= 0 {
				continue
			}
			queue := manager.targetingPlan.BaseQueues[uid][object.TypeShip]
			if len(queue) == 0 {
				t.Fatalf("ship %s has no reachable ship target for %s", uid, group.Name)
			}
		}
	}
}
