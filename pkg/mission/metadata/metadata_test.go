package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/faction"
)

// TestGetLocalizesTraditionalChineseMissionMetadata 验证关卡名称与描述跟随 zh-Hant 切换。
// 场景：设置页切到繁体中文后进入任务选择页；PearlHarbor1941 在 missions.json5 中配置了简繁英俄日五套名称与描述。
// 预期：zh-Hant 返回「珍珠港襲擊-1941」且描述含「珍珠港內」，en 返回英文名；失败表示新字段未进入 displayNames/descriptions 映射。
// 覆盖：SetLanguage 依次切 zh-Hant/en，调用 Get 后断言 DisplayName 与 Description，Cleanup 复位原语言。
func TestGetLocalizesTraditionalChineseMissionMetadata(t *testing.T) {
	previousLanguage := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(previousLanguage)) })

	i18n.SetLanguage(string(i18n.LanguageZhHant))
	md := Get("PearlHarbor1941")
	require.Equal(t, "珍珠港襲擊-1941", md.DisplayName)
	require.Contains(t, md.Description, "珍珠港內")

	i18n.SetLanguage(string(i18n.LanguageEnglish))
	md = Get("PearlHarbor1941")
	require.Equal(t, "Attack on Pearl Harbor - 1941", md.DisplayName)
}

// TestAvailableMissionsFiltersCategoriesAndPreservesOrder 验证任务列表按分类过滤且保持配置顺序。
// 场景：构造乱序 map（classic-third/first/second、tutorial 与 test），order 为 first、test、tutorial、second、third。
// 预期：classic 返回 first/second/third，tutorial 返回 tutorial，test 返回 test；失败意味着顺序随机漂移或分类泄漏。
// 覆盖：三组 require.Equal 精确比较切片元素与顺序，覆盖跨分类穿插。
func TestAvailableMissionsFiltersCategoriesAndPreservesOrder(t *testing.T) {
	metadata := map[string]MissionMetadata{
		"classic-third":  {Name: "classic-third", Category: MissionCategoryClassic},
		"classic-first":  {Name: "classic-first", Category: MissionCategoryClassic},
		"classic-second": {Name: "classic-second", Category: MissionCategoryClassic},
		"tutorial":       {Name: "tutorial", Category: MissionCategoryTutorial},
		"test":           {Name: "test", Category: MissionCategoryTest},
	}
	order := []string{"classic-first", "test", "tutorial", "classic-second", "classic-third"}

	require.Equal(t, []string{
		"classic-first",
		"classic-second",
		"classic-third",
	}, availableMissions(metadata, order, MissionCategoryClassic))
	require.Equal(t, []string{"tutorial"}, availableMissions(metadata, order, MissionCategoryTutorial))
	require.Equal(t, []string{"test"}, availableMissions(metadata, order, MissionCategoryTest))
}

// TestNormalizeMissionCategory 验证任务分类缺省值与非法值校验。
// 场景：分别传入空串、合法 "test"/"tutorial"、未知 "training"。
// 预期：空串归一为 classic，"test"/"tutorial" 保留，"training" 返回含 unknown mission category 的错误。
// 覆盖：三次 NoError+Equal 和一次 ErrorContains，覆盖缺省、两个合法值、非法值四条分支。
func TestNormalizeMissionCategory(t *testing.T) {
	category, err := normalizeMissionCategory("")
	require.NoError(t, err)
	require.Equal(t, MissionCategoryClassic, category)

	category, err = normalizeMissionCategory("test")
	require.NoError(t, err)
	require.Equal(t, MissionCategoryTest, category)

	category, err = normalizeMissionCategory("tutorial")
	require.NoError(t, err)
	require.Equal(t, MissionCategoryTutorial, category)

	_, err = normalizeMissionCategory("training")
	require.ErrorContains(t, err, "unknown mission category")
}

// TestConfiguredMissionOrder 验证 missions.json5 初始化后的关卡分类与顺序稳定。
// 场景：包 init 加载配置后，分别调用 AvailableMissions(classic)、tutorial 与 test。
// 预期：classic 返回 10 关，tutorial 返回 TutorialBasicControls 等 6 关，test 返回 TestAll 等 7 关。
// 覆盖：三组完整切片相等断言，防止配置顺序被改、关卡误分类或漏注册。
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
		"TutorialBasicControls",
		"TutorialFormation",
		"TutorialSubmarine",
		"TutorialAviation",
		"TutorialReinforce",
		"TutorialCombat",
	}, AvailableMissions(MissionCategoryTutorial))
	require.Equal(t, []string{
		"TestAll",
		"TestAntiAircraft",
		"MidwayFourCornersTest",
		"MidwayAllTypesDuelTest",
		"ASWTest",
		"SubmarineTest",
		"RamTest",
	}, AvailableMissions(MissionCategoryTest))
}

// TestStatsForPlayerSideSplitsAllyAndEnemy 验证按所选阵营把双方统计拆成 ally/enemy，非法阵营回退第一阵营。
// 场景：任务选择页切换 P1/P2；P1 的 10 舰/2 增援应显示为我方，P2 的 3/1 显示为敌方，反之亦然。
// 预期：选 P1→ally(10,2)/enemy(3,1)；选 P2→ally(3,1)/enemy(10,2)；
// 非法 "bad" 归一为 PlayerSides[0]=P1，仍得到 ally(10,2)/enemy(3,1)。
// 覆盖：表驱动三个 side 调用 StatsForPlayerSide，逐字段断言 ally/enemy 的 ShipCount/ReinforceCount。
func TestStatsForPlayerSideSplitsAllyAndEnemy(t *testing.T) {
	md := MissionMetadata{
		PlayerSides: []faction.Side{faction.SideP1, faction.SideP2},
		SideStats: map[faction.Side]MissionSideStats{
			faction.SideP1: {ShipCount: 10, ReinforceCount: 2},
			faction.SideP2: {ShipCount: 3, ReinforceCount: 1},
		},
	}

	cases := []struct {
		name  string
		side  faction.Side
		ally  MissionSideStats
		enemy MissionSideStats
	}{
		{name: "P1 selected", side: faction.SideP1, ally: MissionSideStats{10, 2}, enemy: MissionSideStats{3, 1}},
		{name: "P2 selected", side: faction.SideP2, ally: MissionSideStats{3, 1}, enemy: MissionSideStats{10, 2}},
		{
			name:  "invalid falls back to first",
			side:  faction.Side("bad"),
			ally:  MissionSideStats{10, 2},
			enemy: MissionSideStats{3, 1},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ally, enemy := md.StatsForPlayerSide(tc.side)
			require.Equal(t, tc.ally, ally)
			require.Equal(t, tc.enemy, enemy)
		})
	}
}
