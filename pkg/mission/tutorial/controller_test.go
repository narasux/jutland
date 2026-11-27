package tutorial

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

func newControllerTestState() *state.MissionState {
	ms := &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		Player: state.MissionPlayerState{
			CurPlayer: faction.HumanAlpha,
		},
		Arena: state.MissionArenaState{Ships: map[string]*objUnit.BattleShip{}},
		UI:    state.MissionUIState{GameOpts: state.GameOptions{Zoom: state.DefaultZoom()}},
	}
	ms.View.Layout.Width = 1280
	ms.View.Layout.Height = 720
	return ms
}

func addControllerTestShip(ms *state.MissionState, uid string, pos objPos.MapPos) *objUnit.BattleShip {
	ship := &objUnit.BattleShip{
		Uid:          uid,
		Name:         uid,
		Type:         objUnit.ShipTypeDestroyer,
		CurHP:        100,
		CurPos:       pos,
		BelongPlayer: faction.HumanAlpha,
		GroupID:      object.GroupIDNone,
	}
	ms.Arena.Ships[uid] = ship
	return ship
}

// skipStepDwell 把步骤开始时间回拨，绕过最短展示时间，便于断言状态推进。
func skipStepDwell(c *Controller) {
	c.stepStartedAt = time.Now().Add(-tutorialMinStepTime)
}

// TestControllerWalksBasicControls 验证基础操作章节按选中/移动/缩放/地图/暂停顺序推进。
// 场景：1280x720 的测试状态，先跳过欢迎信息卡，依次构造选中舰、位移、缩放、
// 打开地图、返回、暂停、恢复，逐步调用 Observe。
// 预期：stepIndex 从 1 依次推进到 8，最后 Finished 为 true；失败意味着某个完成条件没被状态变化触发，
// 或步骤顺序与 tutorials.json5 不一致。
// 覆盖：真实 TutorialBasicControls 脚本 + 最小 BattleShip/MissionState，断言每一步的索引与最终状态。
func TestControllerWalksBasicControls(t *testing.T) {
	ms := newControllerTestState()
	ship := addControllerTestShip(ms, "HA/DD-1", objPos.New(20, 36))

	controller, err := New("TutorialBasicControls", ms)
	require.NoError(t, err)
	require.Equal(t, 0, controller.stepIndex)

	// 跳过欢迎信息卡，进入“选中舰队”。
	controller.advance(ms)
	require.Equal(t, 1, controller.stepIndex)

	ms.Interaction.SelectedShips = []string{ship.Uid}
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 2, controller.stepIndex)

	ship.CurPos = objPos.NewR(24, 36)
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 3, controller.stepIndex)

	ms.UI.GameOpts.Zoom = 8
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 4, controller.stepIndex)

	ms.Core.MissionStatus = state.MissionInMap
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 5, controller.stepIndex)

	ms.Core.MissionStatus = state.MissionRunning
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 6, controller.stepIndex)

	ms.Core.MissionStatus = state.MissionPaused
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 7, controller.stepIndex)

	ms.Core.MissionStatus = state.MissionRunning
	skipStepDwell(controller)
	controller.Observe(ms)
	require.True(t, controller.Finished())
	require.True(t, controller.View().Finished)
}

// TestControllerAbortAndExit 验证主动退出会标记 ExitAborted 且视图隐藏。
// 场景：教学刚进入第一步（欢迎信息卡）就调用 Abort，模拟玩家按卡片退出。
// 预期：ExitRequested 为 true、ExitKind 为 ExitAborted、View.Visible 为 false；
// 失败意味着退出后仍会继续绘制教学卡或无法离开教学。
// 覆盖：New + Abort + ExitRequested/ExitKind/View 四个断言。
func TestControllerAbortAndExit(t *testing.T) {
	ms := newControllerTestState()
	controller, err := New("TutorialBasicControls", ms)
	require.NoError(t, err)

	controller.Abort()

	require.True(t, controller.ExitRequested())
	require.Equal(t, ExitAborted, controller.ExitKind())
	require.False(t, controller.View().Visible)
}

