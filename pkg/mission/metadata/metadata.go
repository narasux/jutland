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
	// 初始位置
	InitCameraPos obj.MapPos
	// 初始战舰
	InitShips []InitShipMetadata
}

type InitShipMetadata struct {
	ShipName     obj.ShipName
	Pos          obj.MapPos
	Rotation     float64
	BelongPlayer faction.Player
}

var missionMetadata = map[Mission]MissionMetadata{
	MissionDefault: {
		Name:          "默认关卡",
		InitCameraPos: obj.NewMapPos(30, 30),
		MapCfg:        mapcfg.GetByName(mapcfg.MapDefault),
		MaxShipCount:  5,
		InitShips: []InitShipMetadata{
			// 己方舰队
			{obj.ShipDefault, obj.NewMapPos(40, 33), 90, faction.HumanAlpha},
			{obj.ShipDefault, obj.NewMapPos(42, 35), 90, faction.HumanAlpha},
			{obj.ShipDefault, obj.NewMapPos(40, 48), 90, faction.HumanAlpha},
			{obj.ShipDefault, obj.NewMapPos(42, 50), 90, faction.HumanAlpha},
			// 敌人舰队
			{obj.ShipDefault, obj.NewMapPos(70, 35), 90, faction.ComputerAlpha},
			{obj.ShipDefault, obj.NewMapPos(65, 42), 215, faction.ComputerAlpha},
			{obj.ShipDefault, obj.NewMapPos(70, 50), 270, faction.ComputerAlpha},
		},
	},
}

// Get 获取任务元配置
func Get(mission Mission) MissionMetadata {
	return missionMetadata[mission]
}
