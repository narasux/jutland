package metadata

import (
	"sort"

	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// MissionMetadata 任务元配置
type MissionMetadata struct {
	Name        string
	DisplayName string
	MapCfg      *mapcfg.MapCfg
	// 最大战舰数量
	MaxShipCount int
	// 初始资金
	InitFunds int64
	// 初始位置
	InitCameraPos objPos.MapPos
	// 关卡描述
	Description string
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

var missionMetadata map[string]MissionMetadata

// Get 获取任务元配置
func Get(mission string) MissionMetadata {
	return missionMetadata[mission]
}

// AvailableMissions 获取可用任务列表
func AvailableMissions() []string {
	keys := lo.Keys(missionMetadata)
	sort.Strings(keys)
	return keys
}