// TestCalcLayoutKeepsButtonsInsideCard 验证步骤卡与按钮始终落在屏幕内且互不重叠。
// 场景：1280x720 与 1920x1080 两种基准分辨率，教学卡需要避开右上侧栏和屏幕边缘。
// 预期：卡片四边在屏幕内；Next/Skip/Exit 都在卡片内；Skip 在 Next 左侧且不与其重叠；
// 收起把手在屏幕上且不与 Exit 重叠；失败意味着按钮可能点不到、被裁切或互相抢点击。
// 覆盖：表驱动两种分辨率，比较矩形边界、Next/Skip 水平关系，以及 Toggle 与 Exit 的相交情况。
func TestCalcLayoutKeepsButtonsInsideCard(t *testing.T) {
	for _, size := range []struct {
		name   string
		width  float64
		height float64
	}{
		{name: "1280x720", width: 1280, height: 720},
		{name: "1920x1080", width: 1920, height: 1080},
	} {
		t.Run(size.name, func(t *testing.T) {
			ui := CalcLayout(size.width, size.height, false)
			require.GreaterOrEqual(t, ui.Card.X, 0.0)
			require.GreaterOrEqual(t, ui.Card.Y, 0.0)
			require.LessOrEqual(t, ui.Card.X+ui.Card.W, size.width)
			require.LessOrEqual(t, ui.Card.Y+ui.Card.H, size.height)
			for _, rect := range []Rect{ui.Next, ui.Skip, ui.Exit} {
				require.GreaterOrEqual(t, rect.X, ui.Card.X)
				require.GreaterOrEqual(t, rect.Y, ui.Card.Y)
				require.LessOrEqual(t, rect.X+rect.W, ui.Card.X+ui.Card.W)
				require.LessOrEqual(t, rect.Y+rect.H, ui.Card.Y+ui.Card.H)
			}
			require.LessOrEqual(t, ui.Skip.X+ui.Skip.W, ui.Next.X)
			require.GreaterOrEqual(t, ui.Toggle.X, ui.Card.X)
			require.GreaterOrEqual(t, ui.Toggle.Y, 0.0)
			require.LessOrEqual(t, ui.Toggle.X+ui.Toggle.W, ui.Card.X+ui.Card.W)
			require.False(t, rectsOverlap(ui.Toggle, ui.Exit), "收起把手不能压住退出按钮")
		})
	}
}

