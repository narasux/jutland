package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/unitpanel"
)

// countingHandler 记录玩家侧处理器被调用的次数，用于验证托管路由。
type countingHandler struct {
	calls int
}

func (h *countingHandler) Handle(
	map[string]instr.Instruction, *state.MissionState,
) map[string]instr.Instruction {
	h.calls++
	return nil
}

// newAutopilotTestMissionState 构造托管测试所需的最小任务状态：双方阵营、空战场。
func newAutopilotTestMissionState() *state.MissionState {
	return &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		Player: state.MissionPlayerState{
			CurPlayer: faction.HumanAlpha,
			CurEnemy:  faction.ComputerAlpha,
		},
		Arena: state.MissionArenaState{
			Ships:  map[string]*objUnit.BattleShip{},
			Planes: map[string]*objUnit.Plane{},
		},
	}
}

// TestPlayerAIControlRoutesPlayerHandler 验证托管开关只切换玩家侧处理器，并在关闭后恢复人类输入。
// 场景：玩家侧先挂一个计数处理器；未托管时 updateInstructions 应调用它；开启托管后应改用
// ComputerDecisionHandler 且不再调用人类处理器；重复开启不重建处理器；关闭后恢复人类处理器。
// 预期：human.calls 依次为 1/1/2；playerAlphaAuto 在开启后非 nil、重复开启保持同一指针、关闭后为 nil。
// 覆盖：NewWithState + 计数处理器，连续四次 updateInstructions，断言调用次数与自动处理器生命周期。
func TestPlayerAIControlRoutesPlayerHandler(t *testing.T) {
	m := NewWithState(newAutopilotTestMissionState())
	humanHandler := &countingHandler{}
	m.playerAlphaHandler = humanHandler

	m.updateInstructions()
	require.Equal(t, 1, humanHandler.calls)
	require.Nil(t, m.playerAlphaAuto)

	m.state.Core.PlayerAIControl = true
	m.updateInstructions()
	require.Equal(t, 1, humanHandler.calls)
	require.NotNil(t, m.playerAlphaAuto)
	autoHandler := m.playerAlphaAuto

	m.updateInstructions()
	require.Equal(t, autoHandler, m.playerAlphaAuto)
	require.Equal(t, 1, humanHandler.calls)

	m.state.Core.PlayerAIControl = false
	m.updateInstructions()
	require.Equal(t, 2, humanHandler.calls)
	require.Nil(t, m.playerAlphaAuto)
}

