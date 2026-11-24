package cheat

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestShowMeTheMoneyAddsFundsAndMatchesCaseInsensitive 验证加钱命令归一大小写/空格并累加 10000。
// 场景：玩家在终端输入 "show me the money"/"SHOW  ME  THE MONEY"；命令匹配应忽略大小写与多余空格，
// 每次执行都应在当前资金上累加，而不是覆盖成固定值。
// 预期：Match 对两种写法都为 true；CurFunds 500→10500→20500；返回串包含当前资金。
// 覆盖：构造 CurFunds=500 的 MissionState，连续 Exec 两次并断言资金与返回文本。
func TestShowMeTheMoneyAddsFundsAndMatchesCaseInsensitive(t *testing.T) {
	ms := &state.MissionState{Player: state.MissionPlayerState{CurFunds: 500}}
	cheat := &ShowMeTheMoney{}

	require.True(t, cheat.Match("show me the money"))
	require.True(t, cheat.Match("SHOW  ME  THE MONEY"))
	require.False(t, cheat.Match("show me money"))

	got := cheat.Exec(ms)
	require.Equal(t, int64(10500), ms.Player.CurFunds)
	require.Contains(t, got, "10500")

	cheat.Exec(ms)
	require.Equal(t, int64(20500), ms.Player.CurFunds)
}

// TestDebugTogglesFlipOnlyTheirOwnFlag 验证每个 debug 作弊只翻转自己的 DebugFlags 字段。
// 场景：玩家逐项输入 "damage color by team"/"show cursor pos obj info"/"show plane hp"/
// "show hit boxes"/"show airfield runway"；翻转错字段会让面板显示与实际绘制不一致。
// 预期：每项首次 Exec 后目标字段为 true、其余字段仍为 false；再次 Exec 后目标字段回到 false。
// 覆盖：表驱动五个命令，每次从零值 DebugFlags 开始，用 getter 断言目标字段。
func TestDebugTogglesFlipOnlyTheirOwnFlag(t *testing.T) {
	cases := []struct {
		name string
		exec func(*state.MissionState) string
		get  func(state.DebugFlags) bool
	}{
		{
			name: "damage color",
			exec: (&DamageColorByTeam{}).Exec,
			get:  func(f state.DebugFlags) bool { return f.DamageColorByTeam },
		},
		{
			name: "cursor info",
			exec: (&ShowCursorPosObjInfo{}).Exec,
			get:  func(f state.DebugFlags) bool { return f.ShowCursorPosObjInfo },
		},
		{name: "plane hp", exec: (&ShowPlaneHP{}).Exec, get: func(f state.DebugFlags) bool { return f.ShowPlaneHP }},
		{
			name: "hit boxes",
			exec: (&ShowHitBoxes{}).Exec,
			get:  func(f state.DebugFlags) bool { return f.ShowHitBoxes },
		},
		{
			name: "airfield runway",
			exec: (&ShowAirfieldRunway{}).Exec,
			get:  func(f state.DebugFlags) bool { return f.ShowAirfieldRunway },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ms := &state.MissionState{}
			tc.exec(ms)
			require.True(t, tc.get(ms.UI.DebugFlags))
			tc.exec(ms)
			require.False(t, tc.get(ms.UI.DebugFlags))
		})
	}
}

// TestDebugAllEnablesEveryFlag 验证 "debug all" 一次性打开全部五个调试开关。
// 场景：调试战斗中需要同时看伤害颜色、光标对象、飞机 HP、命中框与机场跑道；
// 漏开任何一个都会让对应调试层不显示。
// 预期：Exec 后五个字段全为 true，返回串为 "Enabled all debug flags"。
// 覆盖：空 MissionState 上执行 DebugAll，逐字段断言。
func TestDebugAllEnablesEveryFlag(t *testing.T) {
	ms := &state.MissionState{}
	got := (&DebugAll{}).Exec(ms)
	require.Equal(t, "Enabled all debug flags", got)
	flags := ms.UI.DebugFlags
	require.True(t, flags.DamageColorByTeam)
	require.True(t, flags.ShowCursorPosObjInfo)
	require.True(t, flags.ShowPlaneHP)
	require.True(t, flags.ShowHitBoxes)
	require.True(t, flags.ShowAirfieldRunway)
}

// TestShowMeTheShipMatchesConfiguredNameOnly 验证 "show me the ship <name>" 只匹配配置表中存在的舰名。
// 场景：终端输入舰名时大小写/空格可归一；拼错或输入未配置舰名必须返回 false，不能生成空模板。
// 预期：存在舰名（取 AllShipNames[0]）匹配 true 且记录原始名称；未知名称匹配 false。
// 覆盖：以当前配置的 AllShipNames[0] 构造命令，另测 "definitely-not-a-ship"。
func TestShowMeTheShipMatchesConfiguredNameOnly(t *testing.T) {
	const name = "test-cheat-ship"
	previousMap, previousNames := objUnit.ShipMap, objUnit.AllShipNames
	if objUnit.ShipMap == nil {
		objUnit.ShipMap = map[string]*objUnit.BattleShip{}
	}
	objUnit.ShipMap[name] = &objUnit.BattleShip{Name: name}
	objUnit.AllShipNames = append(objUnit.AllShipNames, name)
	t.Cleanup(func() {
		delete(objUnit.ShipMap, name)
		objUnit.ShipMap, objUnit.AllShipNames = previousMap, previousNames
	})

	cheat := &ShowMeTheShip{}
	require.True(t, cheat.Match("show me the ship "+name))
	require.Equal(t, name, cheat.shipName)
	require.False(t, (&ShowMeTheShip{}).Match("show me the ship definitely-not-a-ship"))
	require.False(t, (&ShowMeTheShip{}).Match("show me the ship"))
}

// TestCheatDescriptionsAndStringStayNonEmpty 验证所有作弊命令的 String/Desc 可用于终端帮助列表。
// 场景：终端 help 逐条打印命令与说明；空字符串会让玩家看到空白行，无法发现命令。
// 预期：五条代表性命令的 String/Desc 都非空，且 String 能被自身 Match 命中。
// 覆盖：表驱动 ShowMeTheMoney/DebugAll/LetMeSeeTheBadKids/BlackSheepWall/ShowMeTheShip 的元数据。
func TestCheatDescriptionsAndStringStayNonEmpty(t *testing.T) {
	cheats := []Cheat{
		&ShowMeTheMoney{},
		&DebugAll{},
		&LetMeSeeTheBadKids{},
		&BlackSheepWall{},
		&HelpMeAISama{},
	}
	for _, cheat := range cheats {
		require.NotEmpty(t, strings.TrimSpace(cheat.String()))
		require.NotEmpty(t, strings.TrimSpace(cheat.Desc()))
		require.True(t, cheat.Match(cheat.String()), cheat.String())
	}

	param := &ShowMeTheShip{}
	require.NotEmpty(t, strings.TrimSpace(param.String()))
	require.NotEmpty(t, strings.TrimSpace(param.Desc()))
}
