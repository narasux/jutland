package metadata

import (
	"github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// 任务元配置
type MissionMetadata struct {
	Name   string
	MapCfg *mapcfg.MapCfg
	// 最大战舰数量
	MaxShipCount int
	// 初始位置
	InitCameraPos object.MapPos
	// 初始战舰
	InitShips []InitShipMetadata
}

type InitShipMetadata struct {
	ShipName object.ShipName
	Pos      object.MapPos
	Rotation float64
}

var missionMetadata = map[Mission]MissionMetadata{
	MissionDefault: {
		Name:          "默认关卡",
		InitCameraPos: object.MapPos{MX: 32, MY: 32},
		MapCfg:        mapcfg.GetByName(mapcfg.MapDefault),
		MaxShipCount:  5,
		InitShips: []InitShipMetadata{
			{object.ShipDefault, object.MapPos{MX: 32, MY: 32, RX: 32, RY: 32}, 90},
		},
	},
}

// Get 获取任务元配置
func Get(mission Mission) MissionMetadata {
	return missionMetadata[mission]
}