// rectsOverlap 判断两个矩形是否有面积交叠（仅边界相接不算）。
func rectsOverlap(a, b Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

// TestCalcLayoutCollapsedBarSticksToBottom 验证收起态只剩贴底窄条，把手落在条内。
// 场景：1280x720 与 1920x1080 下分别计算收起布局。
// 预期：窄条底边贴住屏幕底边、水平居中、把手完全在窄条内；失败意味着收起后仍会挡面板，
// 或者玩家点不到展开把手。
// 覆盖：表驱动两种分辨率，断言 Card 的 Y+H 等于屏幕高、左右留白相等、Toggle 在 Card 内。
func TestCalcLayoutCollapsedBarSticksToBottom(t *testing.T) {
	for _, size := range []struct {
		name   string
		width  float64
		height float64
	}{
		{name: "1280x720", width: 1280, height: 720},
		{name: "1920x1080", width: 1920, height: 1080},
	} {
		t.Run(size.name, func(t *testing.T) {
			ui := CalcLayout(size.width, size.height, true)
			require.True(t, ui.Collapsed)
			require.InDelta(t, size.height, ui.Card.Y+ui.Card.H, 1e-6)
			require.InDelta(t, ui.Card.X, size.width-(ui.Card.X+ui.Card.W), 1e-6)
			require.GreaterOrEqual(t, ui.Toggle.X, ui.Card.X)
			require.LessOrEqual(t, ui.Toggle.X+ui.Toggle.W, ui.Card.X+ui.Card.W)
			require.GreaterOrEqual(t, ui.Toggle.Y, ui.Card.Y)
			require.LessOrEqual(t, ui.Toggle.Y+ui.Toggle.H, ui.Card.Y+ui.Card.H)
		})
	}
}

// TestControllerStuckHintAppearsForSlowSteps 验证操作步骤停太久会显示额外提示。
// 场景：进入基础操作第二章“选中舰队”，把步骤开始时间手动回拨 13 秒（阈值 12 秒）。
// 预期：View.StuckHint 非空；失败意味着玩家卡住时只会一直看到原提示。
// 覆盖：New + advance 到 guided 步骤，修改 stepStartedAt 后调用 View 检查 StuckHint。
func TestControllerStuckHintAppearsForSlowSteps(t *testing.T) {
	ms := newControllerTestState()
	controller, err := New("TutorialBasicControls", ms)
	require.NoError(t, err)
	controller.advance(ms)
	controller.stepStartedAt = time.Now().Add(-13 * time.Second)

	require.NotEmpty(t, controller.View().StuckHint)
}

// TestScriptLookup 验证教学任务能查到脚本且步骤数固定。
// 场景：configs/missions.json5 与 configs/tutorials.json5 已加载，检查六个教学章节。
// 预期：六个任务都有脚本，步骤数依次为 8/6/5/6/5/5；失败意味着脚本漏配或任务改名。
// 覆盖：ScriptForMission 五次、StepCount 五次、脚本 ID 与基础/航空目标点坐标断言。
func TestScriptLookup(t *testing.T) {
	basic, ok := ScriptForMission("TutorialBasicControls")
	require.True(t, ok)
	require.Equal(t, "basic-controls", basic.ID)
	require.Len(t, basic.Steps, 8)
	require.Equal(t, 8, StepCount("TutorialBasicControls"))
	require.NotNil(t, basic.Steps[2].Target)
	require.Equal(t, 18, basic.Steps[2].Target.MX)
	require.Equal(t, 30, basic.Steps[2].Target.MY)

	formation, ok := ScriptForMission("TutorialFormation")
	require.True(t, ok)
	require.Equal(t, "formation", formation.ID)
	require.Len(t, formation.Steps, 6)
	require.Equal(t, 6, StepCount("TutorialFormation"))

	submarine, ok := ScriptForMission("TutorialSubmarine")
	require.True(t, ok)
	require.Equal(t, "submarine", submarine.ID)
	require.Equal(t, 5, StepCount("TutorialSubmarine"))

	aviation, ok := ScriptForMission("TutorialAviation")
	require.True(t, ok)
	require.Equal(t, "aviation", aviation.ID)
	require.Equal(t, 6, StepCount("TutorialAviation"))
	require.NotNil(t, aviation.Steps[2].Target)
	require.Equal(t, 19, aviation.Steps[2].Target.MX)
	require.NotNil(t, aviation.Steps[3].Target)
	require.Equal(t, 21, aviation.Steps[3].Target.MX)
	require.Equal(t, 34, aviation.Steps[3].Target.MY)
	require.NotNil(t, aviation.Steps[4].Target)
	require.Equal(t, 70, aviation.Steps[4].Target.MX)
	require.Equal(t, 22, aviation.Steps[4].Target.MY)

	reinforce, ok := ScriptForMission("TutorialReinforce")
	require.True(t, ok)
	require.Equal(t, "reinforce", reinforce.ID)
	require.Equal(t, 5, StepCount("TutorialReinforce"))

	combat, ok := ScriptForMission("TutorialCombat")
	require.True(t, ok)
	require.Equal(t, "combat", combat.ID)
	require.Equal(t, 5, StepCount("TutorialCombat"))

	require.Equal(t, 0, StepCount("PearlHarbor1941"))
}

// TestSonarStepRequiresHoldingV 验证声呐步骤要连续按住 V 一段时间才推进。
// 场景：潜艇章节直接进入最后的「看声呐范围」步骤，第一帧 ShowSonarRanges=true 表示玩家刚按下 V；
// 中途松开一次，再重新按住并把计时回拨到超过 tutorialSonarHoldTime。
// 预期：刚按下不完成（圈还在画），松开清零，按住够时长后才 Finished；失败意味着声呐范围只显示一帧，
// 玩家还没看清覆盖圈就被翻到完成卡。
// 覆盖：enterStep 直接进入第 5 步，分「刚按下 / 松开 / 按住够久」三次 Observe，断言步数与 finished。
func TestSonarStepRequiresHoldingV(t *testing.T) {
	ms := newControllerTestState()
	controller, err := New("TutorialSubmarine", ms)
	require.NoError(t, err)

	const sonarStep = 4
	controller.enterStep(sonarStep, ms)

	ms.UI.ShowSonarRanges = true
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, sonarStep, controller.StepIndex())
	require.False(t, controller.Finished(), "刚按下 V 不应立即完成")

	ms.UI.ShowSonarRanges = false
	skipStepDwell(controller)
	controller.Observe(ms)
	require.True(t, controller.sonarHeldSince.IsZero(), "松开 V 后按住计时应清零")
	require.False(t, controller.Finished())

	ms.UI.ShowSonarRanges = true
	skipStepDwell(controller)
	controller.Observe(ms)
	controller.sonarHeldSince = time.Now().Add(-tutorialSonarHoldTime)
	skipStepDwell(controller)
	controller.Observe(ms)
	require.True(t, controller.Finished())
}

