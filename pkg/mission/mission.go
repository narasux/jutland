package mission

import "github.com/narasux/jutland/pkg/resources/mapcfg"

// 任务元配置
type MissionMetadata struct {
	Name    string
	InitPos MapPos
	MapCfg  *mapcfg.MapCfg
}

var missionMetadata = map[Mission]MissionMetadata{
	MissionDefault: {
		Name:    "默认关卡",
		InitPos: MapPos{MX: 32, MY: 32},
		MapCfg:  mapcfg.GetByName(mapcfg.MapDefault),
	},
}
