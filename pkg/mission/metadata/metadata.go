package metadata

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// 任务元配置
type MissionMetadata struct {
	Name   string
	MapCfg *mapcfg.MapCfg
	// 最大战舰数量
	MaxShipCount int
	// 初始资金
	InitFunds int64
	// 初始位置
	InitCameraPos obj.MapPos
	// 初始战舰
	InitShips []InitShipMetadata
	// 初始增援点
	InitReinforcePoints []InitReinforcePointMetadata
}

// InitShipMetadata ...
type InitShipMetadata struct {
	ShipName     string
	Pos          obj.MapPos
	Rotation     float64
	BelongPlayer faction.Player
}

// InitReinforcePointMetadata ...
type InitReinforcePointMetadata struct {
	Pos               obj.MapPos
	Rotation          float64
	RallyPos          obj.MapPos
	BelongPlayer      faction.Player
	MaxOncomingShip   int
	ProvidedShipNames []string
}

var missionMetadata map[string]MissionMetadata

// Get 获取任务元配置
func Get(mission string) MissionMetadata {
	return missionMetadata[mission]
}
