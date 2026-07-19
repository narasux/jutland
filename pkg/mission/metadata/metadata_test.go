package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAvailableMissionsFiltersCategoriesAndPreservesOrder(t *testing.T) {
	metadata := map[string]MissionMetadata{
		"classic-third":  {Name: "classic-third", Category: MissionCategoryClassic},
		"classic-first":  {Name: "classic-first", Category: MissionCategoryClassic},
		"classic-second": {Name: "classic-second", Category: MissionCategoryClassic},
		"test":           {Name: "test", Category: MissionCategoryTest},
	}
	order := []string{"classic-first", "test", "classic-second", "classic-third"}

	require.Equal(t, []string{
		"classic-first",
		"classic-second",
		"classic-third",
	}, availableMissions(metadata, order, MissionCategoryClassic))
	require.Equal(t, []string{"test"}, availableMissions(metadata, order, MissionCategoryTest))
}

func TestNormalizeMissionCategory(t *testing.T) {
	category, err := normalizeMissionCategory("")
	require.NoError(t, err)
	require.Equal(t, MissionCategoryClassic, category)

	category, err = normalizeMissionCategory("test")
	require.NoError(t, err)
	require.Equal(t, MissionCategoryTest, category)

	_, err = normalizeMissionCategory("training")
	require.ErrorContains(t, err, "unknown mission category")
}

func TestConfiguredMissionOrder(t *testing.T) {
	require.Equal(t, []string{
		"PearlHarbor1941",
		"ManilaBay1941",
		"WakeIsland1941",
		"DarwinHarbour1942",
		"Midway1942",
		"IslandChainEncounter1942",
		"Guam1944",
		"Saipan1944",
		"Palau1944",
		"Samar1944",
	}, AvailableMissions(MissionCategoryClassic))
	require.Equal(t, []string{
		"TestAll",
		"TestAntiAircraft",
	}, AvailableMissions(MissionCategoryTest))
}
