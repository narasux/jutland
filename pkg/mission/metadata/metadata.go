package metadata

import (
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// MissionCategory 任务关卡分类。
type MissionCategory string

const (
	MissionCategoryClassic MissionCategory = "classic"
	MissionCategoryTest    MissionCategory = "test"
)

// MissionMetadata 任务元配置
type MissionMetadata struct {
	Name         string
	DisplayName  string
	Category     MissionCategory
	MapCfg       *mapcfg.MapCfg
	displayNames map[i18n.Language]string
	// 最大战舰数量
	MaxShipCount int
	// 初始资金
	InitFunds int64
	// 初始位置
	InitCameraPos objPos.MapPos
	// 关卡描述
	Description  string
	descriptions map[i18n.Language]string
	// 统计信息（加载时计算）
	AllyShipCount       int // 我方初始舰船数
	EnemyShipCount      int // 敌方初始舰船数
	AllyReinforceCount  int // 我方增援点数量
	EnemyReinforceCount int // 敌方增援点数量
	OilPlatformCount    int // 油井数量
	TotalReinforceSlots int // 全部增援槽位总数
	// 初始战舰
	InitShips []InitShipMetadata
	// 初始增援点
	InitReinforcePoints []InitReinforcePointMetadata
	// 初始油井
	InitOilPlatforms []InitOilPlatformMetadata
}

// InitShipMetadata ...
type InitShipMetadata struct {
	ShipName     string
	Pos          objPos.MapPos
	Rotation     float64
	BelongPlayer faction.Player
}

// InitReinforcePointMetadata ...
type InitReinforcePointMetadata struct {
	Pos               objPos.MapPos
	Rotation          float64
	RallyPos          objPos.MapPos
	BelongPlayer      faction.Player
	MaxOncomingShip   int
	ProvidedShipNames []string
}

// InitOilPlatformMetadata ...
type InitOilPlatformMetadata struct {
	Pos    objPos.MapPos
	Radius int
	Yield  int
}

var (
	missionMetadata map[string]MissionMetadata
	missionOrder    []string // 保留 missions.json5 中的书写顺序
)

// Get 获取任务元配置
func Get(mission string) MissionMetadata {
	md := missionMetadata[mission]
	if value := i18n.LocalizedValue(md.displayNames); value != "" {
		md.DisplayName = value
	}
	if value := i18n.LocalizedValue(md.descriptions); value != "" {
		md.Description = value
	}
	return md
}

// AvailableMissions 获取可用任务列表
func AvailableMissions(category MissionCategory) []string {
	return availableMissions(missionMetadata, missionOrder, category)
}

func availableMissions(
	metadata map[string]MissionMetadata,
	order []string,
	category MissionCategory,
) []string {
	missions := make([]string, 0, len(order))
	for _, name := range order {
		md := metadata[name]
		if md.Category == category {
			missions = append(missions, name)
		}
	}
	return missions
}
