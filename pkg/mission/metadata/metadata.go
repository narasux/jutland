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
	// 各配置阵营的初始相机位置
	InitCameraPositions map[faction.Side]objPos.MapPos
	// 关卡描述
	Description  string
	descriptions map[i18n.Language]string
	// 可选阵营与分阵营统计信息（加载时计算）
	PlayerSides         []faction.Side
	SideStats           map[faction.Side]MissionSideStats
	OilPlatformCount    int // 油井数量
	TotalReinforceSlots int // 全部增援槽位总数
	// 初始战舰
	InitShips []InitShipMetadata
	// 初始增援点
	InitReinforcePoints []InitReinforcePointMetadata
	// 初始油井
	InitOilPlatforms []InitOilPlatformMetadata
	// 初始陆地机场
	InitAirfields []InitAirfieldMetadata
}

// MissionSideStats 单个任务阵营的统计信息。
type MissionSideStats struct {
	ShipCount      int
	ReinforceCount int
}

// InitShipMetadata ...
type InitShipMetadata struct {
	ShipName   string
	Pos        objPos.MapPos
	Rotation   float64
	BelongSide faction.Side
}

// InitReinforcePointMetadata ...
type InitReinforcePointMetadata struct {
	Pos               objPos.MapPos
	Rotation          float64
	RallyPos          objPos.MapPos
	BelongSide        faction.Side
	MaxOncomingShip   int
	ProvidedShipNames []string
}

// InitOilPlatformMetadata ...
type InitOilPlatformMetadata struct {
	Pos    objPos.MapPos
	Radius int
	Yield  int
}

// InitAirfieldMetadata 陆地机场元配置。
type InitAirfieldMetadata struct {
	Pos          objPos.MapPos
	Rotation     float64
	RunwayLength float64
	RunwayWidth  float64
	BelongSide   faction.Side
	TakeOffTime  float64
	// TakeoffPoints 跑道起飞点数：<=0 缺省（双点并行），1 = 单机串行起飞，
	// >=2 = 双机并行（缺省值）。用于重轰炸机单架间隔起飞。
	TakeoffPoints int
	PlaneGroups   []InitAirfieldGroupMetadata
}

// InitAirfieldGroupMetadata 陆地机场驻场机队元配置。
type InitAirfieldGroupMetadata struct {
	Name string
	// MaxCount 最大数量（同时决定停机坪容量）
	MaxCount int64
	// InitCount 初始停放数量（元数据阶段已把缺省值补齐为 MaxCount）
	InitCount int64
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

// NormalizePlayerSide 将阵营修正为该任务可选的阵营。
func (m MissionMetadata) NormalizePlayerSide(side faction.Side) faction.Side {
	if m.HasPlayerSide(side) {
		return side
	}
	if len(m.PlayerSides) > 0 {
		return m.PlayerSides[0]
	}
	return faction.SideP1
}

// HasPlayerSide 判断任务是否包含指定阵营。
func (m MissionMetadata) HasPlayerSide(side faction.Side) bool {
	for _, playerSide := range m.PlayerSides {
		if playerSide == side {
			return true
		}
	}
	return false
}

// StatsForPlayerSide 按所选阵营返回我方与敌方统计信息。
func (m MissionMetadata) StatsForPlayerSide(side faction.Side) (
	ally MissionSideStats,
	enemy MissionSideStats,
) {
	side = m.NormalizePlayerSide(side)
	for _, playerSide := range m.PlayerSides {
		stats := m.SideStats[playerSide]
		if playerSide == side {
			ally = stats
		} else {
			enemy.ShipCount += stats.ShipCount
			enemy.ReinforceCount += stats.ReinforceCount
		}
	}
	return ally, enemy
}

// CameraPosForPlayerSide 返回所选配置阵营的初始相机位置。
func (m MissionMetadata) CameraPosForPlayerSide(side faction.Side) objPos.MapPos {
	side = m.NormalizePlayerSide(side)
	if pos, ok := m.InitCameraPositions[side]; ok {
		return pos
	}
	return objPos.New(0, 0)
}

// AvailableMissions 获取可用任务列表
func AvailableMissions(category MissionCategory) []string {
	return availableMissions(missionMetadata, missionOrder, category)
}

// AllMissions 获取全部任务名称（按 missions.json5 书写顺序），供启动期配置校验使用。
func AllMissions() []string {
	names := make([]string, len(missionOrder))
	copy(names, missionOrder)
	return names
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
