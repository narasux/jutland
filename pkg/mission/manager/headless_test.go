package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestUpdateHeadlessAdvancesWithoutUI 验证无头更新只推进模拟相位，不读输入/相机/音频，也不依赖 Ebiten 窗口。
// 场景：双方各一艘存活舰，任务 running；headless=true、sync targeting=true、跳过命令相位。
// 预期：调用两次后 simTick/Core.SimTick 为 2；状态保持 running；没有 observer/窗口也能完成
// command(optional)/support/combat/status/stats 相位。
// 覆盖：最小 MissionState + 两舰 + GameMarks/ShipUidGenerators，调用 SetSimulationOptions 与 UpdateHeadless。
func TestUpdateHeadlessAdvancesWithoutUI(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	ms := &state.MissionState{
		Core: state.MissionCoreState{MissionStatus: state.MissionRunning},
		Player: state.MissionPlayerState{
			CurPlayer: faction.HumanAlpha,
		},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				"human": {
					Uid: "human", Name: "Human", Type: objUnit.ShipTypeDestroyer,
					CurHP: 100, TotalHP: 100, BelongPlayer: faction.HumanAlpha,
				},
				"enemy": {
					Uid: "enemy", Name: "Enemy", Type: objUnit.ShipTypeDestroyer,
					CurHP: 100, TotalHP: 100, BelongPlayer: faction.ComputerAlpha,
				},
			},
			Planes:            map[string]*objUnit.Plane{},
			ShipUidGenerators: map[faction.Player]*objUnit.ShipUidGenerator{},
		},
		UI: state.MissionUIState{GameMarks: map[objMark.ID]*objMark.Mark{}},
	}
	ms.Stats = state.NewBattleStats()

	m := &MissionManager{state: ms, instructionSet: NewInstructionSet()}
	m.SetSimulationOptions(SimulationOptions{Headless: true, SyncTargeting: true, SkipCommands: true})

	status, err := m.UpdateHeadless()
	require.NoError(t, err)
	require.Equal(t, state.MissionRunning, status)
	require.Equal(t, int64(1), m.simTick)
	require.Equal(t, int64(1), ms.Core.SimTick)

	status, err = m.UpdateHeadless()
	require.NoError(t, err)
	require.Equal(t, state.MissionRunning, status)
	require.Equal(t, int64(2), m.simTick)
	require.Equal(t, int64(2), ms.Core.SimTick)
}
