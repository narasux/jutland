package metadata

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/yosuke-furukawa/json5/encoding/json5"

	"github.com/narasux/jutland/pkg/config"
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
	ShipName     string
	Pos          obj.MapPos
	Rotation     float64
	BelongPlayer faction.Player
}

var missionMetadata map[string]MissionMetadata

// Get 获取任务元配置
func Get(mission string) MissionMetadata {
	return missionMetadata[mission]
}

type rawInitShipMetadata struct {
	Name         string `json:"name"`
	Pos          []int  `json:"pos"`
	Rotation     int    `json:"rotation"`
	BelongPlayer string `json:"belongPlayer"`
}

type rawMissionMetadata struct {
	Name          string                `json:"name"`
	InitCameraPos [2]int                `json:"initCameraPos"`
	MapName       string                `json:"mapName"`
	MaxShipCount  int                   `json:"maxShipCount"`
	InitShips     []rawInitShipMetadata `json:"initShips"`
}

func init() {
	file, err := os.Open(filepath.Join(config.ConfigBaseDir, "missions.json5"))
	if err != nil {
		log.Fatal("failed to open missions.json5: ", err)
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)

	var misMDs []rawMissionMetadata
	if err = json5.Unmarshal(bytes, &misMDs); err != nil {
		log.Fatal("failed to unmarshal missions.json5: ", err)
	}

	missionMetadata = make(map[string]MissionMetadata)

	for _, md := range misMDs {
		initShips := []InitShipMetadata{}
		for _, shipMD := range md.InitShips {
			initShips = append(initShips, InitShipMetadata{
				ShipName:     shipMD.Name,
				Pos:          obj.NewMapPos(shipMD.Pos[0], shipMD.Pos[1]),
				Rotation:     float64(shipMD.Rotation),
				BelongPlayer: faction.Player(shipMD.BelongPlayer),
			})
		}
		missionMetadata[md.Name] = MissionMetadata{
			Name:          md.Name,
			InitCameraPos: obj.NewMapPos(md.InitCameraPos[0], md.InitCameraPos[1]),
			MapCfg:        mapcfg.GetByName(md.MapName),
			MaxShipCount:  md.MaxShipCount,
			InitShips:     initShips,
		}
	}
}
