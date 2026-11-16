package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// statusTestManager 构造只含胜负判定所需字段的管理器：任务进行中 + 给定舰船/沉没舰。
func statusTestManager(ships []*objUnit.BattleShip, destroyed []*objUnit.BattleShip) *MissionManager {
	byUID := map[string]*objUnit.BattleShip{}
	for _, ship := range ships {
		byUID[ship.Uid] = ship
	}
	return &MissionManager{state: &state.MissionState{
		Core:   state.MissionCoreState{MissionStatus: state.MissionRunning},
		Player: state.MissionPlayerState{CurPlayer: faction.HumanAlpha},
		Arena:  state.MissionArenaState{Ships: byUID, DestroyedShips: destroyed},
	}}
}

// TestUpdateMissionStatusResolvesWinLossFromShips 验证任务进行中时按双方舰船存亡结算胜负。
// 场景：敌方最后一艘舰被击沉→胜利；我方最后一艘舰被击沉→失败；双方仍有舰或沉没动画未播完→继续。
// 预期：仅我方有舰→Success；仅敌方有舰→Failed；双方都有/空战场→保持 Running；
// DestroyedShips 非空时即使船已移出 Ships 也不能提前结算。
// 覆盖：表驱动构造五组 Ships/DestroyedShips，调用 updateMissionStatus 后断言 MissionStatus。
func TestUpdateMissionStatusResolvesWinLossFromShips(t *testing.T) {
	own := &objUnit.BattleShip{Uid: "own", BelongPlayer: faction.HumanAlpha}
	enemy := &objUnit.BattleShip{Uid: "enemy", BelongPlayer: faction.ComputerAlpha}

	cases := []struct {
		name      string
		ships     []*objUnit.BattleShip
		destroyed []*objUnit.BattleShip
		want      state.MissionStatus
	}{
		{name: "both sides remain", ships: []*objUnit.BattleShip{own, enemy}, want: state.MissionRunning},
		{name: "enemy fleet wiped", ships: []*objUnit.BattleShip{own}, want: state.MissionSuccess},
		{name: "own fleet wiped", ships: []*objUnit.BattleShip{enemy}, want: state.MissionFailed},
		{name: "empty arena", want: state.MissionFailed},
		{
			name:      "sinking animation delays verdict",
			ships:     []*objUnit.BattleShip{},
			destroyed: []*objUnit.BattleShip{enemy},
			want:      state.MissionRunning,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := statusTestManager(tc.ships, tc.destroyed)
			manager.updateMissionStatus()
			require.Equal(t, tc.want, manager.state.Core.MissionStatus)
		})
	}
}

// TestUpdateMissionStatusLeavesOtherStatesAlone 验证非 Running 状态不会被误改成 Running/Success。
// 场景：任务已暂停或正在看全屏地图/终端/建筑；胜负判定只应在 Running 分支执行，不能重置玩家当前界面。
// 预期：Paused/InMap/InTerminal/InBuilding 在无输入时保持原状态。
// 覆盖：对四个状态各构造管理器，调用 updateMissionStatus 后断言状态不变。
func TestUpdateMissionStatusLeavesOtherStatesAlone(t *testing.T) {
	for _, status := range []state.MissionStatus{
		state.MissionPaused,
		state.MissionInMap,
		state.MissionInTerminal,
		state.MissionInBuilding,
	} {
		t.Run(string(status), func(t *testing.T) {
			manager := statusTestManager([]*objUnit.BattleShip{
				{Uid: "own", BelongPlayer: faction.HumanAlpha},
				{Uid: "enemy", BelongPlayer: faction.ComputerAlpha},
			}, nil)
			manager.state.Core.MissionStatus = status
			manager.updateMissionStatus()
			require.Equal(t, status, manager.state.Core.MissionStatus)
		})
	}
}