// TestScoutStepWaitsForTakeoff 验证侦察步骤要等飞机离舰后才推进。
// 场景：航空章节进入「派侦察机」步骤（进入时没有手动侦察机），随后出现一架仍在甲板滑跑、
// Flight.Phase=taking_off 的手动侦察机，再调用 FinishTakeoff 让它转入巡航。
// 预期：滑跑阶段不推进，转入巡航后才进入下一步；失败意味着 Shift+右键那一帧就翻页，
// 玩家看不到侦察机从甲板起飞。
// 覆盖：enterStep 进入第 3 步，构造带 ScoutManual 的 Plane 并手工切换 Flight.Phase，
// 两次 Observe 断言 stepIndex 停在 2 再推进到 3。
func TestScoutStepWaitsForTakeoff(t *testing.T) {
	ms := newControllerTestState()
	controller, err := New("TutorialAviation", ms)
	require.NoError(t, err)
	controller.enterStep(2, ms)

	plane := &objUnit.Plane{
		Uid: "scout", Type: objUnit.PlaneTypeScout, CurHP: 10, TotalHP: 10,
		ScoutManual: true, BelongPlayer: faction.HumanAlpha,
	}
	plane.Flight.Phase = objUnit.PlaneFlightPhaseTakingOff
	ms.Arena.PutPlane(plane)

	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 2, controller.StepIndex(), "飞机还在甲板滑跑时不应推进")

	plane.FinishTakeoff()
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 3, controller.StepIndex())
}

// TestAirfieldStepCompletesOnProductionSwitch 验证机场步骤只在玩家切换生产机型后推进。
// 场景：航空章节进入「陆地机场与生产」步骤，场上有一架 P1 机场，初始生产机型 F4F-3；
// 先保持机型不变调用 Observe，再把 CurProducing 改成 SBD-3 调用 Observe。
// 预期：未切换时停在原步骤，切换后才进入下一步；失败意味着卡片承诺的「选择生产机型」没有被识别。
// 覆盖：enterStep 进入第 5 步，构造 Airfield（Uid/BelongPlayer/CurProducing）放入 Arena，
// 两次 Observe 断言 stepIndex 从 4 推进到 5。
func TestAirfieldStepCompletesOnProductionSwitch(t *testing.T) {
	ms := newControllerTestState()
	controller, err := New("TutorialAviation", ms)
	require.NoError(t, err)

	airfield := &objBuilding.Airfield{
		Uid: "af-1", Pos: objPos.New(70, 22), BelongPlayer: faction.HumanAlpha, CurProducing: "F4F-3",
	}
	ms.Arena.Airfields = map[string]*objBuilding.Airfield{airfield.Uid: airfield}
	ms.View.Camera.Width, ms.View.Camera.Height = 15, 10
	ms.Core.MissionMD.MapCfg = &mapcfg.MapCfg{Width: 128, Height: 128}
	controller.enterStep(4, ms)
	// setup focus-airfield 应把镜头居中到机场（70,22），玩家一进这一步就能看到跑道。
	require.InDelta(t, 62.5, ms.View.Camera.Pos.RX, 1e-6)
	require.InDelta(t, 17, ms.View.Camera.Pos.RY, 1e-6)

	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 4, controller.StepIndex(), "未切换生产机型时不应推进")

	airfield.CurProducing = "SBD-3"
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 5, controller.StepIndex())
}

