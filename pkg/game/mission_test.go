package game

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/mission/manager"
	"github.com/narasux/jutland/pkg/mission/metadata"
)

// TestSelectMissionCategoryStartsAtFirstMission 验证切换任务分类后落到该分类第一关。
// 场景：当前 classic 的最后一关 Samar1944，切到 test 再切回 classic，不能保留旧关卡。
// 预期：返回 true 且分别选中 TestAll、PearlHarbor1941；失败意味着分类与关卡不同步。
// 覆盖：Game 初始 curMissionCategory=classic、curMission=Samar1944，两次 selectMissionCategory 后断言。
func TestSelectMissionCategoryStartsAtFirstMission(t *testing.T) {
	g := &Game{
		curMissionCategory: metadata.MissionCategoryClassic,
		curMission:         "Samar1944",
	}

	require.True(t, g.selectMissionCategory(metadata.MissionCategoryTest))
	require.Equal(t, metadata.MissionCategoryTest, g.curMissionCategory)
	require.Equal(t, "TestAll", g.curMission)

	g.curMission = "TestAntiAircraft"
	require.True(t, g.selectMissionCategory(metadata.MissionCategoryClassic))
	require.Equal(t, metadata.MissionCategoryClassic, g.curMissionCategory)
	require.Equal(t, "PearlHarbor1941", g.curMission)
}

// TestOtherMissionCategoryTogglesBothWays 验证经典/测试两个任务分类可双向互切。
// 场景：关卡选择页只有 classic 与 test 两个分类，切换函数必须覆盖两个方向。
// 预期：classic→test、test→classic；失败意味着某一方向会停在原分类。
// 覆盖：直接调用 otherMissionCategory 两次并断言返回值。
func TestOtherMissionCategoryTogglesBothWays(t *testing.T) {
	require.Equal(
		t,
		metadata.MissionCategoryTest,
		otherMissionCategory(metadata.MissionCategoryClassic),
	)
	require.Equal(
		t,
		metadata.MissionCategoryClassic,
		otherMissionCategory(metadata.MissionCategoryTest),
	)
}

// TestCycleMissionWrapsWithinGivenCategory 验证 cycleMission 在分类内首尾循环而不越界。
// 场景：classic 共 10 关，Samar1944 在末位 +1、PearlHarbor1941 在首位 -1；test 共 7 关同理。
// 预期：得到 PearlHarbor1941、Samar1944、MidwayFourCornersTest、RamTest。
// 覆盖：分别取 AvailableMissions(classic/test)，用 ±1 偏移调用四次并断言结果。
func TestCycleMissionWrapsWithinGivenCategory(t *testing.T) {
	classic := metadata.AvailableMissions(metadata.MissionCategoryClassic)
	tests := metadata.AvailableMissions(metadata.MissionCategoryTest)

	require.Equal(t, "PearlHarbor1941", cycleMission(classic, "Samar1944", 1))
	require.Equal(t, "Samar1944", cycleMission(classic, "PearlHarbor1941", -1))
	require.Equal(t, "MidwayFourCornersTest", cycleMission(tests, "TestAntiAircraft", 1))
	require.Equal(t, "RamTest", cycleMission(tests, "TestAll", -1))
}

// TestStartMissionLoadingResetsPreviousMission 验证开始新关卡会丢弃旧管理器并复位加载界面。
// 场景：上一局仍在 MissionRunning，任务管理器和音频播放器均非 nil，三个加载标志均为 true。
// 预期：mode=MissionLoading、missionMgr=nil，Ready/MissionRunningDrawn/LoadedAudioPlayed 全为 false。
// 覆盖：构造带 manager/player 与全 true 标志的 Game，调用一次 startMissionLoading 后逐项断言。
func TestStartMissionLoadingResetsPreviousMission(t *testing.T) {
	g := &Game{
		mode:       GameModeMissionRunning,
		missionMgr: &manager.MissionManager{},
		player:     &audio.Player{},
		objStates: &objStates{LoadingInterface: &loadingInterface{
			Ready:               true,
			MissionRunningDrawn: true,
			LoadedAudioPlayed:   true,
		}},
	}

	g.startMissionLoading()

	require.Equal(t, GameMode(GameModeMissionLoading), g.mode)
	require.Nil(t, g.missionMgr)
	require.False(t, g.objStates.LoadingInterface.Ready)
	require.False(t, g.objStates.LoadingInterface.MissionRunningDrawn)
	require.False(t, g.objStates.LoadingInterface.LoadedAudioPlayed)
}
