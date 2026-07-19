package game

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/audio"
	"github.com/narasux/jutland/pkg/mission/manager"
	"github.com/narasux/jutland/pkg/mission/metadata"
)

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

func TestCycleMissionWrapsWithinGivenCategory(t *testing.T) {
	classic := metadata.AvailableMissions(metadata.MissionCategoryClassic)
	tests := metadata.AvailableMissions(metadata.MissionCategoryTest)

	require.Equal(t, "PearlHarbor1941", cycleMission(classic, "Samar1944", 1))
	require.Equal(t, "Samar1944", cycleMission(classic, "PearlHarbor1941", -1))
	require.Equal(t, "TestAll", cycleMission(tests, "TestAntiAircraft", 1))
	require.Equal(t, "TestAntiAircraft", cycleMission(tests, "TestAll", -1))
}

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