// TestCombatEnemyHitCompletionFiltersWeaponAndActor 验证敌舰命中按武器与攻击者筛选，且不认步骤前的旧记录。
// 场景：战斗章节进入「对舰鱼雷」步骤（基线 Tick=200），先写入一条 200 帧的旧炮弹命中，
// 再写入 201 帧的飞机鱼雷命中，最后写入 202 帧的舰船鱼雷命中。
// 预期：旧记录与飞机鱼雷都不推进（actor=ship），只有舰船鱼雷命中后步骤前进；失败意味着
// 教学会被历史命中或舰载机鱼雷提前完成。
// 覆盖：enterStep 进入第 2 步，逐条追加 Stats.BattleLog 后 Observe，断言 stepIndex 1 → 2。
func TestCombatEnemyHitCompletionFiltersWeaponAndActor(t *testing.T) {
	ms := newControllerTestState()
	controller, err := New("TutorialCombat", ms)
	require.NoError(t, err)
	ms.Core.SimTick = 200
	controller.enterStep(1, ms)

	// 步骤开始前（同一帧）的炮弹命中：不能算本步骤的战果。
	ms.Stats.BattleLog = append(ms.Stats.BattleLog, state.BattleLogEntry{
		Tick: 200, ActorPlayer: faction.HumanAlpha, TargetPlayer: faction.ComputerAlpha,
		Action: state.BattleLogShipHit, Hit: objUnit.HitInfo{WeaponType: "shell"},
	})
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 1, controller.StepIndex())

	// 飞机投下的鱼雷命中：actor 不是舰船，不满足「对舰鱼雷」步骤。
	ms.Stats.BattleLog = append(ms.Stats.BattleLog, state.BattleLogEntry{
		Tick: 201, ActorPlayer: faction.HumanAlpha, ActorIsPlane: true,
		TargetPlayer: faction.ComputerAlpha, Action: state.BattleLogShipHit,
		Hit: objUnit.HitInfo{WeaponType: "torpedo"},
	})
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 1, controller.StepIndex())

	ms.Stats.BattleLog = append(ms.Stats.BattleLog, state.BattleLogEntry{
		Tick: 202, ActorPlayer: faction.HumanAlpha, TargetPlayer: faction.ComputerAlpha,
		Action: state.BattleLogShipHit, Hit: objUnit.HitInfo{WeaponType: "torpedo"},
	})
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 2, controller.StepIndex())
}

// TestCombatPlaneShotDownCompletion 验证对空步骤只认我方击落敌机的战况。
// 场景：进入「对空拦截」步骤（基线 Tick=400），依次写入敌机坠毁、我方飞机被击落、
// 我方舰船防空击落敌机三条记录。
// 预期：前两条不推进，第三条才完成；失败意味着一架坠毁敌机或我方损失也能让教学过关。
// 覆盖：enterStep 进入第 4 步，逐条追加 BattleLog 后 Observe，断言 stepIndex 3 → 4。
func TestCombatPlaneShotDownCompletion(t *testing.T) {
	ms := newControllerTestState()
	controller, err := New("TutorialCombat", ms)
	require.NoError(t, err)
	ms.Core.SimTick = 400
	controller.enterStep(4, ms)

	// 敌机自行坠毁：Action 不是 shotDown，且没有攻击者。
	ms.Stats.BattleLog = append(ms.Stats.BattleLog, state.BattleLogEntry{
		Tick: 401, TargetPlayer: faction.ComputerAlpha, TargetIsPlane: true,
		Action: state.BattleLogPlaneLost,
	})
	// 我方飞机被敌机击落：我方是受害者，不算「击落敌机」。
	ms.Stats.BattleLog = append(ms.Stats.BattleLog, state.BattleLogEntry{
		Tick: 402, ActorPlayer: faction.ComputerAlpha, TargetPlayer: faction.HumanAlpha,
		TargetIsPlane: true, Action: state.BattleLogPlaneShotDown,
	})
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 4, controller.StepIndex())

	// 我方舰船防空击落敌机。
	ms.Stats.BattleLog = append(ms.Stats.BattleLog, state.BattleLogEntry{
		Tick: 403, ActorPlayer: faction.HumanAlpha, TargetPlayer: faction.ComputerAlpha,
		TargetIsPlane: true, Action: state.BattleLogPlaneShotDown,
	})
	skipStepDwell(controller)
	controller.Observe(ms)
	require.True(t, controller.Finished())
}