// TestPreparePlayerFleetForAIRestoresPlayerUnits 验证开启托管时恢复玩家侧武器/飞机/机场并清空手动目标。
// 场景：玩家舰船主炮被禁用、舰载机被禁用、手锁定敌方目标；玩家机场被停用；敌方单位保持同样设置。
// 预期：玩家舰船的 MainGunDisabled/Aircraft.Disable 归 false、AttackTarget 清空；玩家机场 Disabled/Aircraft.Disable
// 归 false；敌方舰船和机场的所有标志保持不变。
// 覆盖：构造双方舰船与机场，调用 syncPlayerAIControl（触发 preparePlayerFleetForAI），逐项断言。
func TestPreparePlayerFleetForAIRestoresPlayerUnits(t *testing.T) {
	ms := &state.MissionState{
		Core: state.MissionCoreState{
			MissionStatus:   state.MissionRunning,
			PlayerAIControl: true,
		},
		Player: state.MissionPlayerState{CurPlayer: faction.HumanAlpha},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				"own": {
					Uid: "own", Name: "own", Type: objUnit.ShipTypeDestroyer,
					BelongPlayer: faction.HumanAlpha, CurHP: 100, TotalHP: 100,
					AttackTarget: "enemy",
					Weapon:       objUnit.ShipWeapon{MainGunDisabled: true},
					Aircraft:     objUnit.ShipAircraft{Disable: true},
				},
				"enemy": {
					Uid: "enemy", Name: "enemy", Type: objUnit.ShipTypeDestroyer,
					BelongPlayer: faction.ComputerAlpha, CurHP: 100, TotalHP: 100,
					AttackTarget: "own",
					Weapon:       objUnit.ShipWeapon{MainGunDisabled: true},
					Aircraft:     objUnit.ShipAircraft{Disable: true},
				},
			},
			Airfields: map[string]*objBuilding.Airfield{
				"own-af": {
					Uid: "own-af", BelongPlayer: faction.HumanAlpha, Disabled: true,
					Aircraft: objUnit.ShipAircraft{Disable: true},
				},
				"enemy-af": {
					Uid: "enemy-af", BelongPlayer: faction.ComputerAlpha, Disabled: true,
					Aircraft: objUnit.ShipAircraft{Disable: true},
				},
			},
			Planes: map[string]*objUnit.Plane{},
		},
	}
	m := &MissionManager{state: ms, instructionSet: NewInstructionSet()}

	m.syncPlayerAIControl()

	require.NotNil(t, m.playerAlphaAuto)
	own := ms.Arena.Ships["own"]
	require.False(t, own.Weapon.MainGunDisabled)
	require.False(t, own.Aircraft.Disable)
	require.Empty(t, own.AttackTarget)
	require.False(t, ms.Arena.Airfields["own-af"].Disabled)
	require.False(t, ms.Arena.Airfields["own-af"].Aircraft.Disable)

	enemy := ms.Arena.Ships["enemy"]
	require.True(t, enemy.Weapon.MainGunDisabled)
	require.True(t, enemy.Aircraft.Disable)
	require.Equal(t, "own", enemy.AttackTarget)
	require.True(t, ms.Arena.Airfields["enemy-af"].Disabled)
	require.True(t, ms.Arena.Airfields["enemy-af"].Aircraft.Disable)
}

// TestShouldAutoScoutFollowsFogAndTakeover 验证自动侦察只对电脑侧和“已托管”的玩家侧开放。
// 场景：迷雾开/关、玩家侧托管开/关、base 属于玩家或电脑；玩家未托管时不应自动派机，
// 托管后应与电脑侧一致；迷雾关闭时双方都不派。
// 预期：开着迷雾时玩家未托管=false、玩家托管=true、电脑=true；关闭迷雾时全部=false。
// 覆盖：表驱动 fog/ai/owner 五组输入，调用 shouldAutoScout 断言布尔结果。
func TestShouldAutoScoutFollowsFogAndTakeover(t *testing.T) {
	cases := []struct {
		name  string
		fog   bool
		ai    bool
		owner faction.Player
		want  bool
	}{
		{name: "player no takeover", fog: true, owner: faction.HumanAlpha, want: false},
		{name: "player takeover", fog: true, ai: true, owner: faction.HumanAlpha, want: true},
		{name: "computer", fog: true, owner: faction.ComputerAlpha, want: true},
		{name: "fog off player takeover", fog: false, ai: true, owner: faction.HumanAlpha, want: false},
		{name: "fog off computer", fog: false, owner: faction.ComputerAlpha, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ms := &state.MissionState{
				Core:   state.MissionCoreState{FogOfWar: tc.fog, PlayerAIControl: tc.ai},
				Player: state.MissionPlayerState{CurPlayer: faction.HumanAlpha},
			}
			require.Equal(t, tc.want, shouldAutoScout(ms, tc.owner))
		})
	}
}

