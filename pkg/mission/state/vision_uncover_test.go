package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestUnexploredCentroidReturnsRemainingAreaCenter 验证未探索重心与“全图探明”边界。
// 场景：AI 需要把舰队/侦察机往尚未探索的海域推；全图探明后应返回 false，不能继续往 (0,0) 派机。
// 预期：10×10 图中已探索 x<5 时，剩余区域重心为 (7,4)；全图探索完/nil 视野返回 false。
// 覆盖：直接填充 Explored 字节图，调用 UnexploredCentroid 断言坐标与 ok。
func TestUnexploredCentroidReturnsRemainingAreaCenter(t *testing.T) {
	vision := NewFactionVision(10, 10)
	for y := 0; y < 10; y++ {
		for x := 0; x < 5; x++ {
			vision.Explored[y*10+x] = 1
		}
	}
	got, ok := vision.UnexploredCentroid()
	require.True(t, ok)
	require.Equal(t, objPos.New(7, 4), got)

	for i := range vision.Explored {
		vision.Explored[i] = 1
	}
	_, ok = vision.UnexploredCentroid()
	require.False(t, ok)

	var nilVision *FactionVision
	_, ok = nilVision.UnexploredCentroid()
	require.False(t, ok)
}

// TestBestUncoverPicksExploredFrontierWithFreshTiles 验证前线选点只落在已探索且有暗邻格的边界上。
// 场景：侦察机完成一圈搜索后，AI 选择下一处“揭雾收益高、离起点近”的前线；若选到内部格会原地打转。
// 预期：12×12 图中只探索 x=6 的竖带、从 (0,6) 以视距 2 搜索，结果落在 x=6 的探索带上，
// 且该格存在 4 邻域未探索格；sight<=0 或全图探明时返回 false。
// 覆盖：构造竖带视野，调用 BestUncover 后断言 x 坐标、ExploredAt 与 hasDarkNeighbor。
func TestBestUncoverPicksExploredFrontierWithFreshTiles(t *testing.T) {
	vision := NewFactionVision(12, 12)
	for y := 0; y < 12; y++ {
		vision.Explored[y*12+6] = 1
	}

	got, ok := vision.BestUncover(objPos.New(0, 6), 2)
	require.True(t, ok)
	require.Equal(t, 6, got.MX)
	require.True(t, vision.ExploredAt(got.MX, got.MY))
	require.True(t, hasDarkNeighbor(vision, got.MX, got.MY))

	_, ok = vision.BestUncover(objPos.New(0, 6), 0)
	require.False(t, ok)
	for i := range vision.Explored {
		vision.Explored[i] = 1
	}
	_, ok = vision.BestUncover(objPos.New(0, 6), 2)
	require.False(t, ok)
}

// TestConcealsEnemyUsesCurrentPlayerFog 验证迷雾只遮挡敌方单位，且当前玩家/己方单位永远不遮。
// 场景：绘制敌方舰船前调用 ConcealsEnemy；若把己方也判成隐藏，玩家会看不到自己的舰。
// 预期：己方返回 false；敌方在 Visible 格 false、在未可见格 true；关闭迷雾后全部 false。
// 覆盖：4×4 视野只点亮 (0,0)，分别查询己方/敌方与可见/不可见坐标。
func TestConcealsEnemyUsesCurrentPlayerFog(t *testing.T) {
	vision := NewFactionVision(4, 4)
	vision.Visible[0] = 1
	ms := &MissionState{
		Core: stateCoreWithFog(),
		Player: MissionPlayerState{
			CurPlayer: faction.HumanAlpha,
			Visions:   map[faction.Player]*FactionVision{faction.HumanAlpha: vision},
		},
	}

	require.False(t, ms.ConcealsEnemy(faction.HumanAlpha, 0, 0))
	require.False(t, ms.ConcealsEnemy(faction.ComputerAlpha, 0, 0))
	require.True(t, ms.ConcealsEnemy(faction.ComputerAlpha, 1, 0))

	ms.Core.FogOfWar = false
	require.False(t, ms.ConcealsEnemy(faction.ComputerAlpha, 1, 0))
}

// stateCoreWithFog 构造开启迷雾的 Core，避免每个测试重复写结构体。
func stateCoreWithFog() MissionCoreState {
	return MissionCoreState{FogOfWar: true}
}