// TestCombatDisarmSetupSilencesEnemy 验证战斗教学的 welcome 步骤会关闭敌方武器。
// 场景：场上有一艘 P1 舰与一艘 P2 靶舰，进入带 disarm-enemy 的欢迎步骤。
// 预期：P2 舰的主炮 / 副炮 / 防空 / 鱼雷 / 火箭全部置为禁用，P1 舰不受影响；
// 失败意味着靶舰会还击，教学可能被打断。
// 覆盖：enterStep(0) 后断言双方 Weapon 的 Disabled 标志。
func TestCombatDisarmSetupSilencesEnemy(t *testing.T) {
	ms := newControllerTestState()
	enemy := addControllerTestShip(ms, "CA/enemy", objPos.New(30, 30))
	enemy.BelongPlayer = faction.ComputerAlpha
	ally := addControllerTestShip(ms, "HA/ally", objPos.New(20, 30))

	controller, err := New("TutorialCombat", ms)
	require.NoError(t, err)
	controller.enterStep(0, ms)

	for _, disabled := range []bool{
		enemy.Weapon.MainGunDisabled, enemy.Weapon.SecondaryGunDisabled,
		enemy.Weapon.AntiAircraftGunDisabled, enemy.Weapon.TorpedoDisabled, enemy.Weapon.RocketDisabled,
	} {
		require.True(t, disabled, "敌方靶舰的所有武器都应被关闭")
	}
	if ally.Weapon.MainGunDisabled || ally.Weapon.TorpedoDisabled {
		t.Fatal("己方舰船不应被 disarm-enemy 影响")
	}
}

// TestCombatAirStrikeSetupAndCompletion 验证航空攻击步骤会替航母指定目标，并在舰载机受命后完成。
// 场景：战斗章节进入第 4 步「航空攻击」，场上有一艘己方航母和一艘最近敌舰；
// setup order-air-strike 应把航母 AttackTarget 指向该敌舰，随后放入一架已锁定该敌舰的舰载机。
// 预期：航母已拿到攻击目标，舰载机受命后步骤推进到下一步；失败意味着航母不会自动派机，
// 或空袭判定不认 CurAttackTarget。
// 覆盖：enterStep(3) 触发 setup 并断言 AttackTarget；构造带 CurAttackTarget 的 Plane 后 Observe
// 断言 stepIndex 3 → 4。
func TestCombatAirStrikeSetupAndCompletion(t *testing.T) {
	ms := newControllerTestState()
	enemy := addControllerTestShip(ms, "CA/enemy", objPos.New(30, 30))
	enemy.BelongPlayer = faction.ComputerAlpha
	carrier := &objUnit.BattleShip{
		Uid: "HA/CV-1", Name: "essex", Type: objUnit.ShipTypeAircraftCarrier,
		CurHP: 100, CurPos: objPos.New(20, 30), BelongPlayer: faction.HumanAlpha,
		Aircraft: objUnit.ShipAircraft{HasPlane: true},
	}
	ms.Arena.Ships[carrier.Uid] = carrier

	controller, err := New("TutorialCombat", ms)
	require.NoError(t, err)
	controller.enterStep(3, ms)
	require.Equal(t, enemy.Uid, carrier.AttackTarget, "setup 应给航母指定最近的敌舰")

	ms.Arena.PutPlane(&objUnit.Plane{
		Uid: "HA/TBF-1", Type: objUnit.PlaneTypeTorpedoBomber, CurHP: 10,
		BelongPlayer: faction.HumanAlpha, CurAttackTarget: enemy.Uid,
	})
	skipStepDwell(controller)
	controller.Observe(ms)
	require.Equal(t, 4, controller.StepIndex())
}
