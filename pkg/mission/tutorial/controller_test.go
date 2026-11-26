package tutorial

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
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
	controller.Observe(ms)
	require.Equal(t, 2, controller.stepIndex)

	ship.CurPos = objPos.NewR(24, 36)
	controller.Observe(ms)
	require.Equal(t, 3, controller.stepIndex)

	ms.UI.GameOpts.Zoom = 8
	controller.Observe(ms)
	require.Equal(t, 4, controller.stepIndex)

	ms.Core.MissionStatus = state.MissionInMap
	controller.Observe(ms)
	require.Equal(t, 5, controller.stepIndex)

	ms.Core.MissionStatus = state.MissionRunning
	controller.Observe(ms)
	require.Equal(t, 6, controller.stepIndex)

	ms.Core.MissionStatus = state.MissionPaused
	controller.Observe(ms)
	require.Equal(t, 7, controller.stepIndex)

	ms.Core.MissionStatus = state.MissionRunning
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
// 失败意味着按钮可能点不到或被裁切。
// 覆盖：表驱动两种分辨率，比较矩形边界与 Next/Skip 的水平关系。
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
			ui := CalcLayout(size.width, size.height)
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
// 场景：configs/missions.json5 与 configs/tutorials.json5 已加载，检查五个教学章节。
// 预期：五个任务都有脚本，步骤数依次为 8/6/5/4/5；失败意味着脚本漏配或任务改名。
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
	require.Equal(t, 4, StepCount("TutorialAviation"))
	require.NotNil(t, aviation.Steps[2].Target)
	require.Equal(t, 19, aviation.Steps[2].Target.MX)

	reinforce, ok := ScriptForMission("TutorialReinforce")
	require.True(t, ok)
	require.Equal(t, "reinforce", reinforce.ID)
	require.Equal(t, 5, StepCount("TutorialReinforce"))

	require.Equal(t, 0, StepCount("PearlHarbor1941"))
}
