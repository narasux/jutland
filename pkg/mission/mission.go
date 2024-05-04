package mission

import (
	obj "github.com/narasux/jutland/pkg/object"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// 任务元配置
type MissionMetadata struct {
	Name   string
	MapCfg *mapcfg.MapCfg
	// 初始位置
	InitCameraPos obj.MapPos
	// 初始战舰
	InitShips []InitShipMetadata
}

type InitShipMetadata struct {
	ShipName obj.ShipName
	Pos      obj.MapPos
	Rotate   int
}

var missionMetadata = map[Mission]MissionMetadata{
	MissionDefault: {
		Name:          "默认关卡",
		InitCameraPos: obj.MapPos{MX: 32, MY: 32},
		MapCfg:        mapcfg.GetByName(mapcfg.MapDefault),
		InitShips: []InitShipMetadata{
			{obj.ShipDefault, obj.MapPos{MX: 32, MY: 32}, 90},
		},
	},
}
