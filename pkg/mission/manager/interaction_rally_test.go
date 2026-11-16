package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// TestSetReinforcePointRallyPosRejectsLandAndClamps 验证增援点集结点落在陆地时拒绝，越界坐标夹到地图内。
// 场景：玩家在增援界面点击小地图设置集结点；点到陆地格应提示失败并保留旧集结点，
// 点到地图外/边缘时应把坐标夹到有效范围，而不是把舰船派到越界位置。
// 预期：陆地 (2,1) 触发 RallySetFailedTick=60 且 RallyPos 不变；越界 (99,99) 被夹到 (4,2)。
// 覆盖：构造 5x3 海岸地图（中间两格陆地），先点陆地再点越界水面，逐项断言。
func TestSetReinforcePointRallyPosRejectsLandAndClamps(t *testing.T) {
	manager := &MissionManager{state: &state.MissionState{
		Core: state.MissionCoreState{MissionMD: metadata.MissionMetadata{
			MapCfg: &mapcfg.MapCfg{
				Map:    mapcfg.MapData{"SSSSS", "SSLLS", "SSSSS"},
				Width:  5,
				Height: 3,
			},
		}},
	}}
	rp := objBuilding.NewReinforcePoint(
		objPos.New(0, 0), 0, objPos.New(1, 1), faction.HumanAlpha, 1, nil,
	)

	manager.setReinforcePointRallyPos(rp, objPos.New(2, 1))
	require.Equal(t, 60, manager.state.UI.RallySetFailedTick, "land click should raise failure flag")
	require.Equal(t, objPos.New(1, 1), rp.RallyPos, "land click must keep the old rally point")

	manager.setReinforcePointRallyPos(rp, objPos.NewR(99, 99))
	require.Equal(t, objPos.New(4, 2), rp.RallyPos, "out-of-map click should clamp into the map")
}