// TestHandleUnitPanelActionsBlocksControlsDuringTakeover 验证托管时只保留单位面板的聚焦/相机动作。
// 场景：托管开启时点击武器开关和焦点舰；关闭托管后再点武器开关。
// 预期：托管时不会产生 EnableWeapon 指令，但 FocusedShipUid 仍会更新；取消托管后同一动作会产生 1 条指令。
// 覆盖：handleUnitPanelActions 的 ActionToggleWeapon/ActionFocusShip 两条分支与托管闸门。
func TestHandleUnitPanelActionsBlocksControlsDuringTakeover(t *testing.T) {
	ms := newAutopilotTestMissionState()
	ms.Core.PlayerAIControl = true
	ms.Arena.Ships["own"] = &objUnit.BattleShip{
		Uid: "own", Name: "own", Type: objUnit.ShipTypeDestroyer,
		BelongPlayer: faction.HumanAlpha, CurHP: 100, TotalHP: 100,
	}
	m := &MissionManager{state: ms, instructionSet: NewInstructionSet()}

	weaponAction := unitpanel.Action{
		Kind:       unitpanel.ActionToggleWeapon,
		ShipUids:   []string{"own"},
		WeaponType: objUnit.WeaponTypeAll,
		Enable:     true,
	}
	m.handleUnitPanelActions([]unitpanel.Action{weaponAction})
	require.Empty(t, m.instructionSet.Items())

	m.handleUnitPanelActions([]unitpanel.Action{{Kind: unitpanel.ActionFocusShip, FocusUid: "own"}})
	require.Equal(t, "own", ms.Interaction.FocusedShipUid)

	ms.Core.PlayerAIControl = false
	m.handleUnitPanelActions([]unitpanel.Action{weaponAction})
	require.Len(t, m.instructionSet.Items(), 1)
}

// TestDisablingPlayerAIControlKeepsQueuedInstructions 验证关闭托管不会清空已在执行的指令。
// 场景：玩家侧托管开启时队列里已有一条 ShipMove 指令；随后关闭托管。
// 预期：两次 updateInstructions 后该指令仍然存在；失败意味着切回人工会丢掉 AI 正在走的航路。
// 覆盖：NewWithState + orderTestInstr 占位 ShipMove uid，开关托管后断言 instructionSet.Exists。
func TestDisablingPlayerAIControlKeepsQueuedInstructions(t *testing.T) {
	m := NewWithState(newAutopilotTestMissionState())
	uid := instr.GenInstrUid(instr.NameShipMove, "own")
	m.instructionSet.Add(&orderTestInstr{uid: uid})

	m.state.Core.PlayerAIControl = true
	m.updateInstructions()
	require.True(t, m.instructionSet.Exists(uid))

	m.state.Core.PlayerAIControl = false
	m.updateInstructions()
	require.True(t, m.instructionSet.Exists(uid))
}

// TestPlayerAITakeoverIssuesOrdersForPlayerShips 验证托管后 AI 会为玩家舰船下发移动指令。
// 场景：玩家与敌方各一艘驱逐舰、相距 30 格（超出驻守响应半径），迷雾关闭，SimTick=60 的编组/路径相位；
// 开启 Core.PlayerAIControl 后调用 updateInstructions。
// 预期：指令集出现玩家舰船 own 的 ShipMove/ShipMovePath（二者共享 UID）；失败说明处理器路由或
// AI 快照把玩家舰船当成了敌方/不存在。
// 覆盖：最小 MissionState + NewWithState + 默认 AI 策略，断言 GenInstrUid(NameShipMove, "own") 存在。
func TestPlayerAITakeoverIssuesOrdersForPlayerShips(t *testing.T) {
	previous := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = previous })

	ms := newAutopilotTestMissionState()
	ms.Core.SimTick = 60
	ms.Core.PlayerAIControl = true
	ms.Arena.Ships["own"] = &objUnit.BattleShip{
		Uid: "own", Name: "own", Type: objUnit.ShipTypeDestroyer,
		BelongPlayer: faction.HumanAlpha, CurHP: 100, TotalHP: 100,
		CurPos: objPos.New(10, 10),
	}
	ms.Arena.Ships["enemy"] = &objUnit.BattleShip{
		Uid: "enemy", Name: "enemy", Type: objUnit.ShipTypeDestroyer,
		BelongPlayer: faction.ComputerAlpha, CurHP: 100, TotalHP: 100,
		CurPos: objPos.New(40, 10),
	}
	m := NewWithState(ms)

	m.updateInstructions()

	uid := instr.GenInstrUid(instr.NameShipMove, "own")
	require.Truef(t, m.instructionSet.Exists(uid), "AI takeover did not issue an order for the player ship")
}
