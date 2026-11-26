package game

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/mission/manager"
	"github.com/narasux/jutland/pkg/mission/metadata"
)

// TestSelectMissionCategoryStartsAtFirstMission 验证切换任务分类后落到该分类第一关。
// 场景：当前 classic 的最后一关 Samar1944，依次切到 tutorial、test，再切回 classic。
// 预期：返回 true 且分别选中 TutorialBasicControls、TestAll、PearlHarbor1941；失败意味着分类与关卡不同步。
// 覆盖：Game 初始 classic/Samar1944，三次 selectMissionCategory 后断言分类与首关。
func TestSelectMissionCategoryStartsAtFirstMission(t *testing.T) {
	g := &Game{
		curMissionCategory: metadata.MissionCategoryClassic,
		curMission:         "Samar1944",
	}

	require.True(t, g.selectMissionCategory(metadata.MissionCategoryTutorial))
	require.Equal(t, metadata.MissionCategoryTutorial, g.curMissionCategory)
	require.Equal(t, "TutorialBasicControls", g.curMission)

	g.curMission = "TutorialFormation"
	require.True(t, g.selectMissionCategory(metadata.MissionCategoryTest))
	require.Equal(t, metadata.MissionCategoryTest, g.curMissionCategory)
	require.Equal(t, "TestAll", g.curMission)

	g.curMission = "TestAntiAircraft"
	require.True(t, g.selectMissionCategory(metadata.MissionCategoryClassic))
	require.Equal(t, metadata.MissionCategoryClassic, g.curMissionCategory)
	require.Equal(t, "PearlHarbor1941", g.curMission)
}

// TestNextMissionCategoryCycles 验证经典/教学/测试三分类按方向循环且不越界。
// 场景：关卡选择页有 classic、tutorial、test 三个非空分类，Tab 与 Shift+Tab 都要工作。
// 预期：正向 classic→tutorial→test→classic，反向 classic→test；失败意味着某个方向会停在原分类。
// 覆盖：调用 nextMissionCategory 四次，断言三个正向结果和一个反向结果。
func TestNextMissionCategoryCycles(t *testing.T) {
	require.Equal(t, metadata.MissionCategoryTutorial, nextMissionCategory(metadata.MissionCategoryClassic, 1))
	require.Equal(t, metadata.MissionCategoryTest, nextMissionCategory(metadata.MissionCategoryTutorial, 1))
	require.Equal(t, metadata.MissionCategoryClassic, nextMissionCategory(metadata.MissionCategoryTest, 1))
	require.Equal(t, metadata.MissionCategoryTest, nextMissionCategory(metadata.MissionCategoryClassic, -1))
}

// TestCycleMissionWrapsWithinGivenCategory 验证 cycleMission 在分类内首尾循环而不越界。
// 场景：classic 10 关、test 7 关、tutorial 5 关；各取首尾用 ±1 偏移检查回绕。
// 预期：classic 得 PearlHarbor1941/Samar1944，tutorial 得 TutorialFormation/TutorialBasicControls，test 得 MidwayFourCornersTest/RamTest。
// 覆盖：分别取三类 AvailableMissions，调用 cycleMission 六次并断言结果。
func TestCycleMissionWrapsWithinGivenCategory(t *testing.T) {
	classic := metadata.AvailableMissions(metadata.MissionCategoryClassic)
	tests := metadata.AvailableMissions(metadata.MissionCategoryTest)
	tutorials := metadata.AvailableMissions(metadata.MissionCategoryTutorial)

	require.Equal(t, "PearlHarbor1941", cycleMission(classic, "Samar1944", 1))
	require.Equal(t, "Samar1944", cycleMission(classic, "PearlHarbor1941", -1))
	require.Equal(t, "TutorialFormation", cycleMission(tutorials, "TutorialBasicControls", 1))
	require.Equal(t, "TutorialBasicControls", cycleMission(tutorials, "TutorialFormation", -1))
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
