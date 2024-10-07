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

type rawMissionMetadata struct {
	Name                string                          `json:"name"`
	DisplayName         string                          `json:"displayName"`
	InitFunds           int64                           `json:"initFunds"`
	InitCameraPos       [2]int                          `json:"initCameraPos"`
	MapName             string                          `json:"mapName"`
	MaxShipCount        int                             `json:"maxShipCount"`
	Description         []string                        `json:"description"`
	InitShips           []rawInitShipMetadata           `json:"initShips"`
	InitReinforcePoints []rawInitReinforcePointMetadata `json:"initReinforcePoints"`
	InitOilPlatforms    []rawInitOilPlatformMetadata    `json:"initOilPlatforms"`
}

type rawInitShipMetadata struct {
	Name         string `json:"name"`
	Pos          [2]int `json:"pos"`
	Rotation     int    `json:"rotation"`
	BelongPlayer string `json:"belongPlayer"`
}

type rawInitReinforcePointMetadata struct {
	Pos               [2]int   `json:"pos"`
	Rotation          int      `json:"rotation"`
	RallyPos          [2]int   `json:"rallyPos"`
	BelongPlayer      string   `json:"belongPlayer"`
	MaxOncomingShip   int      `json:"maxOncomingShip"`
	ProvidedShipNames []string `json:"providedShipNames"`
}

type rawInitOilPlatformMetadata struct {
	Pos    [2]int `json:"pos"`
	Radius int    `json:"radius"`
	Yield  int    `json:"yield"`
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
		// 战舰
		initShips := []InitShipMetadata{}
		for _, shipMD := range md.InitShips {
			initShips = append(initShips, InitShipMetadata{
				ShipName:     shipMD.Name,
				Pos:          obj.NewMapPos(shipMD.Pos[0], shipMD.Pos[1]),
				Rotation:     float64(shipMD.Rotation),
				BelongPlayer: faction.Player(shipMD.BelongPlayer),
			})
		}
		// 增援点
		initReinforcePoints := []InitReinforcePointMetadata{}
		for _, rpMD := range md.InitReinforcePoints {
			initReinforcePoints = append(initReinforcePoints, InitReinforcePointMetadata{
				Pos:               obj.NewMapPos(rpMD.Pos[0], rpMD.Pos[1]),
				Rotation:          float64(rpMD.Rotation),
				RallyPos:          obj.NewMapPos(rpMD.RallyPos[0], rpMD.RallyPos[1]),
				BelongPlayer:      faction.Player(rpMD.BelongPlayer),
				MaxOncomingShip:   rpMD.MaxOncomingShip,
				ProvidedShipNames: rpMD.ProvidedShipNames,
			})
		}
		// 油井
		initOilPlatforms := []InitOilPlatformMetadata{}
		for _, opMD := range md.InitOilPlatforms {
			initOilPlatforms = append(initOilPlatforms, InitOilPlatformMetadata{
				Pos:    obj.NewMapPos(opMD.Pos[0], opMD.Pos[1]),
				Radius: opMD.Radius,
				Yield:  opMD.Yield,
			})
		}
		// 元数据
		missionMetadata[md.Name] = MissionMetadata{
			Name:                md.Name,
			DisplayName:         md.DisplayName,
			MaxShipCount:        md.MaxShipCount,
			InitFunds:           md.InitFunds,
			InitCameraPos:       obj.NewMapPos(md.InitCameraPos[0], md.InitCameraPos[1]),
			MapCfg:              mapcfg.GetByName(md.MapName),
			Description:         md.Description,
			InitShips:           initShips,
			InitReinforcePoints: initReinforcePoints,
			InitOilPlatforms:    initOilPlatforms,
		}
	}
}
